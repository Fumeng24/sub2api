//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGatewayRequest_ClonesEscapingScalarStrings(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-6","metadata":{"user_id":"session-123"},"output_config":{"effort":"high"},"messages":[{"role":"user","content":"hi"}]}`)
	parsed, err := ParseGatewayRequest(NewRequestBodyRef(body), "")
	require.NoError(t, err)

	copy(body, []byte(`{"model":"corrupted-body-reuse","metadata":{"user_id":"changed"},"output_config":{"effort":"low"}}`))

	require.Equal(t, "claude-sonnet-4-6", parsed.Model)
	require.Equal(t, "session-123", parsed.MetadataUserID)
	require.Equal(t, "high", parsed.OutputEffort)
}
