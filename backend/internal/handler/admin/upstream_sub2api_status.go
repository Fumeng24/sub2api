package admin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/util/httputil"

	"github.com/gin-gonic/gin"
)

const (
	upstreamSub2APIStatusTTL        = 2 * time.Minute
	upstreamSub2APITokenSkew        = 30 * time.Second
	upstreamSub2APIPageSize         = 100
	upstreamSub2APIProbeConcurrency = 6
	upstreamNewAPIQuotaPerUnit      = 500000.0
	upstreamNewAPISessionTTL        = 30 * time.Minute
)

var errUpstreamSub2APITwoFactor = errors.New("upstream panel account requires two-factor authentication")
var errUpstreamSub2APICloudflareBlocked = errors.New("upstream panel blocked by Cloudflare")

type upstreamSub2APIHTTPClientContextKey struct{}
type upstreamSub2APIProxyURLContextKey struct{}

type upstreamSub2APIStatusClient struct {
	httpClient *http.Client

	mu           sync.Mutex
	statusCache  map[string]upstreamSub2APIStatusCacheEntry
	tokenCache   map[string]upstreamSub2APITokenCacheEntry
	sessionCache map[string]upstreamNewAPISessionCacheEntry
}

type upstreamSub2APICloudflareBlockedError struct {
	statusCode int
	rayID      string
	proxyUsed  bool
}

type upstreamSub2APIStatusCacheEntry struct {
	status  UpstreamSub2APIAccountStatus
	expires time.Time
}

type upstreamSub2APITokenCacheEntry struct {
	token   string
	expires time.Time
}

type upstreamNewAPISessionCacheEntry struct {
	userID  int64
	cookies []*http.Cookie
	expires time.Time
}

type UpstreamSub2APIAccountStatus struct {
	AccountID     int64     `json:"account_id"`
	AccountName   string    `json:"account_name"`
	LocalPlatform string    `json:"local_platform"`
	BaseURL       string    `json:"base_url"`
	UpstreamKind  string    `json:"upstream_kind,omitempty"`
	ProbeSource   string    `json:"probe_source,omitempty"`
	ProxyUsed     bool      `json:"proxy_used"`
	Status        string    `json:"status"`
	Message       string    `json:"message,omitempty"`
	FetchedAt     time.Time `json:"fetched_at"`
	Cached        bool      `json:"cached"`
	Stale         bool      `json:"stale,omitempty"`

	UserBalance                          *float64 `json:"user_balance,omitempty"`
	KeyRemaining                         *float64 `json:"key_remaining,omitempty"`
	BalanceUnit                          string   `json:"balance_unit,omitempty"`
	UsageMode                            string   `json:"usage_mode,omitempty"`
	UsagePlanName                        string   `json:"usage_plan_name,omitempty"`
	UpstreamKeyID                        *int64   `json:"upstream_key_id,omitempty"`
	UpstreamKeyName                      string   `json:"upstream_key_name,omitempty"`
	UpstreamGroupID                      *int64   `json:"upstream_group_id,omitempty"`
	UpstreamGroupName                    string   `json:"upstream_group_name,omitempty"`
	UpstreamGroupPlatform                string   `json:"upstream_group_platform,omitempty"`
	UpstreamGroupDefaultRateMultiplier   *float64 `json:"upstream_group_default_rate_multiplier,omitempty"`
	UpstreamGroupEffectiveRateMultiplier *float64 `json:"upstream_group_effective_rate_multiplier,omitempty"`
}

type upstreamSub2APILoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Requires2FA  bool   `json:"requires_2fa"`
}

type upstreamSub2APIUser struct {
	ID      int64   `json:"id"`
	Email   string  `json:"email"`
	Balance float64 `json:"balance"`
}

type upstreamSub2APIKey struct {
	ID      int64                 `json:"id"`
	Key     string                `json:"key"`
	Name    string                `json:"name"`
	GroupID *int64                `json:"group_id"`
	Group   *upstreamSub2APIGroup `json:"group"`
}

type upstreamSub2APIGroup struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Platform       string  `json:"platform"`
	RateMultiplier float64 `json:"rate_multiplier"`
}

type upstreamSub2APIKeysPage struct {
	Items    []upstreamSub2APIKey `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	Pages    int                  `json:"pages"`
}

type upstreamSub2APIUsage struct {
	Mode      string   `json:"mode"`
	PlanName  string   `json:"planName"`
	Remaining *float64 `json:"remaining"`
	Balance   *float64 `json:"balance"`
	Unit      string   `json:"unit"`
}

type upstreamNewAPILoginUser struct {
	ID         int64 `json:"id"`
	Require2FA bool  `json:"require_2fa"`
}

type upstreamNewAPIUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Group     string `json:"group"`
	Quota     int64  `json:"quota"`
	UsedQuota int64  `json:"used_quota"`
}

type upstreamNewAPIToken struct {
	ID                 int64   `json:"id"`
	UserID             int64   `json:"user_id"`
	Status             int     `json:"status"`
	Name               string  `json:"name"`
	ExpiredTime        int64   `json:"expired_time"`
	RemainQuota        int64   `json:"remain_quota"`
	UsedQuota          int64   `json:"used_quota"`
	UnlimitedQuota     bool    `json:"unlimited_quota"`
	ModelLimitsEnabled bool    `json:"model_limits_enabled"`
	ModelLimits        string  `json:"model_limits"`
	AllowIPs           *string `json:"allow_ips"`
	Group              string  `json:"group"`
	CrossGroupRetry    bool    `json:"cross_group_retry"`
}

// upstreamNewAPITokenUsage is returned by New API's TokenAuthReadOnly route.
// Unlike dashboard endpoints, it can be queried with the relay API key alone.
type upstreamNewAPITokenUsage struct {
	Object             string `json:"object"`
	Name               string `json:"name"`
	TotalGranted       int64  `json:"total_granted"`
	TotalUsed          int64  `json:"total_used"`
	TotalAvailable     int64  `json:"total_available"`
	UnlimitedQuota     bool   `json:"unlimited_quota"`
	ModelLimitsEnabled bool   `json:"model_limits_enabled"`
	ExpiresAt          int64  `json:"expires_at"`
}

type upstreamNewAPITokensPage struct {
	Items    []upstreamNewAPIToken `json:"items"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

type upstreamNewAPIGroupInfo struct {
	Ratio json.RawMessage `json:"ratio"`
	Desc  string          `json:"desc"`
}

func newUpstreamSub2APIStatusClient() *upstreamSub2APIStatusClient {
	return &upstreamSub2APIStatusClient{
		httpClient:   &http.Client{Timeout: 8 * time.Second},
		statusCache:  make(map[string]upstreamSub2APIStatusCacheEntry),
		tokenCache:   make(map[string]upstreamSub2APITokenCacheEntry),
		sessionCache: make(map[string]upstreamNewAPISessionCacheEntry),
	}
}

func (e *upstreamSub2APICloudflareBlockedError) Error() string {
	message := "上游面板被 Cloudflare 拦截，请给该账号配置可用代理，或联系上游将本机出口 IP 加入白名单"
	if e != nil && e.proxyUsed {
		message = "上游面板通过账号代理访问时仍被 Cloudflare 拦截，请更换可用代理，或联系上游将代理出口 IP 加入白名单"
	}
	if e != nil && strings.TrimSpace(e.rayID) != "" {
		message += " (cf-ray: " + strings.TrimSpace(e.rayID) + ")"
	}
	return message
}

func (e *upstreamSub2APICloudflareBlockedError) Is(target error) bool {
	return target == errUpstreamSub2APICloudflareBlocked
}

// GetUpstreamSub2APIStatus returns upstream panel metadata for the requested accounts.
// New API is checked with its API-key-only usage endpoint first, then configured panel
// credentials enrich a successful check with the real wallet balance and key group rate.
// Panel login remains the fallback when the public endpoint is unavailable. Sub2API
// still uses its official authenticated panel endpoints.
func (h *AccountHandler) GetUpstreamSub2APIStatus(c *gin.Context) {
	accountIDs, err := parseUpstreamSub2APIAccountIDs(c.Query("account_ids"))
	if err != nil {
		response.BadRequest(c, "Invalid account_ids")
		return
	}
	if len(accountIDs) == 0 {
		response.BadRequest(c, "account_ids is required")
		return
	}

	accounts, err := h.adminService.GetAccountsByIDs(c.Request.Context(), accountIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	byID := make(map[int64]*service.Account, len(accounts))
	for _, account := range accounts {
		if account != nil {
			byID[account.ID] = account
		}
	}

	force := strings.EqualFold(c.Query("force"), "true") || c.Query("force") == "1"
	targets := make([]*service.Account, 0, len(accounts))
	for _, accountID := range accountIDs {
		account := byID[accountID]
		if !shouldProbeUpstreamSub2APIAccount(account) {
			continue
		}
		targets = append(targets, account)
	}
	result := make([]UpstreamSub2APIAccountStatus, len(targets))
	var wg sync.WaitGroup
	sem := make(chan struct{}, upstreamSub2APIProbeConcurrency)
	for i, account := range targets {
		wg.Add(1)
		go func(i int, account *service.Account) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			result[i] = h.upstreamSub2API.ProbeAccount(c.Request.Context(), account, force)
		}(i, account)
	}
	wg.Wait()

	// Persist only values returned by this probe. Missing fields explicitly clear
	// earlier snapshots so callers never mistake historical data for current data.
	h.persistUpstreamMetadata(c.Request.Context(), result)

	response.Success(c, result)
}

// persistUpstreamMetadata mirrors the latest probe into account.extra. Values that
// were not returned by this attempt are cleared instead of retained.
func (h *AccountHandler) persistUpstreamMetadata(ctx context.Context, results []UpstreamSub2APIAccountStatus) {
	if h.adminService == nil {
		return
	}
	now := time.Now().UTC()
	for i := range results {
		st := results[i]
		if st.AccountID <= 0 {
			continue
		}
		updates := map[string]any{
			"upstream_rate_cached":        nil,
			"upstream_rate_cached_at":     nil,
			"upstream_wallet_cached":      nil,
			"upstream_wallet_cached_at":   nil,
			"upstream_wallet_cached_unit": nil,
		}
		if st.Status == "ok" {
			rate := st.UpstreamGroupEffectiveRateMultiplier
			if rate == nil {
				rate = st.UpstreamGroupDefaultRateMultiplier
			}
			if rate != nil {
				updates["upstream_rate_cached_at"] = now.Format(time.RFC3339)
				updates["upstream_rate_cached"] = *rate
			}
			if st.ProbeSource == "panel_login" && st.UserBalance != nil {
				updates["upstream_wallet_cached"] = *st.UserBalance
				updates["upstream_wallet_cached_at"] = now.Format(time.RFC3339)
				if unit := strings.TrimSpace(st.BalanceUnit); unit != "" {
					updates["upstream_wallet_cached_unit"] = unit
				}
			}
		}
		if err := h.adminService.UpdateAccountExtra(ctx, st.AccountID, updates); err != nil {
			logger.LegacyPrintf("handler.upstream_sub2api_status", "persist upstream metadata failed: account=%d err=%v", st.AccountID, err)
		}
	}
}

func shouldProbeUpstreamSub2APIAccount(account *service.Account) bool {
	if account == nil {
		return false
	}
	if account.Type != service.AccountTypeAPIKey && account.Type != service.AccountTypeUpstream {
		return false
	}
	if strings.TrimSpace(account.GetCredential("api_key")) == "" {
		return false
	}
	email := strings.TrimSpace(account.GetCredential("upstream_sub2api_email"))
	password := strings.TrimSpace(account.GetCredential("upstream_sub2api_password"))
	if email != "" && password != "" {
		return true
	}
	// An explicitly configured New API account can use /api/usage/token/
	// without dashboard credentials. Avoid probing unrelated API-key accounts
	// that have neither panel credentials nor an explicit New API selection.
	return normalizeUpstreamPanelType(account.GetCredential("upstream_panel_type")) == "newapi"
}

func parseUpstreamSub2APIAccountIDs(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int64, 0, len(parts))
	seen := make(map[int64]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid account id %q", part)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func (p *upstreamSub2APIStatusClient) ProbeAccount(ctx context.Context, account *service.Account, force bool) UpstreamSub2APIAccountStatus {
	now := time.Now().UTC()
	status := UpstreamSub2APIAccountStatus{
		AccountID:     account.ID,
		AccountName:   account.Name,
		LocalPlatform: account.Platform,
		Status:        "error",
		FetchedAt:     now,
	}

	root, err := normalizeUpstreamSub2APIBaseURL(account.GetCredential("base_url"))
	if err != nil {
		status.Status = "invalid_base_url"
		status.Message = err.Error()
		return status
	}
	status.BaseURL = root

	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	email := strings.TrimSpace(account.GetCredential("upstream_sub2api_email"))
	password := strings.TrimSpace(account.GetCredential("upstream_sub2api_password"))
	panelType := normalizeUpstreamPanelType(account.GetCredential("upstream_panel_type"))
	if apiKey == "" {
		status.Status = "missing_api_key"
		status.Message = "upstream api key is empty"
		return status
	}
	if (email == "" || password == "") && panelType != "newapi" {
		status.Status = "missing_login"
		status.Message = "upstream panel login account/password is incomplete"
		return status
	}

	probeCtx, proxyURL, err := p.contextForAccount(ctx, account)
	if err != nil {
		status.Status = "request_failed"
		status.Message = "account proxy is invalid: " + err.Error()
		return status
	}
	status.ProxyUsed = proxyURL != ""

	cacheKey := upstreamSub2APIStatusCacheKey(account.ID, root, apiKey, email, password, panelType, proxyURL)
	if !force {
		if cached, ok := p.getCachedStatus(cacheKey, now); ok {
			return cached
		}
	}

	probed := p.probeAccountFresh(probeCtx, status, root, apiKey, email, password, panelType)
	p.setCachedStatus(cacheKey, probed, now.Add(upstreamSub2APIStatusTTL))
	return probed
}

func (p *upstreamSub2APIStatusClient) probeAccountFresh(ctx context.Context, status UpstreamSub2APIAccountStatus, root, apiKey, email, password, panelType string) UpstreamSub2APIAccountStatus {
	switch panelType {
	case "sub2api":
		status.UpstreamKind = "sub2api"
		return p.probeSub2APIFresh(ctx, status, root, apiKey, email, password)
	case "newapi":
		status.UpstreamKind = "newapi"
		return p.probeNewAPIFresh(ctx, status, root, apiKey, email, password)
	default:
		newAPIStatus := status
		newAPIStatus.UpstreamKind = "newapi"
		publicProbe, publicFallbackAllowed := p.probeNewAPIWithAPIKey(ctx, newAPIStatus, root, apiKey)
		if publicProbe.Status == "ok" {
			metadataProbe := p.enrichNewAPIPublicProbe(ctx, publicProbe, root, apiKey, email, password)
			if hasUpstreamWalletAndRate(metadataProbe) || !hasNewAPIPanelCredentials(email, password) {
				return metadataProbe
			}

			// Before the direct New API probe was added, auto detection tried the
			// Sub2API panel protocol first. Some hybrid upstreams expose the public
			// New API key endpoint but only provide usable wallet/rate metadata via
			// the Sub2API panel endpoints. Keep the direct probe as the primary
			// liveness check, then use this narrow metadata fallback when needed.
			sub2APIStatus := status
			sub2APIStatus.UpstreamKind = "sub2api"
			sub2APIProbe := p.probeSub2APIFresh(ctx, sub2APIStatus, root, apiKey, email, password)
			if sub2APIProbe.Status == "ok" {
				return mergeNewAPIPublicUsage(sub2APIProbe, publicProbe)
			}
			return metadataProbe
		}
		if !publicFallbackAllowed {
			return publicProbe
		}

		sub2APIStatus := status
		sub2APIStatus.UpstreamKind = "sub2api"
		probed := p.probeSub2APIFresh(ctx, sub2APIStatus, root, apiKey, email, password)
		if probed.Status == "ok" || probed.Status == "two_factor_required" {
			return probed
		}
		probedNewAPI := p.probeNewAPIWithLoginFresh(ctx, newAPIStatus, root, apiKey, email, password)
		if probedNewAPI.Status == "ok" || probedNewAPI.Status == "two_factor_required" {
			return probedNewAPI
		}
		probed.Message = strings.TrimSpace(fmt.Sprintf("public newapi: %s; sub2api: %s; newapi login: %s", publicProbe.Message, probed.Message, probedNewAPI.Message))
		return probed
	}
}

func (p *upstreamSub2APIStatusClient) probeSub2APIFresh(ctx context.Context, status UpstreamSub2APIAccountStatus, root, apiKey, email, password string) UpstreamSub2APIAccountStatus {
	status.ProbeSource = "panel_login"
	user, userErr := p.fetchCurrentUser(ctx, root, email, password)
	if userErr != nil {
		status.Status = upstreamSub2APILoginStatus(userErr)
		status.Message = userErr.Error()
		return status
	}
	if user != nil {
		status.UserBalance = &user.Balance
		status.BalanceUnit = "USD"
	}

	groupsByID, groupsErr := p.fetchAvailableGroups(ctx, root, email, password)
	if groupsErr != nil {
		status.Status = upstreamSub2APILoginStatus(groupsErr)
		status.Message = groupsErr.Error()
		return status
	}
	rates, ratesErr := p.fetchGroupRates(ctx, root, email, password)
	if ratesErr != nil {
		status.Status = upstreamSub2APILoginStatus(ratesErr)
		status.Message = ratesErr.Error()
		return status
	}

	matchedKey, keyErr := p.findAPIKey(ctx, root, email, password, apiKey)
	if keyErr != nil {
		status.Status = upstreamSub2APILoginStatus(keyErr)
		status.Message = keyErr.Error()
		return status
	}
	if matchedKey == nil {
		status.Status = "key_not_found"
		status.Message = "upstream api key was not found in the logged-in account"
		return status
	}

	status.UpstreamKeyID = &matchedKey.ID
	status.UpstreamKeyName = matchedKey.Name
	status.UpstreamGroupID = matchedKey.GroupID

	var group *upstreamSub2APIGroup
	if matchedKey.Group != nil {
		group = matchedKey.Group
	}
	if group == nil && matchedKey.GroupID != nil {
		group = groupsByID[*matchedKey.GroupID]
	}
	if group != nil {
		status.UpstreamGroupName = group.Name
		status.UpstreamGroupPlatform = group.Platform
		defaultRate := group.RateMultiplier
		status.UpstreamGroupDefaultRateMultiplier = &defaultRate
		effectiveRate := defaultRate
		if matchedKey.GroupID != nil {
			if rate, ok := rates[*matchedKey.GroupID]; ok {
				effectiveRate = rate
			}
		}
		status.UpstreamGroupEffectiveRateMultiplier = &effectiveRate
	}

	if usage, err := p.fetchUsage(ctx, root, apiKey); err == nil && usage != nil {
		status.UsageMode = usage.Mode
		status.UsagePlanName = usage.PlanName
		if usage.Unit != "" {
			status.BalanceUnit = usage.Unit
		}
		if usage.Remaining != nil {
			status.KeyRemaining = usage.Remaining
		}
		if status.UserBalance == nil && usage.Balance != nil {
			status.UserBalance = usage.Balance
		}
	}

	status.Status = "ok"
	return status
}

func (p *upstreamSub2APIStatusClient) probeNewAPIFresh(ctx context.Context, status UpstreamSub2APIAccountStatus, root, apiKey, username, password string) UpstreamSub2APIAccountStatus {
	publicProbe, fallbackAllowed := p.probeNewAPIWithAPIKey(ctx, status, root, apiKey)
	if publicProbe.Status == "ok" {
		return p.enrichNewAPIPublicProbe(ctx, publicProbe, root, apiKey, username, password)
	}
	if !fallbackAllowed {
		return publicProbe
	}
	if !hasNewAPIPanelCredentials(username, password) {
		publicProbe.Status = "public_api_unavailable"
		publicProbe.Message = "New API API-key usage endpoint is unavailable; configure panel login credentials to enable fallback"
		return publicProbe
	}

	loginProbe := p.probeNewAPIWithLoginFresh(ctx, status, root, apiKey, username, password)
	if loginProbe.Status != "ok" {
		loginProbe.Message = strings.TrimSpace(fmt.Sprintf("public api: %s; login fallback: %s", publicProbe.Message, loginProbe.Message))
	}
	return loginProbe
}

func hasNewAPIPanelCredentials(username, password string) bool {
	return strings.TrimSpace(username) != "" && strings.TrimSpace(password) != ""
}

// enrichNewAPIPublicProbe keeps the public key check as the first step, but does not
// treat it as complete metadata. New API's public endpoint has no user wallet or
// key-to-group mapping, so configured panel credentials are required for those fields.
func (p *upstreamSub2APIStatusClient) enrichNewAPIPublicProbe(ctx context.Context, publicProbe UpstreamSub2APIAccountStatus, root, apiKey, username, password string) UpstreamSub2APIAccountStatus {
	if !hasNewAPIPanelCredentials(username, password) {
		return publicProbe
	}

	loginProbe := p.probeNewAPIWithLoginFresh(ctx, publicProbe, root, apiKey, username, password)
	if loginProbe.Status == "ok" {
		return mergeNewAPIPublicUsage(loginProbe, publicProbe)
	}

	// The key was already verified by the public endpoint. Keep that usable status,
	// but make the missing wallet/group metadata explicit instead of showing stale
	// cached rates or pretending that the key quota is the wallet balance.
	loginMessage := strings.TrimSpace(loginProbe.Message)
	if loginMessage == "" {
		loginMessage = "panel login did not return wallet or group metadata"
	}
	publicProbe.Message = "API key verified; wallet/group metadata unavailable: " + loginMessage
	return publicProbe
}

// mergeNewAPIPublicUsage keeps the public API-key quota semantics while taking
// wallet and group metadata from an authenticated panel probe.
func mergeNewAPIPublicUsage(metadataProbe, publicProbe UpstreamSub2APIAccountStatus) UpstreamSub2APIAccountStatus {
	if publicProbe.UsageMode != "" {
		metadataProbe.UsageMode = publicProbe.UsageMode
	}
	if publicProbe.UsageMode == "unlimited" {
		metadataProbe.KeyRemaining = nil
	} else if publicProbe.KeyRemaining != nil {
		metadataProbe.KeyRemaining = publicProbe.KeyRemaining
	}
	if publicProbe.UpstreamKeyName != "" {
		metadataProbe.UpstreamKeyName = publicProbe.UpstreamKeyName
	}
	return metadataProbe
}

func hasUpstreamWalletAndRate(status UpstreamSub2APIAccountStatus) bool {
	if status.Status != "ok" || status.UserBalance == nil {
		return false
	}
	return status.UpstreamGroupEffectiveRateMultiplier != nil || status.UpstreamGroupDefaultRateMultiplier != nil
}

func (p *upstreamSub2APIStatusClient) probeNewAPIWithAPIKey(ctx context.Context, status UpstreamSub2APIAccountStatus, root, apiKey string) (UpstreamSub2APIAccountStatus, bool) {
	status.ProbeSource = "api_key"
	var usage upstreamNewAPITokenUsage
	httpStatus, _, err := p.doNewAPIJSON(
		ctx,
		http.MethodGet,
		joinUpstreamSub2APIURL(root, "/api/usage/token/"),
		nil,
		apiKey,
		nil,
		&usage,
	)
	if err == nil && usage.Object == "token_usage" {
		status.UpstreamKeyName = strings.TrimSpace(usage.Name)
		if usage.UnlimitedQuota {
			// New API may retain a negative total_available for unlimited keys as
			// historical accounting. The explicit flag is authoritative.
			status.UsageMode = "unlimited"
		} else {
			remaining := newAPIQuotaToUSD(usage.TotalAvailable)
			status.KeyRemaining = &remaining
			status.BalanceUnit = "USD"
			status.UsageMode = "key_quota"
		}
		status.Status = "ok"
		return status, false
	}
	if err == nil {
		err = errors.New("New API API-key usage endpoint returned an unsupported response")
		// A successful HTTP response that is not the New API token-usage schema
		// means this upstream does not expose the public probe we rely on.
		// Permit the configured panel-login fallback in that case.
		status.Status = newAPIPublicProbeStatus(httpStatus, err)
		status.Message = err.Error()
		return status, true
	}
	status.Status = newAPIPublicProbeStatus(httpStatus, err)
	status.Message = err.Error()
	return status, shouldFallbackFromNewAPIPublicProbe(httpStatus, err)
}

func newAPIPublicProbeStatus(httpStatus int, err error) string {
	if errors.Is(err, errUpstreamSub2APICloudflareBlocked) {
		return "cloudflare_blocked"
	}
	if httpStatus == http.StatusNotFound || httpStatus == http.StatusMethodNotAllowed {
		return "public_api_unavailable"
	}
	return "request_failed"
}

func shouldFallbackFromNewAPIPublicProbe(httpStatus int, err error) bool {
	if errors.Is(err, errUpstreamSub2APICloudflareBlocked) {
		return false
	}
	if httpStatus == 0 || httpStatus == http.StatusNotFound || httpStatus == http.StatusMethodNotAllowed {
		return true
	}
	if httpStatus >= http.StatusInternalServerError {
		return true
	}
	var syntaxError *json.SyntaxError
	var typeError *json.UnmarshalTypeError
	return errors.As(err, &syntaxError) || errors.As(err, &typeError)
}

func (p *upstreamSub2APIStatusClient) probeNewAPIWithLoginFresh(ctx context.Context, status UpstreamSub2APIAccountStatus, root, apiKey, username, password string) UpstreamSub2APIAccountStatus {
	status.ProbeSource = "panel_login"
	user, userErr := p.fetchNewAPICurrentUser(ctx, root, username, password)
	if userErr != nil {
		status.Status = upstreamSub2APILoginStatus(userErr)
		status.Message = userErr.Error()
		return status
	}
	if user != nil {
		balance := newAPIQuotaToUSD(user.Quota)
		status.UserBalance = &balance
		status.BalanceUnit = "USD"
	}

	matchedKey, keyErr := p.findNewAPIKey(ctx, root, username, password, apiKey)
	if keyErr != nil {
		status.Status = upstreamSub2APILoginStatus(keyErr)
		status.Message = keyErr.Error()
		return status
	}
	if matchedKey == nil {
		status.Status = "key_not_found"
		status.Message = "upstream api key was not found in the logged-in New API account"
		return status
	}

	status.UpstreamKeyID = &matchedKey.ID
	status.UpstreamKeyName = matchedKey.Name
	if matchedKey.UnlimitedQuota {
		status.UsageMode = "unlimited"
	} else {
		remaining := newAPIQuotaToUSD(matchedKey.RemainQuota)
		status.KeyRemaining = &remaining
	}

	groupName := strings.TrimSpace(matchedKey.Group)
	if groupName == "" && user != nil {
		groupName = strings.TrimSpace(user.Group)
	}
	if groupName != "" {
		status.UpstreamGroupName = groupName
		status.UpstreamGroupPlatform = status.LocalPlatform
		groups, groupsErr := p.fetchNewAPIUserGroups(ctx, root, username, password)
		if groupsErr != nil {
			status.Status = upstreamSub2APILoginStatus(groupsErr)
			status.Message = groupsErr.Error()
			return status
		}
		if group, ok := groups[groupName]; ok {
			if rate, ok := parseNewAPIGroupRatio(group.Ratio); ok {
				status.UpstreamGroupDefaultRateMultiplier = &rate
				status.UpstreamGroupEffectiveRateMultiplier = &rate
			}
		}
	}

	status.Status = "ok"
	return status
}

func upstreamSub2APILoginStatus(err error) string {
	if errors.Is(err, errUpstreamSub2APITwoFactor) {
		return "two_factor_required"
	}
	if errors.Is(err, errUpstreamSub2APICloudflareBlocked) {
		return "cloudflare_blocked"
	}
	return "request_failed"
}

func (p *upstreamSub2APIStatusClient) fetchCurrentUser(ctx context.Context, root, email, password string) (*upstreamSub2APIUser, error) {
	var user upstreamSub2APIUser
	if err := p.getAuthenticatedJSON(ctx, root, email, password, "/api/v1/auth/me", &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (p *upstreamSub2APIStatusClient) fetchAvailableGroups(ctx context.Context, root, email, password string) (map[int64]*upstreamSub2APIGroup, error) {
	var groups []upstreamSub2APIGroup
	if err := p.getAuthenticatedJSON(ctx, root, email, password, "/api/v1/groups/available", &groups); err != nil {
		return nil, err
	}
	out := make(map[int64]*upstreamSub2APIGroup, len(groups))
	for i := range groups {
		group := groups[i]
		out[group.ID] = &group
	}
	return out, nil
}

func (p *upstreamSub2APIStatusClient) fetchGroupRates(ctx context.Context, root, email, password string) (map[int64]float64, error) {
	var raw map[string]float64
	if err := p.getAuthenticatedJSON(ctx, root, email, password, "/api/v1/groups/rates", &raw); err != nil {
		return nil, err
	}
	out := make(map[int64]float64, len(raw))
	for key, rate := range raw {
		id, err := strconv.ParseInt(key, 10, 64)
		if err == nil && id > 0 {
			out[id] = rate
		}
	}
	return out, nil
}

func (p *upstreamSub2APIStatusClient) findAPIKey(ctx context.Context, root, email, password, apiKey string) (*upstreamSub2APIKey, error) {
	matched, _, err := p.findAPIKeyWithPath(ctx, root, email, password, apiKey)
	return matched, err
}

func (p *upstreamSub2APIStatusClient) findAPIKeyWithPath(ctx context.Context, root, email, password, apiKey string) (*upstreamSub2APIKey, string, error) {
	for _, basePath := range []string{"/api/v1/keys", "/api/v1/api-keys"} {
		matchedKey, status, err := p.findAPIKeyAtPath(ctx, root, email, password, apiKey, basePath)
		if err == nil {
			if matchedKey != nil {
				return matchedKey, basePath, nil
			}
			continue
		}
		if status == http.StatusNotFound {
			continue
		}
		return nil, "", err
	}
	return nil, "", errors.New("upstream api key was not found in a supported list endpoint")
}

func (p *upstreamSub2APIStatusClient) findAPIKeyAtPath(ctx context.Context, root, email, password, apiKey, basePath string) (*upstreamSub2APIKey, int, error) {
	page := 1
	for {
		path := fmt.Sprintf("%s?page=%d&page_size=%d", basePath, page, upstreamSub2APIPageSize)
		var keysPage upstreamSub2APIKeysPage
		status, err := p.getAuthenticatedJSONWithStatus(ctx, root, email, password, path, &keysPage)
		if err != nil {
			return nil, status, err
		}
		for i := range keysPage.Items {
			if strings.TrimSpace(keysPage.Items[i].Key) == apiKey {
				return &keysPage.Items[i], status, nil
			}
		}
		if keysPage.Pages > 0 {
			if page >= keysPage.Pages {
				break
			}
		} else if keysPage.Total > 0 {
			if int64(page*upstreamSub2APIPageSize) >= keysPage.Total {
				break
			}
		} else if len(keysPage.Items) < upstreamSub2APIPageSize {
			break
		}
		page++
	}
	return nil, http.StatusOK, nil
}

func (p *upstreamSub2APIStatusClient) fetchUsage(ctx context.Context, root, apiKey string) (*upstreamSub2APIUsage, error) {
	var usage upstreamSub2APIUsage
	if _, err := p.doJSON(ctx, http.MethodGet, joinUpstreamSub2APIURL(root, "/v1/usage"), apiKey, nil, &usage); err != nil {
		return nil, err
	}
	return &usage, nil
}

func (p *upstreamSub2APIStatusClient) fetchNewAPICurrentUser(ctx context.Context, root, username, password string) (*upstreamNewAPIUser, error) {
	var user upstreamNewAPIUser
	if err := p.getNewAPIAuthenticatedJSON(ctx, root, username, password, "/api/user/self", &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (p *upstreamSub2APIStatusClient) fetchNewAPIUserGroups(ctx context.Context, root, username, password string) (map[string]upstreamNewAPIGroupInfo, error) {
	var groups map[string]upstreamNewAPIGroupInfo
	if err := p.getNewAPIAuthenticatedJSON(ctx, root, username, password, "/api/user/self/groups", &groups); err != nil {
		return nil, err
	}
	if groups == nil {
		groups = map[string]upstreamNewAPIGroupInfo{}
	}
	return groups, nil
}

func (p *upstreamSub2APIStatusClient) findNewAPIKey(ctx context.Context, root, username, password, apiKey string) (*upstreamNewAPIToken, error) {
	query := url.Values{}
	query.Set("token", apiKey)
	query.Set("p", "1")
	query.Set("page_size", strconv.Itoa(upstreamSub2APIPageSize))
	path := "/api/token/search?" + query.Encode()
	var page upstreamNewAPITokensPage
	if err := p.getNewAPIAuthenticatedJSON(ctx, root, username, password, path, &page); err != nil {
		return nil, err
	}
	for i := range page.Items {
		return &page.Items[i], nil
	}
	return nil, nil
}

func (p *upstreamSub2APIStatusClient) getAuthenticatedJSON(ctx context.Context, root, email, password, path string, out any) error {
	_, err := p.getAuthenticatedJSONWithStatus(ctx, root, email, password, path, out)
	return err
}

func (p *upstreamSub2APIStatusClient) getAuthenticatedJSONWithStatus(ctx context.Context, root, email, password, path string, out any) (int, error) {
	token, err := p.login(ctx, root, email, password, false)
	if err != nil {
		return 0, err
	}
	status, err := p.doJSON(ctx, http.MethodGet, joinUpstreamSub2APIURL(root, path), token, nil, out)
	if status == http.StatusUnauthorized {
		p.invalidateToken(ctx, root, email, password)
		token, err = p.login(ctx, root, email, password, true)
		if err != nil {
			return status, err
		}
		status, err = p.doJSON(ctx, http.MethodGet, joinUpstreamSub2APIURL(root, path), token, nil, out)
	}
	return status, err
}

func (p *upstreamSub2APIStatusClient) getNewAPIAuthenticatedJSON(ctx context.Context, root, username, password, path string, out any) error {
	session, err := p.loginNewAPI(ctx, root, username, password, false)
	if err != nil {
		return err
	}
	status, _, err := p.doNewAPIJSON(ctx, http.MethodGet, joinUpstreamSub2APIURL(root, path), session, "", nil, out)
	if status == http.StatusUnauthorized {
		p.invalidateNewAPISession(ctx, root, username, password)
		session, err = p.loginNewAPI(ctx, root, username, password, true)
		if err != nil {
			return err
		}
		_, _, err = p.doNewAPIJSON(ctx, http.MethodGet, joinUpstreamSub2APIURL(root, path), session, "", nil, out)
	}
	return err
}

func (p *upstreamSub2APIStatusClient) login(ctx context.Context, root, email, password string, force bool) (string, error) {
	cacheKey := upstreamSub2APITokenCacheKey(upstreamSub2APIProxyURLFromContext(ctx), root, email, password)
	now := time.Now().UTC()
	if !force {
		p.mu.Lock()
		if entry, ok := p.tokenCache[cacheKey]; ok && now.Before(entry.expires) {
			token := entry.token
			p.mu.Unlock()
			return token, nil
		}
		p.mu.Unlock()
	}

	body := map[string]string{
		"email":    email,
		"password": password,
	}
	var loginResp upstreamSub2APILoginResponse
	if _, err := p.doJSON(ctx, http.MethodPost, joinUpstreamSub2APIURL(root, "/api/v1/auth/login"), "", body, &loginResp); err != nil {
		return "", fmt.Errorf("upstream login failed: %w", err)
	}
	if loginResp.Requires2FA {
		return "", errUpstreamSub2APITwoFactor
	}
	token := strings.TrimSpace(loginResp.AccessToken)
	if token == "" {
		return "", errors.New("upstream login returned an empty access token")
	}
	ttl := time.Duration(loginResp.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	expires := now.Add(ttl - upstreamSub2APITokenSkew)
	if !expires.After(now) {
		expires = now.Add(ttl)
	}

	p.mu.Lock()
	p.tokenCache[cacheKey] = upstreamSub2APITokenCacheEntry{token: token, expires: expires}
	p.mu.Unlock()
	return token, nil
}

func (p *upstreamSub2APIStatusClient) loginNewAPI(ctx context.Context, root, username, password string, force bool) (*upstreamNewAPISessionCacheEntry, error) {
	cacheKey := upstreamNewAPISessionCacheKey(upstreamSub2APIProxyURLFromContext(ctx), root, username, password)
	now := time.Now().UTC()
	if !force {
		p.mu.Lock()
		if entry, ok := p.sessionCache[cacheKey]; ok && now.Before(entry.expires) {
			session := cloneUpstreamNewAPISession(entry)
			p.mu.Unlock()
			return &session, nil
		}
		p.mu.Unlock()
	}

	body := map[string]string{
		"username": username,
		"password": password,
	}
	var loginResp upstreamNewAPILoginUser
	status, cookies, err := p.doNewAPIJSON(ctx, http.MethodPost, joinUpstreamSub2APIURL(root, "/api/user/login"), nil, "", body, &loginResp)
	if err != nil {
		return nil, fmt.Errorf("upstream New API login failed: %w", err)
	}
	if status == http.StatusUnauthorized {
		return nil, errors.New("upstream New API login unauthorized")
	}
	if loginResp.Require2FA {
		return nil, errUpstreamSub2APITwoFactor
	}
	if loginResp.ID <= 0 {
		return nil, errors.New("upstream New API login returned an empty user id")
	}
	if len(cookies) == 0 {
		return nil, errors.New("upstream New API login returned no session cookie")
	}
	entry := upstreamNewAPISessionCacheEntry{
		userID:  loginResp.ID,
		cookies: cloneHTTPCookies(cookies),
		expires: now.Add(upstreamNewAPISessionTTL),
	}
	p.mu.Lock()
	p.sessionCache[cacheKey] = cloneUpstreamNewAPISession(entry)
	p.mu.Unlock()
	return &entry, nil
}

func (p *upstreamSub2APIStatusClient) doJSON(ctx context.Context, method, targetURL, bearerToken string, body any, out any) (int, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, targetURL, reqBody)
	if err != nil {
		return 0, err
	}
	setUpstreamPanelRequestHeaders(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(bearerToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearerToken))
	}

	resp, err := p.httpClientForContext(ctx).Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if httputil.IsCloudflareChallengeResponse(resp.StatusCode, resp.Header, data) {
			return resp.StatusCode, newUpstreamSub2APICloudflareBlockedError(resp.StatusCode, resp.Header, data, upstreamSub2APIProxyURLFromContext(ctx))
		}
		return resp.StatusCode, fmt.Errorf("upstream returned HTTP %d: %s", resp.StatusCode, truncateUpstreamSub2APIError(data))
	}
	if out == nil {
		return resp.StatusCode, nil
	}
	payload, err := unwrapUpstreamSub2APIEnvelope(data)
	if err != nil {
		return resp.StatusCode, err
	}
	if len(bytes.TrimSpace(payload)) == 0 || bytes.Equal(bytes.TrimSpace(payload), []byte("null")) {
		return resp.StatusCode, nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return resp.StatusCode, err
	}
	return resp.StatusCode, nil
}

func (p *upstreamSub2APIStatusClient) doNewAPIJSON(ctx context.Context, method, targetURL string, session *upstreamNewAPISessionCacheEntry, bearerToken string, body any, out any) (int, []*http.Cookie, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, targetURL, reqBody)
	if err != nil {
		return 0, nil, err
	}
	setUpstreamPanelRequestHeaders(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(bearerToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearerToken))
	}
	if session != nil {
		if session.userID > 0 {
			req.Header.Set("New-Api-User", strconv.FormatInt(session.userID, 10))
		}
		for _, cookie := range session.cookies {
			if cookie != nil {
				req.AddCookie(cookie)
			}
		}
	}

	resp, err := p.httpClientForContext(ctx).Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return resp.StatusCode, resp.Cookies(), err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if httputil.IsCloudflareChallengeResponse(resp.StatusCode, resp.Header, data) {
			return resp.StatusCode, resp.Cookies(), newUpstreamSub2APICloudflareBlockedError(resp.StatusCode, resp.Header, data, upstreamSub2APIProxyURLFromContext(ctx))
		}
		return resp.StatusCode, resp.Cookies(), fmt.Errorf("upstream returned HTTP %d: %s", resp.StatusCode, truncateUpstreamSub2APIError(data))
	}
	if out == nil {
		return resp.StatusCode, resp.Cookies(), nil
	}
	payload, err := unwrapUpstreamNewAPIEnvelope(data)
	if err != nil {
		return resp.StatusCode, resp.Cookies(), err
	}
	if len(bytes.TrimSpace(payload)) == 0 || bytes.Equal(bytes.TrimSpace(payload), []byte("null")) {
		return resp.StatusCode, resp.Cookies(), nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return resp.StatusCode, resp.Cookies(), err
	}
	return resp.StatusCode, resp.Cookies(), nil
}

func (p *upstreamSub2APIStatusClient) contextForAccount(ctx context.Context, account *service.Account) (context.Context, string, error) {
	proxyURL := upstreamSub2APIAccountProxyURL(account)
	if proxyURL == "" {
		return ctx, "", nil
	}

	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL: proxyURL,
		Timeout:  8 * time.Second,
	})
	if err != nil {
		return ctx, "", err
	}

	proxyCtx := context.WithValue(ctx, upstreamSub2APIHTTPClientContextKey{}, client)
	proxyCtx = context.WithValue(proxyCtx, upstreamSub2APIProxyURLContextKey{}, proxyURL)
	return proxyCtx, proxyURL, nil
}

func upstreamSub2APIAccountProxyURL(account *service.Account) string {
	if account == nil || account.ProxyID == nil || account.Proxy == nil {
		return ""
	}
	return strings.TrimSpace(account.Proxy.URL())
}

func (p *upstreamSub2APIStatusClient) httpClientForContext(ctx context.Context) *http.Client {
	if ctx != nil {
		if client, ok := ctx.Value(upstreamSub2APIHTTPClientContextKey{}).(*http.Client); ok && client != nil {
			return client
		}
	}
	return p.httpClient
}

func upstreamSub2APIProxyURLFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	proxyURL, _ := ctx.Value(upstreamSub2APIProxyURLContextKey{}).(string)
	return strings.TrimSpace(proxyURL)
}

func newUpstreamSub2APICloudflareBlockedError(statusCode int, headers http.Header, body []byte, proxyURL string) error {
	return &upstreamSub2APICloudflareBlockedError{
		statusCode: statusCode,
		rayID:      httputil.ExtractCloudflareRayID(headers, body),
		proxyUsed:  strings.TrimSpace(proxyURL) != "",
	}
}

func setUpstreamPanelRequestHeaders(req *http.Request) {
	if req == nil {
		return
	}
	if strings.TrimSpace(req.Header.Get("Accept")) == "" {
		req.Header.Set("Accept", "application/json, text/plain, */*")
	}
	if strings.TrimSpace(req.Header.Get("Accept-Language")) == "" {
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	}
	if strings.TrimSpace(req.Header.Get("User-Agent")) == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0 Safari/537.36")
	}
}

func unwrapUpstreamSub2APIEnvelope(data []byte) ([]byte, error) {
	return unwrapUpstreamAPIEnvelope(data, "upstream api returned an error")
}

func unwrapUpstreamNewAPIEnvelope(data []byte) ([]byte, error) {
	return unwrapUpstreamAPIEnvelope(data, "upstream New API returned an error")
}

func unwrapUpstreamAPIEnvelope(data []byte, defaultError string) ([]byte, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return trimmed, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return trimmed, nil
	}

	successRaw, hasSuccess := fields["success"]
	codeRaw, hasCode := fields["code"]
	if !hasSuccess && !hasCode {
		return trimmed, nil
	}

	if hasSuccess {
		success, ok := parseUpstreamEnvelopeSuccess(successRaw)
		if !ok {
			return trimmed, nil
		}
		if !success {
			return nil, errors.New(upstreamEnvelopeMessage(fields, defaultError))
		}
		return fields["data"], nil
	}

	success, ok := parseUpstreamEnvelopeCodeSuccess(codeRaw)
	if !ok {
		return trimmed, nil
	}
	if !success {
		return nil, errors.New(upstreamEnvelopeMessage(fields, defaultError))
	}
	return fields["data"], nil
}

func parseUpstreamEnvelopeSuccess(raw json.RawMessage) (bool, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return false, false
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err == nil {
		return value, true
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return number != 0, true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return parseUpstreamEnvelopeSuccessText(text)
	}
	return false, false
}

func parseUpstreamEnvelopeCodeSuccess(raw json.RawMessage) (bool, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return false, false
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err == nil {
		return value, true
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return number == 0 || number == http.StatusOK, true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		text = strings.TrimSpace(strings.ToLower(text))
		switch text {
		case "0", "200", "ok", "success", "true":
			return true, true
		case "false", "error", "failed", "fail":
			return false, true
		default:
			number, err := strconv.ParseFloat(text, 64)
			if err == nil {
				return number == 0 || number == http.StatusOK, true
			}
		}
	}
	return false, false
}

func parseUpstreamEnvelopeSuccessText(text string) (bool, bool) {
	switch strings.TrimSpace(strings.ToLower(text)) {
	case "1", "true", "ok", "success":
		return true, true
	case "0", "false", "error", "failed", "fail":
		return false, true
	default:
		return false, false
	}
}

func upstreamEnvelopeMessage(fields map[string]json.RawMessage, fallback string) string {
	for _, key := range []string{"message", "msg", "error"} {
		raw, ok := fields[key]
		if !ok {
			continue
		}
		var text string
		if err := json.Unmarshal(raw, &text); err == nil {
			if text = strings.TrimSpace(text); text != "" {
				return text
			}
		}
	}
	return fallback
}

func (p *upstreamSub2APIStatusClient) getCachedStatus(key string, now time.Time) (UpstreamSub2APIAccountStatus, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry, ok := p.statusCache[key]
	if !ok || !now.Before(entry.expires) {
		if ok {
			delete(p.statusCache, key)
		}
		return UpstreamSub2APIAccountStatus{}, false
	}
	status := entry.status
	status.Cached = true
	return status, true
}

func (p *upstreamSub2APIStatusClient) setCachedStatus(key string, status UpstreamSub2APIAccountStatus, expires time.Time) {
	status.Cached = false
	p.mu.Lock()
	p.statusCache[key] = upstreamSub2APIStatusCacheEntry{status: status, expires: expires}
	p.mu.Unlock()
}

func (p *upstreamSub2APIStatusClient) invalidateToken(ctx context.Context, root, email, password string) {
	p.mu.Lock()
	delete(p.tokenCache, upstreamSub2APITokenCacheKey(upstreamSub2APIProxyURLFromContext(ctx), root, email, password))
	p.mu.Unlock()
}

func (p *upstreamSub2APIStatusClient) invalidateNewAPISession(ctx context.Context, root, username, password string) {
	p.mu.Lock()
	delete(p.sessionCache, upstreamNewAPISessionCacheKey(upstreamSub2APIProxyURLFromContext(ctx), root, username, password))
	p.mu.Unlock()
}

func normalizeUpstreamSub2APIBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("base_url is empty")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("base_url must use http or https")
	}
	if parsed.Host == "" {
		return "", errors.New("base_url host is empty")
	}
	if parsed.User != nil {
		return "", errors.New("base_url must not contain user information")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("base_url must not contain a query or fragment")
	}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	for _, suffix := range []string{"/api/v1", "/antigravity/v1beta", "/antigravity/v1", "/backend-api/codex", "/v1beta", "/v1"} {
		if path == suffix {
			path = ""
			break
		}
		if strings.HasSuffix(path, suffix) {
			path = strings.TrimRight(strings.TrimSuffix(path, suffix), "/")
			break
		}
	}
	parsed.Path = path
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func joinUpstreamSub2APIURL(root, path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(root, "/") + path
}

func upstreamSub2APIStatusCacheKey(accountID int64, root, apiKey, email, password, panelType, proxyURL string) string {
	return fmt.Sprintf(
		"%d|%s|%s|%s|%s|%s|%s",
		accountID,
		root,
		normalizeUpstreamPanelType(panelType),
		shortUpstreamSub2APIFingerprint(apiKey),
		strings.ToLower(strings.TrimSpace(email)),
		shortUpstreamSub2APIFingerprint(password),
		upstreamSub2APIProxyCacheKey(proxyURL),
	)
}

func upstreamSub2APITokenCacheKey(proxyURL, root, email, password string) string {
	return fmt.Sprintf("%s|%s|%s|%s", upstreamSub2APIProxyCacheKey(proxyURL), root, strings.ToLower(strings.TrimSpace(email)), shortUpstreamSub2APIFingerprint(password))
}

func upstreamNewAPISessionCacheKey(proxyURL, root, username, password string) string {
	return fmt.Sprintf("%s|%s|%s|%s", upstreamSub2APIProxyCacheKey(proxyURL), root, strings.ToLower(strings.TrimSpace(username)), shortUpstreamSub2APIFingerprint(password))
}

func upstreamSub2APIProxyCacheKey(proxyURL string) string {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return "direct"
	}
	return "proxy:" + shortUpstreamSub2APIFingerprint(proxyURL)
}

func normalizeUpstreamPanelType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "sub2api":
		return "sub2api"
	case "newapi", "new-api", "new_api":
		return "newapi"
	default:
		return "auto"
	}
}

func cloneUpstreamNewAPISession(entry upstreamNewAPISessionCacheEntry) upstreamNewAPISessionCacheEntry {
	return upstreamNewAPISessionCacheEntry{
		userID:  entry.userID,
		cookies: cloneHTTPCookies(entry.cookies),
		expires: entry.expires,
	}
}

func cloneHTTPCookies(cookies []*http.Cookie) []*http.Cookie {
	if len(cookies) == 0 {
		return nil
	}
	out := make([]*http.Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		copied := *cookie
		out = append(out, &copied)
	}
	return out
}

func newAPIQuotaToUSD(quota int64) float64 {
	return float64(quota) / upstreamNewAPIQuotaPerUnit
}

func parseNewAPIGroupRatio(raw json.RawMessage) (float64, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return 0, false
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil && number >= 0 {
		return number, true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		value, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err == nil && value >= 0 {
			return value, true
		}
	}
	return 0, false
}

func shortUpstreamSub2APIFingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func truncateUpstreamSub2APIError(data []byte) string {
	text := strings.TrimSpace(string(data))
	if len(text) > 240 {
		return text[:240]
	}
	return text
}
