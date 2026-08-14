package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// GetRecentAccountFirstTokenStats returns user-observed TTFT samples for the scheduler dashboard.
func (r *usageLogRepository) GetRecentAccountFirstTokenStats(ctx context.Context, accountIDs []int64, groupID int64, since time.Time) (map[int64]service.AccountRecentFirstTokenStats, error) {
	result := make(map[int64]service.AccountRecentFirstTokenStats, len(accountIDs))
	if len(accountIDs) == 0 || groupID <= 0 {
		return result, nil
	}

	const query = `
		SELECT
			account_id,
			ROUND(AVG(first_token_ms)::numeric, 2)::float8 AS avg_first_token_ms,
			COUNT(first_token_ms)::bigint AS sample_count
		FROM usage_logs
		WHERE account_id = ANY($1)
		  AND group_id = $2
		  AND created_at >= $3
		  AND first_token_ms IS NOT NULL
		GROUP BY account_id
	`
	rows, err := r.sql.QueryContext(ctx, query, pq.Array(accountIDs), groupID, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var accountID int64
		var stats service.AccountRecentFirstTokenStats
		if err := rows.Scan(&accountID, &stats.AvgFirstTokenMs, &stats.SampleCount); err != nil {
			return nil, err
		}
		result[accountID] = stats
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *usageLogRepository) ListRecentGroupUsers(ctx context.Context, groupID int64, since time.Time, limit int) ([]service.GroupRecentUserUsage, error) {
	if limit <= 0 {
		limit = 1000
	}
	const query = `
		SELECT
			ul.user_id,
			COALESCE(u.email, '') AS email,
			COALESCE(u.username, '') AS username,
			COUNT(*)::bigint AS request_count,
			COALESCE(SUM(ul.actual_cost), 0) AS actual_cost,
			MAX(ul.created_at) AS last_used_at
		FROM usage_logs ul
		JOIN users u ON u.id = ul.user_id
		WHERE ul.group_id = $1
			AND ul.created_at >= $2
			AND ul.actual_cost > 0
			AND u.status = $3
			AND u.deleted_at IS NULL
		GROUP BY ul.user_id, u.email, u.username
		ORDER BY MAX(ul.created_at) DESC
		LIMIT $4
	`
	rows, err := r.sql.QueryContext(ctx, query, groupID, since, service.StatusActive, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	results := make([]service.GroupRecentUserUsage, 0)
	for rows.Next() {
		var row service.GroupRecentUserUsage
		if err := rows.Scan(&row.UserID, &row.Email, &row.Username, &row.RequestCount, &row.ActualCost, &row.LastUsedAt); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
