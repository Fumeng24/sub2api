//go:build unit

package service

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAccountSchedulingBlockReasonAt(t *testing.T) {
	now := time.Now()
	future := now.Add(10 * time.Minute)
	past := now.Add(-10 * time.Minute)

	tests := []struct {
		name    string
		account *Account
		want    AccountSchedulingBlockReason
	}{
		{
			name:    "nil_account",
			account: nil,
			want:    AccountSchedulingBlockInactive,
		},
		{
			name:    "inactive",
			account: &Account{Status: StatusError, Schedulable: true},
			want:    AccountSchedulingBlockInactive,
		},
		{
			name:    "manual_unschedulable",
			account: &Account{Status: StatusActive, Schedulable: false},
			want:    AccountSchedulingBlockManual,
		},
		{
			name: "expired",
			account: &Account{
				Status:             StatusActive,
				Schedulable:        true,
				AutoPauseOnExpired: true,
				ExpiresAt:          &past,
			},
			want: AccountSchedulingBlockExpired,
		},
		{
			name: "overloaded",
			account: &Account{
				Status:        StatusActive,
				Schedulable:   true,
				OverloadUntil: &future,
			},
			want: AccountSchedulingBlockOverloaded,
		},
		{
			name: "rate_limited",
			account: &Account{
				Status:           StatusActive,
				Schedulable:      true,
				RateLimitResetAt: &future,
			},
			want: AccountSchedulingBlockRateLimited,
		},
		{
			name: "temp_unschedulable",
			account: &Account{
				Status:                 StatusActive,
				Schedulable:            true,
				TempUnschedulableUntil: &future,
			},
			want: AccountSchedulingBlockTempUnschedulable,
		},
		{
			name: "quota_exceeded",
			account: &Account{
				Status:      StatusActive,
				Schedulable: true,
				Type:        AccountTypeAPIKey,
				Extra: map[string]any{
					"quota_limit": 10.0,
					"quota_used":  10.0,
				},
			},
			want: AccountSchedulingBlockQuotaExceeded,
		},
		{
			name: "schedulable",
			account: &Account{
				Status:      StatusActive,
				Schedulable: true,
			},
			want: AccountSchedulingBlockNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.account.SchedulingBlockReasonAt(now))
			require.Equal(t, tt.want == AccountSchedulingBlockNone, tt.account.IsSchedulableAt(now))
		})
	}
}

func TestAccountSchedulingBlockReasonForGroupAt_IgnoresSchedulingConfigured(t *testing.T) {
	now := time.Now()
	groupID := int64(10)

	tests := []struct {
		name    string
		account *Account
		want    AccountSchedulingBlockReason
	}{
		{
			name: "configured_group",
			account: &Account{
				Status:        StatusActive,
				Schedulable:   true,
				AccountGroups: []AccountGroup{{GroupID: groupID, SchedulingConfigured: true}},
			},
			want: AccountSchedulingBlockNone,
		},
		{
			name: "legacy_unconfigured_group_still_uses_global_status",
			account: &Account{
				Status:        StatusActive,
				Schedulable:   true,
				AccountGroups: []AccountGroup{{GroupID: groupID, SchedulingConfigured: false}},
			},
			want: AccountSchedulingBlockNone,
		},
		{
			name: "legacy_snapshot_without_group_metadata",
			account: &Account{
				Status:      StatusActive,
				Schedulable: true,
			},
			want: AccountSchedulingBlockNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.account.SchedulingBlockReasonForGroupAt(groupID, now))
			require.Equal(t, tt.want == AccountSchedulingBlockNone, tt.account.IsSchedulableForGroupAt(groupID, now))
		})
	}
}

func TestAccountHardSchedulingBlockReasonAt(t *testing.T) {
	now := time.Now()
	future := now.Add(10 * time.Minute)
	past := now.Add(-10 * time.Minute)

	tests := []struct {
		name    string
		account *Account
		want    AccountSchedulingBlockReason
	}{
		{
			name:    "nil_account",
			account: nil,
			want:    AccountSchedulingBlockInactive,
		},
		{
			name:    "inactive",
			account: &Account{Status: StatusError, Schedulable: true},
			want:    AccountSchedulingBlockInactive,
		},
		{
			name:    "manual_unschedulable",
			account: &Account{Status: StatusActive, Schedulable: false},
			want:    AccountSchedulingBlockManual,
		},
		{
			name: "expired",
			account: &Account{
				Status:             StatusActive,
				Schedulable:        true,
				AutoPauseOnExpired: true,
				ExpiresAt:          &past,
			},
			want: AccountSchedulingBlockExpired,
		},
		{
			name: "quota_exceeded",
			account: &Account{
				Status:      StatusActive,
				Schedulable: true,
				Type:        AccountTypeAPIKey,
				Extra: map[string]any{
					"quota_limit": 10.0,
					"quota_used":  10.0,
				},
			},
			want: AccountSchedulingBlockQuotaExceeded,
		},
		{
			name: "hard_ok_with_runtime_windows",
			account: &Account{
				Status:                 StatusActive,
				Schedulable:            true,
				RateLimitResetAt:       &future,
				OverloadUntil:          &future,
				TempUnschedulableUntil: &future,
			},
			want: AccountSchedulingBlockNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.account.HardSchedulingBlockReasonAt(now))
		})
	}
}

func TestAccountSchedulingSchedulerState(t *testing.T) {
	tests := []struct {
		name   string
		reason AccountSchedulingBlockReason
		want   string
	}{
		{name: "none", reason: AccountSchedulingBlockNone, want: "active"},
		{name: "inactive", reason: AccountSchedulingBlockInactive, want: "error"},
		{name: "manual", reason: AccountSchedulingBlockManual, want: "stopped"},
		{name: "expired", reason: AccountSchedulingBlockExpired, want: "expired"},
		{name: "overloaded", reason: AccountSchedulingBlockOverloaded, want: "overloaded"},
		{name: "rate_limited", reason: AccountSchedulingBlockRateLimited, want: "rate_limited"},
		{name: "temp_unschedulable", reason: AccountSchedulingBlockTempUnschedulable, want: "temp_unschedulable"},
		{name: "quota_exceeded", reason: AccountSchedulingBlockQuotaExceeded, want: "quota_exceeded"},
		{name: "unknown_reason", reason: AccountSchedulingBlockReason("new_reason"), want: "active"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.reason.SchedulerState())
		})
	}
}

func TestAccountSchedulerStateAt(t *testing.T) {
	now := time.Now()
	future := now.Add(10 * time.Minute)

	tests := []struct {
		name    string
		account *Account
		want    string
	}{
		{name: "nil_account", account: nil, want: "unknown"},
		{name: "schedulable", account: &Account{Status: StatusActive, Schedulable: true}, want: "active"},
		{
			name: "rate_limited",
			account: &Account{
				Status:           StatusActive,
				Schedulable:      true,
				RateLimitResetAt: &future,
			},
			want: "rate_limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.account.SchedulerStateAt(now))
		})
	}
}

func TestAccountSchedulabilityClassAt(t *testing.T) {
	now := time.Now()
	future := now.Add(10 * time.Minute)

	tests := []struct {
		name            string
		account         *Account
		wantReason      AccountSchedulingBlockReason
		wantSchedulable bool
		wantStatusError bool
		wantTempLimited bool
		wantRateLimited bool
		wantOverloaded  bool
		wantTempUnsched bool
	}{
		{
			name:            "schedulable",
			account:         &Account{Status: StatusActive, Schedulable: true},
			wantSchedulable: true,
		},
		{
			name:            "status_error",
			account:         &Account{Status: StatusError, Schedulable: true},
			wantReason:      AccountSchedulingBlockInactive,
			wantStatusError: true,
		},
		{
			name:            "rate_limited",
			account:         &Account{Status: StatusActive, Schedulable: true, RateLimitResetAt: &future},
			wantReason:      AccountSchedulingBlockRateLimited,
			wantTempLimited: true,
			wantRateLimited: true,
		},
		{
			name:            "overloaded",
			account:         &Account{Status: StatusActive, Schedulable: true, OverloadUntil: &future},
			wantReason:      AccountSchedulingBlockOverloaded,
			wantTempLimited: true,
			wantOverloaded:  true,
		},
		{
			name:            "temp_unschedulable",
			account:         &Account{Status: StatusActive, Schedulable: true, TempUnschedulableUntil: &future},
			wantReason:      AccountSchedulingBlockTempUnschedulable,
			wantTempLimited: true,
			wantTempUnsched: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.account.SchedulabilityClassAt(now)
			require.Equal(t, tt.wantReason, got.Reason)
			require.Equal(t, tt.wantSchedulable, got.Schedulable)
			require.Equal(t, tt.wantStatusError, got.StatusError)
			require.Equal(t, tt.wantTempLimited, got.TemporarilyLimited)
			require.Equal(t, tt.wantRateLimited, got.RateLimited)
			require.Equal(t, tt.wantOverloaded, got.Overloaded)
			require.Equal(t, tt.wantTempUnsched, got.TempUnschedulable)
		})
	}
}
