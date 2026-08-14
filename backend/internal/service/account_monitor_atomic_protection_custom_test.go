//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAccountMonitorFailureUsesAccountProtection(t *testing.T) {
	blocker := &accountMonitorFailureBlockerStub{}
	svc := &AccountMonitorService{blocker: blocker}
	account := &Account{ID: 38800, Platform: PlatformAnthropic, GroupIDs: []int64{20}}
	until := time.Now().Add(10 * time.Minute)

	applied, skipped := svc.applyMonitorFailureBlock(
		context.Background(),
		account,
		until,
		"account_monitor_consecutive_failures: upstream HTTP 503",
		&AccountMonitor{ID: 1, Model: "claude-opus-4-8"},
		&AccountMonitorCheck{Status: MonitorStatusError, Message: "upstream HTTP 503"},
		accountMonitorFailureClassification{Reason: "account_monitor_consecutive_failures", CooldownMinutes: 10},
	)

	require.True(t, applied)
	require.False(t, skipped)
	require.Equal(t, account.ID, blocker.blockedAccountID)
	require.Empty(t, blocker.groupBlocked)
	require.Len(t, blocker.outboxEvents, 1)
	require.Equal(t, "account", blocker.outboxEvents[0].payload["block_granularity"])
}
