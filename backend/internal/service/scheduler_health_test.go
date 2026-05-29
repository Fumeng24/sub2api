package service

import (
	"testing"
	"time"
)

func TestSchedulerRoleAwareOrderPrefersPrimaryBeforeBackup(t *testing.T) {
	groupID := int64(100)
	primary := &Account{
		ID:       1,
		Priority: 10,
		AccountGroups: []AccountGroup{{
			AccountID:            1,
			GroupID:              groupID,
			Role:                 AccountGroupRolePrimary,
			Weight:               100,
			SortOrder:            10,
			SchedulingConfigured: true,
		}},
	}
	backup := &Account{
		ID:       2,
		Priority: 1,
		AccountGroups: []AccountGroup{{
			AccountID:            2,
			GroupID:              groupID,
			Role:                 AccountGroupRoleBackup,
			Weight:               1000,
			SortOrder:            1,
			SchedulingConfigured: true,
		}},
	}

	scores := buildSchedulerAccountScores(
		[]*Account{backup, primary},
		&groupID,
		"gpt-5.5",
		"/v1/responses",
		nil,
		newAccountSchedulerHealthStats(),
		true,
	)
	order := buildRoleAwareSchedulerOrder(scores, true, "test")

	if len(order) != 2 {
		t.Fatalf("expected 2 scheduler candidates, got %d", len(order))
	}
	if order[0].Account.ID != primary.ID {
		t.Fatalf("expected primary account first, got account %d", order[0].Account.ID)
	}
	if order[1].Account.ID != backup.ID {
		t.Fatalf("expected backup account second, got account %d", order[1].Account.ID)
	}
}

func TestSchedulerFallsBackToBackupWhenPrimaryUnavailable(t *testing.T) {
	groupID := int64(100)
	primary := &Account{
		ID: 1,
		AccountGroups: []AccountGroup{{
			AccountID:            1,
			GroupID:              groupID,
			Role:                 AccountGroupRolePrimary,
			Weight:               100,
			SortOrder:            10,
			SchedulingConfigured: true,
		}},
	}
	backup := &Account{
		ID: 2,
		AccountGroups: []AccountGroup{{
			AccountID:            2,
			GroupID:              groupID,
			Role:                 AccountGroupRoleBackup,
			Weight:               100,
			SortOrder:            20,
			SchedulingConfigured: true,
		}},
	}

	t.Run("primary full", func(t *testing.T) {
		scores := buildSchedulerAccountScores(
			[]*Account{primary, backup},
			&groupID,
			"gpt-5.5",
			"/v1/responses",
			map[int64]*AccountLoadInfo{
				primary.ID: {AccountID: primary.ID, LoadRate: 100},
				backup.ID:  {AccountID: backup.ID, LoadRate: 0},
			},
			newAccountSchedulerHealthStats(),
			true,
		)
		order := buildRoleAwareSchedulerOrder(scores, true, "full")
		if len(order) != 1 || order[0].Account.ID != backup.ID {
			t.Fatalf("expected only backup candidate when primary is full, got %#v", accountIDsFromSchedulerScores(order))
		}
	})

	t.Run("primary circuit open", func(t *testing.T) {
		health := newAccountSchedulerHealthStats()
		for i := 0; i < 3; i++ {
			health.reportFailure(primary.ID, "gpt-5.5", "/v1/responses", "transient", 0)
		}

		scores := buildSchedulerAccountScores(
			[]*Account{primary, backup},
			&groupID,
			"gpt-5.5",
			"/v1/responses",
			nil,
			health,
			true,
		)
		order := buildRoleAwareSchedulerOrder(scores, true, "open")
		if len(order) != 1 || order[0].Account.ID != backup.ID {
			t.Fatalf("expected only backup candidate when primary circuit is open, got %#v", accountIDsFromSchedulerScores(order))
		}
	})
}

func TestSchedulerHealthIsIsolatedByAccountModelAndEndpoint(t *testing.T) {
	groupID := int64(100)
	account := &Account{
		ID: 1,
		AccountGroups: []AccountGroup{{
			AccountID:            1,
			GroupID:              groupID,
			Role:                 AccountGroupRolePrimary,
			Weight:               100,
			SortOrder:            10,
			SchedulingConfigured: true,
		}},
	}
	health := newAccountSchedulerHealthStats()
	for i := 0; i < 3; i++ {
		health.reportFailure(account.ID, "gpt-5.4", "/v1/responses", "transient", 0)
	}

	badModelScores := buildSchedulerAccountScores(
		[]*Account{account},
		&groupID,
		"gpt-5.4",
		"/v1/responses",
		nil,
		health,
		true,
	)
	if len(badModelScores) != 0 {
		t.Fatalf("expected gpt-5.4 on /v1/responses to be blocked, got %d candidates", len(badModelScores))
	}

	otherModelScores := buildSchedulerAccountScores(
		[]*Account{account},
		&groupID,
		"gpt-5.5",
		"/v1/responses",
		nil,
		health,
		true,
	)
	if len(otherModelScores) != 1 {
		t.Fatalf("expected gpt-5.5 on same endpoint to stay available, got %d candidates", len(otherModelScores))
	}

	otherEndpointScores := buildSchedulerAccountScores(
		[]*Account{account},
		&groupID,
		"gpt-5.4",
		"/v1/chat/completions",
		nil,
		health,
		true,
	)
	if len(otherEndpointScores) != 1 {
		t.Fatalf("expected gpt-5.4 on different endpoint to stay available, got %d candidates", len(otherEndpointScores))
	}
}

func TestSchedulerHalfOpenAllowsSingleProbeAfterCooldown(t *testing.T) {
	groupID := int64(100)
	account := &Account{
		ID: 1,
		AccountGroups: []AccountGroup{{
			AccountID:            1,
			GroupID:              groupID,
			Role:                 AccountGroupRolePrimary,
			Weight:               100,
			SortOrder:            10,
			SchedulingConfigured: true,
		}},
	}
	health := newAccountSchedulerHealthStats()
	for i := 0; i < 3; i++ {
		health.reportFailure(account.ID, "gpt-5.5", "/v1/responses", "transient", time.Minute)
	}
	key := makeAccountSchedulerHealthKey(account.ID, "gpt-5.5", "/v1/responses")
	value, ok := health.entries.Load(key)
	if !ok {
		t.Fatal("expected health entry")
	}
	entry, ok := value.(*accountSchedulerHealthEntry)
	if !ok || entry == nil {
		t.Fatalf("unexpected health entry type %T", value)
	}
	entry.mu.Lock()
	entry.cooldownUntil = time.Now().Add(-time.Second)
	entry.mu.Unlock()

	svc := &GatewayService{schedulerHealth: health}
	if !svc.isGatewayAccountSchedulerHealthCandidateAllowed(account.ID, "gpt-5.5", "/v1/responses") {
		t.Fatal("expected expired open circuit to be eligible for half-open candidate scoring")
	}
	if svc.isGatewayAccountSchedulerHealthAllowed(account.ID, "gpt-5.5", "/v1/responses") {
		t.Fatal("expected sticky/strict health check to reject half-open circuit")
	}

	firstScores := buildSchedulerAccountScores(
		[]*Account{account},
		&groupID,
		"gpt-5.5",
		"/v1/responses",
		nil,
		health,
		true,
	)
	if len(firstScores) != 1 {
		t.Fatalf("expected one half-open probe candidate, got %d", len(firstScores))
	}
	if !firstScores[0].HalfOpen || !firstScores[0].Health.HalfOpenProbe {
		t.Fatalf("expected first candidate to consume half-open probe, got half_open=%v probe=%v", firstScores[0].HalfOpen, firstScores[0].Health.HalfOpenProbe)
	}

	secondScores := buildSchedulerAccountScores(
		[]*Account{account},
		&groupID,
		"gpt-5.5",
		"/v1/responses",
		nil,
		health,
		true,
	)
	if len(secondScores) != 0 {
		t.Fatalf("expected second half-open probe to be suppressed while first is in flight, got %d", len(secondScores))
	}
}

func TestSchedulerAccountGroupConfigIsIndependentPerGroup(t *testing.T) {
	primaryGroupID := int64(100)
	backupGroupID := int64(200)
	account := &Account{
		ID:       1,
		Priority: 50,
		AccountGroups: []AccountGroup{
			{
				AccountID:            1,
				GroupID:              primaryGroupID,
				Role:                 AccountGroupRolePrimary,
				Weight:               80,
				SortOrder:            10,
				SchedulingConfigured: true,
			},
			{
				AccountID:            1,
				GroupID:              backupGroupID,
				Role:                 AccountGroupRoleBackup,
				Weight:               25,
				SortOrder:            30,
				SchedulingConfigured: true,
			},
		},
	}

	primaryScores := buildSchedulerAccountScores(
		[]*Account{account},
		&primaryGroupID,
		"gpt-5.5",
		"/v1/responses",
		nil,
		nil,
		true,
	)
	backupScores := buildSchedulerAccountScores(
		[]*Account{account},
		&backupGroupID,
		"gpt-5.5",
		"/v1/responses",
		nil,
		nil,
		true,
	)

	if len(primaryScores) != 1 || len(backupScores) != 1 {
		t.Fatalf("expected one score per group, got primary=%d backup=%d", len(primaryScores), len(backupScores))
	}
	if primaryScores[0].Role != AccountGroupRolePrimary || primaryScores[0].BaseWeight != 80 || primaryScores[0].SortOrder != 10 {
		t.Fatalf("unexpected primary group config: role=%s weight=%d sort=%d", primaryScores[0].Role, primaryScores[0].BaseWeight, primaryScores[0].SortOrder)
	}
	if backupScores[0].Role != AccountGroupRoleBackup || backupScores[0].BaseWeight != 25 || backupScores[0].SortOrder != 30 {
		t.Fatalf("unexpected backup group config: role=%s weight=%d sort=%d", backupScores[0].Role, backupScores[0].BaseWeight, backupScores[0].SortOrder)
	}
}

func TestSchedulerDefaultMigratedGroupConfigIsNotExplicit(t *testing.T) {
	groupID := int64(100)
	a := &Account{
		ID:       1,
		Priority: 20,
		AccountGroups: []AccountGroup{{
			AccountID:            1,
			GroupID:              groupID,
			Priority:             20,
			Role:                 AccountGroupRolePrimary,
			Weight:               100,
			SortOrder:            20,
			SchedulingConfigured: false,
		}},
	}
	b := &Account{
		ID:       2,
		Priority: 10,
		AccountGroups: []AccountGroup{{
			AccountID:            2,
			GroupID:              groupID,
			Priority:             10,
			Role:                 AccountGroupRolePrimary,
			Weight:               100,
			SortOrder:            10,
			SchedulingConfigured: false,
		}},
	}

	scores := buildSchedulerAccountScores(
		[]*Account{a, b},
		&groupID,
		"gpt-5.5",
		"/v1/responses",
		nil,
		nil,
		true,
	)
	if hasExplicitSchedulerGroupConfig(scores) {
		t.Fatal("expected migrated default account_groups rows to be treated as non-explicit scheduler config")
	}

	order := buildRoleAwareSchedulerOrder(scores, false, "default")
	if len(order) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(order))
	}
	if order[0].Account.ID != b.ID || order[1].Account.ID != a.ID {
		t.Fatalf("expected non-explicit config to preserve sorted priority order, got %#v", accountIDsFromSchedulerScores(order))
	}
}

func accountIDsFromSchedulerScores(scores []schedulerAccountScore) []int64 {
	ids := make([]int64, 0, len(scores))
	for _, score := range scores {
		if score.Account != nil {
			ids = append(ids, score.Account.ID)
		}
	}
	return ids
}
