package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const schedulerProtectionAdvisoryLockNamespace = "account-group-scheduler-protection"

// TrySetGroupTempUnschedulableUnlessLast atomically applies a group-scoped
// cooldown only while another schedulable account remains in the same pool.
func (r *accountRepository) TrySetGroupTempUnschedulableUnlessLast(
	ctx context.Context,
	accountID, groupID int64,
	platform string,
	until time.Time,
	reason string,
) (bool, error) {
	platform = strings.TrimSpace(platform)
	if accountID <= 0 {
		return false, fmt.Errorf("scheduler protection: invalid account id %d", accountID)
	}
	if groupID <= 0 {
		return false, fmt.Errorf("scheduler protection: invalid group id %d", groupID)
	}
	if platform == "" {
		return false, fmt.Errorf("scheduler protection: platform is required")
	}
	if until.IsZero() {
		return false, fmt.Errorf("scheduler protection: cooldown deadline is required")
	}

	payload := map[string]string{
		"until":        until.UTC().Format(time.RFC3339),
		"triggered_at": time.Now().UTC().Format(time.RFC3339),
	}
	if value := strings.TrimSpace(reason); value != "" {
		payload["reason"] = value
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("scheduler protection: encode cooldown payload: %w", err)
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return false, fmt.Errorf("scheduler protection: begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	txClient := tx.Client()

	lockKey := fmt.Sprintf("%s:%d:%s", schedulerProtectionAdvisoryLockNamespace, groupID, platform)
	if _, err := txClient.ExecContext(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		lockKey,
	); err != nil {
		return false, fmt.Errorf("scheduler protection: acquire advisory lock: %w", err)
	}

	groupKey := strconv.FormatInt(groupID, 10)
	rows, err := txClient.QueryContext(
		ctx,
		`SELECT a.id, a.type, COALESCE(a.extra, '{}'::jsonb)::text
		 FROM accounts AS a
		 JOIN account_groups AS ag ON ag.account_id = a.id
		 WHERE ag.group_id = $1
		   AND a.platform = $2
		   AND a.deleted_at IS NULL
		   AND a.status = $3
		   AND a.schedulable = TRUE
		   AND (a.temp_unschedulable_until IS NULL OR a.temp_unschedulable_until <= NOW())
		   AND (a.expires_at IS NULL OR a.expires_at > NOW() OR a.auto_pause_on_expired = FALSE)
		   AND (a.overload_until IS NULL OR a.overload_until <= NOW())
		   AND (a.rate_limit_reset_at IS NULL OR a.rate_limit_reset_at <= NOW())
		   AND (
		     a.extra #>> ARRAY['group_temp_unschedulable', $4, 'until'] IS NULL
		     OR (a.extra #>> ARRAY['group_temp_unschedulable', $4, 'until'])::timestamptz <= NOW()
		   )`,
		groupID,
		platform,
		service.StatusActive,
		groupKey,
	)
	if err != nil {
		return false, fmt.Errorf("scheduler protection: query schedulable candidates: %w", err)
	}
	candidateCount := 0
	targetIsCandidate := false
	for rows.Next() {
		var candidateID int64
		var accountType string
		var extraRaw string
		if err := rows.Scan(&candidateID, &accountType, &extraRaw); err != nil {
			_ = rows.Close()
			return false, fmt.Errorf("scheduler protection: scan schedulable candidate: %w", err)
		}
		candidate := service.Account{ID: candidateID, Type: accountType}
		if extraRaw != "" && extraRaw != "null" {
			if err := json.Unmarshal([]byte(extraRaw), &candidate.Extra); err != nil {
				_ = rows.Close()
				return false, fmt.Errorf("scheduler protection: decode candidate quota state: %w", err)
			}
		}
		if candidate.IsAPIKeyOrBedrock() && candidate.IsQuotaExceeded() {
			continue
		}
		candidateCount++
		if candidateID == accountID {
			targetIsCandidate = true
		}
	}
	if err := rows.Close(); err != nil {
		return false, fmt.Errorf("scheduler protection: close candidate rows: %w", err)
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("scheduler protection: iterate schedulable candidates: %w", err)
	}
	if !targetIsCandidate || candidateCount <= 1 {
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("scheduler protection: commit protected no-op: %w", err)
		}
		return false, nil
	}

	result, err := txClient.ExecContext(
		ctx,
		`UPDATE accounts SET
		   extra = jsonb_set(
		     jsonb_set(
		       COALESCE(extra, '{}'::jsonb),
		       '{group_temp_unschedulable}'::text[],
		       COALESCE(extra->'group_temp_unschedulable', '{}'::jsonb),
		       TRUE
		     ),
		     ARRAY['group_temp_unschedulable', $1]::text[],
		     $2::jsonb,
		     TRUE
		   ),
		   updated_at = NOW()
		 WHERE id = $3
		   AND deleted_at IS NULL
		   AND (
		     extra #>> ARRAY['group_temp_unschedulable', $1, 'until'] IS NULL
		     OR (extra #>> ARRAY['group_temp_unschedulable', $1, 'until'])::timestamptz < $4
		   )`,
		groupKey,
		raw,
		accountID,
		until.UTC(),
	)
	if err != nil {
		return false, fmt.Errorf("scheduler protection: update group cooldown: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("scheduler protection: read update result: %w", err)
	}
	if affected != 1 {
		return false, fmt.Errorf("scheduler protection: cooldown update affected %d rows", affected)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("scheduler protection: commit cooldown: %w", err)
	}

	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &accountID, &groupID, nil); err != nil {
		logger.LegacyPrintf(
			"repository.account",
			"[SchedulerOutbox] enqueue protected group temp unschedulable failed: account=%d group=%d err=%v",
			accountID,
			groupID,
			err,
		)
	}
	r.syncSchedulerAccountSnapshot(ctx, accountID)
	return true, nil
}
