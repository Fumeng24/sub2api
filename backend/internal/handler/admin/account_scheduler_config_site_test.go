package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountHandlerUpdateSchedulerConfigUsesNarrowUpdateSurface(t *testing.T) {
	adminSvc := newStubAdminService()
	router := setupAccountMixedChannelRouter(adminSvc)
	accountHandler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.PUT("/api/v1/admin/accounts/:id/scheduler-config", accountHandler.UpdateSchedulerConfig)

	body, _ := json.Marshal(map[string]any{
		"concurrency":     7,
		"load_factor":     11,
		"rate_multiplier": 0.2,
		"manual_rate":     nil,
		"rate_scale":      0.15,
		"credentials":     map[string]any{"api_key": "must-not-be-used"},
		"extra":           map[string]any{"model_rate_limits": "must-not-be-used"},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/accounts/3/scheduler-config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.updatedAccounts, 1)
	update := adminSvc.updatedAccounts[0]
	require.NotNil(t, update.Concurrency)
	require.Equal(t, 7, *update.Concurrency)
	require.NotNil(t, update.LoadFactor)
	require.Equal(t, 11, *update.LoadFactor)
	require.NotNil(t, update.RateMultiplier)
	require.InDelta(t, 0.2, *update.RateMultiplier, 1e-12)
	require.Empty(t, update.Credentials)
	require.Nil(t, update.Extra)

	require.Len(t, adminSvc.updatedExtra, 1)
	require.Contains(t, adminSvc.updatedExtra[0], "manual_rate")
	require.Nil(t, adminSvc.updatedExtra[0]["manual_rate"])
	require.Equal(t, 0.15, adminSvc.updatedExtra[0]["rate_scale"])
}
