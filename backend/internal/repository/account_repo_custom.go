package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbaccountgroup "github.com/Wei-Shaw/sub2api/ent/accountgroup"
	dbpredicate "github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// LoadOpenAIAccountRuntimeStats restores model-scoped scheduler EWMA state
// after a process restart. Only the recent bounded window is loaded; the
// durable group experience sorter remains the long-term source of truth.
func (r *accountRepository) LoadOpenAIAccountRuntimeStats(ctx context.Context) ([]service.OpenAIAccountRuntimeStatRecord, error) {
	if r == nil || r.sql == nil {
		return nil, nil
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT account_id, canonical_model, error_rate_ewma, ttft_ewma,
		       sample_count, ttft_samples, updated_at,
		       transient_failure_streak, transient_last_failure_at, transient_block_until,
		       slow_reserve_marked_at, slow_reserve_last_touched_at,
		       slow_reserve_expires_at, slow_reserve_reason, slow_reserve_ttft_ms
		FROM openai_account_scheduler_runtime_stats
		WHERE updated_at >= NOW() - INTERVAL '24 hours'
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	records := make([]service.OpenAIAccountRuntimeStatRecord, 0)
	for rows.Next() {
		var (
			record                                                              service.OpenAIAccountRuntimeStatRecord
			ttft                                                                sql.NullFloat64
			updatedAt                                                           time.Time
			transientLastFailureAt, transientBlockUntil                         sql.NullTime
			slowReserveMarkedAt, slowReserveLastTouchedAt, slowReserveExpiresAt sql.NullTime
			slowReserveReason                                                   sql.NullString
		)
		if err := rows.Scan(
			&record.AccountID, &record.CanonicalModel, &record.ErrorRateEWMA, &ttft,
			&record.SampleCount, &record.TTFTSamples, &updatedAt,
			&record.TransientFailureStreak, &transientLastFailureAt, &transientBlockUntil,
			&slowReserveMarkedAt, &slowReserveLastTouchedAt, &slowReserveExpiresAt,
			&slowReserveReason, &record.SlowReserveTTFTMs,
		); err != nil {
			return nil, err
		}
		if ttft.Valid {
			value := ttft.Float64
			record.TTFTEWMA = &value
		}
		record.UpdatedAt = updatedAt
		if transientLastFailureAt.Valid {
			record.TransientLastFailureAt = &transientLastFailureAt.Time
		}
		if transientBlockUntil.Valid {
			record.TransientBlockUntil = &transientBlockUntil.Time
		}
		if slowReserveMarkedAt.Valid {
			record.SlowReserveMarkedAt = &slowReserveMarkedAt.Time
		}
		if slowReserveLastTouchedAt.Valid {
			record.SlowReserveLastTouchedAt = &slowReserveLastTouchedAt.Time
		}
		if slowReserveExpiresAt.Valid {
			record.SlowReserveExpiresAt = &slowReserveExpiresAt.Time
		}
		if slowReserveReason.Valid {
			record.SlowReserveReason = slowReserveReason.String
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

// SaveOpenAIAccountRuntimeStats persists a bounded batch of scheduler EWMA
// snapshots. The write is deliberately best-effort from the gateway hot path;
// the in-memory score is still authoritative until the next request.
func (r *accountRepository) SaveOpenAIAccountRuntimeStats(ctx context.Context, records []service.OpenAIAccountRuntimeStatRecord) error {
	if r == nil || r.sql == nil || len(records) == 0 {
		return nil
	}
	const maxBatch = 256
	for start := 0; start < len(records); start += maxBatch {
		end := start + maxBatch
		if end > len(records) {
			end = len(records)
		}
		batch := records[start:end]
		values := make([]string, 0, len(batch))
		args := make([]any, 0, len(batch)*15)
		for i, record := range batch {
			base := i*15 + 1
			placeholders := make([]string, 0, 15)
			for offset := 0; offset < 15; offset++ {
				placeholders = append(placeholders, fmt.Sprintf("$%d", base+offset))
			}
			values = append(values, "("+strings.Join(placeholders, ",")+")")
			args = append(args,
				record.AccountID, record.CanonicalModel, record.ErrorRateEWMA, record.TTFTEWMA,
				record.SampleCount, record.TTFTSamples, record.UpdatedAt,
				record.TransientFailureStreak, record.TransientLastFailureAt, record.TransientBlockUntil,
				record.SlowReserveMarkedAt, record.SlowReserveLastTouchedAt, record.SlowReserveExpiresAt,
				record.SlowReserveReason, record.SlowReserveTTFTMs,
			)
		}
		query := `INSERT INTO openai_account_scheduler_runtime_stats
			(account_id, canonical_model, error_rate_ewma, ttft_ewma, sample_count, ttft_samples, updated_at,
			 transient_failure_streak, transient_last_failure_at, transient_block_until,
			 slow_reserve_marked_at, slow_reserve_last_touched_at, slow_reserve_expires_at,
			 slow_reserve_reason, slow_reserve_ttft_ms)
			VALUES ` + strings.Join(values, ",") + `
			ON CONFLICT (account_id, canonical_model) DO UPDATE SET
				error_rate_ewma = EXCLUDED.error_rate_ewma,
				ttft_ewma = EXCLUDED.ttft_ewma,
				sample_count = EXCLUDED.sample_count,
				ttft_samples = EXCLUDED.ttft_samples,
				updated_at = EXCLUDED.updated_at,
				transient_failure_streak = EXCLUDED.transient_failure_streak,
				transient_last_failure_at = EXCLUDED.transient_last_failure_at,
				transient_block_until = EXCLUDED.transient_block_until,
				slow_reserve_marked_at = EXCLUDED.slow_reserve_marked_at,
				slow_reserve_last_touched_at = EXCLUDED.slow_reserve_last_touched_at,
				slow_reserve_expires_at = EXCLUDED.slow_reserve_expires_at,
				slow_reserve_reason = EXCLUDED.slow_reserve_reason,
				slow_reserve_ttft_ms = EXCLUDED.slow_reserve_ttft_ms`
		if _, err := r.sql.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

// SaveOpenAISchedulerSafetyState updates only breaker/reserve columns. It does
// not touch EWMA fields, so an asynchronous safety notification can never
// replace a newer health score with an older in-memory snapshot.
func (r *accountRepository) SaveOpenAISchedulerSafetyState(ctx context.Context, record service.OpenAISchedulerSafetyStateRecord) error {
	if r == nil || r.sql == nil || record.AccountID <= 0 || strings.TrimSpace(record.CanonicalModel) == "" {
		return nil
	}
	_, err := r.sql.ExecContext(ctx, `
		INSERT INTO openai_account_scheduler_runtime_stats (
			account_id, canonical_model,
			transient_failure_streak, transient_last_failure_at, transient_block_until,
			slow_reserve_marked_at, slow_reserve_last_touched_at, slow_reserve_expires_at,
			slow_reserve_reason, slow_reserve_ttft_ms, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
		ON CONFLICT (account_id, canonical_model) DO UPDATE SET
			transient_failure_streak = EXCLUDED.transient_failure_streak,
			transient_last_failure_at = EXCLUDED.transient_last_failure_at,
			transient_block_until = EXCLUDED.transient_block_until,
			slow_reserve_marked_at = EXCLUDED.slow_reserve_marked_at,
			slow_reserve_last_touched_at = EXCLUDED.slow_reserve_last_touched_at,
			slow_reserve_expires_at = EXCLUDED.slow_reserve_expires_at,
			slow_reserve_reason = EXCLUDED.slow_reserve_reason,
			slow_reserve_ttft_ms = EXCLUDED.slow_reserve_ttft_ms,
			updated_at = NOW()
	`,
		record.AccountID, record.CanonicalModel,
		record.TransientFailureStreak, record.TransientLastFailureAt, record.TransientBlockUntil,
		record.SlowReserveMarkedAt, record.SlowReserveLastTouchedAt, record.SlowReserveExpiresAt,
		record.SlowReserveReason, record.SlowReserveTTFTMs,
	)
	return err
}

func (r *accountRepository) syncSchedulerBucketMembershipCustom(ctx context.Context, account *service.Account) {
	if account == nil || account.IsSchedulerBucketMember() {
		return
	}
	if err := service.RemoveSchedulerAccountFromBuckets(ctx, r.schedulerCache, account.ID); err != nil {
		logger.LegacyPrintf("repository.account", "[Scheduler] remove account from buckets failed: id=%d err=%v", account.ID, err)
	}
}

func configureAccountGroupCreateCustom(builder *dbent.AccountGroupCreate, priority int) {
	builder.
		SetRole(service.AccountGroupRolePrimary).
		SetWeight(100).
		SetSortOrder(priority).
		SetSchedulingConfigured(true)
}

// bindGroupsCustom preserves per-group scheduler configuration for retained
// bindings. The official delete-and-recreate implementation would erase it.
func (r *accountRepository) bindGroupsCustom(ctx context.Context, accountID int64, groupIDs []int64) (bool, error) {
	existingRows, err := r.client.AccountGroup.Query().
		Where(dbaccountgroup.AccountIDEQ(accountID)).
		All(ctx)
	if err != nil {
		return true, err
	}

	existingByGroupID := make(map[int64]*dbent.AccountGroup, len(existingRows))
	existingGroupIDs := make([]int64, 0, len(existingRows))
	for _, row := range existingRows {
		existingByGroupID[row.GroupID] = row
		existingGroupIDs = append(existingGroupIDs, row.GroupID)
	}

	desiredGroupIDs := make([]int64, 0, len(groupIDs))
	desiredSet := make(map[int64]int, len(groupIDs))
	for _, groupID := range groupIDs {
		if groupID <= 0 {
			continue
		}
		if _, exists := desiredSet[groupID]; exists {
			continue
		}
		desiredGroupIDs = append(desiredGroupIDs, groupID)
		desiredSet[groupID] = len(desiredGroupIDs)
	}

	tx, err := r.client.Tx(ctx)
	if err != nil && !errors.Is(err, dbent.ErrTxStarted) {
		return true, err
	}

	var txClient *dbent.Client
	if err == nil {
		defer func() { _ = tx.Rollback() }()
		txClient = tx.Client()
	} else {
		txClient = r.client
	}

	deleteQuery := txClient.AccountGroup.Delete().Where(dbaccountgroup.AccountIDEQ(accountID))
	if len(desiredGroupIDs) > 0 {
		deleteQuery = deleteQuery.Where(dbaccountgroup.GroupIDNotIn(desiredGroupIDs...))
	}
	if _, err := deleteQuery.Exec(ctx); err != nil {
		return true, err
	}

	builders := make([]*dbent.AccountGroupCreate, 0, len(desiredGroupIDs))
	for _, groupID := range desiredGroupIDs {
		sortOrder := desiredSet[groupID]
		if existing, ok := existingByGroupID[groupID]; ok {
			update := txClient.AccountGroup.Update().
				Where(
					dbaccountgroup.AccountIDEQ(accountID),
					dbaccountgroup.GroupIDEQ(groupID),
				).
				SetPriority(sortOrder).
				SetSchedulingConfigured(true)
			if !existing.SchedulingConfigured {
				update.SetSortOrder(sortOrder)
			}
			if _, err := update.Save(ctx); err != nil {
				return true, err
			}
			continue
		}
		builder := txClient.AccountGroup.Create().
			SetAccountID(accountID).
			SetGroupID(groupID).
			SetPriority(sortOrder)
		configureAccountGroupCreateCustom(builder, sortOrder)
		builders = append(builders, builder)
	}

	if len(builders) > 0 {
		if _, err := txClient.AccountGroup.CreateBulk(builders...).Save(ctx); err != nil {
			return true, err
		}
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return true, err
		}
	}

	payload := buildSchedulerGroupPayload(mergeGroupIDs(existingGroupIDs, desiredGroupIDs))
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountGroupsChanged, &accountID, nil, payload); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue bind groups failed: account=%d err=%v", accountID, err)
	}
	return true, nil
}

func (r *accountRepository) SetGroupTempUnschedulable(ctx context.Context, id int64, groupID int64, until time.Time, reason string) error {
	if groupID <= 0 {
		return nil
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
		return err
	}

	client := clientFromContext(ctx, r.client)
	result, err := client.ExecContext(ctx, `UPDATE accounts SET
		extra = jsonb_set(
			jsonb_set(COALESCE(extra, '{}'::jsonb), '{group_temp_unschedulable}'::text[], COALESCE(extra->'group_temp_unschedulable', '{}'::jsonb), true),
			ARRAY['group_temp_unschedulable', $1]::text[], $2::jsonb, true
		),
		updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL`, strconv.FormatInt(groupID, 10), raw, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAccountNotFound
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, &groupID, nil); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue group temp unschedulable failed: account=%d group=%d err=%v", id, groupID, err)
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
	return nil
}

func (r *accountRepository) ClearGroupTempUnschedulable(ctx context.Context, id int64) error {
	groupIDs, err := r.loadAccountGroupIDs(ctx, id)
	if err != nil {
		return err
	}
	client := clientFromContext(ctx, r.client)
	result, err := client.ExecContext(ctx, `UPDATE accounts
		SET extra = COALESCE(extra, '{}'::jsonb) - 'group_temp_unschedulable',
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAccountNotFound
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, buildSchedulerGroupPayload(groupIDs)); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue group cooldown recovery failed: account=%d err=%v", id, err)
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
	return nil
}

// ClearTransient5xxCooldown clears only the cooldown produced by the transient
// upstream 5xx policy. The conditions deliberately exclude manual scheduling
// changes and any other temporary state that happens to share the same column.
func (r *accountRepository) ClearTransient5xxCooldown(ctx context.Context, id int64) (bool, error) {
	result, err := r.sql.ExecContext(ctx, `
		UPDATE accounts
		SET temp_unschedulable_until = NULL,
			temp_unschedulable_reason = NULL,
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL
			AND status = $2
			AND schedulable IS TRUE
			AND temp_unschedulable_until IS NOT NULL
			AND (
				LEFT(temp_unschedulable_reason, LENGTH($3)) = $3
				OR LEFT(temp_unschedulable_reason, LENGTH($4)) = $4
			)
	`, id, service.StatusActive,
		service.Transient5xxCooldownReasonPrefix,
		service.PoolTransient5xxCooldownReasonPrefix)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected == 0 {
		return false, nil
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue transient 5xx recovery failed: account=%d err=%v", id, err)
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
	return true, nil
}

// RenewTransient5xxCooldown extends a matching transient 5xx cooldown while
// its active health probe is still failing. It cannot create a cooldown, change
// its reason, or touch an account the administrator has made unschedulable.
func (r *accountRepository) RenewTransient5xxCooldown(ctx context.Context, id int64, until time.Time) (bool, error) {
	result, err := r.sql.ExecContext(ctx, `
		UPDATE accounts
		SET temp_unschedulable_until = $1,
			updated_at = NOW()
		WHERE id = $2
			AND deleted_at IS NULL
			AND status = $3
			AND schedulable IS TRUE
			AND temp_unschedulable_until IS NOT NULL
			AND temp_unschedulable_until < $1
			AND (
				LEFT(temp_unschedulable_reason, LENGTH($4)) = $4
				OR LEFT(temp_unschedulable_reason, LENGTH($5)) = $5
			)
	`, until, id, service.StatusActive,
		service.Transient5xxCooldownReasonPrefix,
		service.PoolTransient5xxCooldownReasonPrefix)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected == 0 {
		return false, nil
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue transient 5xx cooldown renewal failed: account=%d err=%v", id, err)
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
	return true, nil
}

func (r *accountRepository) setSchedulableCustom(ctx context.Context, id int64, schedulable bool) (bool, error) {
	groupIDs, err := r.loadAccountGroupIDs(ctx, id)
	if err != nil {
		return true, err
	}
	if schedulable {
		_, err = r.sql.ExecContext(ctx, `
			UPDATE accounts
			SET schedulable = TRUE,
				error_message = CASE WHEN status = 'active' THEN '' ELSE error_message END,
				temp_unschedulable_until = CASE WHEN status = 'active' THEN NULL ELSE temp_unschedulable_until END,
				temp_unschedulable_reason = CASE WHEN status = 'active' THEN NULL ELSE temp_unschedulable_reason END,
				updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL`, id)
	} else {
		_, err = r.client.Account.Update().Where(dbaccount.IDEQ(id)).SetSchedulable(false).Save(ctx)
	}
	if err != nil {
		return true, err
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, buildSchedulerGroupPayload(groupIDs)); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue schedulable change failed: account=%d err=%v", id, err)
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
	return true, nil
}

func (r *accountRepository) accountsToSchedulableService(ctx context.Context, accounts []*dbent.Account, now time.Time) ([]service.Account, error) {
	result, err := r.accountsToService(ctx, accounts)
	if err != nil {
		return nil, err
	}
	return filterSchedulableServiceAccounts(result, now), nil
}

func filterSchedulableServiceAccounts(accounts []service.Account, now time.Time) []service.Account {
	filtered := accounts[:0]
	for _, account := range accounts {
		if account.IsSchedulableAt(now) {
			filtered = append(filtered, account)
		}
	}
	return filtered
}

func filterSchedulableServiceAccountsForGroup(accounts []service.Account, groupID int64, now time.Time) []service.Account {
	if groupID <= 0 {
		return filterSchedulableServiceAccounts(accounts, now)
	}
	filtered := accounts[:0]
	for _, account := range accounts {
		if account.IsSchedulableAt(now) {
			filtered = append(filtered, account)
		}
	}
	return filtered
}

func schedulableAccountListOverlayPredicates(now time.Time) []dbpredicate.Account {
	return []dbpredicate.Account{
		notExpiredPredicate(now),
		dbaccount.Or(dbaccount.OverloadUntilIsNil(), dbaccount.OverloadUntilLTE(now)),
	}
}

func hydrateAccountGroupSchedulingCustom(dst *service.AccountGroup, src *dbent.AccountGroup) {
	dst.Role = src.Role
	dst.Weight = src.Weight
	dst.SortOrder = src.SortOrder
	dst.SchedulingConfigured = true
}

func (r *accountRepository) AppendSchedulerOutboxEvent(ctx context.Context, eventType string, accountID *int64, groupID *int64, payload map[string]any) error {
	return enqueueSchedulerOutbox(ctx, r.sql, eventType, accountID, groupID, payload)
}
