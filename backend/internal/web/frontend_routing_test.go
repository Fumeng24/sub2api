package web

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldBypassEmbeddedFrontendRequest(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   bool
	}{
		{name: "spa dashboard", method: http.MethodGet, path: "/dashboard", want: false},
		{name: "spa usage", method: http.MethodGet, path: "/usage", want: false},
		{name: "spa messages", method: http.MethodGet, path: "/messages", want: false},
		{name: "spa public pricing", method: http.MethodGet, path: "/pricing", want: false},
		{name: "versioned API", method: http.MethodGet, path: "/v1/models", want: true},
		{name: "bare models API", method: http.MethodGet, path: "/models", want: true},
		{name: "bare responses websocket", method: http.MethodGet, path: "/responses", want: true},
		{name: "bare live sideband", method: http.MethodGet, path: "/live/call_123", want: true},
		{name: "bare billing API", method: http.MethodGet, path: "/sub2api/billing", want: true},
		{name: "bare chat API", method: http.MethodPost, path: "/chat/completions", want: true},
		{name: "bare messages API", method: http.MethodPost, path: "/messages", want: true},
		{name: "bare embeddings API", method: http.MethodPost, path: "/embeddings", want: true},
		{name: "unknown POST is not SPA", method: http.MethodPost, path: "/unknown", want: true},
		{name: "unknown DELETE is not SPA", method: http.MethodDelete, path: "/unknown", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, shouldBypassEmbeddedFrontendRequest(tt.method, tt.path))
		})
	}
}
