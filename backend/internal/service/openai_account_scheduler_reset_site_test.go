package service

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestOpenAIQuotaHeadroomFactor_LegacyWindowKeysRemainCompatible(t *testing.T) {
	now := time.Now()
	account := &Account{Extra: map[string]any{
		"codex_usage_updated_at": now.Format(time.RFC3339),
		"codex_7d_used_percent":  40.0,
		"codex_5h_used_percent":  95.0,
		"codex_5h_reset_at":      now.Add(time.Hour).Format(time.RFC3339),
	}}

	require.Equal(t, 0.3, openAIQuotaHeadroomFactor(account, now))
}

func TestOpenAIQuotaHeadroomFactor_StaleSnapshotIsNeutral(t *testing.T) {
	now := time.Now()
	account := &Account{Extra: map[string]any{
		"codex_usage_updated_at": now.Add(-9 * time.Hour).Format(time.RFC3339),
		"codex_7d_used_percent":  99.0,
	}}

	require.Equal(t, openAIQuotaHeadroomNeutralFactor, openAIQuotaHeadroomFactor(account, now))
}
