package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *channelMonitorRepository) UpdateSortOrders(ctx context.Context, updates []service.ChannelMonitorSortOrderUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	sortOrderByID := make(map[int64]int, len(updates))
	monitorIDs := make([]int64, 0, len(updates))
	for _, update := range updates {
		if update.ID <= 0 {
			continue
		}
		if _, exists := sortOrderByID[update.ID]; !exists {
			monitorIDs = append(monitorIDs, update.ID)
		}
		sortOrderByID[update.ID] = update.SortOrder
	}
	if len(monitorIDs) == 0 {
		return nil
	}

	var existingCount int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM channel_monitors WHERE id = ANY($1)`, pq.Array(monitorIDs)).Scan(&existingCount); err != nil {
		return err
	}
	if existingCount != len(monitorIDs) {
		return service.ErrChannelMonitorNotFound
	}

	args := make([]any, 0, len(monitorIDs)*2+1)
	caseClauses := make([]string, 0, len(monitorIDs))
	placeholder := 1
	for _, id := range monitorIDs {
		caseClauses = append(caseClauses, fmt.Sprintf("WHEN $%d THEN $%d", placeholder, placeholder+1))
		args = append(args, id, sortOrderByID[id])
		placeholder += 2
	}
	args = append(args, pq.Array(monitorIDs))
	query := fmt.Sprintf(`
		UPDATE channel_monitors
		SET sort_order = CASE id
			%s
			ELSE sort_order
		END,
			updated_at = NOW()
		WHERE id = ANY($%d)
	`, strings.Join(caseClauses, "\n\t\t\t"), placeholder)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != int64(len(monitorIDs)) {
		return service.ErrChannelMonitorNotFound
	}
	return nil
}

func cloneInt64PtrRepo(in *int64) *int64 {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}
