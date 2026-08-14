//go:build !embed

package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

const (
	// NonceHTMLPlaceholder is the placeholder for nonce in HTML script tags.
	NonceHTMLPlaceholder = "__CSP_NONCE_VALUE__"
	frontendDistEnv      = "SUB2API_FRONTEND_DIST"
)

// externalFrontendServer serves the external frontend with settings injection.
type externalFrontendServer struct {
	distFS          fs.FS
	fileServer      http.Handler
	baseHTML        []byte
	entryScriptPath string
	stylesheetPaths []string
	cache           *HTMLCache
	settings        PublicSettingsProvider
	overrideDir     string
}

func newExternalFrontendServer(settingsProvider PublicSettingsProvider) (*externalFrontendServer, error) {
	distDir := externalFrontendDistDir()
	if distDir == "" {
		return nil, errors.New("frontend dist not found")
	}

	distFS := os.DirFS(distDir)
	baseHTML, err := readFrontendBaseHTML(distFS)
	if err != nil {
		return nil, err
	}

	cache := NewHTMLCache()
	cache.SetBaseHTML(baseHTML)

	return &externalFrontendServer{
		distFS:          distFS,
		fileServer:      http.FileServer(http.Dir(distDir)),
		baseHTML:        baseHTML,
		entryScriptPath: extractFrontendEntryScript(baseHTML),
		stylesheetPaths: extractFrontendStylesheets(baseHTML),
		cache:           cache,
		settings:        settingsProvider,
		overrideDir:     filepath.Join("data", "public"),
	}, nil
}

// InvalidateCache invalidates the HTML cache.
func (s *externalFrontendServer) InvalidateCache() {
	if s != nil && s.cache != nil {
		s.cache.Invalidate()
	}
}

// Middleware returns the Gin middleware handler.
func (s *externalFrontendServer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestPath := c.Request.URL.Path
		if shouldBypassEmbeddedFrontendRequest(c.Request.Method, requestPath) {
			c.Next()
			return
		}

		cleanPath := cleanFrontendPath(requestPath)
		if cleanPath == "" || cleanPath == "index.html" {
			s.serveIndexHTML(c)
			return
		}

		if !s.fileExists(cleanPath) {
			if isFrontendEntryRequest(cleanPath) {
				serveFrontendEntryShim(c, s.entryScriptPath, s.stylesheetPaths)
				return
			}
			if isFrontendAssetRequest(cleanPath) {
				c.String(http.StatusNotFound, "Frontend asset not found")
				c.Abort()
				return
			}
			s.serveIndexHTML(c)
			return
		}

		if s.tryServeOverride(c, cleanPath) {
			return
		}

		c.Request.URL.Path = "/" + cleanPath
		s.fileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

func (s *externalFrontendServer) fileExists(name string) bool {
	if !fs.ValidPath(name) {
		return false
	}
	file, err := s.distFS.Open(name)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

func (s *externalFrontendServer) tryServeOverride(c *gin.Context, cleanPath string) bool {
	if s.overrideDir == "" {
		return false
	}
	filePath := filepath.Join(s.overrideDir, filepath.Clean("/"+cleanPath))
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}
	c.File(filePath)
	c.Abort()
	return true
}

func (s *externalFrontendServer) serveIndexHTML(c *gin.Context) {
	nonce := middleware.GetNonceFromContext(c)

	if cached := s.cache.Get(); cached != nil {
		if match := c.GetHeader("If-None-Match"); match == cached.ETag {
			c.Status(http.StatusNotModified)
			c.Abort()
			return
		}

		content := replaceNoncePlaceholder(cached.Content, nonce)
		c.Header("ETag", cached.ETag)
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
		c.Abort()
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	settings, err := s.settings.GetPublicSettingsForInjection(ctx)
	if err != nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", s.baseHTML)
		c.Abort()
		return
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", s.baseHTML)
		c.Abort()
		return
	}

	rendered := s.injectSettings(settingsJSON)
	s.cache.Set(rendered, settingsJSON)

	content := replaceNoncePlaceholder(rendered, nonce)
	if cached := s.cache.Get(); cached != nil {
		c.Header("ETag", cached.ETag)
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	c.Abort()
}

func (s *externalFrontendServer) injectSettings(settingsJSON []byte) []byte {
	script := []byte(`<script nonce="` + NonceHTMLPlaceholder + `">window.__APP_CONFIG__=` + string(settingsJSON) + `;</script>`)
	headClose := []byte("</head>")
	result := bytes.Replace(s.baseHTML, headClose, append(script, headClose...), 1)
	return injectSiteTitle(result, settingsJSON)
}

func injectSiteTitle(html, settingsJSON []byte) []byte {
	var cfg struct {
		SiteName string `json:"site_name"`
	}
	if err := json.Unmarshal(settingsJSON, &cfg); err != nil || cfg.SiteName == "" {
		return html
	}

	titleStart := bytes.Index(html, []byte("<title>"))
	titleEnd := bytes.Index(html, []byte("</title>"))
	if titleStart == -1 || titleEnd == -1 || titleEnd <= titleStart {
		return html
	}

	existingTitle := strings.TrimSpace(string(html[titleStart+len("<title>") : titleEnd]))
	if !isReplaceableDefaultSiteTitle(existingTitle) {
		return html
	}

	newTitle := []byte("<title>" + cfg.SiteName + " - AI API Gateway</title>")
	result := make([]byte, 0, len(html)-titleEnd+titleStart+len(newTitle))
	result = append(result, html[:titleStart]...)
	result = append(result, newTitle...)
	result = append(result, html[titleEnd+len("</title>"):]...)
	return result
}

func isReplaceableDefaultSiteTitle(title string) bool {
	return strings.HasPrefix(title, "Sub2API") || strings.HasPrefix(title, "Wegoo's API")
}

func replaceNoncePlaceholder(html []byte, nonce string) []byte {
	return bytes.ReplaceAll(html, []byte(NonceHTMLPlaceholder), []byte(nonce))
}

func serveExternalFrontend() gin.HandlerFunc {
	distDir := externalFrontendDistDir()
	if distDir == "" {
		return func(c *gin.Context) {
			if shouldBypassEmbeddedFrontendRequest(c.Request.Method, c.Request.URL.Path) {
				c.Next()
				return
			}
			c.String(http.StatusNotFound, "Frontend assets not found. Set SUB2API_FRONTEND_DIST or place files in frontend/dist.")
			c.Abort()
		}
	}

	distFS := os.DirFS(distDir)
	fileServer := http.FileServer(http.Dir(distDir))
	overrideDir := filepath.Join("data", "public")
	baseHTML, _ := readFrontendBaseHTML(distFS)
	entryScriptPath := extractFrontendEntryScript(baseHTML)
	stylesheetPaths := extractFrontendStylesheets(baseHTML)

	return func(c *gin.Context) {
		requestPath := c.Request.URL.Path
		if shouldBypassEmbeddedFrontendRequest(c.Request.Method, requestPath) {
			c.Next()
			return
		}

		cleanPath := cleanFrontendPath(requestPath)
		if cleanPath == "" || cleanPath == "index.html" {
			serveIndexHTML(c, distFS)
			return
		}

		if file, err := distFS.Open(cleanPath); err == nil {
			_ = file.Close()
			if tryServeOverrideFile(c, overrideDir, cleanPath) {
				return
			}
			c.Request.URL.Path = "/" + cleanPath
			fileServer.ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}

		if isFrontendEntryRequest(cleanPath) {
			serveFrontendEntryShim(c, entryScriptPath, stylesheetPaths)
			return
		}

		if isFrontendAssetRequest(cleanPath) {
			c.String(http.StatusNotFound, "Frontend asset not found")
			c.Abort()
			return
		}

		serveIndexHTML(c, distFS)
	}
}

func tryServeOverrideFile(c *gin.Context, overrideDir, cleanPath string) bool {
	if overrideDir == "" {
		return false
	}
	filePath := filepath.Join(overrideDir, filepath.Clean("/"+cleanPath))
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}
	c.File(filePath)
	c.Abort()
	return true
}

var frontendEntryScriptPattern = regexp.MustCompile(`(?i)<script\b[^>]*\bsrc=["']([^"']*/assets/index-[^"']+\.js)["'][^>]*>`)
var frontendStylesheetPattern = regexp.MustCompile(`(?i)<link\b[^>]*\brel=["']stylesheet["'][^>]*\bhref=["']([^"']+\.css)["'][^>]*>`)

func extractFrontendEntryScript(html []byte) string {
	matches := frontendEntryScriptPattern.FindSubmatch(html)
	if len(matches) < 2 {
		return ""
	}
	return string(matches[1])
}

func extractFrontendStylesheets(html []byte) []string {
	matches := frontendStylesheetPattern.FindAllSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}

	paths := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		path := string(match[1])
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

func readFrontendBaseHTML(fsys fs.FS) ([]byte, error) {
	file, err := fsys.Open("index.html")
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func isFrontendEntryRequest(cleanPath string) bool {
	return cleanPath == "src/main.ts" ||
		(strings.HasPrefix(cleanPath, "assets/index-") && strings.HasSuffix(cleanPath, ".js"))
}

func isFrontendAssetRequest(cleanPath string) bool {
	if strings.HasPrefix(cleanPath, "assets/") || strings.HasPrefix(cleanPath, "src/") {
		return true
	}

	switch cleanPath {
	case "favicon.ico", "logo.png", "robots.txt", "manifest.webmanifest", "site.webmanifest":
		return true
	}

	switch strings.ToLower(filepath.Ext(cleanPath)) {
	case ".js", ".mjs", ".css", ".map", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico", ".woff", ".woff2", ".ttf", ".json", ".txt", ".xml", ".webmanifest", ".ts":
		return true
	default:
		return false
	}
}

func serveFrontendEntryShim(c *gin.Context, entryScriptPath string, stylesheetPaths []string) {
	if entryScriptPath == "" {
		c.String(http.StatusNotFound, "Frontend entry not found")
		c.Abort()
		return
	}

	scriptParts := make([]string, 0, len(stylesheetPaths)*5+3)
	for _, stylesheetPath := range stylesheetPaths {
		scriptParts = append(scriptParts,
			"if(!document.querySelector('link[href=\"",
			jsStringLiteralContent(stylesheetPath),
			"\"]')){const l=document.createElement('link');l.rel='stylesheet';l.href=",
			strconv.Quote(stylesheetPath),
			";document.head.appendChild(l);}\n",
		)
	}
	scriptParts = append(scriptParts, "import ", strconv.Quote(entryScriptPath), ";\n")

	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/javascript; charset=utf-8", []byte(strings.Join(scriptParts, "")))
	c.Abort()
}

func jsStringLiteralContent(value string) string {
	quoted := strconv.Quote(value)
	return quoted[1 : len(quoted)-1]
}

func serveIndexHTML(c *gin.Context, fsys fs.FS) {
	content, err := readFrontendBaseHTML(fsys)
	if err != nil {
		c.String(http.StatusNotFound, "Frontend not found")
		c.Abort()
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	c.Abort()
}

func hasExternalFrontend() bool {
	return externalFrontendDistDir() != ""
}

func externalFrontendDistDir() string {
	for _, candidate := range frontendDistCandidates() {
		if frontendDistExists(candidate) {
			return candidate
		}
	}
	return ""
}

func frontendDistCandidates() []string {
	candidates := make([]string, 0, 5)
	if envDir := strings.TrimSpace(os.Getenv(frontendDistEnv)); envDir != "" {
		candidates = append(candidates, envDir)
	}
	candidates = append(candidates,
		filepath.Join("frontend", "dist"),
		filepath.Join("backend", "internal", "web", "dist"),
		filepath.Join("web", "dist"),
		"dist",
		filepath.Join("/opt", "sub2api", "frontend", "dist"),
	)
	return candidates
}

func frontendDistExists(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "index.html"))
	return err == nil && !info.IsDir()
}

func cleanFrontendPath(requestPath string) string {
	clean := pathpkg.Clean("/" + strings.TrimPrefix(requestPath, "/"))
	if clean == "/" || clean == "." {
		return ""
	}
	return strings.TrimPrefix(clean, "/")
}
