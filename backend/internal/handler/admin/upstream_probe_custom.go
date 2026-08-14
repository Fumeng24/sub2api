package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	entupstream "github.com/Wei-Shaw/sub2api/ent/upstream"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"github.com/gin-gonic/gin"
)

type upstreamProbeWallet struct {
	Balance *float64 `json:"balance,omitempty"`
	Unit    string   `json:"unit,omitempty"`
}

type upstreamProbeKey struct {
	ID             *int64   `json:"id,omitempty"`
	Name           string   `json:"name,omitempty"`
	GroupID        *int64   `json:"group_id,omitempty"`
	GroupName      string   `json:"group_name,omitempty"`
	UnlimitedQuota bool     `json:"unlimited_quota"`
	Remaining      *float64 `json:"remaining,omitempty"`
	Unit           string   `json:"unit,omitempty"`
}

type upstreamProbeGroup struct {
	ID             *int64   `json:"id,omitempty"`
	Name           string   `json:"name"`
	Platform       string   `json:"platform,omitempty"`
	Description    string   `json:"description,omitempty"`
	RateMultiplier *float64 `json:"rate_multiplier,omitempty"`
	Models         []string `json:"models,omitempty"`
}

type upstreamProtocolCapability struct {
	Platform  string    `json:"platform"`
	Status    string    `json:"status"`
	Models    []string  `json:"models"`
	Message   string    `json:"message,omitempty"`
	FetchedAt time.Time `json:"fetched_at"`
}

type upstreamProbeMetadata struct {
	DetectedKind     string                                    `json:"detected_kind,omitempty"`
	ProbeSource      string                                    `json:"probe_source,omitempty"`
	ManagementStatus string                                    `json:"management_status"`
	ManagementHint   string                                    `json:"management_hint,omitempty"`
	Wallet           *upstreamProbeWallet                      `json:"wallet,omitempty"`
	Key              *upstreamProbeKey                         `json:"key,omitempty"`
	Groups           []upstreamProbeGroup                      `json:"groups"`
	Protocols        []upstreamProtocolCapability              `json:"protocols"`
	AccountBilling   map[string]upstreamAccountBillingMetadata `json:"account_billing,omitempty"`
	Refresh          *upstreamRefreshMetadata                  `json:"refresh,omitempty"`
	FetchedAt        time.Time                                 `json:"fetched_at"`
}

type upstreamProbeOutcome struct {
	metadata             upstreamProbeMetadata
	status               entupstream.Status
	errorMsg             string
	detectedKindVerified bool
}

type upstreamModelProbeRequest struct {
	Platform  string `json:"platform"`
	GroupName string `json:"group_name"`
	Model     string `json:"model"`
}

var errUpstreamNewAPIManagementUserID = errors.New("NewAPI management access token is configured, but management_user_id is missing or invalid")

type upstreamModelProbeResult struct {
	Success    bool       `json:"success"`
	Platform   string     `json:"platform"`
	GroupName  string     `json:"group_name"`
	Model      string     `json:"model"`
	LatencyMs  int64      `json:"latency_ms"`
	StatusCode int        `json:"status_code,omitempty"`
	Status     string     `json:"status"`
	Message    string     `json:"message,omitempty"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

func (h *UpstreamHandler) Probe(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	item, err := h.refreshUpstreamByID(c.Request.Context(), id, true)
	if err != nil {
		writeUpstreamQueryError(c, err)
		return
	}
	duplicateCount, _ := h.client.Upstream.Query().
		Where(entupstream.BaseURLEQ(item.BaseURL), entupstream.IDNEQ(item.ID)).
		Count(c.Request.Context())
	response.Success(c, buildUpstreamView(item, true, duplicateCount))
}

// TestModel uses the same real invocation and verification cache as the batch
// endpoint. It never changes an account allowlist or scheduler state.
func (h *UpstreamHandler) TestModel(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	var req upstreamModelProbeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	req.Platform = strings.ToLower(strings.TrimSpace(req.Platform))
	req.GroupName = strings.TrimSpace(req.GroupName)
	req.Model = strings.TrimSpace(req.Model)
	if !isManagedUpstreamPlatform(req.Platform) || req.GroupName == "" || req.Model == "" {
		response.BadRequest(c, "platform, group_name, and model are required")
		return
	}
	item, err := h.queryUpstream(c, id, true)
	if err != nil {
		writeUpstreamQueryError(c, err)
		return
	}
	metadata, err := parseUpstreamProbeMetadata(item.Metadata)
	if err != nil {
		response.InternalError(c, "Failed to load upstream capability metadata")
		return
	}
	if !upstreamProbeGroupExists(metadata.Groups, req.Platform, req.GroupName) {
		response.BadRequest(c, "Selected upstream group is not available")
		return
	}

	batch := h.probeUpstreamModels(c.Request.Context(), item, metadata, upstreamModelsProbeRequest{
		Platform: req.Platform, GroupName: req.GroupName, Models: []string{req.Model},
	})
	if len(batch.Results) != 1 {
		response.Success(c, upstreamModelProbeResult{
			Platform: req.Platform, GroupName: req.GroupName, Model: req.Model,
			Status:  service.ModelCapabilityStatusUnconfirmed,
			Message: batch.Message,
		})
		return
	}
	response.Success(c, batch.Results[0])
}

func safeUpstreamModelProbeError(err error, item *dbent.Upstream, extraSecrets ...string) string {
	var message string
	var syncErr *service.UpstreamModelSyncError
	if errors.As(err, &syncErr) {
		message = redactUpstreamText(syncErr.SafeMessage(), item)
	} else {
		message = safeUpstreamProbeError(err, item)
	}
	for _, secret := range extraSecrets {
		secret = strings.TrimSpace(secret)
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "[REDACTED]")
		}
	}
	return message
}

func (h *UpstreamHandler) probeUpstream(ctx context.Context, item *dbent.Upstream) upstreamProbeOutcome {
	metadata, managementOK, detectedKindVerified, managementErr := h.probeUpstreamManagement(ctx, item)
	now := metadata.FetchedAt

	metadata.Protocols = h.probeUpstreamProtocols(ctx, item, now)
	isSub2API := item.Kind == entupstream.KindSub2api || metadata.DetectedKind == entupstream.KindSub2api.String()
	isNewAPI := item.Kind == entupstream.KindNewapi || metadata.DetectedKind == entupstream.KindNewapi.String()
	if managementOK && isSub2API {
		if channels, err := h.fetchSub2APIAvailableChannels(ctx, item, service.PlatformOpenAI); err == nil {
			applySub2APIModelCatalogue(&metadata, channels)
		}
	}
	if managementOK && isNewAPI {
		summary := h.applyNewAPIGroupModelCatalogue(ctx, item, &metadata)
		applyNewAPIProtocolFromGroupCatalogue(&metadata, summary)
	}
	attachProtocolModelsToGroups(&metadata)
	protocolOK := false
	for _, protocol := range metadata.Protocols {
		if protocol.Status == "ok" {
			protocolOK = true
			break
		}
	}

	status := entupstream.StatusError
	errorMsg := metadata.ManagementHint
	switch {
	case managementOK && protocolOK:
		status = entupstream.StatusHealthy
		errorMsg = ""
	case managementOK || protocolOK:
		status = entupstream.StatusDegraded
		if errorMsg == "" {
			_, errorMsg = lightweightRefreshStatus(metadata)
			if errorMsg == "" {
				errorMsg = "Some upstream capabilities could not be verified"
			}
		}
	default:
		if errorMsg == "" {
			errorMsg = "No upstream management or model endpoint could be verified"
		}
	}
	if managementErr != nil && errorMsg == "" {
		errorMsg = safeUpstreamProbeError(managementErr, item)
	}
	return upstreamProbeOutcome{
		metadata:             metadata,
		status:               status,
		errorMsg:             errorMsg,
		detectedKindVerified: detectedKindVerified,
	}
}

// probeUpstreamManagement intentionally excludes protocol/model discovery. It is
// shared by the full manual probe and the lightweight periodic refresh path.
func (h *UpstreamHandler) probeUpstreamManagement(ctx context.Context, item *dbent.Upstream) (upstreamProbeMetadata, bool, bool, error) {
	now := time.Now().UTC()
	metadata := upstreamProbeMetadata{
		ManagementStatus: "unavailable",
		Groups:           []upstreamProbeGroup{},
		Protocols:        []upstreamProtocolCapability{},
		AccountBilling:   map[string]upstreamAccountBillingMetadata{},
		FetchedAt:        now,
	}

	probeAccount := h.transientUpstreamAccount(ctx, item, service.PlatformOpenAI, upstreamAPIKeyForPlatform(item.Credentials, service.PlatformOpenAI))
	probeCtx, _, proxyErr := h.panelClient.contextForAccount(ctx, probeAccount)
	if proxyErr != nil {
		metadata.ManagementStatus = "error"
		metadata.ManagementHint = "Configured proxy is invalid"
		return metadata, false, false, proxyErr
	}

	managementOK := false
	detectedKindVerified := false
	detectedKind := item.Kind.String()
	var managementErr error

	switch item.Kind {
	case entupstream.KindNewapi:
		managementOK, managementErr = h.probeNewAPIManagement(probeCtx, item, &metadata)
		detectedKindVerified = managementOK
		detectedKind = entupstream.KindNewapi.String()
	case entupstream.KindSub2api:
		managementOK, managementErr = h.probeSub2APIManagement(probeCtx, item, &metadata)
		detectedKindVerified = managementOK
		detectedKind = entupstream.KindSub2api.String()
	default:
		// Auto detection starts with NewAPI's API-key-only endpoint. This keeps
		// API-key-only upstreams usable without forcing administrators to create
		// a dashboard login just to identify the site type.
		publicProbeOK := false
		if apiKey := upstreamAPIKeyForPlatform(item.Credentials, service.PlatformOpenAI); apiKey != "" {
			publicAccount := h.transientUpstreamAccount(ctx, item, service.PlatformOpenAI, apiKey)
			publicAccount.Credentials["upstream_panel_type"] = entupstream.KindNewapi.String()
			publicStatus := h.panelClient.ProbeAccount(probeCtx, publicAccount, true)
			if publicStatus.UpstreamKind == entupstream.KindNewapi.String() && publicStatus.Status == "ok" {
				publicProbeOK = true
				detectedKindVerified = true
				detectedKind = entupstream.KindNewapi.String()
				metadata.ProbeSource = publicStatus.ProbeSource
				metadata.Key = probeKeyFromPublicStatus(publicStatus)
			} else if message := strings.TrimSpace(publicStatus.Message); message != "" {
				managementErr = errors.New(message)
			}
		}
		if upstreamCredentialPresent(item.Credentials, upstreamCredentialManagementAccessToken) {
			managementOK, managementErr = h.probeNewAPIManagement(probeCtx, item, &metadata)
			if managementOK {
				detectedKindVerified = true
				detectedKind = entupstream.KindNewapi.String()
			}
		}
		if !managementOK && hasUpstreamPanelLogin(item.Credentials) {
			newAPIMetadata := upstreamProbeMetadata{Groups: []upstreamProbeGroup{}, Protocols: []upstreamProtocolCapability{}, FetchedAt: now}
			if ok, err := h.probeNewAPIManagement(probeCtx, item, &newAPIMetadata); ok {
				if metadata.Key != nil && newAPIMetadata.Key == nil {
					newAPIMetadata.Key = metadata.Key
				}
				metadata = newAPIMetadata
				managementOK = true
				managementErr = nil
				detectedKindVerified = true
				detectedKind = entupstream.KindNewapi.String()
			} else {
				managementErr = err
			}
		}
		if !managementOK && !publicProbeOK && hasUpstreamPanelLogin(item.Credentials) {
			sub2APIMetadata := upstreamProbeMetadata{Groups: []upstreamProbeGroup{}, Protocols: []upstreamProtocolCapability{}, FetchedAt: now}
			if ok, err := h.probeSub2APIManagement(probeCtx, item, &sub2APIMetadata); ok {
				metadata = sub2APIMetadata
				managementOK = true
				managementErr = nil
				detectedKindVerified = true
				detectedKind = entupstream.KindSub2api.String()
			} else if managementErr == nil {
				managementErr = err
			}
		}
	}

	metadata.DetectedKind = detectedKind
	if managementOK {
		metadata.ManagementStatus = "ok"
		metadata.ManagementHint = ""
	} else {
		metadata.ManagementStatus, metadata.ManagementHint = upstreamManagementFailure(item, managementErr)
	}
	return metadata, managementOK, detectedKindVerified, managementErr
}

func (h *UpstreamHandler) probeNewAPIManagement(ctx context.Context, item *dbent.Upstream, metadata *upstreamProbeMetadata) (bool, error) {
	root := item.BaseURL
	apiKey := upstreamAPIKeyForPlatform(item.Credentials, service.PlatformOpenAI)
	accessToken := upstreamCredentialString(item.Credentials, upstreamCredentialManagementAccessToken)
	username := upstreamCredentialString(item.Credentials, upstreamCredentialUsername)
	password := upstreamCredentialString(item.Credentials, upstreamCredentialPassword)

	if accessToken != "" {
		metadata.ProbeSource = "management_access_token"
		var user upstreamNewAPIUser
		session := upstreamNewAPIAccessTokenSession(item.Credentials)
		if session == nil {
			return false, errUpstreamNewAPIManagementUserID
		}
		if _, _, err := h.panelClient.doNewAPIJSON(ctx, http.MethodGet, joinUpstreamSub2APIURL(root, "/api/user/self"), session, accessToken, nil, &user); err != nil {
			return false, err
		}
		balance := newAPIQuotaToUSD(user.Quota)
		metadata.Wallet = &upstreamProbeWallet{Balance: &balance, Unit: "USD"}

		var groups map[string]upstreamNewAPIGroupInfo
		if _, _, err := h.panelClient.doNewAPIJSON(ctx, http.MethodGet, joinUpstreamSub2APIURL(root, "/api/user/self/groups"), session, accessToken, nil, &groups); err != nil {
			return false, err
		}
		metadata.Groups = newAPIProbeGroups(groups)
		if apiKey != "" {
			key, err := h.findNewAPIKeyWithAccessToken(ctx, root, accessToken, session, apiKey)
			if err != nil {
				return false, err
			}
			metadata.Key = newAPIProbeKey(key)
		}
		return true, nil
	}

	if username == "" || password == "" {
		return false, errors.New("NewAPI management access token is not configured")
	}
	metadata.ProbeSource = "panel_login"
	user, err := h.panelClient.fetchNewAPICurrentUser(ctx, root, username, password)
	if err != nil {
		return false, err
	}
	if user != nil {
		balance := newAPIQuotaToUSD(user.Quota)
		metadata.Wallet = &upstreamProbeWallet{Balance: &balance, Unit: "USD"}
	}
	groups, err := h.panelClient.fetchNewAPIUserGroups(ctx, root, username, password)
	if err != nil {
		return false, err
	}
	metadata.Groups = newAPIProbeGroups(groups)
	if apiKey != "" {
		key, err := h.panelClient.findNewAPIKey(ctx, root, username, password, apiKey)
		if err != nil {
			return false, err
		}
		metadata.Key = newAPIProbeKey(key)
	}
	return true, nil
}

func (h *UpstreamHandler) probeSub2APIManagement(ctx context.Context, item *dbent.Upstream, metadata *upstreamProbeMetadata) (bool, error) {
	username := upstreamCredentialString(item.Credentials, upstreamCredentialUsername)
	password := upstreamCredentialString(item.Credentials, upstreamCredentialPassword)
	if username == "" || password == "" {
		return false, errors.New("Sub2API panel username or password is not configured")
	}
	metadata.ProbeSource = "panel_login"
	user, err := h.panelClient.fetchCurrentUser(ctx, item.BaseURL, username, password)
	if err != nil {
		return false, err
	}
	if user != nil {
		balance := user.Balance
		metadata.Wallet = &upstreamProbeWallet{Balance: &balance, Unit: "USD"}
	}
	groups, err := h.panelClient.fetchAvailableGroups(ctx, item.BaseURL, username, password)
	if err != nil {
		return false, err
	}
	rates, err := h.panelClient.fetchGroupRates(ctx, item.BaseURL, username, password)
	if err != nil {
		return false, err
	}
	metadata.Groups = sub2APIProbeGroups(groups, rates)
	apiKey := upstreamAPIKeyForPlatform(item.Credentials, service.PlatformOpenAI)
	if apiKey != "" {
		key, err := h.panelClient.findAPIKey(ctx, item.BaseURL, username, password, apiKey)
		if err != nil {
			return false, err
		}
		if key != nil {
			probeKey := &upstreamProbeKey{ID: &key.ID, Name: key.Name, GroupID: key.GroupID, Unit: "USD"}
			if key.Group != nil {
				probeKey.GroupName = key.Group.Name
			}
			if probeKey.GroupName == "" && key.GroupID != nil {
				if group := groups[*key.GroupID]; group != nil {
					probeKey.GroupName = group.Name
				}
			}
			metadata.Key = probeKey
		}
	}
	return true, nil
}

func (h *UpstreamHandler) findNewAPIKeyWithAccessToken(ctx context.Context, root, accessToken string, session *upstreamNewAPISessionCacheEntry, apiKey string) (*upstreamNewAPIToken, error) {
	query := url.Values{}
	query.Set("token", apiKey)
	query.Set("p", "1")
	query.Set("page_size", strconv.Itoa(upstreamSub2APIPageSize))
	var page upstreamNewAPITokensPage
	_, _, err := h.panelClient.doNewAPIJSON(
		ctx,
		http.MethodGet,
		joinUpstreamSub2APIURL(root, "/api/token/search?"+query.Encode()),
		session,
		accessToken,
		nil,
		&page,
	)
	if err != nil {
		return nil, err
	}
	if len(page.Items) == 0 {
		return nil, nil
	}
	return &page.Items[0], nil
}

func (h *UpstreamHandler) probeUpstreamProtocols(ctx context.Context, item *dbent.Upstream, fetchedAt time.Time) []upstreamProtocolCapability {
	platforms := []string{service.PlatformAnthropic, service.PlatformOpenAI, service.PlatformGemini, service.PlatformGrok}
	capabilities := make([]upstreamProtocolCapability, 0, len(platforms))
	for _, platform := range platforms {
		capability := upstreamProtocolCapability{Platform: platform, Status: "unavailable", Models: []string{}, FetchedAt: fetchedAt}
		apiKey := upstreamAPIKeyForPlatform(item.Credentials, platform)
		if apiKey == "" {
			capability.Status = "missing_api_key"
			capability.Message = "No API key is configured for this protocol"
			capabilities = append(capabilities, capability)
			continue
		}
		if h.accountTestService == nil {
			capability.Status = "unavailable"
			capability.Message = "Model capability probe is unavailable"
			capabilities = append(capabilities, capability)
			continue
		}
		account := h.transientUpstreamAccount(ctx, item, platform, apiKey)
		models, err := h.accountTestService.FetchUpstreamSupportedModels(ctx, account)
		if err != nil {
			capability.Status = "error"
			capability.Message = safeUpstreamProbeError(err, item)
			capabilities = append(capabilities, capability)
			continue
		}
		capability.Status = "ok"
		capability.Models = models
		capabilities = append(capabilities, capability)
	}
	return capabilities
}

func (h *UpstreamHandler) transientUpstreamAccount(ctx context.Context, item *dbent.Upstream, platform, apiKey string) *service.Account {
	credentials := map[string]any{
		"base_url": item.BaseURL,
		"api_key":  apiKey,
	}
	if item.Kind != entupstream.KindAuto {
		credentials["upstream_panel_type"] = item.Kind.String()
	}
	username := upstreamCredentialString(item.Credentials, upstreamCredentialUsername)
	password := upstreamCredentialString(item.Credentials, upstreamCredentialPassword)
	if username != "" {
		credentials["upstream_sub2api_email"] = username
	}
	if password != "" {
		credentials["upstream_sub2api_password"] = password
	}
	account := &service.Account{
		ID:          -item.ID,
		Name:        item.Name,
		Platform:    platform,
		Type:        service.AccountTypeAPIKey,
		Credentials: credentials,
		Extra:       map[string]any{},
		ProxyID:     item.ProxyID,
		Concurrency: 1,
		Status:      service.StatusActive,
		Schedulable: false,
	}
	if item.ProxyID != nil && h.adminService != nil {
		if proxy, err := h.adminService.GetProxy(ctx, *item.ProxyID); err == nil {
			account.Proxy = proxy
		}
	}
	return account
}

func upstreamNewAPIAccessTokenSession(credentials map[string]any) *upstreamNewAPISessionCacheEntry {
	raw := upstreamCredentialString(credentials, upstreamCredentialManagementUserID)
	if raw == "" {
		return nil
	}
	userID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || userID <= 0 {
		return nil
	}
	return &upstreamNewAPISessionCacheEntry{userID: userID}
}

func hasUpstreamPanelLogin(credentials map[string]any) bool {
	return upstreamCredentialPresent(credentials, upstreamCredentialUsername) && upstreamCredentialPresent(credentials, upstreamCredentialPassword)
}

func upstreamManagementFailure(item *dbent.Upstream, err error) (string, string) {
	if item.Kind == entupstream.KindNewapi && !upstreamCredentialPresent(item.Credentials, upstreamCredentialManagementAccessToken) && !hasUpstreamPanelLogin(item.Credentials) {
		return "missing_management_credentials", "Configure a NewAPI management access token to read the real wallet and group rates"
	}
	if item.Kind == entupstream.KindSub2api && !hasUpstreamPanelLogin(item.Credentials) {
		return "missing_management_credentials", "Configure the Sub2API panel username and password"
	}
	if item.Kind == entupstream.KindAuto && !upstreamCredentialPresent(item.Credentials, upstreamCredentialManagementAccessToken) && !hasUpstreamPanelLogin(item.Credentials) {
		return "missing_management_credentials", "Configure management credentials before probing wallet and group rates"
	}
	if errors.Is(err, errUpstreamNewAPIManagementUserID) {
		return "missing_management_user_id", errUpstreamNewAPIManagementUserID.Error()
	}
	if err == nil {
		return "unavailable", "The upstream did not expose management metadata"
	}
	return "error", safeUpstreamProbeError(err, item)
}

func newAPIProbeGroups(groups map[string]upstreamNewAPIGroupInfo) []upstreamProbeGroup {
	result := make([]upstreamProbeGroup, 0, len(groups))
	for name, info := range groups {
		group := upstreamProbeGroup{Name: strings.TrimSpace(name), Description: strings.TrimSpace(info.Desc), Models: []string{}}
		if rate, ok := parseNewAPIGroupRatio(info.Ratio); ok {
			group.RateMultiplier = &rate
		}
		if group.Name != "" {
			result = append(result, group)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func sub2APIProbeGroups(groups map[int64]*upstreamSub2APIGroup, rates map[int64]float64) []upstreamProbeGroup {
	result := make([]upstreamProbeGroup, 0, len(groups))
	for id, source := range groups {
		if source == nil {
			continue
		}
		groupID := id
		rate := source.RateMultiplier
		if effective, ok := rates[id]; ok {
			rate = effective
		}
		result = append(result, upstreamProbeGroup{
			ID:             &groupID,
			Name:           strings.TrimSpace(source.Name),
			Platform:       strings.TrimSpace(source.Platform),
			RateMultiplier: &rate,
			Models:         []string{},
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Platform == result[j].Platform {
			return result[i].Name < result[j].Name
		}
		return result[i].Platform < result[j].Platform
	})
	return result
}

func newAPIProbeKey(key *upstreamNewAPIToken) *upstreamProbeKey {
	if key == nil {
		return nil
	}
	id := key.ID
	result := &upstreamProbeKey{
		ID:             &id,
		Name:           strings.TrimSpace(key.Name),
		GroupName:      strings.TrimSpace(key.Group),
		UnlimitedQuota: key.UnlimitedQuota,
		Unit:           "USD",
	}
	if !key.UnlimitedQuota {
		remaining := newAPIQuotaToUSD(key.RemainQuota)
		result.Remaining = &remaining
	}
	return result
}

func probeKeyFromPublicStatus(status UpstreamSub2APIAccountStatus) *upstreamProbeKey {
	if status.UpstreamKeyID == nil && strings.TrimSpace(status.UpstreamKeyName) == "" &&
		status.KeyRemaining == nil && status.UsageMode == "" {
		return nil
	}
	result := &upstreamProbeKey{
		ID:             status.UpstreamKeyID,
		Name:           strings.TrimSpace(status.UpstreamKeyName),
		GroupID:        status.UpstreamGroupID,
		GroupName:      strings.TrimSpace(status.UpstreamGroupName),
		Remaining:      status.KeyRemaining,
		Unit:           strings.TrimSpace(status.BalanceUnit),
		UnlimitedQuota: status.UsageMode == "unlimited",
	}
	if result.Unit == "" {
		result.Unit = "USD"
	}
	return result
}

func attachProtocolModelsToGroups(metadata *upstreamProbeMetadata) {
	if metadata == nil {
		return
	}
	// Sub2API exposes group-scoped capabilities through /channels/available.
	// Falling back to the protocol-wide catalogue here would incorrectly mark
	// groups omitted by that endpoint as supporting every model on the platform.
	if strings.EqualFold(strings.TrimSpace(metadata.DetectedKind), entupstream.KindSub2api.String()) {
		return
	}
	modelsByPlatform := make(map[string][]string, len(metadata.Protocols))
	for _, protocol := range metadata.Protocols {
		if protocol.Status == "ok" {
			modelsByPlatform[protocol.Platform] = protocol.Models
		}
	}
	for i := range metadata.Groups {
		platform := strings.TrimSpace(metadata.Groups[i].Platform)
		if platform == "" || len(metadata.Groups[i].Models) > 0 {
			continue
		}
		if models, ok := modelsByPlatform[platform]; ok {
			metadata.Groups[i].Models = append([]string(nil), models...)
		}
	}
}

func upstreamMetadataMap(metadata upstreamProbeMetadata) (map[string]any, error) {
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func safeUpstreamProbeError(err error, item *dbent.Upstream) string {
	if err == nil {
		return ""
	}
	message := redactUpstreamText(err.Error(), item)
	message = strings.TrimSpace(message)
	if runeCount := len([]rune(message)); runeCount > 240 {
		message = string([]rune(message)[:240])
	}
	if message == "" {
		return fmt.Sprintf("upstream probe failed: %T", err)
	}
	return message
}

var upstreamSensitiveMetadataKeys = []string{
	upstreamCredentialAPIKey,
	upstreamCredentialOpenAIAPIKey,
	upstreamCredentialAnthropicAPIKey,
	upstreamCredentialGeminiAPIKey,
	upstreamCredentialGrokAPIKey,
	upstreamCredentialManagementAccessToken,
	upstreamCredentialManagementUserID,
	upstreamCredentialUsername,
	upstreamCredentialPassword,
	"base_url",
}

// safeUpstreamMetadata protects responses built from previously persisted probe data.
// Older probe records may contain fields that were not sanitized when they were saved.
func safeUpstreamMetadata(metadata map[string]any, item *dbent.Upstream) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	redacted := logredact.RedactMap(metadata, upstreamSensitiveMetadataKeys...)
	result, ok := redactUpstreamMetadataValue(redacted, item).(map[string]any)
	if !ok || result == nil {
		return map[string]any{}
	}
	return result
}

func redactUpstreamMetadataValue(value any, item *dbent.Upstream) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, nested := range typed {
			result[key] = redactUpstreamMetadataValue(nested, item)
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, nested := range typed {
			result[index] = redactUpstreamMetadataValue(nested, item)
		}
		return result
	case string:
		return redactUpstreamText(typed, item)
	default:
		return value
	}
}

func redactUpstreamText(message string, item *dbent.Upstream) string {
	message = logredact.RedactText(message, upstreamSensitiveMetadataKeys...)
	if item == nil {
		return message
	}
	if baseURL := strings.TrimSpace(item.BaseURL); baseURL != "" {
		message = strings.ReplaceAll(message, baseURL, "upstream")
	}
	for _, credential := range upstreamCredentialValues(item.Credentials) {
		// Avoid replacing short values such as a one-digit user id throughout an
		// otherwise harmless message. Key-shaped credentials are always longer.
		if len([]rune(credential)) < 4 {
			continue
		}
		message = strings.ReplaceAll(message, credential, "[REDACTED]")
	}
	return message
}

func upstreamCredentialValues(credentials map[string]any) []string {
	seen := make(map[string]struct{})
	values := make([]string, 0)
	appendValue := func(raw string) {
		value := strings.TrimSpace(raw)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	for _, key := range []string{
		upstreamCredentialAPIKey,
		upstreamCredentialOpenAIAPIKey,
		upstreamCredentialAnthropicAPIKey,
		upstreamCredentialGeminiAPIKey,
		upstreamCredentialGrokAPIKey,
		upstreamCredentialManagementAccessToken,
		upstreamCredentialManagementUserID,
		upstreamCredentialUsername,
		upstreamCredentialPassword,
	} {
		appendValue(upstreamCredentialString(credentials, key))
	}

	// generated_group_keys also stores non-secret metadata such as group_name,
	// kind, and created_at. Redacting every nested string caused legitimate
	// group names to be displayed as [REDACTED]. Only the generated API key is a
	// credential here.
	rawGenerated, ok := credentials[upstreamCredentialGeneratedGroupKeys]
	if !ok {
		return values
	}
	encoded, err := json.Marshal(rawGenerated)
	if err != nil {
		return values
	}
	var generated map[string]storedGeneratedUpstreamKey
	if err := json.Unmarshal(encoded, &generated); err != nil {
		return values
	}
	for _, stored := range generated {
		appendValue(stored.APIKey)
	}
	return values
}
