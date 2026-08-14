package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAccountFromServiceShallow_RedactsSiteSensitiveCredentialsAndExtra(t *testing.T) {
	src := &service.Account{
		ID:       1,
		Name:     "test",
		Platform: "anthropic",
		Type:     "oauth",
		Credentials: map[string]any{
			"upstream_sub2api_password": "sub2api-secret",
			"base_url":                  "https://api.example.com",
		},
		Extra: map[string]any{
			"access_token": "extra-secret",
			"safe":         "value",
		},
	}

	got := AccountFromServiceShallow(src)
	require.NotNil(t, got)
	require.NotContains(t, got.Credentials, "upstream_sub2api_password")
	require.True(t, got.CredentialsStatus["has_upstream_sub2api_password"])
	require.Equal(t, "https://api.example.com", got.Credentials["base_url"])
	require.NotContains(t, got.Extra, "access_token")
	require.Equal(t, "value", got.Extra["safe"])

	raw, err := json.Marshal(got)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "sub2api-secret")
	require.NotContains(t, string(raw), "extra-secret")
	require.Equal(t, "sub2api-secret", src.Credentials["upstream_sub2api_password"])
	require.Equal(t, "extra-secret", src.Extra["access_token"])
}

func TestAccountFromServiceShallow_TempUnschedulableReasonIsCurrentAndDisplaySafe(t *testing.T) {
	future := time.Now().Add(time.Minute)
	src := &service.Account{
		ID:                      1,
		Name:                    "openai",
		Platform:                service.PlatformOpenAI,
		Type:                    service.AccountTypeAPIKey,
		TempUnschedulableUntil:  &future,
		TempUnschedulableReason: `{"status_code":0,"matched_keyword":"openai_stream_error","error_message":"stream failed"}`,
	}

	got := AccountFromServiceShallow(src)
	require.NotNil(t, got)
	require.NotNil(t, got.TempUnschedulableUntil)
	require.Equal(t, "openai_stream_error", got.TempUnschedulableReason)
	require.NotNil(t, got.TempUnschedulableStatusCode)
	require.Equal(t, 0, *got.TempUnschedulableStatusCode)
}

func TestAccountFromServiceShallow_ExpiredTempUnschedulableReasonIsHidden(t *testing.T) {
	past := time.Now().Add(-time.Minute)
	src := &service.Account{
		ID:                      1,
		Name:                    "openai",
		Platform:                service.PlatformOpenAI,
		Type:                    service.AccountTypeAPIKey,
		TempUnschedulableUntil:  &past,
		TempUnschedulableReason: "legacy token refresh failed",
	}

	got := AccountFromServiceShallow(src)
	require.NotNil(t, got)
	require.Nil(t, got.TempUnschedulableUntil)
	require.Empty(t, got.TempUnschedulableReason)
	require.Nil(t, got.TempUnschedulableStatusCode)
}
