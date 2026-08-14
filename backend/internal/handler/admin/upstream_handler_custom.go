package admin

import (
	"database/sql"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"reflect"
	"sort"
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
	upstreamCredentialAPIKey                = "api_key"
	upstreamCredentialOpenAIAPIKey          = "openai_api_key"
	upstreamCredentialAnthropicAPIKey       = "anthropic_api_key"
	upstreamCredentialGeminiAPIKey          = "gemini_api_key"
	upstreamCredentialGrokAPIKey            = "grok_api_key"
	upstreamCredentialManagementAccessToken = "management_access_token"
	upstreamCredentialManagementUserID      = "management_user_id"
	upstreamCredentialUsername              = "username"
	upstreamCredentialPassword              = "password"
	upstreamCredentialGeneratedGroupKeys    = "generated_group_keys"
)

var editableUpstreamCredentialKeys = map[string]struct{}{
	upstreamCredentialAPIKey:                {},
	upstreamCredentialOpenAIAPIKey:          {},
	upstreamCredentialAnthropicAPIKey:       {},
	upstreamCredentialGeminiAPIKey:          {},
	upstreamCredentialGrokAPIKey:            {},
	upstreamCredentialManagementAccessToken: {},
	upstreamCredentialManagementUserID:      {},
	upstreamCredentialUsername:              {},
	upstreamCredentialPassword:              {},
}

type UpstreamHandler struct {
	client             *dbent.Client
	adminService       service.AdminService
	accountTestService *service.AccountTestService
	panelClient        *upstreamSub2APIStatusClient
	refreshLocks       sync.Map
	accountChangeLocks sync.Map
	generatedKeyLocks  sync.Map
	modelVerifications sync.Map
	refreshRunner      *upstreamRefreshRunner
}

func NewUpstreamHandler(
	client *dbent.Client,
	adminService service.AdminService,
	accountTestService *service.AccountTestService,
	accountHandler *AccountHandler,
	leaderLock service.LeaderLockCache,
	db *sql.DB,
) *UpstreamHandler {
	panelClient := newUpstreamSub2APIStatusClient()
	if accountHandler != nil && accountHandler.upstreamSub2API != nil {
		panelClient = accountHandler.upstreamSub2API
	}
	handler := &UpstreamHandler{
		client:             client,
		adminService:       adminService,
		accountTestService: accountTestService,
		panelClient:        panelClient,
	}
	handler.refreshRunner = newUpstreamRefreshRunner(handler, leaderLock, db)
	handler.refreshRunner.Start()
	return handler
}

// Stop terminates the periodic upstream metadata refresh worker.
func (h *UpstreamHandler) Stop() {
	if h != nil && h.refreshRunner != nil {
		h.refreshRunner.Stop()
	}
}

type upstreamMutationRequest struct {
	Name             string            `json:"name"`
	BaseURL          string            `json:"base_url"`
	Kind             string            `json:"kind"`
	ProxyID          *int64            `json:"proxy_id"`
	ClearProxy       bool              `json:"clear_proxy"`
	Credentials      map[string]string `json:"credentials"`
	ClearCredentials []string          `json:"clear_credentials"`
}

type upstreamCredentialStatus struct {
	HasAPIKey                bool `json:"has_api_key"`
	HasOpenAIAPIKey          bool `json:"has_openai_api_key"`
	HasAnthropicAPIKey       bool `json:"has_anthropic_api_key"`
	HasGeminiAPIKey          bool `json:"has_gemini_api_key"`
	HasGrokAPIKey            bool `json:"has_grok_api_key"`
	HasManagementAccessToken bool `json:"has_management_access_token"`
	HasManagementUserID      bool `json:"has_management_user_id"`
	HasUsername              bool `json:"has_username"`
	HasPassword              bool `json:"has_password"`
	GeneratedGroupKeyCount   int  `json:"generated_group_key_count"`
}

type upstreamAccountSummary struct {
	ID                           int64    `json:"id"`
	Name                         string   `json:"name"`
	Platform                     string   `json:"platform"`
	Type                         string   `json:"type"`
	Status                       string   `json:"status"`
	Schedulable                  bool     `json:"schedulable"`
	UpstreamID                   *int64   `json:"upstream_id,omitempty"`
	GroupIDs                     []int64  `json:"group_ids"`
	Generated                    bool     `json:"generated"`
	UpstreamGroupID              *int64   `json:"upstream_group_id,omitempty"`
	UpstreamGroup                string   `json:"upstream_group,omitempty"`
	UpstreamGroupRateMultiplier  *float64 `json:"upstream_group_rate_multiplier,omitempty"`
	UpstreamGroupRateSource      string   `json:"upstream_group_rate_source,omitempty"`
	UpstreamGroupStale           bool     `json:"upstream_group_stale"`
	UpstreamGroupChangeSupported bool     `json:"upstream_group_change_supported"`
	UpstreamGroupChangeReason    string   `json:"upstream_group_change_reason,omitempty"`
}

type upstreamLocalGroupSummary struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

type upstreamView struct {
	ID                    int64                       `json:"id"`
	Name                  string                      `json:"name"`
	BaseURL               string                      `json:"base_url"`
	Kind                  string                      `json:"kind"`
	ProxyID               *int64                      `json:"proxy_id"`
	ProxyName             string                      `json:"proxy_name,omitempty"`
	Status                string                      `json:"status"`
	LastProbeAt           *time.Time                  `json:"last_probe_at,omitempty"`
	LastProbeError        *string                     `json:"last_probe_error,omitempty"`
	Metadata              map[string]any              `json:"metadata"`
	CredentialStatus      upstreamCredentialStatus    `json:"credential_status"`
	AccountCount          int                         `json:"account_count"`
	LocalGroups           []upstreamLocalGroupSummary `json:"local_groups"`
	Accounts              []upstreamAccountSummary    `json:"accounts,omitempty"`
	DuplicateBaseURLCount int                         `json:"duplicate_base_url_count"`
	CreatedAt             time.Time                   `json:"created_at"`
	UpdatedAt             time.Time                   `json:"updated_at"`
}

type upstreamBindRequest struct {
	AccountIDs  []int64 `json:"account_ids"`
	AllowRebind bool    `json:"allow_rebind"`
}

type upstreamUnbindRequest struct {
	AccountIDs     []int64 `json:"account_ids"`
	DeleteAccounts *bool   `json:"delete_accounts"`
}

type upstreamBindResult struct {
	UpdatedAccountIDs []int64 `json:"updated_account_ids"`
	DeletedAccountIDs []int64 `json:"deleted_account_ids,omitempty"`
	UnchangedIDs      []int64 `json:"unchanged_account_ids"`
}

func (h *UpstreamHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	query := h.client.Upstream.Query()
	if search != "" {
		query.Where(entupstream.Or(
			entupstream.NameContainsFold(search),
			entupstream.BaseURLContainsFold(search),
		))
	}
	total, err := query.Clone().Count(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to count upstreams")
		return
	}
	items, err := query.
		WithProxy().
		WithAccounts(func(q *dbent.AccountQuery) {
			q.WithGroups()
		}).
		Order(dbent.Asc(entupstream.FieldName), dbent.Asc(entupstream.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to list upstreams")
		return
	}
	views := make([]upstreamView, 0, len(items))
	for _, item := range items {
		views = append(views, buildUpstreamView(item, false, 0))
	}
	response.Paginated(c, views, int64(total), page, pageSize)
}

func (h *UpstreamHandler) Get(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	item, err := h.queryUpstream(c, id, true)
	if err != nil {
		writeUpstreamQueryError(c, err)
		return
	}
	duplicateCount, _ := h.client.Upstream.Query().
		Where(entupstream.BaseURLEQ(item.BaseURL), entupstream.IDNEQ(item.ID)).
		Count(c.Request.Context())
	response.Success(c, buildUpstreamView(item, true, duplicateCount))
}

func (h *UpstreamHandler) Create(c *gin.Context) {
	var req upstreamMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		response.BadRequest(c, "name is required")
		return
	}
	baseURL, err := normalizeUpstreamSub2APIBaseURL(req.BaseURL)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	kind, err := parseUpstreamKind(req.Kind, entupstream.KindAuto)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	credentials, err := mergeUpstreamCredentials(nil, req.Credentials, req.ClearCredentials)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.validateUpstreamProxy(c, req.ProxyID); err != nil {
		return
	}
	item, err := h.client.Upstream.Create().
		SetName(name).
		SetBaseURL(baseURL).
		SetKind(kind).
		SetCredentials(credentials).
		SetNillableProxyID(req.ProxyID).
		Save(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to create upstream")
		return
	}
	item, err = h.queryUpstream(c, item.ID, true)
	if err != nil {
		response.InternalError(c, "Upstream was created but could not be loaded")
		return
	}
	duplicateCount, _ := h.client.Upstream.Query().
		Where(entupstream.BaseURLEQ(item.BaseURL), entupstream.IDNEQ(item.ID)).
		Count(c.Request.Context())
	response.Created(c, buildUpstreamView(item, true, duplicateCount))
}

func (h *UpstreamHandler) Update(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	lock := h.refreshLock(id)
	lock.Lock()
	defer lock.Unlock()
	existing, err := h.client.Upstream.Get(c.Request.Context(), id)
	if err != nil {
		writeUpstreamQueryError(c, err)
		return
	}
	var req upstreamMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = existing.Name
	}
	baseURL := existing.BaseURL
	if strings.TrimSpace(req.BaseURL) != "" {
		baseURL, err = normalizeUpstreamSub2APIBaseURL(req.BaseURL)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}
	kind, err := parseUpstreamKind(req.Kind, existing.Kind)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	credentials, err := mergeUpstreamCredentials(existing.Credentials, req.Credentials, req.ClearCredentials)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	proxyID := existing.ProxyID
	if req.ClearProxy {
		proxyID = nil
	} else if req.ProxyID != nil {
		proxyID = req.ProxyID
	}
	if err := h.validateUpstreamProxy(c, proxyID); err != nil {
		return
	}
	probeIdentityChanged := existing.BaseURL != baseURL || existing.Kind != kind ||
		!reflect.DeepEqual(existing.Credentials, credentials) || !equalOptionalInt64(existing.ProxyID, proxyID)
	update := h.client.Upstream.UpdateOneID(id).
		SetName(name).
		SetBaseURL(baseURL).
		SetKind(kind).
		SetCredentials(credentials)
	if proxyID == nil {
		update.ClearProxyID()
	} else {
		update.SetProxyID(*proxyID)
	}
	if probeIdentityChanged {
		update.SetStatus(entupstream.StatusUnknown).
			ClearLastProbeAt().
			ClearLastProbeError().
			SetMetadata(map[string]any{})
	}
	if _, err := update.Save(c.Request.Context()); err != nil {
		response.InternalError(c, "Failed to update upstream")
		return
	}
	item, err := h.queryUpstream(c, id, true)
	if err != nil {
		response.InternalError(c, "Upstream was updated but could not be loaded")
		return
	}
	duplicateCount, _ := h.client.Upstream.Query().
		Where(entupstream.BaseURLEQ(item.BaseURL), entupstream.IDNEQ(item.ID)).
		Count(c.Request.Context())
	response.Success(c, buildUpstreamView(item, true, duplicateCount))
}

func (h *UpstreamHandler) Delete(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	lock := h.refreshLock(id)
	lock.Lock()
	defer lock.Unlock()
	if _, err := h.client.Upstream.Get(c.Request.Context(), id); err != nil {
		writeUpstreamQueryError(c, err)
		return
	}
	ctx := c.Request.Context()
	count, err := h.client.Account.Query().Where(entaccount.UpstreamID(id)).Count(ctx)
	if err != nil {
		response.InternalError(c, "Failed to inspect bound accounts")
		return
	}
	force := c.Query("force") == "1" || strings.EqualFold(c.Query("force"), "true")
	if count > 0 && !force {
		response.Error(c, http.StatusConflict, "Unbind existing accounts before deleting this upstream")
		return
	}
	tx, err := h.client.Tx(ctx)
	if err != nil {
		response.InternalError(c, "Failed to start delete transaction")
		return
	}
	defer func() { _ = tx.Rollback() }()
	if count > 0 {
		if _, err := tx.Account.Update().Where(entaccount.UpstreamID(id)).ClearUpstreamID().Save(ctx); err != nil {
			response.InternalError(c, "Failed to unbind accounts")
			return
		}
	}
	if err := tx.Upstream.DeleteOneID(id).Exec(ctx); err != nil {
		response.InternalError(c, "Failed to delete upstream")
		return
	}
	if err := tx.Commit(); err != nil {
		response.InternalError(c, "Failed to commit delete transaction")
		return
	}
	response.Success(c, gin.H{"deleted": true, "unbound_account_count": count})
}

func (h *UpstreamHandler) ListBindCandidates(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	if _, err := h.client.Upstream.Get(c.Request.Context(), id); err != nil {
		writeUpstreamQueryError(c, err)
		return
	}
	page, pageSize := response.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	query := h.client.Account.Query().Where(
		entaccount.TypeIn(service.AccountTypeAPIKey, service.AccountTypeUpstream),
		entaccount.UpstreamIDIsNil(),
	)
	if search != "" {
		query.Where(entaccount.NameContainsFold(search))
	}
	total, err := query.Clone().Count(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to count accounts")
		return
	}
	accounts, err := query.WithGroups().
		Order(dbent.Asc(entaccount.FieldName), dbent.Asc(entaccount.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to list accounts")
		return
	}
	items := make([]upstreamAccountSummary, 0, len(accounts))
	for _, account := range accounts {
		items = append(items, buildUpstreamAccountSummary(account, nil, nil))
	}
	response.Paginated(c, items, int64(total), page, pageSize)
}

func (h *UpstreamHandler) BindAccounts(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	if _, err := h.client.Upstream.Get(c.Request.Context(), id); err != nil {
		writeUpstreamQueryError(c, err)
		return
	}
	var req upstreamBindRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.AccountIDs) == 0 {
		response.BadRequest(c, "account_ids is required")
		return
	}
	ids := dedupePositiveInt64(req.AccountIDs)
	ctx := c.Request.Context()
	accounts, err := h.client.Account.Query().Where(entaccount.IDIn(ids...)).All(ctx)
	if err != nil {
		response.InternalError(c, "Failed to load accounts")
		return
	}
	if len(accounts) != len(ids) {
		response.BadRequest(c, "One or more accounts do not exist")
		return
	}
	result := upstreamBindResult{}
	toUpdate := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		if account.Type != service.AccountTypeAPIKey && account.Type != service.AccountTypeUpstream {
			response.BadRequest(c, "Only API key accounts can be bound to an upstream")
			return
		}
		if account.UpstreamID != nil && *account.UpstreamID == id {
			result.UnchangedIDs = append(result.UnchangedIDs, account.ID)
			continue
		}
		if account.UpstreamID != nil && *account.UpstreamID != id && !req.AllowRebind {
			response.Error(c, http.StatusConflict, "One or more accounts are already bound to another upstream")
			return
		}
		toUpdate = append(toUpdate, account.ID)
	}
	if len(toUpdate) > 0 {
		if _, err := h.client.Account.Update().Where(entaccount.IDIn(toUpdate...)).SetUpstreamID(id).Save(ctx); err != nil {
			response.InternalError(c, "Failed to bind accounts")
			return
		}
		result.UpdatedAccountIDs = toUpdate
	}
	sort.Slice(result.UpdatedAccountIDs, func(i, j int) bool { return result.UpdatedAccountIDs[i] < result.UpdatedAccountIDs[j] })
	sort.Slice(result.UnchangedIDs, func(i, j int) bool { return result.UnchangedIDs[i] < result.UnchangedIDs[j] })
	response.Success(c, result)
}

func (h *UpstreamHandler) UnbindAccounts(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	var req upstreamUnbindRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.AccountIDs) == 0 {
		response.BadRequest(c, "account_ids is required")
		return
	}
	ids := dedupePositiveInt64(req.AccountIDs)
	ctx := c.Request.Context()
	boundIDs, err := h.client.Account.Query().
		Where(entaccount.IDIn(ids...), entaccount.UpstreamID(id)).
		IDs(ctx)
	if err != nil {
		response.InternalError(c, "Failed to load bound accounts")
		return
	}
	deleteAccounts := req.DeleteAccounts == nil || *req.DeleteAccounts
	result := upstreamBindResult{UnchangedIDs: differenceInt64(ids, boundIDs)}
	if deleteAccounts && len(boundIDs) > 0 {
		if h.adminService == nil {
			response.InternalError(c, "Account deletion service is unavailable")
			return
		}
		for _, accountID := range boundIDs {
			if err := h.adminService.DeleteAccount(ctx, accountID); err != nil {
				response.InternalError(c, "Failed to delete unbound account")
				return
			}
			result.DeletedAccountIDs = append(result.DeletedAccountIDs, accountID)
		}
	} else if len(boundIDs) > 0 {
		if _, err := h.client.Account.Update().
			Where(entaccount.IDIn(boundIDs...), entaccount.UpstreamID(id)).
			ClearUpstreamID().
			Save(ctx); err != nil {
			response.InternalError(c, "Failed to unbind accounts")
			return
		}
		result.UpdatedAccountIDs = append(result.UpdatedAccountIDs, boundIDs...)
	}
	sort.Slice(result.UpdatedAccountIDs, func(i, j int) bool { return result.UpdatedAccountIDs[i] < result.UpdatedAccountIDs[j] })
	sort.Slice(result.DeletedAccountIDs, func(i, j int) bool { return result.DeletedAccountIDs[i] < result.DeletedAccountIDs[j] })
	sort.Slice(result.UnchangedIDs, func(i, j int) bool { return result.UnchangedIDs[i] < result.UnchangedIDs[j] })
	response.Success(c, result)
}

func (h *UpstreamHandler) queryUpstream(c *gin.Context, id int64, withAccounts bool) (*dbent.Upstream, error) {
	query := h.client.Upstream.Query().Where(entupstream.ID(id)).WithProxy()
	if withAccounts {
		query.WithAccounts(func(q *dbent.AccountQuery) {
			q.WithGroups().Order(dbent.Asc(entaccount.FieldName), dbent.Asc(entaccount.FieldID))
		})
	}
	return query.Only(c.Request.Context())
}

func buildUpstreamView(item *dbent.Upstream, includeAccounts bool, duplicateCount int) upstreamView {
	var lastProbeError *string
	if item.LastProbeError != nil {
		safe := safeUpstreamProbeError(errors.New(*item.LastProbeError), item)
		if safe != "" {
			lastProbeError = &safe
		}
	}
	view := upstreamView{
		ID:                    item.ID,
		Name:                  item.Name,
		BaseURL:               item.BaseURL,
		Kind:                  item.Kind.String(),
		ProxyID:               item.ProxyID,
		Status:                item.Status.String(),
		LastProbeAt:           item.LastProbeAt,
		LastProbeError:        lastProbeError,
		Metadata:              safeUpstreamMetadata(item.Metadata, item),
		CredentialStatus:      buildUpstreamCredentialStatus(item.Credentials),
		AccountCount:          len(item.Edges.Accounts),
		LocalGroups:           buildUpstreamLocalGroupSummaries(item.Edges.Accounts),
		DuplicateBaseURLCount: duplicateCount,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}
	if item.Edges.Proxy != nil {
		view.ProxyName = item.Edges.Proxy.Name
	}
	if includeAccounts {
		metadata, _ := parseUpstreamProbeMetadata(item.Metadata)
		view.Accounts = make([]upstreamAccountSummary, 0, len(item.Edges.Accounts))
		for _, account := range item.Edges.Accounts {
			view.Accounts = append(view.Accounts, buildUpstreamAccountSummary(account, item, &metadata))
		}
		sort.SliceStable(view.Accounts, func(i, j int) bool {
			left := strings.ToLower(view.Accounts[i].Name)
			right := strings.ToLower(view.Accounts[j].Name)
			if left == right {
				if view.Accounts[i].Name == view.Accounts[j].Name {
					return view.Accounts[i].ID < view.Accounts[j].ID
				}
				return view.Accounts[i].Name < view.Accounts[j].Name
			}
			return left < right
		})
	}
	return view
}

func buildUpstreamLocalGroupSummaries(accounts []*dbent.Account) []upstreamLocalGroupSummary {
	groupsByID := make(map[int64]upstreamLocalGroupSummary)
	for _, account := range accounts {
		for _, group := range account.Edges.Groups {
			groupsByID[group.ID] = upstreamLocalGroupSummary{
				ID:       group.ID,
				Name:     group.Name,
				Platform: group.Platform,
			}
		}
	}
	groups := make([]upstreamLocalGroupSummary, 0, len(groupsByID))
	for _, group := range groupsByID {
		groups = append(groups, group)
	}
	sort.SliceStable(groups, func(i, j int) bool {
		left := strings.ToLower(groups[i].Name)
		right := strings.ToLower(groups[j].Name)
		if left == right {
			if groups[i].Name == groups[j].Name {
				return groups[i].ID < groups[j].ID
			}
			return groups[i].Name < groups[j].Name
		}
		return left < right
	})
	return groups
}

func buildUpstreamAccountSummary(item *dbent.Account, upstream *dbent.Upstream, metadata *upstreamProbeMetadata) upstreamAccountSummary {
	groupIDs := make([]int64, 0, len(item.Edges.Groups))
	for _, group := range item.Edges.Groups {
		groupIDs = append(groupIDs, group.ID)
	}
	sort.Slice(groupIDs, func(i, j int) bool { return groupIDs[i] < groupIDs[j] })
	generated, _ := item.Extra["upstream_generated"].(bool)
	upstreamGroup, _ := item.Extra["upstream_group_name"].(string)
	upstreamGroupID, _ := numericInt64(item.Extra["upstream_group_id"])
	summary := upstreamAccountSummary{
		ID:            item.ID,
		Name:          item.Name,
		Platform:      item.Platform,
		Type:          item.Type,
		Status:        item.Status,
		Schedulable:   item.Schedulable,
		UpstreamID:    item.UpstreamID,
		GroupIDs:      groupIDs,
		Generated:     generated,
		UpstreamGroup: strings.TrimSpace(upstreamGroup),
	}
	if upstreamGroupID > 0 {
		summary.UpstreamGroupID = &upstreamGroupID
	}
	if metadata != nil {
		if billing, ok := metadata.AccountBilling[strconv.FormatInt(item.ID, 10)]; ok {
			if billing.UpstreamGroupID != nil {
				summary.UpstreamGroupID = billing.UpstreamGroupID
			}
			if name := strings.TrimSpace(billing.UpstreamGroupName); name != "" {
				summary.UpstreamGroup = name
			}
			summary.UpstreamGroupStale = billing.Stale || billing.Status != "ok"
			switch {
			case billing.GroupEffectiveRateMultiplier != nil:
				summary.UpstreamGroupRateMultiplier = billing.GroupEffectiveRateMultiplier
				summary.UpstreamGroupRateSource = "effective"
			case billing.GroupDefaultRateMultiplier != nil:
				summary.UpstreamGroupRateMultiplier = billing.GroupDefaultRateMultiplier
				summary.UpstreamGroupRateSource = "default"
			}
		} else if summary.UpstreamGroup != "" {
			summary.UpstreamGroupStale = true
		}
		if summary.UpstreamGroupRateMultiplier == nil && summary.UpstreamGroup != "" {
			if group := findUpstreamProbeGroup(metadata.Groups, item.Platform, summary.UpstreamGroup); group != nil && group.RateMultiplier != nil {
				rate := *group.RateMultiplier
				summary.UpstreamGroupRateMultiplier = &rate
				summary.UpstreamGroupRateSource = "catalogue"
			}
		}
	}
	summary.UpstreamGroupChangeSupported, summary.UpstreamGroupChangeReason = upstreamAccountGroupChangeCapability(upstream, item)
	return summary
}

func findUpstreamProbeGroup(groups []upstreamProbeGroup, platform, name string) *upstreamProbeGroup {
	for index := range groups {
		group := &groups[index]
		if !strings.EqualFold(strings.TrimSpace(group.Name), strings.TrimSpace(name)) {
			continue
		}
		if group.Platform == "" || strings.EqualFold(strings.TrimSpace(group.Platform), strings.TrimSpace(platform)) {
			return group
		}
	}
	return nil
}

func numericInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		converted := int64(typed)
		return converted, float64(converted) == typed
	case json.Number:
		parsed, err := typed.Int64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func upstreamAccountGroupChangeCapability(upstream *dbent.Upstream, account *dbent.Account) (bool, string) {
	if upstream == nil || account == nil {
		return false, "upstream management context is unavailable"
	}
	if account.Type != service.AccountTypeAPIKey && account.Type != service.AccountTypeUpstream {
		return false, "only API key accounts can change upstream groups"
	}
	if upstreamCredentialString(account.Credentials, upstreamCredentialAPIKey) == "" {
		return false, "the bound account has no API key"
	}
	switch upstream.Kind {
	case entupstream.KindNewapi:
		if upstreamCredentialPresent(upstream.Credentials, upstreamCredentialManagementAccessToken) {
			if upstreamNewAPIAccessTokenSession(upstream.Credentials) == nil {
				return false, "a valid NewAPI management user ID is required"
			}
			return true, ""
		}
		if hasUpstreamPanelLogin(upstream.Credentials) {
			return true, ""
		}
		return false, "NewAPI management credentials are required"
	case entupstream.KindSub2api:
		if hasUpstreamPanelLogin(upstream.Credentials) {
			return true, ""
		}
		return false, "Sub2API panel credentials are required"
	default:
		return false, "probe and identify the upstream type first"
	}
}

func buildUpstreamCredentialStatus(credentials map[string]any) upstreamCredentialStatus {
	return upstreamCredentialStatus{
		HasAPIKey:                upstreamCredentialPresent(credentials, upstreamCredentialAPIKey),
		HasOpenAIAPIKey:          upstreamCredentialPresent(credentials, upstreamCredentialOpenAIAPIKey),
		HasAnthropicAPIKey:       upstreamCredentialPresent(credentials, upstreamCredentialAnthropicAPIKey),
		HasGeminiAPIKey:          upstreamCredentialPresent(credentials, upstreamCredentialGeminiAPIKey),
		HasGrokAPIKey:            upstreamCredentialPresent(credentials, upstreamCredentialGrokAPIKey),
		HasManagementAccessToken: upstreamCredentialPresent(credentials, upstreamCredentialManagementAccessToken),
		HasManagementUserID:      upstreamCredentialPresent(credentials, upstreamCredentialManagementUserID),
		HasUsername:              upstreamCredentialPresent(credentials, upstreamCredentialUsername),
		HasPassword:              upstreamCredentialPresent(credentials, upstreamCredentialPassword),
		GeneratedGroupKeyCount:   len(generatedUpstreamGroupKeys(credentials)),
	}
}

func mergeUpstreamCredentials(existing map[string]any, incoming map[string]string, clear []string) (map[string]any, error) {
	result := maps.Clone(existing)
	if result == nil {
		result = map[string]any{}
	}
	for key, value := range incoming {
		key = strings.TrimSpace(key)
		if _, ok := editableUpstreamCredentialKeys[key]; !ok {
			return nil, &upstreamValidationError{message: "Unsupported credential field: " + key}
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result[key] = value
	}
	for _, key := range clear {
		key = strings.TrimSpace(key)
		if _, ok := editableUpstreamCredentialKeys[key]; !ok {
			return nil, &upstreamValidationError{message: "Unsupported credential field: " + key}
		}
		delete(result, key)
	}
	return result, nil
}

type upstreamValidationError struct{ message string }

func (e *upstreamValidationError) Error() string { return e.message }

func parseUpstreamKind(raw string, fallback entupstream.Kind) (entupstream.Kind, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	value := entupstream.Kind(strings.ToLower(strings.TrimSpace(raw)))
	if err := entupstream.KindValidator(value); err != nil {
		return "", &upstreamValidationError{message: "kind must be auto, newapi, or sub2api"}
	}
	return value, nil
}

func parseUpstreamID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid upstream ID")
		return 0, false
	}
	return id, true
}

func writeUpstreamQueryError(c *gin.Context, err error) {
	if dbent.IsNotFound(err) {
		response.NotFound(c, "Upstream not found")
		return
	}
	response.InternalError(c, "Failed to load upstream")
}

func (h *UpstreamHandler) validateUpstreamProxy(c *gin.Context, proxyID *int64) error {
	if proxyID == nil {
		return nil
	}
	if *proxyID <= 0 {
		response.BadRequest(c, "Invalid proxy ID")
		return &upstreamValidationError{message: "invalid proxy ID"}
	}
	if h.adminService == nil {
		response.InternalError(c, "Proxy validation is unavailable")
		return &upstreamValidationError{message: "proxy validation unavailable"}
	}
	if _, err := h.adminService.GetProxy(c.Request.Context(), *proxyID); err != nil {
		response.BadRequest(c, "Proxy not found")
		return err
	}
	return nil
}

func upstreamCredentialPresent(credentials map[string]any, key string) bool {
	value, ok := credentials[key]
	if !ok {
		return false
	}
	text, ok := value.(string)
	return ok && strings.TrimSpace(text) != ""
}

func upstreamCredentialString(credentials map[string]any, key string) string {
	value, _ := credentials[key].(string)
	return strings.TrimSpace(value)
}

func upstreamAPIKeyForPlatform(credentials map[string]any, platform string) string {
	key := ""
	switch platform {
	case service.PlatformOpenAI:
		key = upstreamCredentialString(credentials, upstreamCredentialOpenAIAPIKey)
	case service.PlatformAnthropic:
		key = upstreamCredentialString(credentials, upstreamCredentialAnthropicAPIKey)
	case service.PlatformGemini:
		key = upstreamCredentialString(credentials, upstreamCredentialGeminiAPIKey)
	case service.PlatformGrok:
		key = upstreamCredentialString(credentials, upstreamCredentialGrokAPIKey)
	}
	if key == "" {
		key = upstreamCredentialString(credentials, upstreamCredentialAPIKey)
	}
	return key
}

func generatedUpstreamGroupKeys(credentials map[string]any) map[string]any {
	raw, _ := credentials[upstreamCredentialGeneratedGroupKeys].(map[string]any)
	if raw == nil {
		return map[string]any{}
	}
	return raw
}

func equalOptionalInt64(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func dedupePositiveInt64(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func differenceInt64(all, subset []int64) []int64 {
	set := make(map[int64]struct{}, len(subset))
	for _, value := range subset {
		set[value] = struct{}{}
	}
	result := make([]int64, 0, len(all))
	for _, value := range all {
		if _, ok := set[value]; !ok {
			result = append(result, value)
		}
	}
	return result
}
