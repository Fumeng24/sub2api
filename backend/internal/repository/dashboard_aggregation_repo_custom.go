package repository

import (
	"context"
	"database/sql"
	"log"
	"time"
)

func (r *dashboardAggregationRepository) cleanupUsageBillingDedupErrorCustom(ctx context.Context, cutoff time.Time, cause error) (sql.Result, error) {
	if !isUndefinedTableError(cause) {
		return nil, cause
	}
	log.Printf("[DashboardAggregation] usage_billing_dedup_archive missing; deleting old dedup keys without archive")
	return r.sql.ExecContext(ctx, `
		WITH victims AS (
			SELECT ctid
			FROM usage_billing_dedup
			WHERE created_at < $1
			LIMIT $2
		)
		DELETE FROM usage_billing_dedup
		WHERE ctid IN (SELECT ctid FROM victims)
	`, cutoff.UTC(), usageBillingDedupCleanupBatchSize)
}
