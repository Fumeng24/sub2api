package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type usageDashboardExtensionRepo struct {
	service.UsageLogRepository
	groupCalled     bool
	endpointCalled  bool
	groupStart      time.Time
	groupEnd        time.Time
	endpointStart   time.Time
	endpointEnd     time.Time
	groupFilters    usagestats.UsageLogFilters
	endpointFilters usagestats.UsageLogFilters
}

func (s *usageDashboardExtensionRepo) GetGroupStatsWithUsageFilters(
	_ context.Context,
	startTime, endTime time.Time,
	filters usagestats.UsageLogFilters,
) ([]usagestats.GroupStat, error) {
	s.groupCalled = true
	s.groupStart = startTime
	s.groupEnd = endTime
	s.groupFilters = filters
	return []usagestats.GroupStat{{GroupID: 9, GroupName: "Pro", Requests: 3}}, nil
}

func (s *usageDashboardExtensionRepo) GetEndpointStatsWithUsageFilters(
	_ context.Context,
	startTime, endTime time.Time,
	filters usagestats.UsageLogFilters,
) ([]usagestats.EndpointStat, error) {
	s.endpointCalled = true
	s.endpointStart = startTime
	s.endpointEnd = endTime
	s.endpointFilters = filters
	return []usagestats.EndpointStat{{Endpoint: "/v1/responses", Requests: 3}}, nil
}

type usageDashboardExtensionAPIKeyRepo struct {
	service.APIKeyRepository
	keys map[int64]*service.APIKey
}

func (s *usageDashboardExtensionAPIKeyRepo) GetByID(_ context.Context, id int64) (*service.APIKey, error) {
	key, ok := s.keys[id]
	if !ok {
		return nil, service.ErrAPIKeyNotFound
	}
	clone := *key
	return &clone, nil
}

func newUsageDashboardExtensionRouter(usageRepo *usageDashboardExtensionRepo, apiKeyRepo *usageDashboardExtensionAPIKeyRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(usageRepo, nil, nil, nil)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo, nil, nil, nil, nil, nil, nil)
	handler := NewUsageHandler(usageSvc, apiKeySvc, nil, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	router.GET("/usage/dashboard/groups", handler.DashboardGroups)
	router.GET("/usage/dashboard/endpoints", handler.DashboardEndpoints)
	return router
}

func TestUsageDashboardExtensionsUseSharedFiltersAcrossDST(t *testing.T) {
	const query = "?api_key_id=7&group_id=9&model=gpt-5.6&request_type=ws_v2&stream=invalid&billing_type=2&billing_mode=image&start_date=2026-03-08&end_date=2026-03-08&timezone=America%2FNew_York"

	for _, path := range []string{"/usage/dashboard/groups", "/usage/dashboard/endpoints"} {
		t.Run(path, func(t *testing.T) {
			usageRepo := &usageDashboardExtensionRepo{}
			apiKeyRepo := &usageDashboardExtensionAPIKeyRepo{keys: map[int64]*service.APIKey{
				7: {ID: 7, UserID: 42, Status: service.StatusAPIKeyActive},
			}}
			router := newUsageDashboardExtensionRouter(usageRepo, apiKeyRepo)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path+query, nil))

			require.Equal(t, http.StatusOK, rec.Code)
			require.Contains(t, rec.Body.String(), `"start_date":"2026-03-08"`)
			require.Contains(t, rec.Body.String(), `"end_date":"2026-03-08"`)

			var startTime, endTime time.Time
			var filters usagestats.UsageLogFilters
			if path == "/usage/dashboard/groups" {
				require.True(t, usageRepo.groupCalled)
				startTime, endTime, filters = usageRepo.groupStart, usageRepo.groupEnd, usageRepo.groupFilters
			} else {
				require.True(t, usageRepo.endpointCalled)
				startTime, endTime, filters = usageRepo.endpointStart, usageRepo.endpointEnd, usageRepo.endpointFilters
			}

			require.Equal(t, 23*time.Hour, endTime.Sub(startTime))
			require.Equal(t, int64(42), filters.UserID)
			require.Equal(t, int64(7), filters.APIKeyID)
			require.Equal(t, int64(9), filters.GroupID)
			require.Equal(t, "gpt-5.6", filters.Model)
			require.Equal(t, usagestats.ModelSourceRequested, filters.ModelFilterSource)
			require.NotNil(t, filters.RequestType)
			require.Equal(t, int16(service.RequestTypeWSV2), *filters.RequestType)
			require.Nil(t, filters.Stream)
			require.NotNil(t, filters.BillingType)
			require.Equal(t, int8(2), *filters.BillingType)
			require.Equal(t, "image", filters.BillingMode)
		})
	}
}

func TestUsageDashboardExtensionsRejectCrossUserAPIKey(t *testing.T) {
	usageRepo := &usageDashboardExtensionRepo{}
	apiKeyRepo := &usageDashboardExtensionAPIKeyRepo{keys: map[int64]*service.APIKey{
		7: {ID: 7, UserID: 99, Status: service.StatusAPIKeyActive},
	}}
	router := newUsageDashboardExtensionRouter(usageRepo, apiKeyRepo)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/usage/dashboard/endpoints?api_key_id=7", nil))

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.False(t, usageRepo.endpointCalled)
}

func TestUsageDashboardExtensionsRejectInvalidFilters(t *testing.T) {
	for _, query := range []string{
		"start_date=not-a-date",
		"billing_mode=unsupported",
	} {
		t.Run(query, func(t *testing.T) {
			usageRepo := &usageDashboardExtensionRepo{}
			apiKeyRepo := &usageDashboardExtensionAPIKeyRepo{keys: map[int64]*service.APIKey{}}
			router := newUsageDashboardExtensionRouter(usageRepo, apiKeyRepo)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/usage/dashboard/groups?"+query, nil))

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.False(t, usageRepo.groupCalled)
		})
	}
}
