//go:build embed

package web

import (
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type frontendServerCustom struct {
	entryScriptPath string
	stylesheetPaths []string
}

func newFrontendServerCustom(baseHTML []byte) frontendServerCustom {
	return frontendServerCustom{
		entryScriptPath: extractFrontendEntryScript(baseHTML),
		stylesheetPaths: extractFrontendStylesheets(baseHTML),
	}
}

func serveMissingFrontendPathCustom(c *gin.Context, cleanPath string, custom frontendServerCustom) bool {
	if isFrontendEntryRequest(cleanPath) {
		serveFrontendEntryShim(c, custom.entryScriptPath, custom.stylesheetPaths)
		return true
	}
	if isFrontendAssetRequest(cleanPath) {
		c.String(http.StatusNotFound, "Frontend asset not found")
		c.Abort()
		return true
	}
	return false
}

func shouldReplaceEmbeddedSiteTitleCustom(html []byte, titleStart, titleEnd int) bool {
	existingTitle := strings.TrimSpace(string(html[titleStart+len("<title>") : titleEnd]))
	return strings.HasPrefix(existingTitle, "Sub2API") || strings.HasPrefix(existingTitle, "Wegoo's API")
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

func readFrontendBaseHTML(fsys fs.FS) []byte {
	file, err := fsys.Open("index.html")
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil
	}
	return content
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

	var script strings.Builder
	for _, stylesheetPath := range stylesheetPaths {
		script.WriteString("if(!document.querySelector('link[href=\"")
		script.WriteString(jsStringLiteralContent(stylesheetPath))
		script.WriteString("\"]')){const l=document.createElement('link');l.rel='stylesheet';l.href=")
		script.WriteString(strconv.Quote(stylesheetPath))
		script.WriteString(";document.head.appendChild(l);}\n")
	}
	script.WriteString("import ")
	script.WriteString(strconv.Quote(entryScriptPath))
	script.WriteString(";\n")

	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/javascript; charset=utf-8", []byte(script.String()))
	c.Abort()
}

func jsStringLiteralContent(value string) string {
	quoted := strconv.Quote(value)
	return quoted[1 : len(quoted)-1]
}
