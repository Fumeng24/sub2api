package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

// GetEndpointStatsWithFilters returns inbound endpoint stats using the shared usage filter shape.
func (s *UsageService) GetEndpointStatsWithFilters(ctx context.Context, startTime, endTime time.Time, filters usagestats.UsageLogFilters) ([]usagestats.EndpointStat, error) {
	type endpointStatsWithUsageFiltersRepo interface {
		GetEndpointStatsWithUsageFilters(context.Context, time.Time, time.Time, usagestats.UsageLogFilters) ([]usagestats.EndpointStat, error)
	}
	if filterRepo, ok := s.usageRepo.(endpointStatsWithUsageFiltersRepo); ok {
		stats, err := filterRepo.GetEndpointStatsWithUsageFilters(ctx, startTime, endTime, filters)
		if err != nil {
			return nil, fmt.Errorf("get endpoint stats with filters: %w", err)
		}
		return stats, nil
	}

	stats, err := s.usageRepo.GetEndpointStatsWithFilters(ctx, startTime, endTime, filters.UserID, filters.APIKeyID, filters.AccountID, filters.GroupID, filters.Model, filters.RequestType, filters.Stream, filters.BillingType)
	if err != nil {
		return nil, fmt.Errorf("get endpoint stats with filters: %w", err)
	}
	return stats, nil
}
