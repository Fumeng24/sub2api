package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	entaccount "github.com/Wei-Shaw/sub2api/ent/account"
	entupstream "github.com/Wei-Shaw/sub2api/ent/upstream"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	upstreamRefreshScanInterval = time.Minute
	upstreamRefreshInterval     = 5 * time.Minute
	upstreamRefreshProbeTimeout = 75 * time.Second
	upstreamAccountProbeTimeout = 30 * time.Second
	upstreamRefreshMaxPerCycle  = 20
	upstreamRefreshConcurrency  = 4
	upstreamRefreshLeaderTTL    = 10 * time.Minute
	upstreamRefreshLeaderKey    = "upstream:management:refresh:leader"
	managedAccountStatusMaxIDs  = 500
	managedAccountStatusMaxBody = 32 << 10
)

type upstreamRefreshMetadata struct {
	Status              string     `json:"status"`
	Stale               bool       `json:"stale"`
	LastAttemptAt       time.Time  `json:"last_attempt_at"`
	LastSuccessAt       *time.Time `json:"last_success_at,omitempty"`
	NextRefreshAt       time.Time  `json:"next_refresh_at"`
	FailureCount        int        `json:"failure_count,omitempty"`
	AccountSuccessCount int        `json:"account_success_count"`
	AccountFailureCount int        `json:"account_failure_count"`
}

type upstreamAccountBillingMetadata struct {
	AccountID                    int64      `json:"account_id"`
	Status                       string     `json:"status"`
	Message                      string     `json:"message,omitempty"`
	ProbeSource                  string     `json:"probe_source,omitempty"`
	FetchedAt                    time.Time  `json:"fetched_at"`
	LastSuccessAt                *time.Time `json:"last_success_at,omitempty"`
	Stale                        bool       `json:"stale"`
	FailureCount                 int        `json:"failure_count,omitempty"`
	KeyRemaining                 *float64   `json:"key_remaining,omitempty"`
	BalanceUnit                  string     `json:"balance_unit,omitempty"`
	UsageMode                    string     `json:"usage_mode,omitempty"`
	UsagePlanName                string     `json:"usage_plan_name,omitempty"`
	UpstreamKeyID                *int64     `json:"upstream_key_id,omitempty"`
	UpstreamKeyName              string     `json:"upstream_key_name,omitempty"`
	UpstreamGroupID              *int64     `json:"upstream_group_id,omitempty"`
	UpstreamGroupName            string     `json:"upstream_group_name,omitempty"`
	UpstreamGroupPlatform        string     `json:"upstream_group_platform,omitempty"`
	GroupDefaultRateMultiplier   *float64   `json:"group_default_rate_multiplier,omitempty"`
	GroupEffectiveRateMultiplier *float64   `json:"group_effective_rate_multiplier,omitempty"`
}

type refreshManagedUpstreamAccountsRequest struct {
	AccountIDs []int64 `json:"account_ids"`
}

type upstreamRefreshRunner struct {
	handler    *UpstreamHandler
	leaderLock service.LeaderLockCache
	db         *sql.DB
	owner      string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	cycle  sync.Mutex
	start  bool
	stop   bool
}

func newUpstreamRefreshRunner(handler *UpstreamHandler, leaderLock service.LeaderLockCache, db *sql.DB) *upstreamRefreshRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &upstreamRefreshRunner{
		handler:    handler,
		leaderLock: leaderLock,
		db:         db,
		owner:      fmt.Sprintf("%d-%p", time.Now().UnixNano(), handler),
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (r *upstreamRefreshRunner) Start() {
	if r == nil || r.handler == nil || r.handler.client == nil {
		return
	}
	r.mu.Lock()
	if r.start || r.stop {
		r.mu.Unlock()
		return
	}
	r.start = true
	r.wg.Add(1)
	r.mu.Unlock()
	go r.run()
}

func (r *upstreamRefreshRunner) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.stop {
		r.mu.Unlock()
		return
	}
	r.stop = true
	r.cancel()
	r.mu.Unlock()
	r.wg.Wait()
}

func (r *upstreamRefreshRunner) run() {
	defer r.wg.Done()
	r.runDue(r.ctx)
	ticker := time.NewTicker(upstreamRefreshScanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.runDue(r.ctx)
		}
	}
}

func (r *upstreamRefreshRunner) runDue(ctx context.Context) {
	r.cycle.Lock()
	defer r.cycle.Unlock()

	release, acquired := r.acquireLeader(ctx)
	if !acquired {
		return
	}
	defer release()

	items, err := r.handler.client.Upstream.Query().
		Select(entupstream.FieldID, entupstream.FieldMetadata, entupstream.FieldLastProbeAt).
		Order(dbent.Asc(entupstream.FieldID)).
		All(ctx)
	if err != nil {
		logger.LegacyPrintf("handler.upstream_refresh", "list upstreams failed: err=%v", err)
		return
	}
	now := time.Now().UTC()
	due := make([]*dbent.Upstream, 0, len(items))
	for _, item := range items {
		if upstreamRefreshDue(item, now) {
			due = append(due, item)
		}
	}
	sortUpstreamsByRefreshDue(due)
	if len(due) > upstreamRefreshMaxPerCycle {
		due = due[:upstreamRefreshMaxPerCycle]
	}

	sem := make(chan struct{}, upstreamRefreshConcurrency)
	var wg sync.WaitGroup
	for _, item := range due {
		upstreamID := item.ID
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			probeCtx, cancel := context.WithTimeout(ctx, upstreamRefreshProbeTimeout)
			defer cancel()
			if _, err := r.handler.refreshUpstreamByID(probeCtx, upstreamID, false); err != nil && !errors.Is(err, context.Canceled) {
				logger.LegacyPrintf("handler.upstream_refresh", "refresh failed: upstream_id=%d err=%v", upstreamID, err)
			}
		}()
	}
	wg.Wait()
}

func sortUpstreamsByRefreshDue(items []*dbent.Upstream) {
	sort.SliceStable(items, func(i, j int) bool {
		left := upstreamRefreshDueAt(items[i])
		right := upstreamRefreshDueAt(items[j])
		if left.Equal(right) {
			return items[i].ID < items[j].ID
		}
		if left.IsZero() {
			return true
		}
		if right.IsZero() {
			return false
		}
		return left.Before(right)
	})
}

func (r *upstreamRefreshRunner) acquireLeader(ctx context.Context) (func(), bool) {
	lockCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return service.AcquireSingletonLeaderLock(lockCtx, r.leaderLock, r.db, upstreamRefreshLeaderKey, r.owner, upstreamRefreshLeaderTTL)
}

func upstreamRefreshDue(item *dbent.Upstream, now time.Time) bool {
	if item == nil {
		return false
	}
	dueAt := upstreamRefreshDueAt(item)
	return dueAt.IsZero() || !now.Before(dueAt)
}

func upstreamRefreshDueAt(item *dbent.Upstream) time.Time {
	if item == nil {
		return time.Time{}
	}
	metadata, err := parseUpstreamProbeMetadata(item.Metadata)
	if err == nil && metadata.Refresh != nil && !metadata.Refresh.NextRefreshAt.IsZero() {
		return metadata.Refresh.NextRefreshAt
	}
	if item.LastProbeAt == nil {
		return time.Time{}
	}
	return item.LastProbeAt.Add(upstreamRefreshInterval)
}

func (h *UpstreamHandler) refreshLock(upstreamID int64) *sync.Mutex {
	value, _ := h.refreshLocks.LoadOrStore(upstreamID, &sync.Mutex{})
	return value.(*sync.Mutex)
}

func (h *UpstreamHandler) refreshUpstreamByID(ctx context.Context, upstreamID int64, full bool) (*dbent.Upstream, error) {
	if h == nil || h.client == nil {
		return nil, errors.New("upstream refresh is unavailable")
	}
	lock := h.refreshLock(upstreamID)
	lock.Lock()
	defer lock.Unlock()

	item, err := h.client.Upstream.Query().
		Where(entupstream.ID(upstreamID)).
		WithProxy().
		WithAccounts(func(query *dbent.AccountQuery) {
			query.Order(dbent.Asc(entaccount.FieldID))
		}).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	previous, _ := parseUpstreamProbeMetadata(item.Metadata)

	var outcome upstreamProbeOutcome
	managementOK := false
	if full {
		outcome = h.probeUpstream(ctx, item)
		managementOK = outcome.metadata.ManagementStatus == "ok"
	} else {
		metadata, ok, detectedKindVerified, probeErr := h.probeUpstreamManagement(ctx, item)
		managementOK = ok
		outcome = upstreamProbeOutcome{
			metadata:             metadata,
			status:               entupstream.StatusError,
			detectedKindVerified: detectedKindVerified,
		}
		if ok {
			outcome.status = entupstream.StatusHealthy
		} else {
			outcome.errorMsg = metadata.ManagementHint
			if outcome.errorMsg == "" && probeErr != nil {
				outcome.errorMsg = safeUpstreamProbeError(probeErr, item)
			}
		}
	}

	metadata := mergeUpstreamManagementMetadata(previous, outcome.metadata, full)
	accountBilling, accountSuccesses, accountFailures := h.refreshBoundAccountBilling(ctx, item, metadata, previous.AccountBilling)
	metadata.AccountBilling = accountBilling
	metadata.Refresh = nextUpstreamRefreshMetadata(previous.Refresh, item.ID, managementOK, accountSuccesses, accountFailures)

	if accountFailures > 0 {
		if outcome.status == entupstream.StatusHealthy {
			outcome.status = entupstream.StatusDegraded
		}
	}
	if !managementOK && accountSuccesses > 0 && outcome.status == entupstream.StatusError {
		outcome.status = entupstream.StatusDegraded
	}
	// A panel failure only prevents wallet/rate management. If a previously
	// verified protocol capability remains available, the upstream is partially
	// usable rather than completely unavailable.
	if !managementOK && outcome.status == entupstream.StatusError && hasVerifiedUpstreamProtocol(metadata.Protocols) {
		outcome.status = entupstream.StatusDegraded
	}
	if metadata.Refresh.Status == "ok" && !full {
		outcome.status, outcome.errorMsg = lightweightRefreshStatus(metadata)
	}
	if metadata.Refresh.Status == "failed" && strings.TrimSpace(outcome.errorMsg) == "" {
		outcome.errorMsg = "Upstream management metadata could not be refreshed"
	}
	outcome.errorMsg = upstreamRefreshErrorSummary(outcome.errorMsg, managementOK, accountBilling, accountFailures)

	encoded, err := upstreamMetadataMap(metadata)
	if err != nil {
		return nil, err
	}
	update := h.client.Upstream.UpdateOneID(item.ID).
		SetStatus(outcome.status).
		SetLastProbeAt(metadata.Refresh.LastAttemptAt).
		SetUpdatedAt(item.UpdatedAt).
		SetMetadata(encoded)
	if outcome.errorMsg == "" {
		update.ClearLastProbeError()
	} else {
		update.SetLastProbeError(safeUpstreamProbeError(errors.New(outcome.errorMsg), item))
	}
	if item.Kind == entupstream.KindAuto && outcome.detectedKindVerified && metadata.DetectedKind != "" {
		if detected, parseErr := parseUpstreamKind(metadata.DetectedKind, entupstream.KindAuto); parseErr == nil && detected != entupstream.KindAuto {
			update.SetKind(detected)
		}
	}
	if _, err := update.Save(ctx); err != nil {
		return nil, err
	}
	return h.client.Upstream.Query().
		Where(entupstream.ID(item.ID)).
		WithProxy().
		WithAccounts(func(query *dbent.AccountQuery) {
			query.WithGroups().Order(dbent.Desc(entaccount.FieldID))
		}).
		Only(ctx)
}

func upstreamRefreshErrorSummary(
	existing string,
	managementOK bool,
	accountBilling map[string]upstreamAccountBillingMetadata,
	accountFailures int,
) string {
	parts := make([]string, 0, 4)
	if existing = strings.TrimSpace(existing); existing != "" {
		scope := "upstream capability probe"
		if !managementOK {
			scope = "management probe"
		}
		parts = append(parts, scope+": "+existing)
	}
	if accountFailures <= 0 {
		return strings.Join(parts, "; ")
	}

	failures := make([]upstreamAccountBillingMetadata, 0, accountFailures)
	for _, billing := range accountBilling {
		if billing.Stale {
			failures = append(failures, billing)
		}
	}
	sort.Slice(failures, func(i, j int) bool { return failures[i].AccountID < failures[j].AccountID })

	const maxDetails = 2
	detailCount := len(failures)
	if detailCount > maxDetails {
		detailCount = maxDetails
	}
	for index := 0; index < detailCount; index++ {
		billing := failures[index]
		code := strings.TrimSpace(billing.Status)
		if code == "" {
			code = "unknown"
		}
		if code == "ok" && billing.GroupEffectiveRateMultiplier == nil && billing.GroupDefaultRateMultiplier == nil {
			code = "rate_unavailable"
		}
		detail := fmt.Sprintf("account #%d [%s]", billing.AccountID, code)
		if message := strings.TrimSpace(billing.Message); message != "" {
			detail += ": " + message
		}
		parts = append(parts, detail)
	}
	remaining := accountFailures - detailCount
	if remaining > 0 {
		parts = append(parts, fmt.Sprintf("%d more bound account failure(s)", remaining))
	}
	if len(failures) == 0 {
		parts = append(parts, fmt.Sprintf("%d bound account key(s) could not be mapped to an upstream rate", accountFailures))
	}
	return strings.Join(parts, "; ")
}

func lightweightRefreshStatus(metadata upstreamProbeMetadata) (entupstream.Status, string) {
	if len(metadata.Protocols) == 0 {
		return entupstream.StatusHealthy, ""
	}
	failures := make([]string, 0, len(metadata.Protocols))
	for _, protocol := range metadata.Protocols {
		if protocol.Status == "ok" {
			return entupstream.StatusHealthy, ""
		}
		if protocol.Status == "missing_api_key" {
			continue
		}
		message := strings.TrimSpace(protocol.Message)
		if message == "" {
			message = strings.TrimSpace(protocol.Status)
		}
		failures = append(failures, fmt.Sprintf("%s: %s", protocol.Platform, message))
	}
	if len(failures) == 0 {
		return entupstream.StatusHealthy, ""
	}
	return entupstream.StatusDegraded, strings.Join(failures, "; ")
}

func hasVerifiedUpstreamProtocol(protocols []upstreamProtocolCapability) bool {
	for _, protocol := range protocols {
		if strings.EqualFold(strings.TrimSpace(protocol.Status), "ok") {
			return true
		}
	}
	return false
}

func mergeUpstreamManagementMetadata(previous, current upstreamProbeMetadata, full bool) upstreamProbeMetadata {
	if !full {
		current.Protocols = previous.Protocols
		preserveUpstreamGroupCapabilities(current.Groups, previous.Groups)
	}
	if current.Groups == nil {
		current.Groups = []upstreamProbeGroup{}
	}
	if current.Protocols == nil {
		current.Protocols = []upstreamProtocolCapability{}
	}
	attachProtocolModelsToGroups(&current)
	return current
}

func preserveUpstreamGroupCapabilities(current, previous []upstreamProbeGroup) {
	for currentIndex := range current {
		for previousIndex := range previous {
			if !sameUpstreamProbeGroup(current[currentIndex], previous[previousIndex]) {
				continue
			}
			if current[currentIndex].Platform == "" {
				current[currentIndex].Platform = previous[previousIndex].Platform
			}
			if len(current[currentIndex].Models) == 0 && len(previous[previousIndex].Models) > 0 {
				current[currentIndex].Models = append([]string(nil), previous[previousIndex].Models...)
			}
			break
		}
	}
}

func sameUpstreamProbeGroup(left, right upstreamProbeGroup) bool {
	if left.ID != nil || right.ID != nil {
		return left.ID != nil && right.ID != nil && *left.ID == *right.ID
	}
	if left.Name != right.Name {
		return false
	}
	return left.Platform == "" || right.Platform == "" || left.Platform == right.Platform
}

func nextUpstreamRefreshMetadata(previous *upstreamRefreshMetadata, upstreamID int64, managementOK bool, accountSuccesses, accountFailures int) *upstreamRefreshMetadata {
	now := time.Now().UTC()
	result := &upstreamRefreshMetadata{
		Status:              "ok",
		LastAttemptAt:       now,
		NextRefreshAt:       now.Add(upstreamRefreshInterval + upstreamRefreshJitter(upstreamID)),
		AccountSuccessCount: accountSuccesses,
		AccountFailureCount: accountFailures,
	}
	if previous != nil {
		result.LastSuccessAt = previous.LastSuccessAt
	}
	if managementOK && accountFailures == 0 {
		result.LastSuccessAt = &now
		return result
	}
	result.Stale = true
	result.Status = "failed"
	if managementOK || accountSuccesses > 0 {
		result.Status = "partial"
	}
	if previous != nil {
		result.FailureCount = previous.FailureCount + 1
	} else {
		result.FailureCount = 1
	}
	result.NextRefreshAt = now.Add(upstreamRefreshRetryDelay(result.FailureCount) + upstreamRefreshJitter(upstreamID))
	return result
}

func upstreamRefreshRetryDelay(failureCount int) time.Duration {
	if failureCount < 1 {
		failureCount = 1
	}
	delay := time.Minute
	for i := 1; i < failureCount && delay < upstreamRefreshInterval; i++ {
		delay *= 2
	}
	if delay > upstreamRefreshInterval {
		return upstreamRefreshInterval
	}
	return delay
}

func upstreamRefreshJitter(upstreamID int64) time.Duration {
	return time.Duration((upstreamID*17)%31) * time.Second
}

func (h *UpstreamHandler) refreshBoundAccountBilling(
	ctx context.Context,
	item *dbent.Upstream,
	metadata upstreamProbeMetadata,
	previous map[string]upstreamAccountBillingMetadata,
) (map[string]upstreamAccountBillingMetadata, int, int) {
	accounts := append([]*dbent.Account(nil), item.Edges.Accounts...)
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].ID < accounts[j].ID })
	result := make(map[string]upstreamAccountBillingMetadata, len(accounts))
	successes := 0
	failures := 0
	for _, account := range accounts {
		if account == nil {
			continue
		}
		key := strconv.FormatInt(account.ID, 10)
		prior := previous[key]
		apiKey := upstreamCredentialString(account.Credentials, upstreamCredentialAPIKey)
		if (account.Type != service.AccountTypeAPIKey && account.Type != service.AccountTypeUpstream) || apiKey == "" {
			result[key] = upstreamAccountBillingMetadata{
				AccountID:   account.ID,
				Status:      "unsupported",
				Message:     "Bound account does not expose an API key for upstream rate discovery",
				FetchedAt:   time.Now().UTC(),
				BalanceUnit: "USD",
			}
			continue
		}

		probeCtx, cancel := context.WithTimeout(ctx, upstreamAccountProbeTimeout)
		status := h.probeBoundAccountBilling(probeCtx, item, account, metadata, apiKey)
		cancel()
		billing, ok := upstreamAccountBillingFromStatus(item, status, prior)
		result[key] = billing
		if ok {
			successes++
		} else {
			failures++
		}
	}
	return result, successes, failures
}

func (h *UpstreamHandler) probeBoundAccountBilling(ctx context.Context, item *dbent.Upstream, account *dbent.Account, metadata upstreamProbeMetadata, apiKey string) UpstreamSub2APIAccountStatus {
	detectedKind := strings.TrimSpace(metadata.DetectedKind)
	if detectedKind == "" {
		detectedKind = item.Kind.String()
	}
	if detectedKind == entupstream.KindNewapi.String() && upstreamCredentialPresent(item.Credentials, upstreamCredentialManagementAccessToken) {
		return h.probeNewAPIBoundAccountWithAccessToken(ctx, item, account, metadata, apiKey)
	}

	probeAccount := h.transientUpstreamAccount(ctx, item, account.Platform, apiKey)
	probeAccount.ID = account.ID
	probeAccount.Name = account.Name
	if detectedKind == entupstream.KindNewapi.String() || detectedKind == entupstream.KindSub2api.String() {
		probeAccount.Credentials["upstream_panel_type"] = detectedKind
	}
	return h.panelClient.ProbeAccount(ctx, probeAccount, true)
}

func (h *UpstreamHandler) probeNewAPIBoundAccountWithAccessToken(ctx context.Context, item *dbent.Upstream, account *dbent.Account, metadata upstreamProbeMetadata, apiKey string) UpstreamSub2APIAccountStatus {
	now := time.Now().UTC()
	status := UpstreamSub2APIAccountStatus{
		AccountID:     account.ID,
		AccountName:   account.Name,
		LocalPlatform: account.Platform,
		BaseURL:       item.BaseURL,
		UpstreamKind:  entupstream.KindNewapi.String(),
		ProbeSource:   "management_access_token",
		Status:        "request_failed",
		FetchedAt:     now,
		BalanceUnit:   "USD",
	}
	probeAccount := h.transientUpstreamAccount(ctx, item, account.Platform, apiKey)
	probeCtx, _, err := h.panelClient.contextForAccount(ctx, probeAccount)
	if err != nil {
		status.Message = "Configured proxy is invalid"
		return status
	}
	accessToken := upstreamCredentialString(item.Credentials, upstreamCredentialManagementAccessToken)
	session := upstreamNewAPIAccessTokenSession(item.Credentials)
	if session == nil {
		status.Message = errUpstreamNewAPIManagementUserID.Error()
		return status
	}
	key, err := h.findNewAPIKeyWithAccessToken(probeCtx, item.BaseURL, accessToken, session, apiKey)
	if err != nil {
		status.Message = safeUpstreamProbeError(err, item)
		return status
	}
	if key == nil {
		status.Status = "key_not_found"
		status.Message = "Upstream API key was not found in the managed NewAPI account"
		return status
	}
	status.UpstreamKeyID = &key.ID
	status.UpstreamKeyName = strings.TrimSpace(key.Name)
	if key.UnlimitedQuota {
		status.UsageMode = "unlimited"
	} else {
		remaining := newAPIQuotaToUSD(key.RemainQuota)
		status.KeyRemaining = &remaining
		status.UsageMode = "key_quota"
	}
	status.UpstreamGroupName = strings.TrimSpace(key.Group)
	status.UpstreamGroupPlatform = account.Platform
	for _, group := range metadata.Groups {
		if group.Name != status.UpstreamGroupName || group.RateMultiplier == nil {
			continue
		}
		status.UpstreamGroupID = group.ID
		defaultRate := *group.RateMultiplier
		status.UpstreamGroupDefaultRateMultiplier = &defaultRate
		status.UpstreamGroupEffectiveRateMultiplier = &defaultRate
		break
	}
	status.Status = "ok"
	return status
}

func upstreamAccountBillingFromStatus(item *dbent.Upstream, status UpstreamSub2APIAccountStatus, previous upstreamAccountBillingMetadata) (upstreamAccountBillingMetadata, bool) {
	now := status.FetchedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	message := strings.TrimSpace(status.Message)
	if message != "" {
		message = safeUpstreamProbeError(errors.New(message), item)
	}
	current := upstreamAccountBillingMetadata{
		AccountID:                    status.AccountID,
		Status:                       status.Status,
		Message:                      message,
		ProbeSource:                  status.ProbeSource,
		FetchedAt:                    now,
		KeyRemaining:                 status.KeyRemaining,
		BalanceUnit:                  status.BalanceUnit,
		UsageMode:                    status.UsageMode,
		UsagePlanName:                status.UsagePlanName,
		UpstreamKeyID:                status.UpstreamKeyID,
		UpstreamKeyName:              status.UpstreamKeyName,
		UpstreamGroupID:              status.UpstreamGroupID,
		UpstreamGroupName:            status.UpstreamGroupName,
		UpstreamGroupPlatform:        status.UpstreamGroupPlatform,
		GroupDefaultRateMultiplier:   status.UpstreamGroupDefaultRateMultiplier,
		GroupEffectiveRateMultiplier: status.UpstreamGroupEffectiveRateMultiplier,
	}
	rate := current.GroupEffectiveRateMultiplier
	if rate == nil {
		rate = current.GroupDefaultRateMultiplier
	}
	if status.Status == "ok" && rate != nil {
		current.LastSuccessAt = &now
		return current, true
	}

	current.Stale = true
	current.FailureCount = previous.FailureCount + 1
	current.LastSuccessAt = previous.LastSuccessAt
	if current.Message == "" && status.Status == "ok" {
		current.Message = "Upstream key was verified but its group rate was unavailable"
	}
	return current, false
}

// AccountStatuses projects persisted upstream-management data onto runtime
// accounts. It never performs remote I/O; periodic refresh owns that work.
func (h *UpstreamHandler) AccountStatuses(c *gin.Context) {
	accountIDs, err := parseUpstreamSub2APIAccountIDs(c.Query("account_ids"))
	if err != nil {
		response.BadRequest(c, "Invalid account_ids")
		return
	}
	if len(accountIDs) == 0 {
		response.BadRequest(c, "account_ids is required")
		return
	}
	if len(accountIDs) > managedAccountStatusMaxIDs {
		response.BadRequest(c, "account_ids exceeds the maximum of 500")
		return
	}
	statuses, err := h.managedAccountStatuses(c.Request.Context(), accountIDs)
	if err != nil {
		response.InternalError(c, "Failed to load upstream-managed account rates")
		return
	}
	response.Success(c, statuses)
}

// RefreshAccountStatuses performs a lightweight refresh for the distinct
// upstreams that own the requested accounts, then returns their persisted view.
func (h *UpstreamHandler) RefreshAccountStatuses(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, managedAccountStatusMaxBody)
	var req refreshManagedUpstreamAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	for _, accountID := range req.AccountIDs {
		if accountID <= 0 {
			response.BadRequest(c, "Invalid account_ids")
			return
		}
	}
	accountIDs := dedupePositiveInt64(req.AccountIDs)
	if len(accountIDs) == 0 {
		response.BadRequest(c, "account_ids is required")
		return
	}
	if len(req.AccountIDs) > managedAccountStatusMaxIDs || len(accountIDs) > managedAccountStatusMaxIDs {
		response.BadRequest(c, "account_ids exceeds the maximum of 500")
		return
	}
	requestCtx := c.Request.Context()
	accounts, err := h.client.Account.Query().
		Where(entaccount.IDIn(accountIDs...)).
		All(requestCtx)
	if err != nil {
		response.InternalError(c, "Failed to load upstream bindings")
		return
	}
	upstreamIDs := make([]int64, 0, len(accounts))
	seen := make(map[int64]struct{}, len(accounts))
	for _, account := range accounts {
		if account.UpstreamID == nil {
			continue
		}
		if _, exists := seen[*account.UpstreamID]; exists {
			continue
		}
		seen[*account.UpstreamID] = struct{}{}
		upstreamIDs = append(upstreamIDs, *account.UpstreamID)
	}
	sort.Slice(upstreamIDs, func(i, j int) bool { return upstreamIDs[i] < upstreamIDs[j] })

	sem := make(chan struct{}, upstreamRefreshConcurrency)
	var wg sync.WaitGroup
	for _, upstreamID := range upstreamIDs {
		id := upstreamID
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-requestCtx.Done():
				return
			}
			probeCtx, cancel := context.WithTimeout(requestCtx, upstreamRefreshProbeTimeout)
			defer cancel()
			if _, refreshErr := h.refreshUpstreamByID(probeCtx, id, false); refreshErr != nil {
				logger.LegacyPrintf("handler.upstream_refresh", "manual refresh failed: upstream_id=%d err=%v", id, refreshErr)
			}
		}()
	}
	wg.Wait()

	statuses, err := h.managedAccountStatuses(requestCtx, accountIDs)
	if err != nil {
		response.InternalError(c, "Failed to load refreshed upstream account rates")
		return
	}
	response.Success(c, statuses)
}

func (h *UpstreamHandler) managedAccountStatuses(ctx context.Context, accountIDs []int64) ([]UpstreamSub2APIAccountStatus, error) {
	accounts, err := h.client.Account.Query().
		Where(entaccount.IDIn(accountIDs...)).
		WithUpstream().
		All(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]*dbent.Account, len(accounts))
	for _, account := range accounts {
		byID[account.ID] = account
	}
	result := make([]UpstreamSub2APIAccountStatus, 0, len(accounts))
	for _, accountID := range accountIDs {
		account := byID[accountID]
		if account == nil {
			continue
		}
		result = append(result, managedAccountStatus(account))
	}
	return result, nil
}

func managedAccountStatus(account *dbent.Account) UpstreamSub2APIAccountStatus {
	now := time.Now().UTC()
	result := UpstreamSub2APIAccountStatus{
		AccountID:     account.ID,
		AccountName:   account.Name,
		LocalPlatform: account.Platform,
		Status:        "upstream_unbound",
		Message:       "Account is not bound to upstream management",
		FetchedAt:     now,
		Cached:        true,
	}
	item := account.Edges.Upstream
	if item == nil {
		return result
	}
	result.BaseURL = item.BaseURL
	result.UpstreamKind = item.Kind.String()
	result.ProbeSource = "upstream_manager"
	metadata, err := parseUpstreamProbeMetadata(item.Metadata)
	if err != nil {
		result.Status = "invalid_metadata"
		result.Message = "Stored upstream metadata could not be decoded"
		return result
	}
	if metadata.Wallet != nil {
		result.UserBalance = metadata.Wallet.Balance
		result.BalanceUnit = metadata.Wallet.Unit
	}
	billing, ok := metadata.AccountBilling[strconv.FormatInt(account.ID, 10)]
	if !ok {
		result.Status = "not_probed"
		result.Message = "Upstream management has not resolved this account key yet"
		if metadata.Refresh != nil {
			result.FetchedAt = metadata.Refresh.LastAttemptAt
			result.Stale = metadata.Refresh.Stale
		}
		return result
	}
	result.Status = billing.Status
	result.Message = billing.Message
	result.FetchedAt = billing.FetchedAt
	result.Stale = billing.Stale || metadata.ManagementStatus != "ok"
	result.KeyRemaining = billing.KeyRemaining
	if billing.BalanceUnit != "" {
		result.BalanceUnit = billing.BalanceUnit
	}
	result.UsageMode = billing.UsageMode
	result.UsagePlanName = billing.UsagePlanName
	result.UpstreamKeyID = billing.UpstreamKeyID
	result.UpstreamKeyName = billing.UpstreamKeyName
	result.UpstreamGroupID = billing.UpstreamGroupID
	result.UpstreamGroupName = billing.UpstreamGroupName
	result.UpstreamGroupPlatform = billing.UpstreamGroupPlatform
	result.UpstreamGroupDefaultRateMultiplier = billing.GroupDefaultRateMultiplier
	result.UpstreamGroupEffectiveRateMultiplier = billing.GroupEffectiveRateMultiplier
	return result
}
