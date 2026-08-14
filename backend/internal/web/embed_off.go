//go:build !embed

// Package web provides frontend assets for the application.
package web

import (
	"context"

	"github.com/gin-gonic/gin"
)

// PublicSettingsProvider is an interface to fetch public settings.
type PublicSettingsProvider interface {
	GetPublicSettingsForInjection(ctx context.Context) (any, error)
}

// FrontendServer serves the frontend in non-embed builds.
type FrontendServer = externalFrontendServer

// NewFrontendServer creates a frontend server backed by an external dist dir.
func NewFrontendServer(settingsProvider PublicSettingsProvider) (*FrontendServer, error) {
	return newExternalFrontendServer(settingsProvider)
}

// ServeEmbeddedFrontend keeps the legacy middleware name for setup mode.
func ServeEmbeddedFrontend() gin.HandlerFunc {
	return serveExternalFrontend()
}

// HasEmbeddedFrontend reports whether external frontend assets are available.
func HasEmbeddedFrontend() bool {
	return hasExternalFrontend()
}
