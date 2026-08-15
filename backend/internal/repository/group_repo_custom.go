package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbaccountgroup "github.com/Wei-Shaw/sub2api/ent/accountgroup"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"

	entsql "entgo.io/ent/dialect/sql"
)

func (r *groupRepository) getAccountCountCustom(ctx context.Context, groupID int64) (total int64, active int64, handled bool, err error) {
	counts, err := r.loadAccountCountsCustomQuery(ctx, []int64{groupID})
	if err != nil {
		return 0, 0, true, err
	}
	count := counts[groupID]
	return count.Total, count.Active, true, nil
}

func (r *groupRepository) loadAccountCountsCustom(ctx context.Context, groupIDs []int64) (counts map[int64]groupAccountCounts, handled bool, err error) {
	counts, err = r.loadAccountCountsCustomQuery(ctx, groupIDs)
	return counts, true, err
}

func (r *groupRepository) loadAccountCountsCustomQuery(ctx context.Context, groupIDs []int64) (counts map[int64]groupAccountCounts, err error) {
	counts = make(map[int64]groupAccountCounts, len(groupIDs))
	if len(groupIDs) == 0 {
		return counts, nil
	}

	rows, err := r.sql.QueryContext(ctx, `
		SELECT
			ag.group_id,
			`+joinedAccountSchedulabilitySelectColumns+`
		FROM account_groups ag
		JOIN accounts a ON a.id = ag.account_id
		WHERE ag.group_id = ANY($1)
			AND a.deleted_at IS NULL
	`, pq.Array(groupIDs))
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			counts = nil
		}
	}()

	now := time.Now()
	for rows.Next() {
		var (
			groupID int64
			account service.Account
		)
		accountTarget := newAccountSchedulabilityScanTarget(&account)
		destinations := append([]any{&groupID}, accountTarget.destinations()...)
		if err = rows.Scan(destinations...); err != nil {
			return nil, err
		}
		if err = accountTarget.apply(); err != nil {
			return nil, err
		}
		count := counts[groupID]
		count.Total++
		class := account.SchedulabilityClassAt(now)
		if class.Schedulable {
			count.Active++
		}
		if class.TemporarilyLimited {
			count.RateLimited++
		}
		counts[groupID] = count
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return counts, nil
}

func bindAccountsToGroupStatementCustom(accountIDs []int64, groupID int64) (string, []any) {
	return `INSERT INTO account_groups (account_id, group_id, priority, role, weight, sort_order, scheduling_configured, created_at)
		 SELECT account_id, $2, row_number() OVER (ORDER BY ord)::int, 'primary', 100, row_number() OVER (ORDER BY ord)::int, TRUE, NOW()
		 FROM unnest($1::bigint[]) WITH ORDINALITY AS t(account_id, ord)
		 ON CONFLICT (account_id, group_id) DO NOTHING`, []any{pq.Array(accountIDs), groupID}
}

func (r *groupRepository) ListAccountSchedulingConfigs(ctx context.Context, groupID int64) ([]service.AccountSchedulingEntry, error) {
	entries, err := r.client.AccountGroup.Query().
		Where(
			func(s *entsql.Selector) {
				s.Where(entsql.EQ(s.C("group_id"), groupID))
			},
			dbaccountgroup.HasAccountWith(dbaccount.DeletedAtIsNil()),
		).
		WithAccount().
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]service.AccountSchedulingEntry, 0, len(entries))
	for _, entry := range entries {
		item := service.AccountSchedulingEntry{
			AccountSchedulingConfig: service.AccountSchedulingConfig{
				AccountID:            entry.AccountID,
				Priority:             entry.Priority,
				Role:                 entry.Role,
				Weight:               entry.Weight,
				SortOrder:            entry.SortOrder,
				SchedulingConfigured: true,
			},
			GroupID: entry.GroupID,
		}
		if entry.Edges.Account == nil {
			continue
		}
		item.Account = accountEntityToService(entry.Edges.Account)
		item.Account.AccountGroups = []service.AccountGroup{{
			AccountID:            entry.AccountID,
			GroupID:              entry.GroupID,
			Priority:             entry.Priority,
			Role:                 entry.Role,
			Weight:               entry.Weight,
			SortOrder:            entry.SortOrder,
			SchedulingConfigured: true,
		}}
		item.BlockReason = item.Account.SchedulingBlockReasonForGroupAt(groupID, time.Now())
		item.State = item.BlockReason.SchedulerState()
		out = append(out, item)
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.SortOrder != b.SortOrder {
			return a.SortOrder < b.SortOrder
		}
		// Role is metadata only.  The persisted sort_order is the canonical
		// per-group order; use the stable account ID only for malformed ties.
		return a.AccountID < b.AccountID
	})

	return out, nil
}

func (r *groupRepository) UpdateAccountSchedulingConfigs(ctx context.Context, groupID int64, configs []service.AccountSchedulingConfig) error {
	if groupID <= 0 {
		return service.ErrGroupNotFound
	}
	if len(configs) == 0 {
		return nil
	}

	seen := make(map[int64]struct{}, len(configs))
	updates := make([]service.AccountSchedulingConfig, 0, len(configs))
	for _, cfg := range configs {
		if cfg.AccountID <= 0 {
			return fmt.Errorf("invalid account_id")
		}
		if _, exists := seen[cfg.AccountID]; exists {
			return fmt.Errorf("duplicate account_id %d", cfg.AccountID)
		}
		seen[cfg.AccountID] = struct{}{}
		switch cfg.Role {
		case "", service.AccountGroupRolePrimary:
			cfg.Role = service.AccountGroupRolePrimary
		case service.AccountGroupRoleBackup:
			cfg.Role = service.AccountGroupRoleBackup
		default:
			return fmt.Errorf("invalid role %q", cfg.Role)
		}
		if cfg.Weight <= 0 {
			return fmt.Errorf("weight must be > 0")
		}
		if cfg.SortOrder == 0 {
			cfg.SortOrder = len(updates) + 1
		}
		cfg.Priority = cfg.SortOrder
		updates = append(updates, cfg)
	}

	accountIDs := make([]int64, 0, len(updates))
	for _, cfg := range updates {
		accountIDs = append(accountIDs, cfg.AccountID)
	}

	var existingCount int
	if err := scanSingleRow(ctx, r.sql,
		`SELECT COUNT(*)
		 FROM account_groups
		 WHERE group_id = $1 AND account_id = ANY($2)`,
		[]any{groupID, pq.Array(accountIDs)},
		&existingCount,
	); err != nil {
		return err
	}
	if existingCount != len(accountIDs) {
		return fmt.Errorf("account scheduling config contains accounts not bound to group")
	}

	args := make([]any, 0, len(updates)*5+2)
	roleCases := make([]string, 0, len(updates))
	weightCases := make([]string, 0, len(updates))
	priorityCases := make([]string, 0, len(updates))
	sortCases := make([]string, 0, len(updates))
	placeholder := 1
	for _, cfg := range updates {
		roleCases = append(roleCases, fmt.Sprintf("WHEN $%d THEN $%d", placeholder, placeholder+1))
		weightCases = append(weightCases, fmt.Sprintf("WHEN $%d THEN $%d", placeholder, placeholder+2))
		priorityCases = append(priorityCases, fmt.Sprintf("WHEN $%d THEN $%d", placeholder, placeholder+3))
		sortCases = append(sortCases, fmt.Sprintf("WHEN $%d THEN $%d", placeholder, placeholder+4))
		args = append(args, cfg.AccountID, cfg.Role, cfg.Weight, cfg.Priority, cfg.SortOrder)
		placeholder += 5
	}
	args = append(args, pq.Array(accountIDs), groupID)
	idArrayPlaceholder := placeholder
	groupPlaceholder := placeholder + 1

	query := fmt.Sprintf(`
		UPDATE account_groups
		SET
				role = CASE account_id %s ELSE role END,
				weight = CASE account_id %s ELSE weight END,
				priority = CASE account_id %s ELSE priority END,
				sort_order = CASE account_id %s ELSE sort_order END,
				scheduling_configured = TRUE
		WHERE account_id = ANY($%d) AND group_id = $%d
	`, strings.Join(roleCases, " "), strings.Join(weightCases, " "), strings.Join(priorityCases, " "), strings.Join(sortCases, " "), idArrayPlaceholder, groupPlaceholder)

	result, err := r.sql.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != int64(len(updates)) {
		return fmt.Errorf("account scheduling config update affected %d rows, expected %d", affected, len(updates))
	}

	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventGroupChanged, nil, &groupID, nil); err != nil {
		logger.LegacyPrintf("repository.group", "[SchedulerOutbox] enqueue account scheduling config update failed: group=%d err=%v", groupID, err)
	}
	return nil
}
