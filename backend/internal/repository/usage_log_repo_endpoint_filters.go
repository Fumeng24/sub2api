package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

// GetEndpointStatsWithUsageFilters returns inbound endpoint statistics using the shared filter shape.
func (r *usageLogRepository) GetEndpointStatsWithUsageFilters(ctx context.Context, startTime, endTime time.Time, filters usagestats.UsageLogFilters) ([]usagestats.EndpointStat, error) {
	return r.getEndpointStatsByColumnWithFilters(
		ctx,
		"inbound_endpoint",
		startTime,
		endTime,
		filters.UserID,
		filters.APIKeyID,
		filters.AccountID,
		filters.GroupID,
		filters.Model,
		filters.ModelFilterSource,
		filters.RequestType,
		filters.Stream,
		filters.BillingType,
		filters.BillingMode,
	)
}
