package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeOpenAIWSFallbackClientMessageCustom(t *testing.T) {
	message := "upstream request failed at https://private.example/v1 with Bearer secret-token-value"

	sanitized := sanitizeOpenAIWSFallbackClientMessageCustom(http.StatusBadGateway, "upstream_error", message)

	require.Equal(t, ClientFacingTemporaryUnavailableMessage(), sanitized)
}
