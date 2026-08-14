package admin

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type CopyAccountRequest struct {
	Name string `json:"name"`
}

// Copy creates a new account using the same field semantics as exporting one
// account and importing it back into the current instance.
func (h *AccountHandler) Copy(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	var req CopyAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	payload := struct {
		AccountID int64              `json:"account_id"`
		Request   CopyAccountRequest `json:"request"`
	}{AccountID: accountID, Request: req}

	executeAdminIdempotentJSON(c, "admin.accounts.copy", payload, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		source, err := h.adminService.GetAccount(ctx, accountID)
		if err != nil {
			return nil, err
		}
		created, err := h.createAccountCopy(ctx, source, req)
		if err != nil {
			return nil, err
		}
		return h.buildAccountResponseWithRuntime(ctx, created), nil
	})
}

func (h *AccountHandler) createAccountCopy(ctx context.Context, source *service.Account, req CopyAccountRequest) (*service.Account, error) {
	if source == nil {
		return nil, service.ErrAccountNotFound
	}

	var expiresAt *int64
	if source.ExpiresAt != nil {
		value := source.ExpiresAt.Unix()
		expiresAt = &value
	}
	autoPauseOnExpired := source.AutoPauseOnExpired
	item := DataAccount{
		Name:               defaultCopyAccountName(source.Name, req.Name),
		Notes:              source.Notes,
		Platform:           source.Platform,
		Type:               source.Type,
		Credentials:        cloneDataMap(source.Credentials),
		Extra:              cloneDataMap(source.Extra),
		Concurrency:        source.Concurrency,
		Priority:           source.Priority,
		RateMultiplier:     cloneFloat64Ptr(source.RateMultiplier),
		ExpiresAt:          expiresAt,
		AutoPauseOnExpired: &autoPauseOnExpired,
	}
	enrichCredentialsFromIDToken(&item)
	if err := validateDataAccount(item); err != nil {
		return nil, err
	}
	return h.adminService.CreateAccount(ctx, &service.CreateAccountInput{
		Name: item.Name, Notes: item.Notes, Platform: item.Platform, Type: item.Type,
		Credentials: item.Credentials, Extra: item.Extra, ProxyID: cloneInt64Ptr(source.ProxyID),
		Concurrency: item.Concurrency, Priority: item.Priority, RateMultiplier: item.RateMultiplier,
		GroupIDs: nil, ExpiresAt: item.ExpiresAt, AutoPauseOnExpired: item.AutoPauseOnExpired,
		SkipDefaultGroupBind: true,
	})
}

func defaultCopyAccountName(sourceName, requestedName string) string {
	name := strings.TrimSpace(requestedName)
	if name == "" {
		name = strings.TrimSpace(sourceName)
		if name == "" {
			name = "account"
		}
	}
	if len(name) > 100 {
		name = name[:100]
	}
	return name
}

func cloneDataMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	out := make(map[string]any, len(src))
	for key, value := range src {
		out[key] = cloneDataValue(value)
	}
	return out
}

func cloneDataValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cloneDataMap(v)
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = cloneDataValue(v[i])
		}
		return out
	default:
		return value
	}
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
