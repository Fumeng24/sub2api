package admin

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	entupstream "github.com/Wei-Shaw/sub2api/ent/upstream"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	maxUpstreamBatchProbeModels             = 100
	upstreamModelProbeConcurrency           = 3
	upstreamModelProbeTimeout               = 90 * time.Second
	upstreamModelProbeMaxAttempts           = 3
	upstreamModelProbeRetryBaseDelay        = 200 * time.Millisecond
	upstreamModelProbeMaxRetryAfter         = 2 * time.Second
	upstreamModelVerificationTTL            = 15 * time.Minute
	upstreamNewAPIGroupCatalogueConcurrency = 4
	upstreamGeneratedKeyCleanupTimeout      = 30 * time.Second
)

var errUpstreamNewAPIGroupModelsEmpty = errors.New("The selected NewAPI group returned no supported models")

type upstreamNewAPIGroupCatalogueSummary struct {
	TotalGroups     int
	AvailableGroups int
	EmptyGroups     int
	FailedGroups    int
	Models          []string
}

type upstreamModelsProbeRequest struct {
	Platform  string   `json:"platform"`
	GroupName string   `json:"group_name"`
	Models    []string `json:"models"`
	APIKey    string   `json:"api_key,omitempty"`
}

type upstreamModelsProbeResponse struct {
	Success         bool                       `json:"success"`
	Platform        string                     `json:"platform"`
	GroupName       string                     `json:"group_name"`
	Status          string                     `json:"status"`
	Message         string                     `json:"message,omitempty"`
	Source          string                     `json:"source,omitempty"`
	LatencyMs       int64                      `json:"latency_ms"`
	AvailableModels []string                   `json:"available_models"`
	Results         []upstreamModelProbeResult `json:"results"`
}

type upstreamSub2APIAvailableChannel struct {
	Platforms []upstreamSub2APIAvailableChannelPlatform `json:"platforms"`
}

type upstreamSub2APIAvailableChannelPlatform struct {
	Platform        string                                 `json:"platform"`
	Groups          []upstreamSub2APIAvailableChannelGroup `json:"groups"`
	SupportedModels []upstreamSub2APIAvailableChannelModel `json:"supported_models"`
}

type upstreamSub2APIAvailableChannelGroup struct {
	ID              int64                                  `json:"id"`
	Name            string                                 `json:"name"`
	Platform        string                                 `json:"platform"`
	SupportedModels []upstreamSub2APIAvailableChannelModel `json:"supported_models"`
}

type upstreamSub2APIAvailableChannelModel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Model    string `json:"model"`
	ModelID  string `json:"model_id"`
	Platform string `json:"platform"`
}

type upstreamModelVerificationKey struct {
	UpstreamID     int64
	ProxyID        int64
	Platform       string
	GroupName      string
	Model          string
	KeyFingerprint [32]byte
}

type upstreamModelVerification struct {
	VerifiedAt time.Time
	ExpiresAt  time.Time
}

type upstreamModelProbeTaskResult struct {
	Index int
	Entry upstreamModelProbeResult
}

// ProbeModels discovers candidates when models is empty. A non-empty request
// invokes every model through the selected group's actual API key and protocol;
// catalogue membership alone never marks a model as supported.
func (h *UpstreamHandler) ProbeModels(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	var req upstreamModelsProbeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	req.Platform = strings.ToLower(strings.TrimSpace(req.Platform))
	req.GroupName = strings.TrimSpace(req.GroupName)
	req.APIKey = strings.TrimSpace(req.APIKey)
	req.Models = dedupeStrings(req.Models)
	if !isManagedUpstreamPlatform(req.Platform) || req.GroupName == "" {
		response.BadRequest(c, "platform and group_name are required")
		return
	}
	if len(req.Models) > maxUpstreamBatchProbeModels {
		response.BadRequest(c, fmt.Sprintf("models cannot contain more than %d entries", maxUpstreamBatchProbeModels))
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

	response.Success(c, h.probeUpstreamModels(c.Request.Context(), item, metadata, req))
}

func (h *UpstreamHandler) probeUpstreamModels(
	ctx context.Context,
	item *dbent.Upstream,
	metadata upstreamProbeMetadata,
	req upstreamModelsProbeRequest,
) upstreamModelsProbeResponse {
	startedAt := time.Now()
	result := upstreamModelsProbeResponse{
		Platform:        req.Platform,
		GroupName:       req.GroupName,
		Status:          "error",
		AvailableModels: []string{},
		Results:         make([]upstreamModelProbeResult, 0, len(req.Models)),
	}
	if len(req.Models) == 0 {
		models, source, catalogueErr := h.fetchLiveUpstreamGroupModels(
			ctx, item, metadata, req.Platform, req.GroupName, req.APIKey,
		)
		result.Source = source
		result.LatencyMs = time.Since(startedAt).Milliseconds()
		if catalogueErr != nil {
			// A management catalogue is useful for discovery, but it is not the
			// authority for model support. Keep the UI usable when /v1/models or a
			// panel catalogue is unavailable; every returned candidate still has to
			// pass the explicit real-request probe before it can be selected.
			if candidates, candidateSource := h.fallbackUpstreamModelCandidates(
				ctx, item, metadata, req.Platform, req.GroupName,
			); len(candidates) > 0 {
				result.AvailableModels = candidates
				result.Source = candidateSource
				result.Success = true
				result.Status = "candidates"
				result.Message = "Upstream model catalogue is unavailable; candidates require an actual model request to confirm support"
				return result
			}
			result.Message = safeUpstreamModelProbeError(catalogueErr, item, req.APIKey)
			return result
		}
		result.AvailableModels = models
		result.Success = true
		result.Status = "ok"
		return result
	}

	account, apiKey, probeSource, err := h.resolveActualUpstreamModelProbeAccount(
		ctx, item, metadata, req.Platform, req.GroupName, req.APIKey,
	)
	if err != nil {
		message := safeUpstreamModelProbeError(err, item, req.APIKey)
		result.Message = message
		result.Results = unconfirmedUpstreamModelResults(req, message)
		result.LatencyMs = time.Since(startedAt).Milliseconds()
		return result
	}
	result.Source = probeSource

	result.Results = h.runActualUpstreamModelProbes(ctx, item, account, apiKey, req)
	result.LatencyMs = time.Since(startedAt).Milliseconds()
	result.Success, result.Status, result.Message = summarizeActualUpstreamModelProbes(result.Results)
	return result
}

func (h *UpstreamHandler) resolveActualUpstreamModelProbeAccount(
	ctx context.Context,
	item *dbent.Upstream,
	metadata upstreamProbeMetadata,
	platform, groupName, apiKeyOverride string,
) (*service.Account, string, string, error) {
	if h == nil || h.accountTestService == nil || item == nil {
		return nil, "", "", errors.New("Model capability probe is unavailable")
	}
	if apiKey := strings.TrimSpace(apiKeyOverride); apiKey != "" {
		return h.transientUpstreamAccount(ctx, item, platform, apiKey), apiKey, "request_override", nil
	}
	_, apiKey, source := h.resolveUpstreamGroupProbeAccount(ctx, item, metadata, platform, groupName)
	if apiKey != "" {
		return h.transientUpstreamAccount(ctx, item, platform, apiKey), apiKey, source, nil
	}
	current, apiKey, source, err := h.ensureUpstreamGroupProbeKey(ctx, item, metadata, platform, groupName)
	if err != nil {
		return nil, "", "", err
	}
	if current == nil || apiKey == "" {
		return nil, "", "", errors.New("No API key is available for the selected upstream group")
	}
	return h.transientUpstreamAccount(ctx, current, platform, apiKey), apiKey, source, nil
}

func (h *UpstreamHandler) runActualUpstreamModelProbes(
	ctx context.Context,
	item *dbent.Upstream,
	account *service.Account,
	apiKey string,
	req upstreamModelsProbeRequest,
) []upstreamModelProbeResult {
	workers := upstreamModelProbeConcurrency
	if len(req.Models) < workers {
		workers = len(req.Models)
	}
	// Keep the HTTP connection-pool limit aligned with the bounded worker pool.
	// Three concurrent probes keep large model lists practical without creating
	// an unbounded burst at the upstream.
	account.Concurrency = workers
	tasks := make(chan int)
	completed := make(chan upstreamModelProbeTaskResult, len(req.Models))
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range tasks {
				model := req.Models[index]
				probeCtx, cancel := context.WithTimeout(ctx, upstreamModelProbeTimeout)
				startedAt := time.Now()
				probe := h.probeUpstreamModelWithRetry(probeCtx, account, model)
				cancel()
				entry := upstreamModelProbeResult{
					Platform:   req.Platform,
					GroupName:  req.GroupName,
					Model:      model,
					LatencyMs:  time.Since(startedAt).Milliseconds(),
					Status:     probe.Status,
					StatusCode: probe.StatusCode,
					Message:    safeUpstreamModelProbeError(errors.New(probe.Message), item, apiKey),
				}
				if probe.Status == service.ModelCapabilityStatusOK {
					entry.Success = true
					verifiedAt, expiresAt := h.rememberUpstreamModelVerification(
						item, req.Platform, req.GroupName, model, apiKey,
					)
					entry.VerifiedAt = &verifiedAt
					entry.ExpiresAt = &expiresAt
				}
				completed <- upstreamModelProbeTaskResult{Index: index, Entry: entry}
			}
		}()
	}
	go func() {
		defer close(tasks)
		for index := range req.Models {
			select {
			case tasks <- index:
			case <-ctx.Done():
				return
			}
		}
	}()
	wg.Wait()
	close(completed)

	results := make([]upstreamModelProbeResult, len(req.Models))
	seen := make([]bool, len(req.Models))
	for result := range completed {
		results[result.Index] = result.Entry
		seen[result.Index] = true
	}
	for index, ok := range seen {
		if ok {
			continue
		}
		results[index] = upstreamModelProbeResult{
			Platform: req.Platform, GroupName: req.GroupName, Model: req.Models[index],
			Status: service.ModelCapabilityStatusUnconfirmed, Message: "Model verification was cancelled",
		}
	}
	return results
}

func (h *UpstreamHandler) probeUpstreamModelWithRetry(
	ctx context.Context,
	account *service.Account,
	model string,
) service.ModelCapabilityProbeResult {
	var result service.ModelCapabilityProbeResult
	for attempt := 1; attempt <= upstreamModelProbeMaxAttempts; attempt++ {
		result = h.accountTestService.ProbeAPIKeyModel(ctx, account, model)
		if result.StatusCode != http.StatusTooManyRequests || attempt == upstreamModelProbeMaxAttempts {
			return result
		}

		delay, retry := upstreamModelProbeRetryDelay(result.RetryAfter, attempt)
		if !retry {
			return result
		}
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return result
		}
	}
	return result
}

func upstreamModelProbeRetryDelay(retryAfter time.Duration, attempt int) (time.Duration, bool) {
	if retryAfter > 0 {
		if retryAfter > upstreamModelProbeMaxRetryAfter {
			return 0, false
		}
		return retryAfter, true
	}
	delay := upstreamModelProbeRetryBaseDelay << (attempt - 1)
	if delay > upstreamModelProbeMaxRetryAfter {
		delay = upstreamModelProbeMaxRetryAfter
	}
	return delay, true
}

func unconfirmedUpstreamModelResults(req upstreamModelsProbeRequest, message string) []upstreamModelProbeResult {
	results := make([]upstreamModelProbeResult, 0, len(req.Models))
	for _, model := range req.Models {
		results = append(results, upstreamModelProbeResult{
			Platform: req.Platform, GroupName: req.GroupName, Model: model,
			Status: service.ModelCapabilityStatusUnconfirmed, Message: message,
		})
	}
	return results
}

func summarizeActualUpstreamModelProbes(results []upstreamModelProbeResult) (bool, string, string) {
	if len(results) == 0 {
		return false, "error", "No models were tested"
	}
	okCount := 0
	unsupportedCount := 0
	for _, result := range results {
		switch result.Status {
		case service.ModelCapabilityStatusOK:
			okCount++
		case service.ModelCapabilityStatusUnsupported:
			unsupportedCount++
		}
	}
	if okCount == len(results) {
		return true, "ok", ""
	}
	unconfirmedCount := len(results) - okCount - unsupportedCount
	status := "partial"
	if okCount == 0 && unsupportedCount == 0 {
		status = "error"
	}
	return false, status, fmt.Sprintf(
		"%d available, %d unsupported, %d temporarily unconfirmed",
		okCount, unsupportedCount, unconfirmedCount,
	)
}

func (h *UpstreamHandler) rememberUpstreamModelVerification(
	item *dbent.Upstream,
	platform, groupName, model, apiKey string,
) (time.Time, time.Time) {
	now := time.Now().UTC()
	expiresAt := now.Add(upstreamModelVerificationTTL)
	key := buildUpstreamModelVerificationKey(item, platform, groupName, model, apiKey)
	h.modelVerifications.Store(key, upstreamModelVerification{VerifiedAt: now, ExpiresAt: expiresAt})
	return now, expiresAt
}

func buildUpstreamModelVerificationKey(
	item *dbent.Upstream,
	platform, groupName, model, apiKey string,
) upstreamModelVerificationKey {
	upstreamID := int64(0)
	proxyID := int64(0)
	baseURL := ""
	if item != nil {
		upstreamID = item.ID
		baseURL = strings.TrimSpace(item.BaseURL)
		if item.ProxyID != nil {
			proxyID = *item.ProxyID
		}
	}
	return upstreamModelVerificationKey{
		UpstreamID:     upstreamID,
		ProxyID:        proxyID,
		Platform:       strings.ToLower(strings.TrimSpace(platform)),
		GroupName:      normalizeUpstreamGroupKey(groupName),
		Model:          strings.TrimSpace(model),
		KeyFingerprint: sha256.Sum256([]byte(baseURL + "\x00" + strings.TrimSpace(apiKey))),
	}
}

func (h *UpstreamHandler) unverifiedUpstreamModels(
	item *dbent.Upstream,
	platform, groupName string,
	models []string,
	apiKey string,
	now time.Time,
) []string {
	missing := make([]string, 0)
	for _, model := range models {
		key := buildUpstreamModelVerificationKey(item, platform, groupName, model, apiKey)
		value, ok := h.modelVerifications.Load(key)
		if !ok {
			missing = append(missing, model)
			continue
		}
		verification, ok := value.(upstreamModelVerification)
		if !ok || !verification.ExpiresAt.After(now) {
			h.modelVerifications.Delete(key)
			missing = append(missing, model)
		}
	}
	return missing
}

func (h *UpstreamHandler) fetchLiveUpstreamGroupModels(
	ctx context.Context,
	item *dbent.Upstream,
	metadata upstreamProbeMetadata,
	platform, groupName, apiKeyOverride string,
) ([]string, string, error) {
	if h == nil || item == nil {
		return nil, "", errors.New("Upstream model capability probe is unavailable")
	}
	// A key supplied for the account being planned is more specific than the
	// upstream management catalogue. Validate it first so an inaccessible key is
	// never represented as usable just because another key can see the group.
	if apiKeyOverride != "" {
		if h.accountTestService == nil {
			return nil, "", errors.New("Model capability probe is unavailable")
		}
		account := h.transientUpstreamAccount(ctx, item, platform, apiKeyOverride)
		models, err := h.accountTestService.FetchUpstreamSupportedModels(ctx, account)
		if err != nil {
			return nil, "", err
		}
		return dedupeStrings(models), "request_override", nil
	}

	kind := item.Kind
	if kind == entupstream.KindAuto {
		switch strings.ToLower(strings.TrimSpace(metadata.DetectedKind)) {
		case entupstream.KindNewapi.String():
			kind = entupstream.KindNewapi
		case entupstream.KindSub2api.String():
			kind = entupstream.KindSub2api
		}
	}

	var managementErr error
	switch {
	case h.panelClient == nil:
		managementErr = errors.New("Upstream management model endpoint is unavailable")
	case kind == entupstream.KindNewapi:
		if models, err := h.fetchNewAPIGroupModels(ctx, item, groupName); err == nil {
			return models, "newapi_group_models", nil
		} else {
			managementErr = err
		}
	case kind == entupstream.KindSub2api:
		if models, err := h.fetchSub2APIGroupModels(ctx, item, platform, groupName); err == nil {
			return models, "sub2api_available_channels", nil
		} else {
			managementErr = err
		}
	}

	account, key, source := h.resolveUpstreamGroupProbeAccount(ctx, item, metadata, platform, groupName)
	if key != "" {
		if account == nil {
			account = h.transientUpstreamAccount(ctx, item, platform, key)
		}
		if h.accountTestService == nil {
			return nil, "", errors.New("Model capability probe is unavailable")
		}
		models, err := h.accountTestService.FetchUpstreamSupportedModels(ctx, account)
		if err == nil {
			return dedupeStrings(models), source, nil
		}
		return nil, "", err
	}
	if generatedModels, generatedSource, generatedErr := h.probeUpstreamGroupWithGeneratedKey(ctx, item, metadata, platform, groupName); generatedErr == nil {
		return generatedModels, generatedSource, nil
	} else {
		// The generated key is scoped to the selected group, so its failure is
		// more specific than an earlier management-catalogue failure.
		managementErr = generatedErr
	}
	if managementErr != nil {
		return nil, "", managementErr
	}
	return nil, "", errors.New("No verifiable API key or management model endpoint is available for the selected group")
}

// ensureUpstreamGroupProbeKey returns a key scoped to the selected upstream
// group. Creating a key through the management API is sufficient to establish
// that the credential exists; /v1/models is deliberately not a prerequisite
// because many upstreams reject or omit that catalogue endpoint.
func (h *UpstreamHandler) ensureUpstreamGroupProbeKey(
	ctx context.Context,
	item *dbent.Upstream,
	metadata upstreamProbeMetadata,
	platform, groupName string,
) (*dbent.Upstream, string, string, error) {
	if h == nil || item == nil || h.client == nil || h.accountTestService == nil {
		return nil, "", "", errors.New("No model capability probe is available for the selected group")
	}
	kind := item.Kind
	if kind == entupstream.KindAuto {
		switch strings.ToLower(strings.TrimSpace(metadata.DetectedKind)) {
		case entupstream.KindNewapi.String():
			kind = entupstream.KindNewapi
		case entupstream.KindSub2api.String():
			kind = entupstream.KindSub2api
		}
	}
	target := findUpstreamProbeGroup(metadata.Groups, platform, groupName)
	if target == nil {
		return nil, "", "", errors.New("The selected upstream group is not present in the latest probe")
	}
	spec := normalizedUpstreamAccountGenerationSpec{
		upstreamAccountGenerationSpec: upstreamAccountGenerationSpec{
			Platform:          platform,
			UpstreamGroupName: groupName,
			UpstreamGroupID:   target.ID,
		},
	}

	lock := h.generatedKeyLock(item.ID, groupName)
	lock.Lock()
	defer lock.Unlock()
	current, err := h.client.Upstream.Get(ctx, item.ID)
	if err != nil {
		return nil, "", "", err
	}
	if stored := lookupStoredGeneratedKey(current.Credentials, groupName); stored.APIKey != "" {
		return current, stored.APIKey, "stored_group_key", nil
	}

	managementItem := *current
	managementItem.Kind = kind
	if !canCreateRemoteUpstreamKey(&managementItem, spec) {
		return nil, "", "", errors.New("No management credentials are available to create a key for the selected upstream group")
	}
	if h.panelClient == nil {
		return nil, "", "", errors.New("Upstream key management is unavailable")
	}
	created, err := h.createRemoteUpstreamGroupKey(ctx, &managementItem, spec)
	if err != nil {
		return nil, "", "", err
	}
	if err := h.persistGeneratedUpstreamKey(ctx, item.ID, created); err != nil {
		cleanupErr := runUpstreamGeneratedKeyCleanup(ctx, func(cleanupCtx context.Context) error {
			return h.deleteRemoteUpstreamGroupKey(cleanupCtx, &managementItem, created)
		})
		if cleanupErr != nil {
			return nil, "", "", fmt.Errorf("verified group key storage failed and remote cleanup failed: %w", cleanupErr)
		}
		return nil, "", "", errors.New("The verified upstream group key could not be stored locally")
	}
	current.Credentials = mergeStoredGeneratedKey(current.Credentials, created)
	return current, created.APIKey, "generated_group_key", nil
}

// probeUpstreamGroupWithGeneratedKey handles groups which are visible in the
// management directory but intentionally omitted from the public catalogue
// (for example a Sub2API codefree group). The generated key is persisted first;
// the catalogue request is only an optional discovery step.
func (h *UpstreamHandler) probeUpstreamGroupWithGeneratedKey(
	ctx context.Context,
	item *dbent.Upstream,
	metadata upstreamProbeMetadata,
	platform, groupName string,
) ([]string, string, error) {
	current, apiKey, source, err := h.ensureUpstreamGroupProbeKey(ctx, item, metadata, platform, groupName)
	if err != nil {
		return nil, "", err
	}
	account := h.transientUpstreamAccount(ctx, current, platform, apiKey)
	models, probeErr := h.accountTestService.FetchUpstreamSupportedModels(ctx, account)
	models = dedupeStrings(models)
	if probeErr != nil || len(models) == 0 {
		if probeErr != nil {
			return nil, "", fmt.Errorf("generated group key model probe failed: %w", probeErr)
		}
		return nil, "", errors.New("The selected upstream group returned no supported models")
	}
	return models, source, nil
}

func (h *UpstreamHandler) fallbackUpstreamModelCandidates(
	ctx context.Context,
	item *dbent.Upstream,
	metadata upstreamProbeMetadata,
	platform, groupName string,
) ([]string, string) {
	models := make([]string, 0)
	if target := findUpstreamProbeGroup(metadata.Groups, platform, groupName); target != nil {
		models = append(models, target.Models...)
	}
	for _, protocol := range metadata.Protocols {
		if strings.EqualFold(strings.TrimSpace(protocol.Platform), strings.TrimSpace(platform)) {
			models = append(models, protocol.Models...)
		}
	}
	// id=0 deliberately uses the platform's built-in candidate catalogue. It
	// is only a probe candidate source and is never persisted as support.
	if h != nil && h.adminService != nil {
		if candidates, err := h.adminService.GetGroupModelsListCandidates(ctx, 0, platform); err == nil {
			models = append(models, candidates...)
		}
	}
	models = dedupeStrings(models)
	if len(models) == 0 {
		return nil, ""
	}
	return models, "local_candidates"
}

func runUpstreamGeneratedKeyCleanup(parent context.Context, cleanup func(context.Context) error) error {
	if cleanup == nil {
		return nil
	}
	if parent == nil {
		parent = context.Background()
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), upstreamGeneratedKeyCleanupTimeout)
	defer cancel()
	return cleanup(cleanupCtx)
}

func (h *UpstreamHandler) fetchNewAPIGroupModels(ctx context.Context, item *dbent.Upstream, groupName string) ([]string, error) {
	probeAccount := h.transientUpstreamAccount(ctx, item, service.PlatformOpenAI, "")
	probeCtx, _, err := h.panelClient.contextForAccount(ctx, probeAccount)
	if err != nil {
		return nil, err
	}
	path := "/api/user/models?" + url.Values{"group": []string{groupName}}.Encode()
	var models []string
	accessToken := upstreamCredentialString(item.Credentials, upstreamCredentialManagementAccessToken)
	if accessToken != "" {
		session := upstreamNewAPIAccessTokenSession(item.Credentials)
		if session == nil {
			return nil, errUpstreamNewAPIManagementUserID
		}
		if _, _, err := h.panelClient.doNewAPIJSON(
			probeCtx, http.MethodGet, joinUpstreamSub2APIURL(item.BaseURL, path), session, accessToken, nil, &models,
		); err != nil {
			return nil, err
		}
	} else {
		username := upstreamCredentialString(item.Credentials, upstreamCredentialUsername)
		password := upstreamCredentialString(item.Credentials, upstreamCredentialPassword)
		if username == "" || password == "" {
			return nil, errors.New("NewAPI management credentials are not configured")
		}
		if err := h.panelClient.getNewAPIAuthenticatedJSON(probeCtx, item.BaseURL, username, password, path, &models); err != nil {
			return nil, err
		}
	}
	models = dedupeStrings(models)
	if len(models) == 0 {
		return nil, errUpstreamNewAPIGroupModelsEmpty
	}
	return models, nil
}

// applyNewAPIGroupModelCatalogue enriches a full manual probe with NewAPI's
// official group-scoped model directory. The default relay key's /v1/models
// response is key-specific and must not be treated as the whole site's model
// catalogue.
func (h *UpstreamHandler) applyNewAPIGroupModelCatalogue(
	ctx context.Context,
	item *dbent.Upstream,
	metadata *upstreamProbeMetadata,
) upstreamNewAPIGroupCatalogueSummary {
	summary := upstreamNewAPIGroupCatalogueSummary{}
	if h == nil || item == nil || metadata == nil || len(metadata.Groups) == 0 {
		return summary
	}

	type catalogueResult struct {
		index  int
		models []string
		err    error
	}
	results := make(chan catalogueResult, len(metadata.Groups))
	sem := make(chan struct{}, upstreamNewAPIGroupCatalogueConcurrency)
	var wg sync.WaitGroup

	for index := range metadata.Groups {
		groupName := strings.TrimSpace(metadata.Groups[index].Name)
		if groupName == "" {
			continue
		}
		summary.TotalGroups++
		wg.Add(1)
		go func(index int, groupName string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results <- catalogueResult{index: index, err: ctx.Err()}
				return
			}
			models, err := h.fetchNewAPIGroupModels(ctx, item, groupName)
			results <- catalogueResult{index: index, models: models, err: err}
		}(index, groupName)
	}

	wg.Wait()
	close(results)
	for result := range results {
		if result.err != nil {
			if errors.Is(result.err, errUpstreamNewAPIGroupModelsEmpty) {
				summary.EmptyGroups++
			} else {
				summary.FailedGroups++
			}
			continue
		}
		models := dedupeStrings(result.models)
		if len(models) == 0 {
			summary.EmptyGroups++
			continue
		}
		metadata.Groups[result.index].Models = models
		summary.AvailableGroups++
		summary.Models = append(summary.Models, models...)
	}
	summary.Models = dedupeStrings(summary.Models)
	return summary
}

func applyNewAPIProtocolFromGroupCatalogue(
	metadata *upstreamProbeMetadata,
	summary upstreamNewAPIGroupCatalogueSummary,
) {
	if metadata == nil || summary.AvailableGroups == 0 {
		return
	}
	for index := range metadata.Protocols {
		protocol := &metadata.Protocols[index]
		if protocol.Platform != service.PlatformOpenAI {
			continue
		}
		protocol.Status = "ok"
		protocol.Models = append([]string(nil), summary.Models...)
		protocol.Message = ""
		return
	}
	metadata.Protocols = append(metadata.Protocols, upstreamProtocolCapability{
		Platform:  service.PlatformOpenAI,
		Status:    "ok",
		Models:    append([]string(nil), summary.Models...),
		FetchedAt: metadata.FetchedAt,
	})
}

func (h *UpstreamHandler) fetchSub2APIGroupModels(
	ctx context.Context,
	item *dbent.Upstream,
	platform, groupName string,
) ([]string, error) {
	channels, err := h.fetchSub2APIAvailableChannels(ctx, item, platform)
	if err != nil {
		return nil, err
	}
	models := modelsForSub2APIGroup(channels, platform, groupName)
	if len(models) == 0 {
		return nil, errors.New("The selected Sub2API group is not exposed by the upstream model catalogue")
	}
	return models, nil
}

func (h *UpstreamHandler) fetchSub2APIAvailableChannels(
	ctx context.Context,
	item *dbent.Upstream,
	platform string,
) ([]upstreamSub2APIAvailableChannel, error) {
	username := upstreamCredentialString(item.Credentials, upstreamCredentialUsername)
	password := upstreamCredentialString(item.Credentials, upstreamCredentialPassword)
	if username == "" || password == "" {
		return nil, errors.New("Sub2API panel username or password is not configured")
	}
	probeAccount := h.transientUpstreamAccount(ctx, item, platform, "")
	probeCtx, _, err := h.panelClient.contextForAccount(ctx, probeAccount)
	if err != nil {
		return nil, err
	}
	var channels []upstreamSub2APIAvailableChannel
	if err := h.panelClient.getAuthenticatedJSON(
		probeCtx, item.BaseURL, username, password, "/api/v1/channels/available", &channels,
	); err != nil {
		return nil, err
	}
	return channels, nil
}

func modelsForSub2APIGroup(channels []upstreamSub2APIAvailableChannel, platform, groupName string) []string {
	models := make([]string, 0)
	for _, channel := range channels {
		for _, section := range channel.Platforms {
			if !strings.EqualFold(strings.TrimSpace(section.Platform), platform) {
				continue
			}
			for _, group := range section.Groups {
				if !strings.EqualFold(strings.TrimSpace(group.Name), strings.TrimSpace(groupName)) {
					continue
				}
				for _, model := range group.SupportedModels {
					models = append(models, upstreamAvailableChannelModelID(model))
				}
				// Older Sub2API versions expose models on the channel's platform
				// section instead of repeating them on every group. This fallback
				// remains scoped to the exact channel section containing the group;
				// it never mixes another channel or platform into the result.
				if len(group.SupportedModels) == 0 {
					for _, model := range section.SupportedModels {
						models = append(models, upstreamAvailableChannelModelID(model))
					}
				}
			}
		}
	}
	return dedupeStrings(models)
}

func upstreamAvailableChannelModelID(model upstreamSub2APIAvailableChannelModel) string {
	for _, value := range []string{model.ID, model.Name, model.Model, model.ModelID} {
		if value = strings.TrimSpace(value); value != "" {
			return strings.TrimPrefix(value, "models/")
		}
	}
	return ""
}

func applySub2APIModelCatalogue(metadata *upstreamProbeMetadata, channels []upstreamSub2APIAvailableChannel) {
	if metadata == nil {
		return
	}
	modelsByPlatform := make(map[string][]string)
	for index := range metadata.Groups {
		group := &metadata.Groups[index]
		models := modelsForSub2APIGroup(channels, group.Platform, group.Name)
		if len(models) == 0 {
			continue
		}
		group.Models = models
		modelsByPlatform[group.Platform] = append(modelsByPlatform[group.Platform], models...)
	}

	for platform, models := range modelsByPlatform {
		models = dedupeStrings(models)
		if len(models) == 0 {
			continue
		}
		found := false
		for index := range metadata.Protocols {
			if metadata.Protocols[index].Platform != platform {
				continue
			}
			metadata.Protocols[index].Status = "ok"
			metadata.Protocols[index].Models = models
			metadata.Protocols[index].Message = ""
			found = true
			break
		}
		if !found {
			metadata.Protocols = append(metadata.Protocols, upstreamProtocolCapability{
				Platform:  platform,
				Status:    "ok",
				Models:    models,
				FetchedAt: metadata.FetchedAt,
			})
		}
	}
}

func upstreamProbeGroupExists(groups []upstreamProbeGroup, platform, groupName string) bool {
	for _, group := range groups {
		if !strings.EqualFold(strings.TrimSpace(group.Name), strings.TrimSpace(groupName)) {
			continue
		}
		if group.Platform == "" || strings.EqualFold(strings.TrimSpace(group.Platform), platform) {
			return true
		}
	}
	return false
}

func (h *UpstreamHandler) resolveUpstreamGroupProbeAccount(
	ctx context.Context,
	item *dbent.Upstream,
	metadata upstreamProbeMetadata,
	platform, groupName string,
) (*service.Account, string, string) {
	if stored := lookupStoredGeneratedKey(item.Credentials, groupName); stored.APIKey != "" {
		return nil, stored.APIKey, "stored_group_key"
	}

	for _, billing := range metadata.AccountBilling {
		if billing.AccountID <= 0 || !strings.EqualFold(strings.TrimSpace(billing.UpstreamGroupName), strings.TrimSpace(groupName)) {
			continue
		}
		if billing.UpstreamGroupPlatform != "" && !strings.EqualFold(billing.UpstreamGroupPlatform, platform) {
			continue
		}
		if h.adminService != nil {
			account, err := h.adminService.GetAccount(ctx, billing.AccountID)
			if err == nil && account != nil && account.Platform == platform {
				if key := strings.TrimSpace(account.GetCredential("api_key")); key != "" {
					return account, key, "bound_group_account"
				}
			}
		}
	}

	defaultKey := upstreamAPIKeyForPlatform(item.Credentials, platform)
	if defaultKey == "" {
		return nil, "", ""
	}
	if metadata.Key != nil && strings.EqualFold(strings.TrimSpace(metadata.Key.GroupName), strings.TrimSpace(groupName)) {
		return nil, defaultKey, "stored_default_key"
	}
	matchingGroups := 0
	for _, group := range metadata.Groups {
		if group.Platform == "" || strings.EqualFold(strings.TrimSpace(group.Platform), platform) {
			matchingGroups++
		}
	}
	if matchingGroups == 1 {
		return nil, defaultKey, "stored_default_key"
	}
	return nil, "", ""
}
