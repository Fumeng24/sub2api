package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestCopyAccountUsesExportImportAccountFields(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()
	h := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/:id/copy", h.Copy)
	service.SetDefaultIdempotencyCoordinator(nil)

	proxyID := int64(11)
	rateMultiplier := 0.75
	expiresAt := time.Unix(1893456000, 0).UTC()
	loadFactor := 9
	adminSvc.accounts = []service.Account{
		{
			ID:         21,
			Name:       "upstream",
			Notes:      siteStringPtr("note"),
			Platform:   service.PlatformOpenAI,
			Type:       service.AccountTypeUpstream,
			ProxyID:    &proxyID,
			Status:     service.StatusError,
			LoadFactor: &loadFactor,
			Credentials: map[string]any{
				"api_key": "sk-secret",
				"nested":  map[string]any{"token": "raw"},
			},
			Extra: map[string]any{
				"upstream_group_id": float64(123),
			},
			Concurrency:        7,
			Priority:           42,
			RateMultiplier:     &rateMultiplier,
			ExpiresAt:          &expiresAt,
			AutoPauseOnExpired: false,
			GroupIDs:           []int64{100, 200},
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/21/copy", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdAccounts, 1)
	input := adminSvc.createdAccounts[0]
	require.Equal(t, "upstream", input.Name)
	require.Equal(t, "note", *input.Notes)
	require.Equal(t, service.PlatformOpenAI, input.Platform)
	require.Equal(t, service.AccountTypeUpstream, input.Type)
	require.Equal(t, proxyID, *input.ProxyID)
	require.Equal(t, "sk-secret", input.Credentials["api_key"])
	require.Equal(t, map[string]any{"token": "raw"}, input.Credentials["nested"])
	require.Equal(t, float64(123), input.Extra["upstream_group_id"])
	require.Equal(t, 7, input.Concurrency)
	require.Equal(t, 42, input.Priority)
	require.Equal(t, rateMultiplier, *input.RateMultiplier)
	require.Equal(t, expiresAt.Unix(), *input.ExpiresAt)
	require.False(t, *input.AutoPauseOnExpired)
	require.True(t, input.SkipDefaultGroupBind)
	require.Nil(t, input.GroupIDs)
	require.Nil(t, input.LoadFactor, "copy should match data import/export fields and not copy load_factor")
}

func siteStringPtr(value string) *string {
	return &value
}
