package service

import (
	"strings"
	"testing"
)

func TestOpsCleanupDeletedCountsStringIncludesSchedulerTables(t *testing.T) {
	got := (opsCleanupDeletedCounts{opsCleanupDeletedCountsCustom: opsCleanupDeletedCountsCustom{schedulerOutbox: 12, schedulerHistory: 34}}).String()
	if !strings.Contains(got, "scheduler_outbox=12") {
		t.Fatalf("String() = %q, want scheduler_outbox count", got)
	}
	if !strings.Contains(got, "scheduler_history=34") {
		t.Fatalf("String() = %q, want scheduler_history count", got)
	}
}
