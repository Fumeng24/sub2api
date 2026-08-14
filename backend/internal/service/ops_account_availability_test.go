package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAccountAvailabilityFlags_UsesSchedulingBlockReason(t *testing.T) {
	now := time.Now()
	future := now.Add(10 * time.Minute)
	past := now.Add(-10 * time.Minute)

	tests := []struct {
		name                  string
		account               *Account
		wantBlockReason       string
		wantAvailable         bool
		wantRateLimited       bool
		wantOverloaded        bool
		wantTempUnschedulable bool
		wantError             bool
	}{
		{
			name:          "available",
			account:       &Account{Status: StatusActive, Schedulable: true},
			wantAvailable: true,
		},
		{
			name:            "status_error",
			account:         &Account{Status: StatusError, Schedulable: true},
			wantBlockReason: AccountSchedulingBlockInactive.String(),
			wantError:       true,
		},
		{
			name:            "expired",
			account:         &Account{Status: StatusActive, Schedulable: true, AutoPauseOnExpired: true, ExpiresAt: &past},
			wantBlockReason: AccountSchedulingBlockExpired.String(),
		},
		{
			name:            "quota_exceeded",
			account:         &Account{Status: StatusActive, Schedulable: true, Type: AccountTypeAPIKey, Extra: map[string]any{"quota_limit": 1.0, "quota_used": 1.0}},
			wantBlockReason: AccountSchedulingBlockQuotaExceeded.String(),
		},
		{
			name:            "rate_limited",
			account:         &Account{Status: StatusActive, Schedulable: true, RateLimitResetAt: &future},
			wantBlockReason: AccountSchedulingBlockRateLimited.String(),
			wantRateLimited: true,
		},
		{
			name:            "overloaded",
			account:         &Account{Status: StatusActive, Schedulable: true, OverloadUntil: &future},
			wantBlockReason: AccountSchedulingBlockOverloaded.String(),
			wantOverloaded:  true,
		},
		{
			name:                  "temp_unschedulable",
			account:               &Account{Status: StatusActive, Schedulable: true, TempUnschedulableUntil: &future},
			wantBlockReason:       AccountSchedulingBlockTempUnschedulable.String(),
			wantTempUnschedulable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := accountAvailabilityFlags(tt.account, now)
			require.Equal(t, tt.wantBlockReason, got.BlockReason)
			require.Equal(t, tt.wantAvailable, got.IsAvailable)
			require.Equal(t, tt.wantRateLimited, got.IsRateLimited)
			require.Equal(t, tt.wantOverloaded, got.IsOverloaded)
			require.Equal(t, tt.wantTempUnschedulable, got.IsTempUnschedulable)
			require.Equal(t, tt.wantError, got.HasError)
		})
	}
}
