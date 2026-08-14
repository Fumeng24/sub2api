package admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	entaccount "github.com/Wei-Shaw/sub2api/ent/account"
	entupstream "github.com/Wei-Shaw/sub2api/ent/upstream"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	upstreamAccountGroupChangeTimeout = 90 * time.Second
	upstreamAccountRollbackTimeout    = 30 * time.Second
)

type upstreamAccountGroupChangeRequest struct {
	GroupName string `json:"group_name"`
	GroupID   *int64 `json:"group_id"`
}

type upstreamAccountGroupChangeResponse struct {
	Account upstreamAccountSummary `json:"account"`
	Models  []string               `json:"models"`
	Warning string                 `json:"warning,omitempty"`
}

type upstreamManagedNewAPIAuth struct {
	ctx         context.Context
	session     *upstreamNewAPISessionCacheEntry
	accessToken string
}

type upstreamAccountRenameItem struct {
	AccountID    int64  `json:"account_id"`
	UpstreamID   int64  `json:"upstream_id"`
	CurrentName  string `json:"current_name"`
	ProposedName string `json:"proposed_name,omitempty"`
	Action       string `json:"action"`
	Reason       string `json:"reason,omitempty"`
}

type upstreamAccountRenamePreview struct {
	Renames int                         `json:"renames"`
	Skips   int                         `json:"skips"`
	Items   []upstreamAccountRenameItem `json:"items"`
}

type upstreamAccountRenameApplyResult struct {
	Renamed int                         `json:"renamed"`
	Skipped int                         `json:"skipped"`
	Failed  int                         `json:"failed"`
	Items   []upstreamAccountRenameItem `json:"items"`
}

func (h *UpstreamHandler) accountChangeLock(accountID int64) *sync.Mutex {
	value, _ := h.accountChangeLocks.LoadOrStore(accountID, &sync.Mutex{})
	return value.(*sync.Mutex)
}

// ChangeAccountUpstreamGroup changes the remote key group first, verifies the
// remote state and model catalogue, and only then updates the local account.
func (h *UpstreamHandler) ChangeAccountUpstreamGroup(c *gin.Context) {
	upstreamID, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	accountID, err := strconv.ParseInt(c.Param("account_id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req upstreamAccountGroupChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	req.GroupName = strings.TrimSpace(req.GroupName)
	if req.GroupName == "" {
		response.BadRequest(c, "group_name is required")
		return
	}

	accountLock := h.accountChangeLock(accountID)
	accountLock.Lock()
	defer accountLock.Unlock()
	refreshLock := h.refreshLock(upstreamID)
	refreshLock.Lock()
	defer refreshLock.Unlock()

	ctx := c.Request.Context()
	item, err := h.client.Upstream.Query().
		Where(entupstream.ID(upstreamID)).
		WithProxy().
		Only(ctx)
	if err != nil {
		writeUpstreamQueryError(c, err)
		return
	}
	account, err := h.client.Account.Query().
		Where(entaccount.ID(accountID), entaccount.UpstreamID(upstreamID)).
		WithGroups().
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			response.NotFound(c, "Bound account not found")
		} else {
			response.InternalError(c, "Failed to load bound account")
		}
		return
	}
	if supported, reason := upstreamAccountGroupChangeCapability(item, account); !supported {
		response.Error(c, http.StatusConflict, reason)
		return
	}
	if h.adminService == nil || h.accountTestService == nil || h.panelClient == nil {
		response.InternalError(c, "Upstream account management is unavailable")
		return
	}

	metadata, err := parseUpstreamProbeMetadata(item.Metadata)
	if err != nil {
		response.InternalError(c, "Failed to load upstream capability metadata")
		return
	}
	if metadata.ManagementStatus != "ok" || len(metadata.Groups) == 0 {
		response.Error(c, http.StatusConflict, "The upstream group catalogue is not currently verified; probe the upstream first")
		return
	}
	target, err := resolveUpstreamAccountTargetGroup(metadata.Groups, account.Platform, req.GroupName, req.GroupID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if target.RateMultiplier == nil {
		response.Error(c, http.StatusConflict, "The selected upstream group rate is not currently verifiable")
		return
	}
	operationCtx, cancelOperation := context.WithTimeout(context.WithoutCancel(ctx), upstreamAccountGroupChangeTimeout)
	defer cancelOperation()
	apiKey := upstreamCredentialString(account.Credentials, upstreamCredentialAPIKey)
	rollback, remoteKeyID, err := h.changeRemoteAccountGroup(operationCtx, item, apiKey, *target)
	if err != nil {
		response.Error(c, http.StatusBadGateway, safeUpstreamProbeError(err, item))
		return
	}
	rollbackRemote := func(cause string) bool {
		if rollback == nil {
			return true
		}
		if rollbackErr := runUpstreamAccountRollback(operationCtx, rollback); rollbackErr != nil {
			slog.Error("upstream account group rollback failed",
				"upstream_id", upstreamID,
				"account_id", accountID,
				"cause", cause,
				"error", safeUpstreamProbeError(rollbackErr, item),
			)
			return false
		}
		return true
	}

	billingStatus := h.probeBoundAccountBilling(operationCtx, item, account, metadata, apiKey)
	if billingStatus.UpstreamKeyID == nil {
		billingStatus.UpstreamKeyID = remoteKeyID
	}
	if !changedAccountBillingMatchesTarget(billingStatus, *target, account.Platform) {
		rolledBack := rollbackRemote("upstream group billing verification failed")
		message := "The selected upstream group or its effective rate could not be verified with the account API key"
		if !rolledBack {
			message += "; remote state requires manual verification"
		}
		response.Error(c, http.StatusBadGateway, message)
		return
	}
	rate := billingStatus.UpstreamGroupEffectiveRateMultiplier
	if rate == nil {
		rate = billingStatus.UpstreamGroupDefaultRateMultiplier
	}

	probeAccount := h.transientUpstreamAccount(operationCtx, item, account.Platform, apiKey)
	probeAccount.ID = account.ID
	probeAccount.Name = account.Name
	models, modelErr := h.accountTestService.FetchUpstreamSupportedModels(operationCtx, probeAccount)
	models = dedupeStrings(models)
	if modelErr != nil || len(models) == 0 {
		rolledBack := rollbackRemote("model catalogue verification failed")
		message := "The selected upstream group did not return a verifiable model allowlist"
		if !rolledBack {
			message += "; remote state requires manual verification"
		}
		response.Error(c, http.StatusBadGateway, message)
		return
	}

	credentials := maps.Clone(account.Credentials)
	if credentials == nil {
		credentials = map[string]any{}
	}
	credentials["model_mapping"] = identityModelMapping(models)
	extra := maps.Clone(account.Extra)
	if extra == nil {
		extra = map[string]any{}
	}
	extra["upstream_group_name"] = target.Name
	if target.ID != nil {
		extra["upstream_group_id"] = *target.ID
	} else {
		delete(extra, "upstream_group_id")
	}
	extra["upstream_group_changed_at"] = time.Now().UTC().Format(time.RFC3339)
	name := defaultGeneratedAccountName(item.Name, target.Name, account.Platform)
	if _, err := h.adminService.UpdateAccount(operationCtx, account.ID, &service.UpdateAccountInput{
		Name:           name,
		Credentials:    credentials,
		Extra:          extra,
		RateMultiplier: rate,
	}); err != nil {
		rolledBack := rollbackRemote("local account update failed")
		message := "The upstream group was not applied because the local account update failed"
		if !rolledBack {
			message += "; remote state requires manual verification"
		}
		response.Error(c, http.StatusInternalServerError, message)
		return
	}

	warning := ""
	if err := h.persistChangedAccountBilling(operationCtx, item, &metadata, account, *target, billingStatus, models); err != nil {
		warning = "The account was changed, but its upstream status snapshot will update on the next refresh"
	}
	updated, err := h.client.Account.Query().
		Where(entaccount.ID(account.ID)).
		WithGroups().
		Only(operationCtx)
	if err != nil {
		account.Name = name
		account.Credentials = credentials
		account.Extra = extra
		account.RateMultiplier = *rate
		updated = account
		warning = "The account was changed, but its latest local view will update on the next refresh"
	}
	item.Metadata, _ = upstreamMetadataMap(metadata)
	response.Success(c, upstreamAccountGroupChangeResponse{
		Account: buildUpstreamAccountSummary(updated, item, &metadata),
		Models:  models,
		Warning: warning,
	})
}

func changedAccountBillingMatchesTarget(status UpstreamSub2APIAccountStatus, target upstreamProbeGroup, platform string) bool {
	if status.Status != "ok" || !strings.EqualFold(strings.TrimSpace(status.UpstreamGroupName), strings.TrimSpace(target.Name)) {
		return false
	}
	if target.ID != nil && (status.UpstreamGroupID == nil || *status.UpstreamGroupID != *target.ID) {
		return false
	}
	if status.UpstreamGroupPlatform != "" && !strings.EqualFold(strings.TrimSpace(status.UpstreamGroupPlatform), strings.TrimSpace(platform)) {
		return false
	}
	return status.UpstreamGroupEffectiveRateMultiplier != nil || status.UpstreamGroupDefaultRateMultiplier != nil
}

func runUpstreamAccountRollback(parent context.Context, rollback func(context.Context) error) error {
	if rollback == nil {
		return nil
	}
	if parent == nil {
		parent = context.Background()
	}
	rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), upstreamAccountRollbackTimeout)
	defer cancel()
	return rollback(rollbackCtx)
}

func resolveUpstreamAccountTargetGroup(groups []upstreamProbeGroup, platform, groupName string, groupID *int64) (*upstreamProbeGroup, error) {
	var matches []*upstreamProbeGroup
	for index := range groups {
		group := &groups[index]
		if !strings.EqualFold(strings.TrimSpace(group.Name), groupName) {
			continue
		}
		if groupID != nil && (group.ID == nil || *group.ID != *groupID) {
			continue
		}
		if group.Platform != "" && !strings.EqualFold(strings.TrimSpace(group.Platform), strings.TrimSpace(platform)) {
			continue
		}
		matches = append(matches, group)
	}
	if len(matches) == 0 {
		return nil, errors.New("The selected upstream group is not available for this account protocol")
	}
	if len(matches) > 1 {
		return nil, errors.New("The selected upstream group is ambiguous; select it again after refreshing the upstream")
	}
	return matches[0], nil
}

func (h *UpstreamHandler) changeRemoteAccountGroup(
	ctx context.Context,
	item *dbent.Upstream,
	apiKey string,
	target upstreamProbeGroup,
) (func(context.Context) error, *int64, error) {
	switch item.Kind {
	case entupstream.KindNewapi:
		return h.changeNewAPIAccountGroup(ctx, item, apiKey, target.Name)
	case entupstream.KindSub2api:
		if target.ID == nil {
			return nil, nil, errors.New("The selected Sub2API group has no verified group ID")
		}
		return h.changeSub2APIAccountGroup(ctx, item, apiKey, *target.ID)
	default:
		return nil, nil, errors.New("Probe and identify the upstream type before changing groups")
	}
}

func (h *UpstreamHandler) managedNewAPIAuth(ctx context.Context, item *dbent.Upstream, platform string) (upstreamManagedNewAPIAuth, error) {
	probeAccount := h.transientUpstreamAccount(ctx, item, platform, "")
	probeCtx, _, err := h.panelClient.contextForAccount(ctx, probeAccount)
	if err != nil {
		return upstreamManagedNewAPIAuth{}, err
	}
	auth := upstreamManagedNewAPIAuth{
		ctx:         probeCtx,
		accessToken: upstreamCredentialString(item.Credentials, upstreamCredentialManagementAccessToken),
	}
	if auth.accessToken != "" {
		auth.session = upstreamNewAPIAccessTokenSession(item.Credentials)
		if auth.session == nil {
			return upstreamManagedNewAPIAuth{}, errUpstreamNewAPIManagementUserID
		}
		return auth, nil
	}
	username := upstreamCredentialString(item.Credentials, upstreamCredentialUsername)
	password := upstreamCredentialString(item.Credentials, upstreamCredentialPassword)
	if username == "" || password == "" {
		return upstreamManagedNewAPIAuth{}, errors.New("NewAPI management credentials are not configured")
	}
	auth.session, err = h.panelClient.loginNewAPI(probeCtx, item.BaseURL, username, password, false)
	if err != nil {
		return upstreamManagedNewAPIAuth{}, err
	}
	return auth, nil
}

func (h *UpstreamHandler) changeNewAPIAccountGroup(
	ctx context.Context,
	item *dbent.Upstream,
	apiKey, targetGroup string,
) (func(context.Context) error, *int64, error) {
	auth, err := h.managedNewAPIAuth(ctx, item, service.PlatformOpenAI)
	if err != nil {
		return nil, nil, err
	}
	key, err := h.findNewAPIKeyWithAccessToken(auth.ctx, item.BaseURL, auth.accessToken, auth.session, apiKey)
	if err != nil {
		return nil, nil, err
	}
	if key == nil {
		return nil, nil, errors.New("The bound API key was not found in the managed NewAPI account")
	}
	previous := *key
	if strings.EqualFold(strings.TrimSpace(previous.Group), strings.TrimSpace(targetGroup)) {
		keyID := previous.ID
		return func(context.Context) error { return nil }, &keyID, nil
	}
	rollback := func(rollbackCtx context.Context) error {
		rollbackAuth, authErr := h.managedNewAPIAuth(rollbackCtx, item, service.PlatformOpenAI)
		if authErr != nil {
			return authErr
		}
		if updateErr := h.updateManagedNewAPITokenGroup(rollbackAuth, item, previous, previous.Group); updateErr != nil {
			return updateErr
		}
		verified, verifyErr := h.findNewAPIKeyWithAccessToken(
			rollbackAuth.ctx, item.BaseURL, rollbackAuth.accessToken, rollbackAuth.session, apiKey,
		)
		if verifyErr != nil {
			return verifyErr
		}
		if verified == nil || !strings.EqualFold(strings.TrimSpace(verified.Group), strings.TrimSpace(previous.Group)) {
			return errors.New("NewAPI group rollback could not be verified")
		}
		return nil
	}
	if err := h.updateManagedNewAPITokenGroup(auth, item, previous, targetGroup); err != nil {
		if rollbackErr := runUpstreamAccountRollback(ctx, rollback); rollbackErr != nil {
			return nil, nil, fmt.Errorf("NewAPI group update failed and rollback requires manual verification: %w", err)
		}
		return nil, nil, err
	}
	verified, err := h.findNewAPIKeyWithAccessToken(auth.ctx, item.BaseURL, auth.accessToken, auth.session, apiKey)
	if err != nil || verified == nil || !strings.EqualFold(strings.TrimSpace(verified.Group), strings.TrimSpace(targetGroup)) {
		rollbackErr := runUpstreamAccountRollback(ctx, rollback)
		if err != nil {
			if rollbackErr != nil {
				return nil, nil, fmt.Errorf("NewAPI group update could not be verified and rollback requires manual verification: %w", err)
			}
			return nil, nil, fmt.Errorf("NewAPI group update could not be verified: %w", err)
		}
		if rollbackErr != nil {
			return nil, nil, errors.New("NewAPI group update could not be verified and rollback requires manual verification")
		}
		return nil, nil, errors.New("NewAPI group update could not be verified")
	}
	keyID := previous.ID
	return rollback, &keyID, nil
}

func (h *UpstreamHandler) updateManagedNewAPITokenGroup(auth upstreamManagedNewAPIAuth, item *dbent.Upstream, key upstreamNewAPIToken, groupName string) error {
	payload := map[string]any{
		"id":                   key.ID,
		"name":                 key.Name,
		"status":               key.Status,
		"expired_time":         key.ExpiredTime,
		"remain_quota":         key.RemainQuota,
		"unlimited_quota":      key.UnlimitedQuota,
		"model_limits_enabled": key.ModelLimitsEnabled,
		"model_limits":         key.ModelLimits,
		"allow_ips":            key.AllowIPs,
		"group":                groupName,
		"cross_group_retry":    key.CrossGroupRetry,
	}
	_, _, err := h.panelClient.doNewAPIJSON(
		auth.ctx,
		http.MethodPut,
		joinUpstreamSub2APIURL(item.BaseURL, "/api/token/"),
		auth.session,
		auth.accessToken,
		payload,
		nil,
	)
	return err
}

func (h *UpstreamHandler) changeSub2APIAccountGroup(
	ctx context.Context,
	item *dbent.Upstream,
	apiKey string,
	targetGroupID int64,
) (func(context.Context) error, *int64, error) {
	probeAccount := h.transientUpstreamAccount(ctx, item, service.PlatformOpenAI, "")
	probeCtx, _, err := h.panelClient.contextForAccount(ctx, probeAccount)
	if err != nil {
		return nil, nil, err
	}
	username := upstreamCredentialString(item.Credentials, upstreamCredentialUsername)
	password := upstreamCredentialString(item.Credentials, upstreamCredentialPassword)
	key, basePath, err := h.panelClient.findAPIKeyWithPath(probeCtx, item.BaseURL, username, password, apiKey)
	if err != nil {
		return nil, nil, err
	}
	if key == nil {
		return nil, nil, errors.New("The bound API key was not found in the managed Sub2API account")
	}
	if key.GroupID == nil {
		return nil, nil, errors.New("The current Sub2API key group is unknown, so a safe rollback is not possible")
	}
	previousGroupID := *key.GroupID
	if previousGroupID == targetGroupID {
		keyID := key.ID
		return func(context.Context) error { return nil }, &keyID, nil
	}
	update := func(callCtx context.Context, groupID int64) error {
		callProbeAccount := h.transientUpstreamAccount(callCtx, item, service.PlatformOpenAI, "")
		callProbeCtx, _, contextErr := h.panelClient.contextForAccount(callCtx, callProbeAccount)
		if contextErr != nil {
			return contextErr
		}
		token, loginErr := h.panelClient.login(callProbeCtx, item.BaseURL, username, password, false)
		if loginErr != nil {
			return loginErr
		}
		path := basePath + "/" + strconv.FormatInt(key.ID, 10)
		_, updateErr := h.panelClient.doJSON(callProbeCtx, http.MethodPut, joinUpstreamSub2APIURL(item.BaseURL, path), token, map[string]any{"group_id": groupID}, nil)
		return updateErr
	}
	verify := func(callCtx context.Context, groupID int64) error {
		callProbeAccount := h.transientUpstreamAccount(callCtx, item, service.PlatformOpenAI, "")
		callProbeCtx, _, contextErr := h.panelClient.contextForAccount(callCtx, callProbeAccount)
		if contextErr != nil {
			return contextErr
		}
		verified, _, verifyErr := h.panelClient.findAPIKeyWithPath(callProbeCtx, item.BaseURL, username, password, apiKey)
		if verifyErr != nil {
			return verifyErr
		}
		if verified == nil || verified.GroupID == nil || *verified.GroupID != groupID {
			return errors.New("Sub2API key group could not be verified")
		}
		return nil
	}
	rollback := func(rollbackCtx context.Context) error {
		if updateErr := update(rollbackCtx, previousGroupID); updateErr != nil {
			return updateErr
		}
		return verify(rollbackCtx, previousGroupID)
	}
	if err := update(ctx, targetGroupID); err != nil {
		if rollbackErr := runUpstreamAccountRollback(ctx, rollback); rollbackErr != nil {
			return nil, nil, fmt.Errorf("Sub2API group update failed and rollback requires manual verification: %w", err)
		}
		return nil, nil, err
	}
	if err := verify(ctx, targetGroupID); err != nil {
		rollbackErr := runUpstreamAccountRollback(ctx, rollback)
		if rollbackErr != nil {
			return nil, nil, fmt.Errorf("Sub2API group update could not be verified and rollback requires manual verification: %w", err)
		}
		return nil, nil, fmt.Errorf("Sub2API group update could not be verified: %w", err)
	}
	keyID := key.ID
	return rollback, &keyID, nil
}

func (h *UpstreamHandler) persistChangedAccountBilling(
	ctx context.Context,
	item *dbent.Upstream,
	metadata *upstreamProbeMetadata,
	account *dbent.Account,
	target upstreamProbeGroup,
	billingStatus UpstreamSub2APIAccountStatus,
	models []string,
) error {
	if item == nil || metadata == nil || account == nil {
		return errors.New("upstream account status is unavailable")
	}
	key := strconv.FormatInt(account.ID, 10)
	if metadata.AccountBilling == nil {
		metadata.AccountBilling = map[string]upstreamAccountBillingMetadata{}
	}
	previous := metadata.AccountBilling[key]
	billing, ok := upstreamAccountBillingFromStatus(item, billingStatus, previous)
	if !ok {
		return errors.New("verified upstream account billing could not be persisted")
	}
	billing.ProbeSource = "group_change_verified"
	metadata.AccountBilling[key] = billing
	for index := range metadata.Groups {
		if sameUpstreamProbeGroup(metadata.Groups[index], target) {
			metadata.Groups[index].Models = append([]string(nil), models...)
			break
		}
	}
	encoded, err := upstreamMetadataMap(*metadata)
	if err != nil {
		return err
	}
	if _, err := h.client.Upstream.UpdateOneID(item.ID).SetMetadata(encoded).Save(ctx); err != nil {
		return err
	}
	item.Metadata = encoded
	return nil
}

func (h *UpstreamHandler) RenameAccountsPreview(c *gin.Context) {
	preview, err := h.buildUpstreamAccountRenamePreview(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to build account rename preview")
		return
	}
	response.Success(c, preview)
}

func (h *UpstreamHandler) RenameAccountsApply(c *gin.Context) {
	if h == nil || h.adminService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Upstream account management is unavailable")
		return
	}
	preview, err := h.buildUpstreamAccountRenamePreview(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to build account rename preview")
		return
	}
	result := upstreamAccountRenameApplyResult{Items: make([]upstreamAccountRenameItem, 0, len(preview.Items))}
	for _, item := range preview.Items {
		if item.Action != "rename" {
			result.Skipped++
			result.Items = append(result.Items, item)
			continue
		}
		lock := h.accountChangeLock(item.AccountID)
		lock.Lock()
		_, updateErr := h.adminService.UpdateAccount(c.Request.Context(), item.AccountID, &service.UpdateAccountInput{Name: item.ProposedName})
		lock.Unlock()
		if updateErr != nil {
			item.Action = "failed"
			item.Reason = "local account update failed"
			result.Failed++
		} else {
			item.Action = "renamed"
			result.Renamed++
		}
		result.Items = append(result.Items, item)
	}
	response.Success(c, result)
}

func (h *UpstreamHandler) buildUpstreamAccountRenamePreview(ctx context.Context) (upstreamAccountRenamePreview, error) {
	result := upstreamAccountRenamePreview{Items: []upstreamAccountRenameItem{}}
	items, err := h.client.Upstream.Query().
		WithAccounts(func(query *dbent.AccountQuery) {
			query.Order(dbent.Asc(entaccount.FieldName), dbent.Asc(entaccount.FieldID))
		}).
		Order(dbent.Asc(entupstream.FieldName), dbent.Asc(entupstream.FieldID)).
		All(ctx)
	if err != nil {
		return result, err
	}
	for _, upstream := range items {
		metadata, _ := parseUpstreamProbeMetadata(upstream.Metadata)
		for _, account := range upstream.Edges.Accounts {
			entry := upstreamAccountRenameItem{
				AccountID:   account.ID,
				UpstreamID:  upstream.ID,
				CurrentName: account.Name,
				Action:      "skip",
			}
			billing, ok := metadata.AccountBilling[strconv.FormatInt(account.ID, 10)]
			if !ok || billing.Status != "ok" || billing.Stale || strings.TrimSpace(billing.UpstreamGroupName) == "" {
				entry.Reason = "upstream group is not currently verified"
				result.Skips++
				result.Items = append(result.Items, entry)
				continue
			}
			entry.ProposedName = defaultGeneratedAccountName(upstream.Name, billing.UpstreamGroupName, account.Platform)
			if entry.ProposedName == account.Name {
				entry.Reason = "already uses the automatic name"
				result.Skips++
			} else {
				entry.Action = "rename"
				result.Renames++
			}
			result.Items = append(result.Items, entry)
		}
	}
	return result, nil
}
