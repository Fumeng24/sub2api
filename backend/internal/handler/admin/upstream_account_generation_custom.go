package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"sort"
	"strconv"
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
	upstreamNewAPIGeneratedKeyReadMaxAttempts    = 3
	upstreamNewAPIGeneratedKeyReadRetryBaseDelay = time.Second
)

type upstreamAccountGenerationRequest struct {
	Accounts []upstreamAccountGenerationSpec `json:"accounts"`
}

type upstreamAccountGenerationSpec struct {
	Name              string   `json:"name"`
	Platform          string   `json:"platform"`
	UpstreamGroupName string   `json:"upstream_group_name"`
	UpstreamGroupID   *int64   `json:"upstream_group_id"`
	Models            []string `json:"models"`
	LocalGroupIDs     []int64  `json:"local_group_ids"`
	Concurrency       int      `json:"concurrency"`
	Priority          *int     `json:"priority"`
	RateMultiplier    *float64 `json:"rate_multiplier"`
	APIKey            string   `json:"api_key"`
}

type upstreamAccountGenerationPreview struct {
	Index                 int      `json:"index"`
	Name                  string   `json:"name"`
	Platform              string   `json:"platform"`
	UpstreamGroupName     string   `json:"upstream_group_name"`
	UpstreamGroupID       *int64   `json:"upstream_group_id,omitempty"`
	Models                []string `json:"models"`
	LocalGroupIDs         []int64  `json:"local_group_ids"`
	Concurrency           int      `json:"concurrency"`
	Priority              int      `json:"priority"`
	RateMultiplier        *float64 `json:"rate_multiplier,omitempty"`
	Action                string   `json:"action"`
	ExistingAccountID     *int64   `json:"existing_account_id,omitempty"`
	KeySource             string   `json:"key_source,omitempty"`
	WillCreateUpstreamKey bool     `json:"will_create_upstream_key"`
	Warnings              []string `json:"warnings"`
	Errors                []string `json:"errors"`
}

type upstreamAccountGenerationPreviewResponse struct {
	Valid   bool                               `json:"valid"`
	Creates int                                `json:"creates"`
	Skips   int                                `json:"skips"`
	Items   []upstreamAccountGenerationPreview `json:"items"`
	specs   []normalizedUpstreamAccountGenerationSpec
}

type normalizedUpstreamAccountGenerationSpec struct {
	upstreamAccountGenerationSpec
	Name           string
	Models         []string
	LocalGroupIDs  []int64
	Concurrency    int
	Priority       int
	ResolvedAPIKey string
	KeySource      string
	Action         string
	ExistingID     *int64
}

type upstreamAccountGenerationResult struct {
	Index             int    `json:"index"`
	Success           bool   `json:"success"`
	Skipped           bool   `json:"skipped"`
	AccountID         *int64 `json:"account_id,omitempty"`
	ExistingAccountID *int64 `json:"existing_account_id,omitempty"`
	Error             string `json:"error,omitempty"`
}

type storedGeneratedUpstreamKey struct {
	APIKey    string    `json:"api_key"`
	ID        int64     `json:"id"`
	Kind      string    `json:"kind"`
	GroupName string    `json:"group_name"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *UpstreamHandler) PreviewAccounts(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	item, err := h.queryUpstream(c, id, true)
	if err != nil {
		writeUpstreamQueryError(c, err)
		return
	}
	var req upstreamAccountGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Accounts) == 0 {
		response.BadRequest(c, "accounts is required")
		return
	}
	preview := h.previewUpstreamAccounts(c.Request.Context(), item, req.Accounts)
	response.Success(c, preview)
}

func (h *UpstreamHandler) GenerateAccounts(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	item, err := h.queryUpstream(c, id, true)
	if err != nil {
		writeUpstreamQueryError(c, err)
		return
	}
	var req upstreamAccountGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Accounts) == 0 {
		response.BadRequest(c, "accounts is required")
		return
	}
	preview := h.previewUpstreamAccounts(c.Request.Context(), item, req.Accounts)
	if !preview.Valid {
		response.Error(c, http.StatusConflict, "Account generation preview contains errors")
		return
	}

	results := make([]upstreamAccountGenerationResult, 0, len(preview.specs))
	metadata, _ := parseUpstreamProbeMetadata(item.Metadata)
	for index, spec := range preview.specs {
		result := upstreamAccountGenerationResult{Index: index}
		if spec.Action == "skip" {
			result.Success = true
			result.Skipped = true
			result.ExistingAccountID = spec.ExistingID
			results = append(results, result)
			continue
		}

		apiKey := spec.ResolvedAPIKey
		if spec.KeySource == "remote_new_key" {
			verifiedModels, _, probeErr := h.probeUpstreamGroupWithGeneratedKey(
				c.Request.Context(), item, metadata, spec.Platform, spec.UpstreamGroupName,
			)
			if probeErr != nil {
				result.Error = safeUpstreamProbeError(probeErr, item)
				results = append(results, result)
				continue
			}
			if unsupported := modelsNotInCatalogue(spec.Models, verifiedModels); len(unsupported) > 0 {
				result.Error = "One or more selected models are not available from the generated upstream group key"
				results = append(results, result)
				continue
			}
			current, loadErr := h.client.Upstream.Get(c.Request.Context(), item.ID)
			if loadErr != nil {
				result.Error = "The verified upstream group key could not be loaded"
				results = append(results, result)
				continue
			}
			stored := lookupStoredGeneratedKey(current.Credentials, spec.UpstreamGroupName)
			if stored.APIKey == "" {
				result.Error = "The verified upstream group key is unavailable"
				results = append(results, result)
				continue
			}
			apiKey = stored.APIKey
			item.Credentials = current.Credentials
		}
		if unverified := h.unverifiedUpstreamModels(
			item,
			spec.Platform,
			spec.UpstreamGroupName,
			spec.Models,
			apiKey,
			time.Now().UTC(),
		); len(unverified) > 0 {
			result.Error = "Models no longer have a valid successful request verification: " + strings.Join(unverified, ", ")
			results = append(results, result)
			continue
		}

		credentials := map[string]any{
			"base_url":      item.BaseURL,
			"api_key":       apiKey,
			"model_mapping": identityModelMapping(spec.Models),
		}
		extra := map[string]any{
			"upstream_generated":     true,
			"upstream_group_name":    spec.UpstreamGroupName,
			"upstream_generated_at":  time.Now().UTC().Format(time.RFC3339),
			"upstream_management_id": item.ID,
		}
		if spec.UpstreamGroupID != nil {
			extra["upstream_group_id"] = *spec.UpstreamGroupID
		}
		created, createErr := h.adminService.CreateAccount(c.Request.Context(), &service.CreateAccountInput{
			Name:                 spec.Name,
			Platform:             spec.Platform,
			Type:                 service.AccountTypeAPIKey,
			Credentials:          credentials,
			Extra:                extra,
			ProxyID:              item.ProxyID,
			UpstreamID:           &item.ID,
			Concurrency:          spec.Concurrency,
			Priority:             spec.Priority,
			RateMultiplier:       spec.RateMultiplier,
			GroupIDs:             spec.LocalGroupIDs,
			SkipDefaultGroupBind: true,
		})
		if createErr != nil {
			result.Error = "Local account creation failed"
			results = append(results, result)
			continue
		}
		result.Success = true
		result.AccountID = &created.ID
		results = append(results, result)
	}
	response.Success(c, gin.H{"results": results})
}

func modelsNotInCatalogue(selected, available []string) []string {
	allowed := make(map[string]struct{}, len(available))
	for _, model := range available {
		allowed[model] = struct{}{}
	}
	missing := make([]string, 0)
	for _, model := range selected {
		if _, ok := allowed[model]; !ok {
			missing = append(missing, model)
		}
	}
	return missing
}

func (h *UpstreamHandler) previewUpstreamAccounts(ctx context.Context, item *dbent.Upstream, specs []upstreamAccountGenerationSpec) upstreamAccountGenerationPreviewResponse {
	responseValue := upstreamAccountGenerationPreviewResponse{
		Valid: true,
		Items: make([]upstreamAccountGenerationPreview, 0, len(specs)),
		specs: make([]normalizedUpstreamAccountGenerationSpec, 0, len(specs)),
	}
	metadata, _ := parseUpstreamProbeMetadata(item.Metadata)
	knownGroups := make(map[string]upstreamProbeGroup, len(metadata.Groups))
	for _, group := range metadata.Groups {
		knownGroups[normalizeUpstreamGroupKey(group.Name)] = group
	}
	protocolModels := make(map[string]map[string]struct{}, len(metadata.Protocols))
	for _, protocol := range metadata.Protocols {
		models := make(map[string]struct{}, len(protocol.Models))
		for _, model := range protocol.Models {
			models[model] = struct{}{}
		}
		protocolModels[protocol.Platform] = models
	}
	currentKeyGroup := ""
	if metadata.Key != nil {
		currentKeyGroup = normalizeUpstreamGroupKey(metadata.Key.GroupName)
	}

	for index, input := range specs {
		normalized := normalizedUpstreamAccountGenerationSpec{upstreamAccountGenerationSpec: input}
		normalized.Platform = strings.ToLower(strings.TrimSpace(input.Platform))
		normalized.UpstreamGroupName = strings.TrimSpace(input.UpstreamGroupName)
		normalized.Models = dedupeStrings(input.Models)
		normalized.LocalGroupIDs = dedupePositiveInt64(input.LocalGroupIDs)
		normalized.Concurrency = input.Concurrency
		if normalized.Concurrency <= 0 {
			normalized.Concurrency = 3
		}
		normalized.Priority = 50
		if input.Priority != nil {
			normalized.Priority = *input.Priority
		}
		normalized.Name = strings.TrimSpace(input.Name)
		if normalized.Name == "" {
			normalized.Name = defaultGeneratedAccountName(item.Name, normalized.UpstreamGroupName, normalized.Platform)
		}
		preview := upstreamAccountGenerationPreview{
			Index:             index,
			Name:              normalized.Name,
			Platform:          normalized.Platform,
			UpstreamGroupName: normalized.UpstreamGroupName,
			UpstreamGroupID:   normalized.UpstreamGroupID,
			Models:            normalized.Models,
			LocalGroupIDs:     normalized.LocalGroupIDs,
			Concurrency:       normalized.Concurrency,
			Priority:          normalized.Priority,
			RateMultiplier:    normalized.RateMultiplier,
			Action:            "create",
			Warnings:          []string{},
			Errors:            []string{},
		}

		if !isManagedUpstreamPlatform(normalized.Platform) {
			preview.Errors = append(preview.Errors, "Unsupported platform")
		}
		if normalized.UpstreamGroupName == "" {
			preview.Errors = append(preview.Errors, "Upstream group is required")
		} else if len(knownGroups) > 0 {
			group, exists := knownGroups[normalizeUpstreamGroupKey(normalized.UpstreamGroupName)]
			if !exists {
				preview.Errors = append(preview.Errors, "The selected upstream group is not present in the latest probe")
			} else {
				if normalized.UpstreamGroupID == nil && group.ID != nil {
					normalized.UpstreamGroupID = group.ID
					preview.UpstreamGroupID = group.ID
				}
				if normalized.RateMultiplier == nil && group.RateMultiplier != nil {
					rate := *group.RateMultiplier
					normalized.RateMultiplier = &rate
					preview.RateMultiplier = &rate
				}
				if group.Platform != "" && group.Platform != normalized.Platform {
					preview.Errors = append(preview.Errors, "The selected upstream group belongs to another platform")
				}
			}
		}

		if len(normalized.Models) == 0 {
			normalized.Models = sortedStringSet(protocolModels[normalized.Platform])
			preview.Models = normalized.Models
		}
		if len(normalized.Models) == 0 {
			preview.Errors = append(preview.Errors, "Select at least one model")
		} else if known := protocolModels[normalized.Platform]; len(known) > 0 {
			for _, model := range normalized.Models {
				if _, exists := known[model]; !exists {
					preview.Warnings = append(preview.Warnings, "Model was not returned by the latest protocol probe: "+model)
				}
			}
		}
		if len(normalized.LocalGroupIDs) == 0 {
			preview.Errors = append(preview.Errors, "Select at least one local group")
		} else if err := h.validateGeneratedAccountGroups(ctx, normalized.Platform, normalized.LocalGroupIDs); err != nil {
			preview.Errors = append(preview.Errors, err.Error())
		}
		if normalized.Concurrency > 10000 {
			preview.Errors = append(preview.Errors, "Concurrency is too large")
		}
		if normalized.Priority < 0 {
			preview.Errors = append(preview.Errors, "Priority must be non-negative")
		}
		if normalized.RateMultiplier != nil && *normalized.RateMultiplier < 0 {
			preview.Errors = append(preview.Errors, "Rate multiplier must be non-negative")
		}

		existingAccount := findGeneratedAccount(item.Edges.Accounts, normalized.Platform, normalized.UpstreamGroupName)
		apiKey, keySource := h.resolveGeneratedAccountAPIKey(item, normalized, currentKeyGroup)
		normalized.ResolvedAPIKey = apiKey
		normalized.KeySource = keySource
		preview.KeySource = keySource
		preview.WillCreateUpstreamKey = keySource == "remote_new_key"
		if keySource == "missing" && existingAccount == nil {
			preview.Errors = append(preview.Errors, "No key is available for the selected upstream group; configure management credentials or provide a group API key")
		}

		if existingAccount != nil {
			normalized.Action = "skip"
			normalized.ExistingID = &existingAccount.ID
			preview.Action = "skip"
			preview.ExistingAccountID = &existingAccount.ID
			preview.Warnings = append(preview.Warnings, "An account for this platform and upstream group is already bound")
			responseValue.Skips++
		} else {
			normalized.Action = "create"
			responseValue.Creates++
			if normalized.KeySource == "remote_new_key" {
				preview.Errors = append(preview.Errors, "Test the selected models first so a group-scoped key can be verified")
			} else if normalized.ResolvedAPIKey != "" {
				unverified := h.unverifiedUpstreamModels(
					item,
					normalized.Platform,
					normalized.UpstreamGroupName,
					normalized.Models,
					normalized.ResolvedAPIKey,
					time.Now().UTC(),
				)
				if len(unverified) > 0 {
					preview.Errors = append(preview.Errors,
						"Models require a successful current-group request before they can enter the whitelist: "+strings.Join(unverified, ", "),
					)
				}
			}
		}
		if len(preview.Errors) > 0 {
			responseValue.Valid = false
		}
		responseValue.Items = append(responseValue.Items, preview)
		responseValue.specs = append(responseValue.specs, normalized)
	}
	return responseValue
}

func (h *UpstreamHandler) resolveGeneratedAccountAPIKey(item *dbent.Upstream, spec normalizedUpstreamAccountGenerationSpec, currentKeyGroup string) (string, string) {
	if key := strings.TrimSpace(spec.APIKey); key != "" {
		return key, "request_override"
	}
	if stored := lookupStoredGeneratedKey(item.Credentials, spec.UpstreamGroupName); stored.APIKey != "" {
		return stored.APIKey, "stored_group_key"
	}
	platformKey := upstreamAPIKeyForPlatform(item.Credentials, spec.Platform)
	selectedGroup := normalizeUpstreamGroupKey(spec.UpstreamGroupName)
	if platformKey != "" && (currentKeyGroup == "" || selectedGroup == currentKeyGroup) {
		return platformKey, "stored_default_key"
	}
	if canCreateRemoteUpstreamKey(item, spec) {
		return "", "remote_new_key"
	}
	return "", "missing"
}

func canCreateRemoteUpstreamKey(item *dbent.Upstream, spec normalizedUpstreamAccountGenerationSpec) bool {
	if item == nil || strings.TrimSpace(spec.UpstreamGroupName) == "" {
		return false
	}
	switch item.Kind {
	case entupstream.KindNewapi:
		return upstreamCredentialPresent(item.Credentials, upstreamCredentialManagementAccessToken) || hasUpstreamPanelLogin(item.Credentials)
	case entupstream.KindSub2api:
		return spec.UpstreamGroupID != nil && hasUpstreamPanelLogin(item.Credentials)
	default:
		return false
	}
}

func (h *UpstreamHandler) validateGeneratedAccountGroups(ctx context.Context, platform string, ids []int64) error {
	groups, err := h.adminService.GetAllGroupsByPlatform(ctx, platform)
	if err != nil {
		return errors.New("Failed to load local groups")
	}
	allowed := make(map[int64]struct{}, len(groups))
	for _, group := range groups {
		allowed[group.ID] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := allowed[id]; !ok {
			return fmt.Errorf("Local group #%d is not an active %s group", id, platform)
		}
	}
	return nil
}

func (h *UpstreamHandler) createRemoteUpstreamGroupKey(ctx context.Context, item *dbent.Upstream, spec normalizedUpstreamAccountGenerationSpec) (storedGeneratedUpstreamKey, error) {
	switch item.Kind {
	case entupstream.KindNewapi:
		return h.createNewAPIRemoteGroupKey(ctx, item, spec)
	case entupstream.KindSub2api:
		return h.createSub2APIRemoteGroupKey(ctx, item, spec)
	default:
		return storedGeneratedUpstreamKey{}, errors.New("probe and identify the upstream type before generating a group key")
	}
}

func (h *UpstreamHandler) createNewAPIRemoteGroupKey(ctx context.Context, item *dbent.Upstream, spec normalizedUpstreamAccountGenerationSpec) (storedGeneratedUpstreamKey, error) {
	probeAccount := h.transientUpstreamAccount(ctx, item, spec.Platform, "")
	probeCtx, _, err := h.panelClient.contextForAccount(ctx, probeAccount)
	if err != nil {
		return storedGeneratedUpstreamKey{}, err
	}
	accessToken := upstreamCredentialString(item.Credentials, upstreamCredentialManagementAccessToken)
	session := upstreamNewAPIAccessTokenSession(item.Credentials)
	if accessToken == "" {
		username := upstreamCredentialString(item.Credentials, upstreamCredentialUsername)
		password := upstreamCredentialString(item.Credentials, upstreamCredentialPassword)
		session, err = h.panelClient.loginNewAPI(probeCtx, item.BaseURL, username, password, false)
		if err != nil {
			return storedGeneratedUpstreamKey{}, err
		}
	}
	name := remoteGeneratedKeyName(item.ID, spec.UpstreamGroupName)
	payload := map[string]any{
		"name":                 name,
		"expired_time":         -1,
		"remain_quota":         0,
		"unlimited_quota":      true,
		"model_limits_enabled": false,
		"group":                spec.UpstreamGroupName,
		"cross_group_retry":    false,
	}
	if _, _, err := h.panelClient.doNewAPIJSON(probeCtx, http.MethodPost, joinUpstreamSub2APIURL(item.BaseURL, "/api/token/"), session, accessToken, payload, nil); err != nil {
		return storedGeneratedUpstreamKey{}, err
	}

	query := url.Values{}
	query.Set("keyword", name)
	query.Set("p", "1")
	query.Set("page_size", strconv.Itoa(upstreamSub2APIPageSize))
	var page upstreamNewAPITokensPage
	if _, _, err := h.panelClient.doNewAPIJSON(probeCtx, http.MethodGet, joinUpstreamSub2APIURL(item.BaseURL, "/api/token/search?"+query.Encode()), session, accessToken, nil, &page); err != nil {
		return storedGeneratedUpstreamKey{}, err
	}
	var created *upstreamNewAPIToken
	for i := range page.Items {
		if page.Items[i].Name == name {
			created = &page.Items[i]
			break
		}
	}
	if created == nil {
		return storedGeneratedUpstreamKey{}, errors.New("NewAPI created the key but did not return it in search")
	}
	keyPath := "/api/token/" + strconv.FormatInt(created.ID, 10) + "/key"
	apiKey, err := h.readNewAPIGeneratedKeyWithRetry(probeCtx, item.BaseURL, keyPath, session, accessToken)
	if err != nil {
		cleanupErr := runUpstreamGeneratedKeyCleanup(ctx, func(cleanupCtx context.Context) error {
			return h.deleteRemoteUpstreamGroupKey(cleanupCtx, item, storedGeneratedUpstreamKey{
				ID:        created.ID,
				Kind:      entupstream.KindNewapi.String(),
				GroupName: spec.UpstreamGroupName,
			})
		})
		if cleanupErr != nil {
			return storedGeneratedUpstreamKey{}, errors.Join(
				err,
				fmt.Errorf("remote generated key cleanup after retrieval failure: %w", cleanupErr),
			)
		}
		return storedGeneratedUpstreamKey{}, err
	}
	return storedGeneratedUpstreamKey{
		APIKey:    apiKey,
		ID:        created.ID,
		Kind:      entupstream.KindNewapi.String(),
		GroupName: spec.UpstreamGroupName,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (h *UpstreamHandler) readNewAPIGeneratedKeyWithRetry(
	ctx context.Context,
	baseURL, keyPath string,
	session *upstreamNewAPISessionCacheEntry,
	accessToken string,
) (string, error) {
	for attempt := 1; attempt <= upstreamNewAPIGeneratedKeyReadMaxAttempts; attempt++ {
		var keyResponse struct {
			Key string `json:"key"`
		}
		status, _, err := h.panelClient.doNewAPIJSON(
			ctx,
			http.MethodPost,
			joinUpstreamSub2APIURL(baseURL, keyPath),
			session,
			accessToken,
			nil,
			&keyResponse,
		)
		if err == nil {
			apiKey := strings.TrimSpace(keyResponse.Key)
			if apiKey == "" {
				return "", errors.New("NewAPI returned an empty generated key")
			}
			return apiKey, nil
		}
		if status != http.StatusTooManyRequests || attempt == upstreamNewAPIGeneratedKeyReadMaxAttempts {
			return "", err
		}

		delay := upstreamNewAPIGeneratedKeyReadRetryBaseDelay << (attempt - 1)
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
			return "", fmt.Errorf("NewAPI generated key retrieval canceled: %w", ctx.Err())
		}
	}
	return "", errors.New("NewAPI generated key retrieval failed")
}

func (h *UpstreamHandler) createSub2APIRemoteGroupKey(ctx context.Context, item *dbent.Upstream, spec normalizedUpstreamAccountGenerationSpec) (storedGeneratedUpstreamKey, error) {
	if spec.UpstreamGroupID == nil {
		return storedGeneratedUpstreamKey{}, errors.New("Sub2API group ID is required")
	}
	probeAccount := h.transientUpstreamAccount(ctx, item, spec.Platform, "")
	probeCtx, _, err := h.panelClient.contextForAccount(ctx, probeAccount)
	if err != nil {
		return storedGeneratedUpstreamKey{}, err
	}
	username := upstreamCredentialString(item.Credentials, upstreamCredentialUsername)
	password := upstreamCredentialString(item.Credentials, upstreamCredentialPassword)
	token, err := h.panelClient.login(probeCtx, item.BaseURL, username, password, false)
	if err != nil {
		return storedGeneratedUpstreamKey{}, err
	}
	payload := map[string]any{
		"name":     remoteGeneratedKeyName(item.ID, spec.UpstreamGroupName),
		"group_id": *spec.UpstreamGroupID,
	}
	var created upstreamSub2APIKey
	var lastErr error
	for _, path := range []string{"/api/v1/api-keys", "/api/v1/keys"} {
		if _, err := h.panelClient.doJSON(probeCtx, http.MethodPost, joinUpstreamSub2APIURL(item.BaseURL, path), token, payload, &created); err == nil {
			lastErr = nil
			break
		} else {
			lastErr = err
		}
	}
	if lastErr != nil {
		return storedGeneratedUpstreamKey{}, lastErr
	}
	if strings.TrimSpace(created.Key) == "" {
		return storedGeneratedUpstreamKey{}, errors.New("Sub2API returned an empty generated key")
	}
	return storedGeneratedUpstreamKey{
		APIKey:    strings.TrimSpace(created.Key),
		ID:        created.ID,
		Kind:      entupstream.KindSub2api.String(),
		GroupName: spec.UpstreamGroupName,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (h *UpstreamHandler) persistGeneratedUpstreamKey(ctx context.Context, upstreamID int64, key storedGeneratedUpstreamKey) error {
	lock := h.refreshLock(upstreamID)
	lock.Lock()
	defer lock.Unlock()
	item, err := h.client.Upstream.Get(ctx, upstreamID)
	if err != nil {
		return err
	}
	credentials := mergeStoredGeneratedKey(item.Credentials, key)
	_, err = h.client.Upstream.UpdateOneID(upstreamID).SetCredentials(credentials).Save(ctx)
	return err
}

func (h *UpstreamHandler) generatedKeyLock(upstreamID int64, groupName string) *sync.Mutex {
	key := strconv.FormatInt(upstreamID, 10) + ":" + normalizeUpstreamGroupKey(groupName)
	value, _ := h.generatedKeyLocks.LoadOrStore(key, &sync.Mutex{})
	return value.(*sync.Mutex)
}

func (h *UpstreamHandler) deleteRemoteUpstreamGroupKey(ctx context.Context, item *dbent.Upstream, key storedGeneratedUpstreamKey) error {
	if item == nil || key.ID <= 0 {
		return errors.New("generated upstream key has no remote ID")
	}
	switch key.Kind {
	case entupstream.KindNewapi.String():
		auth, err := h.managedNewAPIAuth(ctx, item, service.PlatformOpenAI)
		if err != nil {
			return err
		}
		status, _, err := h.panelClient.doNewAPIJSON(
			auth.ctx,
			http.MethodDelete,
			joinUpstreamSub2APIURL(item.BaseURL, "/api/token/"+strconv.FormatInt(key.ID, 10)),
			auth.session,
			auth.accessToken,
			nil,
			nil,
		)
		if err != nil {
			return fmt.Errorf("NewAPI generated key cleanup failed with HTTP %d: %w", status, err)
		}
		return nil
	case entupstream.KindSub2api.String():
		username := upstreamCredentialString(item.Credentials, upstreamCredentialUsername)
		password := upstreamCredentialString(item.Credentials, upstreamCredentialPassword)
		probeAccount := h.transientUpstreamAccount(ctx, item, service.PlatformOpenAI, "")
		probeCtx, _, err := h.panelClient.contextForAccount(ctx, probeAccount)
		if err != nil {
			return err
		}
		token, err := h.panelClient.login(probeCtx, item.BaseURL, username, password, false)
		if err != nil {
			return err
		}
		id := strconv.FormatInt(key.ID, 10)
		var lastErr error
		for _, path := range []string{"/api/v1/api-keys/" + id, "/api/v1/keys/" + id} {
			status, err := h.panelClient.doJSON(probeCtx, http.MethodDelete, joinUpstreamSub2APIURL(item.BaseURL, path), token, nil, nil)
			if err == nil {
				return nil
			}
			lastErr = fmt.Errorf("HTTP %d: %w", status, err)
			if status != http.StatusNotFound {
				break
			}
		}
		return lastErr
	default:
		return errors.New("unsupported generated upstream key type")
	}
}

func mergeStoredGeneratedKey(credentials map[string]any, key storedGeneratedUpstreamKey) map[string]any {
	result := maps.Clone(credentials)
	if result == nil {
		result = map[string]any{}
	}
	stored := maps.Clone(generatedUpstreamGroupKeys(result))
	encoded, _ := json.Marshal(key)
	var value map[string]any
	_ = json.Unmarshal(encoded, &value)
	stored[normalizeUpstreamGroupKey(key.GroupName)] = value
	result[upstreamCredentialGeneratedGroupKeys] = stored
	return result
}

func lookupStoredGeneratedKey(credentials map[string]any, groupName string) storedGeneratedUpstreamKey {
	raw, ok := generatedUpstreamGroupKeys(credentials)[normalizeUpstreamGroupKey(groupName)]
	if !ok {
		return storedGeneratedUpstreamKey{}
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return storedGeneratedUpstreamKey{}
	}
	var result storedGeneratedUpstreamKey
	if err := json.Unmarshal(data, &result); err != nil {
		return storedGeneratedUpstreamKey{}
	}
	return result
}

func parseUpstreamProbeMetadata(raw map[string]any) (upstreamProbeMetadata, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return upstreamProbeMetadata{}, err
	}
	var result upstreamProbeMetadata
	if err := json.Unmarshal(data, &result); err != nil {
		return upstreamProbeMetadata{}, err
	}
	return result, nil
}

func findGeneratedAccount(accounts []*dbent.Account, platform, groupName string) *dbent.Account {
	normalizedGroup := normalizeUpstreamGroupKey(groupName)
	for _, account := range accounts {
		if account == nil || account.Platform != platform {
			continue
		}
		storedGroup, _ := account.Extra["upstream_group_name"].(string)
		if normalizeUpstreamGroupKey(storedGroup) == normalizedGroup {
			return account
		}
	}
	return nil
}

func isManagedUpstreamPlatform(platform string) bool {
	switch platform {
	case service.PlatformOpenAI, service.PlatformAnthropic, service.PlatformGemini, service.PlatformGrok:
		return true
	default:
		return false
	}
}

func identityModelMapping(models []string) map[string]any {
	result := make(map[string]any, len(models))
	for _, model := range models {
		result[model] = model
	}
	return result
}

func defaultGeneratedAccountName(upstreamName, groupName, platform string) string {
	parts := []string{strings.TrimSpace(upstreamName), strings.TrimSpace(groupName), strings.ToUpper(strings.TrimSpace(platform))}
	name := strings.Join(parts, " / ")
	runes := []rune(name)
	if len(runes) > 100 {
		name = string(runes[:100])
	}
	return name
}

func remoteGeneratedKeyName(upstreamID int64, groupName string) string {
	slugRunes := make([]rune, 0, len(groupName))
	for _, value := range strings.ToLower(strings.TrimSpace(groupName)) {
		if (value >= 'a' && value <= 'z') || (value >= '0' && value <= '9') || value == '-' || value == '_' {
			slugRunes = append(slugRunes, value)
		} else if len(slugRunes) == 0 || slugRunes[len(slugRunes)-1] != '-' {
			slugRunes = append(slugRunes, '-')
		}
	}
	slug := strings.Trim(string(slugRunes), "-")
	if slug == "" {
		slug = "group"
	}
	name := fmt.Sprintf("sub2api-%d-%s-%d", upstreamID, slug, time.Now().UnixNano())
	if len(name) > 50 {
		name = name[:50]
	}
	return name
}

func normalizeUpstreamGroupKey(groupName string) string {
	return strings.ToLower(strings.TrimSpace(groupName))
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortedStringSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
