package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeUpstreamErrorMessage_PreservesOfficialAndCustomRedaction(t *testing.T) {
	message := "request https://example.com/v1?key=secret-query failed Authorization: Bearer secret-token-value"

	sanitized := sanitizeUpstreamErrorMessage(message)

	require.Contains(t, sanitized, "key=***")
	require.Contains(t, strings.ToLower(sanitized), "authorization: ***")
	require.NotContains(t, sanitized, "secret-query")
	require.NotContains(t, sanitized, "secret-token-value")
}
