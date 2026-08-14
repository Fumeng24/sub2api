//go:build embed

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectSiteTitleCustom(t *testing.T) {
	html := []byte(`<html><head><title>低成本大模型 API 聚合平台 - GPT Claude Gemini Codex API - Wegoo AI</title></head><body></body></html>`)
	settingsJSON := []byte(`{"site_name":"MyCustomSite"}`)

	result := injectSiteTitle(html, settingsJSON)

	assert.Equal(t, string(html), string(result))
	assert.NotContains(t, string(result), "MyCustomSite - AI API Gateway")
}

func TestInjectSiteTitleCustomReplacesWegooFallback(t *testing.T) {
	html := []byte(`<html><head><title>Wegoo's API - AI API Gateway</title></head><body></body></html>`)
	settingsJSON := []byte(`{"site_name":"MyCustomSite"}`)

	result := injectSiteTitle(html, settingsJSON)

	assert.Contains(t, string(result), "<title>MyCustomSite - AI API Gateway</title>")
	assert.NotContains(t, string(result), "Wegoo's API")
}

func TestFrontendServerMiddlewareCustom(t *testing.T) {
	t.Run("serves_legacy_vite_entry_as_javascript_shim", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)
		require.NotEmpty(t, server.entryScriptPath)

		router := gin.New()
		router.Use(server.Middleware())

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/src/main.ts", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/javascript")
		assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		assert.Contains(t, w.Body.String(), `import "`)
		assert.Contains(t, w.Body.String(), server.entryScriptPath)
		assert.NotContains(t, w.Body.String(), "<!doctype html>")
	})

	t.Run("serves_stale_index_asset_as_current_entry_shim", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		router.Use(server.Middleware())

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/index-stalehash.js", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/javascript")
		assert.Contains(t, w.Body.String(), server.entryScriptPath)
		assert.NotContains(t, w.Body.String(), "<!doctype html>")
	})

	t.Run("returns_404_for_missing_static_assets", func(t *testing.T) {
		provider := &mockSettingsProvider{
			settings: map[string]string{"test": "value"},
		}

		server, err := NewFrontendServer(provider)
		require.NoError(t, err)

		router := gin.New()
		router.Use(server.Middleware())

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/missing.css", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NotContains(t, w.Body.String(), "<!doctype html>")
	})
}

func TestServeEmbeddedFrontendCustom(t *testing.T) {
	t.Run("serves_legacy_vite_entry_as_javascript_shim", func(t *testing.T) {
		middleware := ServeEmbeddedFrontend()

		router := gin.New()
		router.Use(middleware)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/src/main.ts", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/javascript")
		assert.Contains(t, w.Body.String(), `import "/assets/index-`)
		assert.NotContains(t, w.Body.String(), "<!doctype html>")
	})

	t.Run("returns_404_for_missing_static_assets", func(t *testing.T) {
		middleware := ServeEmbeddedFrontend()

		router := gin.New()
		router.Use(middleware)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NotContains(t, w.Body.String(), "<!doctype html>")
	})
}
