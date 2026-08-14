package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// AccountMonitorRepository 账号监控持久化接口（由 repository 层实现）。
type AccountMonitorRepository interface {
	Create(ctx context.Context, m *AccountMonitor) error
	Update(ctx context.Context, m *AccountMonitor) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*AccountMonitor, error)
	GetByAccountID(ctx context.Context, accountID int64) (*AccountMonitor, error)
	List(ctx context.Context) ([]*AccountMonitor, error)
	ListEnabled(ctx context.Context) ([]*AccountMonitor, error)
	UpdateLastCheckedAt(ctx context.Context, id int64, at time.Time) error

	InsertChecks(ctx context.Context, checks []*AccountMonitorCheck) error
	// LatestChecks 返回每个 monitorID 最近一次探测（按 model 取主记录即可，这里取最近一条）。
	LatestChecks(ctx context.Context, monitorIDs []int64) (map[int64]*AccountMonitorCheck, error)
	// Availability1h 返回每个 monitorID 近 1 小时的可用率（百分比）。
	Availability1h(ctx context.Context, monitorIDs []int64) (map[int64]float64, error)
	// AvgLatency1h 返回每个 monitorID 近 1 小时的平均探测响应耗时（ms）。无样本的 monitorID 不在结果里。
	AvgLatency1h(ctx context.Context, monitorIDs []int64) (map[int64]float64, error)
	// RecentChecks 返回每个 monitorID 最近 limit 条探测（newest-first），供彩虹条渲染。
	RecentChecks(ctx context.Context, monitorIDs []int64, limit int) (map[int64][]*AccountMonitorCheck, error)
	DeleteChecksOlderThan(ctx context.Context, cutoff time.Time) (int, error)
}

// accountMonitorTimelinePoints 彩虹条返回的最近探测点数（与渠道监控用户视图一致）。
const accountMonitorTimelinePoints = 60

// accountMonitorAccountReader 读取被监控账号的最小接口（由 *AccountService 满足）。
type accountMonitorAccountReader interface {
	GetByID(ctx context.Context, id int64) (*Account, error)
}

type AccountMonitorRecovery interface {
	RecoverAccountAfterSuccessfulTest(ctx context.Context, accountID int64) (*SuccessfulTestRecoveryResult, error)
}

type accountMonitorModelRecovery interface {
	RecoverAccountModelAfterSuccessfulTest(ctx context.Context, accountID int64, model string) (*SuccessfulTestRecoveryResult, error)
}

type AccountMonitorFailureBlocker interface {
	SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error
	SetModelRateLimit(ctx context.Context, id int64, scope string, resetAt time.Time, reason ...string) error
}

// AccountMonitorService 账号监控管理服务（CRUD + RunCheck + 聚合）。
type AccountMonitorService struct {
	repo     AccountMonitorRepository
	account  accountMonitorAccountReader
	recovery AccountMonitorRecovery
	blocker  AccountMonitorFailureBlocker

	// scheduler 由 wire 通过 SetScheduler 注入；CRUD 后回调即时同步任务。nil 时为 no-op。
	scheduler AccountMonitorScheduler
}

// AccountMonitorScheduler 调度器接口，供 service 在 CRUD 时回调（setter 注入避免依赖环）。
type AccountMonitorScheduler interface {
	Schedule(m *AccountMonitor)
	Unschedule(id int64)
}

// NewAccountMonitorService 创建账号监控服务。
func NewAccountMonitorService(repo AccountMonitorRepository, account accountMonitorAccountReader) *AccountMonitorService {
	return &AccountMonitorService{repo: repo, account: account}
}

func (s *AccountMonitorService) SetRecovery(recovery AccountMonitorRecovery) {
	s.recovery = recovery
}

func (s *AccountMonitorService) SetFailureBlocker(blocker AccountMonitorFailureBlocker) {
	s.blocker = blocker
}

// SetScheduler 注入调度器（wire 在 runner 创建后回调）。
func (s *AccountMonitorService) SetScheduler(sched AccountMonitorScheduler) {
	s.scheduler = sched
}

// ---------- CRUD ----------

// Create 为一个 api_key 类账号创建监控。account_id 唯一，重复创建报错。
func (s *AccountMonitorService) Create(ctx context.Context, p AccountMonitorCreateParams) (*AccountMonitor, error) {
	if p.AccountID <= 0 {
		return nil, ErrAccountMonitorInvalidAccountID
	}
	// 创建监控时仍要求账号当前可用，避免给废弃/停用账号创建长期任务。
	acc, err := s.eligibleAccountForCreate(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	if existing, _ := s.repo.GetByAccountID(ctx, p.AccountID); existing != nil {
		return nil, ErrAccountMonitorExists
	}

	m := &AccountMonitor{
		AccountID:       p.AccountID,
		Provider:        firstNonEmpty(strings.TrimSpace(p.Provider), inferProviderFromAccount(acc)),
		Model:           firstNonEmpty(strings.TrimSpace(p.Model), accountMonitorDefaultModel),
		Enabled:         p.Enabled,
		IntervalSeconds: orDefaultInterval(p.IntervalSeconds),
		JitterSeconds:   p.JitterSeconds,
		CreatedBy:       p.CreatedBy,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("create account monitor: %w", err)
	}
	if s.scheduler != nil {
		s.scheduler.Schedule(m)
	}
	return m, nil
}

// Update 更新监控配置（指针字段为 nil 则不改）。
func (s *AccountMonitorService) Update(ctx context.Context, id int64, p AccountMonitorUpdateParams) (*AccountMonitor, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Provider != nil {
		m.Provider = strings.TrimSpace(*p.Provider)
	}
	if p.Model != nil {
		m.Model = firstNonEmpty(strings.TrimSpace(*p.Model), accountMonitorDefaultModel)
	}
	if p.Enabled != nil {
		m.Enabled = *p.Enabled
	}
	if p.IntervalSeconds != nil {
		m.IntervalSeconds = orDefaultInterval(*p.IntervalSeconds)
	}
	if p.JitterSeconds != nil {
		m.JitterSeconds = *p.JitterSeconds
	}
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, fmt.Errorf("update account monitor: %w", err)
	}
	if s.scheduler != nil {
		// Schedule 内部据 Enabled 自动选择 Unschedule 或重建任务。
		s.scheduler.Schedule(m)
	}
	return m, nil
}

// Delete 删除监控（及其历史由 DB CASCADE 清除）。
func (s *AccountMonitorService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.scheduler != nil {
		s.scheduler.Unschedule(id)
	}
	return nil
}

// Get 查询单个监控。
func (s *AccountMonitorService) Get(ctx context.Context, id int64) (*AccountMonitor, error) {
	return s.repo.GetByID(ctx, id)
}

// List 列出所有监控。
func (s *AccountMonitorService) List(ctx context.Context) ([]*AccountMonitor, error) {
	return s.repo.List(ctx)
}

// ListEnabledMonitors 供 runner 启动时建立任务表。
func (s *AccountMonitorService) ListEnabledMonitors(ctx context.Context) ([]*AccountMonitor, error) {
	return s.repo.ListEnabled(ctx)
}

// ---------- 探测 ----------

// RunCheck 执行一次探测：实时读账号凭证 → 账号监控轻量探测 → 落库。
// 返回的 error 仅表示"无法探测"（账号不合格等）；探测本身失败会作为一条 check 记录落库。
func (s *AccountMonitorService) RunCheck(ctx context.Context, id int64) (*AccountMonitorCheck, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	acc, eligErr := s.eligibleAccountForProbe(ctx, m.AccountID)
	if eligErr != nil {
		// 账号不合格：记一条 error，便于 admin 在 UI 看到原因。
		errCheck := s.buildErrorCheck(m, eligErr.Error())
		s.persistChecks(ctx, m, []*AccountMonitorCheck{errCheck})
		return errCheck, eligErr
	}

	endpoint := resolveAccountEndpoint(acc)
	if verr := validateAccountMonitorEndpoint(endpoint); verr != nil {
		errCheck := s.buildErrorCheck(m, "invalid account base_url: "+verr.Error())
		s.persistChecks(ctx, m, []*AccountMonitorCheck{errCheck})
		return errCheck, verr
	}

	apiKey := strings.TrimSpace(acc.GetCredential("api_key"))
	pingMs := pingEndpointOrigin(ctx, endpoint)
	res := runAccountMonitorCheckForModel(ctx, m.Provider, endpoint, apiKey, m.Model)
	res.PingLatencyMs = pingMs

	check := &AccountMonitorCheck{
		AccountMonitorID: m.ID,
		Model:            m.Model,
		Status:           res.Status,
		LatencyMs:        res.LatencyMs,
		PingLatencyMs:    res.PingLatencyMs,
		Message:          res.Message,
		CheckedAt:        res.CheckedAt,
	}
	s.persistChecks(ctx, m, []*AccountMonitorCheck{check})
	s.recoverAccountAfterSuccessfulMonitor(ctx, acc, check)
	s.blockAccountAfterConsecutiveMonitorFailures(ctx, m, acc, check)
	return check, nil
}

// ---------- 聚合（admin 视图）----------

// StatusByAccountID 返回所有监控的聚合状态，按 account_id 索引（前端 SchedulerView 按行匹配）。
func (s *AccountMonitorService) StatusByAccountID(ctx context.Context) (map[int64]*AccountMonitorStatus, error) {
	monitors, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]*AccountMonitorStatus, len(monitors))
	if len(monitors) == 0 {
		return out, nil
	}
	ids := make([]int64, 0, len(monitors))
	for _, m := range monitors {
		ids = append(ids, m.ID)
	}
	// timeline（newest-first）一次取出，其首元素即"最近一次"，省掉单独的 LatestChecks 查询。
	timelines, _ := s.repo.RecentChecks(ctx, ids, accountMonitorTimelinePoints)
	avail, _ := s.repo.Availability1h(ctx, ids)
	avgLatency, _ := s.repo.AvgLatency1h(ctx, ids)

	for _, m := range monitors {
		tl := timelines[m.ID]
		st := &AccountMonitorStatus{
			MonitorID:       m.ID,
			AccountID:       m.AccountID,
			Model:           m.Model,
			Enabled:         m.Enabled,
			IntervalSeconds: m.IntervalSeconds,
			Availability1h:  avail[m.ID],
			LastCheckedAt:   m.LastCheckedAt,
			Timeline:        tl,
		}
		if v, ok := avgLatency[m.ID]; ok {
			st.AvgLatency1h = &v
		}
		if len(tl) > 0 {
			latest := tl[0]
			st.LatestStatus = latest.Status
			st.LatestLatency = latest.LatencyMs
			st.PingLatencyMs = latest.PingLatencyMs
		}
		out[m.AccountID] = st
	}
	return out, nil
}

// ---------- 内部 helper ----------

// eligibleAccountForCreate 读取并校验账号：创建监控时必须是 active 的 api_key 类账号且有 api_key 凭证。
func (s *AccountMonitorService) eligibleAccountForCreate(ctx context.Context, accountID int64) (*Account, error) {
	acc, err := s.eligibleAccountForProbe(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if !acc.IsActive() {
		return nil, ErrAccountMonitorNotEligible
	}
	return acc, nil
}

// eligibleAccountForProbe 读取并校验探针必要条件。
// 定时探针必须无视本机调度状态/status=error，否则账号一旦被本机标错就只能手工复位。
func (s *AccountMonitorService) eligibleAccountForProbe(ctx context.Context, accountID int64) (*Account, error) {
	acc, err := s.account.GetByID(ctx, accountID)
	if err != nil {
		return nil, ErrAccountMonitorNotEligible
	}
	if acc == nil || acc.Type != AccountTypeAPIKey {
		return nil, ErrAccountMonitorNotEligible
	}
	if strings.TrimSpace(acc.GetCredential("api_key")) == "" {
		return nil, ErrAccountMonitorNotEligible
	}
	return acc, nil
}

func (s *AccountMonitorService) recoverAccountAfterSuccessfulMonitor(ctx context.Context, acc *Account, check *AccountMonitorCheck) {
	if s == nil || s.recovery == nil || acc == nil || check == nil {
		return
	}
	switch check.Status {
	case MonitorStatusOperational, MonitorStatusDegraded:
	default:
		return
	}
	if acc.IsOpenAIApiKey() && strings.TrimSpace(check.Model) != "" {
		if recovery, ok := s.recovery.(accountMonitorModelRecovery); ok {
			if _, err := recovery.RecoverAccountModelAfterSuccessfulTest(ctx, acc.ID, check.Model); err == nil {
				return
			}
		}
	}
	if _, err := s.recovery.RecoverAccountAfterSuccessfulTest(ctx, acc.ID); err != nil {
		// 探针恢复是 best-effort；失败不影响监控记录。
		return
	}
}

const (
	accountMonitorFailureBlockThreshold       = 3
	accountMonitorFailureBlockCooldownMinutes = 5
	accountMonitorHardFailureCooldownHours    = 24
)

func (s *AccountMonitorService) blockAccountAfterConsecutiveMonitorFailures(ctx context.Context, m *AccountMonitor, acc *Account, check *AccountMonitorCheck) {
	if s == nil || s.blocker == nil || s.repo == nil || m == nil || acc == nil || check == nil {
		return
	}
	if !isAccountMonitorBlockableStatus(check.Status) {
		return
	}
	recent, err := s.repo.RecentChecks(ctx, []int64{m.ID}, accountMonitorFailureBlockThreshold)
	if err != nil {
		return
	}
	checks := recent[m.ID]
	if len(checks) < accountMonitorFailureBlockThreshold {
		return
	}
	for _, c := range checks {
		if c == nil || !isAccountMonitorBlockableStatus(c.Status) {
			return
		}
	}
	blockAcc := s.latestAccountForMonitorBlock(ctx, acc)
	if s.blockModelAfterMonitorModelUnsupported(ctx, m, blockAcc, check) {
		return
	}
	classification := classifyAccountMonitorFailure(check.Message)
	reason := classification.Reason
	if reason == "" {
		reason = groupReserveReasonMonitor
	}
	if strings.TrimSpace(check.Message) != "" {
		reason += ": " + truncateMessage(check.Message)
	}
	cooldown := time.Duration(accountMonitorFailureBlockCooldownMinutes) * time.Minute
	if classification.Hard {
		cooldown = time.Duration(accountMonitorHardFailureCooldownHours) * time.Hour
	}
	until := time.Now().Add(cooldown)
	applied, skipped := s.applyMonitorFailureBlock(ctx, blockAcc, until, reason, m, check, classification)
	if skipped && !applied {
		return
	}
}

func (s *AccountMonitorService) latestAccountForMonitorBlock(ctx context.Context, acc *Account) *Account {
	if s == nil || s.account == nil || acc == nil || acc.ID <= 0 {
		return acc
	}
	fresh, err := s.account.GetByID(ctx, acc.ID)
	if err != nil || fresh == nil {
		return acc
	}
	return fresh
}

type accountMonitorFailureClassification struct {
	Hard            bool
	Reason          string
	CooldownMinutes int
}

func classifyAccountMonitorFailure(message string) accountMonitorFailureClassification {
	normalized := strings.ToLower(strings.TrimSpace(message))
	normalized = strings.NewReplacer("_", " ", "-", " ", "\n", " ", "\r", " ", "\t", " ").Replace(normalized)
	normalized = strings.Join(strings.Fields(normalized), " ")
	code := monitorStatusCodeFromMessage(message)
	hard := func(reason string) accountMonitorFailureClassification {
		return accountMonitorFailureClassification{
			Hard:            true,
			Reason:          reason,
			CooldownMinutes: accountMonitorHardFailureCooldownHours * 60,
		}
	}
	if code == 401 {
		return hard("account_monitor_auth_failed")
	}
	// A 410 endpoint-migrated response is a permanent provider configuration
	// failure, not a transient outage. Keeping it on a five-minute cooldown
	// makes the monitor re-enable the same unusable credential repeatedly and
	// can send users back to it when the cooldown expires.
	if code == 410 || strings.Contains(normalized, "endpoint migrated") ||
		strings.Contains(normalized, "endpoint is not served") {
		return hard("account_monitor_endpoint_migrated")
	}
	if code == 403 {
		switch {
		case strings.Contains(normalized, "insufficient balance"),
			strings.Contains(normalized, "insufficient account balance"),
			strings.Contains(normalized, "insufficient user quota"),
			strings.Contains(normalized, "quota insufficient"),
			strings.Contains(normalized, "余额不足"),
			strings.Contains(normalized, "额度不足"),
			strings.Contains(normalized, "剩余额度"):
			return hard("account_monitor_insufficient_balance")
		case strings.Contains(normalized, "group deleted"),
			strings.Contains(normalized, "group disabled"),
			strings.Contains(normalized, "所属分组已删除"),
			strings.Contains(normalized, "所属分组已停用"):
			return hard("account_monitor_upstream_group_unavailable")
		case strings.Contains(normalized, "invalid api key"),
			strings.Contains(normalized, "api key invalid"),
			strings.Contains(normalized, "unauthorized"),
			strings.Contains(normalized, "forbidden"),
			strings.Contains(normalized, "无效"):
			return hard("account_monitor_auth_failed")
		}
	}
	return accountMonitorFailureClassification{
		Reason:          "account_monitor_consecutive_failures",
		CooldownMinutes: accountMonitorFailureBlockCooldownMinutes,
	}
}

func isAccountMonitorBlockableStatus(status string) bool {
	switch status {
	case MonitorStatusError, MonitorStatusFailed:
		return true
	default:
		return false
	}
}

func (s *AccountMonitorService) blockModelAfterMonitorModelUnsupported(ctx context.Context, m *AccountMonitor, acc *Account, check *AccountMonitorCheck) bool {
	if s == nil || s.blocker == nil || m == nil || acc == nil || check == nil {
		return false
	}
	if !isUpstreamModelNotFoundError(monitorStatusCodeFromMessage(check.Message), []byte(check.Message)) {
		return false
	}
	modelKey := modelRateLimitKeyForUpstreamModelNotFound(ctx, acc, m.Model)
	if modelKey == "" {
		return false
	}
	alreadyBlocked := acc.isRateLimitActiveForKey(modelKey)
	until := time.Now().Add(upstreamModelNotFoundCooldown)
	if err := s.blocker.SetModelRateLimit(ctx, acc.ID, modelKey, until, upstreamModelNotFoundReason); err != nil {
		return true
	}
	if alreadyBlocked {
		return true
	}
	recordSchedulerBlocked(ctx, s.blocker, acc.ID, firstAccountGroupID(ctx, acc), 0, "account_monitor_model_unsupported", "account_monitor", until, map[string]any{
		"monitor_id":        m.ID,
		"model":             m.Model,
		"model_rate_limit":  modelKey,
		"latest_status":     check.Status,
		"latest_message":    truncateMessage(check.Message),
		"cooldown_minutes":  int(upstreamModelNotFoundCooldown / time.Minute),
		"failure_category":  "account_monitor_model_unsupported",
		"block_granularity": "model",
	})
	return true
}

func monitorStatusCodeFromMessage(message string) int {
	message = strings.ToLower(strings.TrimSpace(message))
	if message == "" {
		return 0
	}
	const marker = "http "
	idx := strings.Index(message, marker)
	if idx < 0 {
		return 0
	}
	start := idx + len(marker)
	end := start
	for end < len(message) && message[end] >= '0' && message[end] <= '9' {
		end++
	}
	if end == start {
		return 0
	}
	code := 0
	for _, ch := range message[start:end] {
		code = code*10 + int(ch-'0')
	}
	return code
}

func (s *AccountMonitorService) applyMonitorFailureBlock(ctx context.Context, acc *Account, until time.Time, reason string, m *AccountMonitor, check *AccountMonitorCheck, classification accountMonitorFailureClassification) (bool, bool) {
	if s == nil || s.blocker == nil || acc == nil {
		return false, false
	}
	monitorID := int64(0)
	model := ""
	latestStatus := ""
	latestMessage := ""
	if m != nil {
		monitorID = m.ID
		model = m.Model
	}
	if check != nil {
		latestStatus = check.Status
		latestMessage = truncateMessage(check.Message)
	}
	if !classification.Hard && acc.IsOpenAIApiKey() && strings.TrimSpace(model) != "" {
		modelKey := modelRateLimitKeyForUpstreamModelNotFound(ctx, acc, model)
		if modelKey != "" {
			if acc.isRateLimitActiveForKey(modelKey) {
				return true, false
			}
			if err := s.blocker.SetModelRateLimit(ctx, acc.ID, modelKey, until, reason); err != nil {
				return false, false
			}
			extra := map[string]any{
				"monitor_id":        monitorID,
				"model":             model,
				"model_rate_limit":  modelKey,
				"latest_status":     latestStatus,
				"latest_message":    latestMessage,
				"failure_threshold": accountMonitorFailureBlockThreshold,
				"cooldown_minutes":  classification.CooldownMinutes,
				"failure_category":  classification.Reason,
				"block_granularity": "model",
			}
			recordSchedulerBlocked(ctx, s.blocker, acc.ID, firstAccountGroupID(ctx, acc), 0, classification.Reason, "account_monitor", until, extra)
			return true, false
		}
	}
	if acc.TempUnschedulableUntil != nil && time.Now().Before(*acc.TempUnschedulableUntil) {
		// A monitor run every minute must not keep extending one incident or flood
		// the scheduler history while the account is already in cooldown.
		return true, false
	}
	extra := map[string]any{
		"monitor_id":        monitorID,
		"model":             model,
		"latest_status":     latestStatus,
		"latest_message":    latestMessage,
		"failure_threshold": accountMonitorFailureBlockThreshold,
		"cooldown_minutes":  classification.CooldownMinutes,
		"failure_category":  classification.Reason,
		"block_granularity": "account",
	}
	if err := s.blocker.SetTempUnschedulable(ctx, acc.ID, until, reason); err != nil {
		return false, false
	}
	recordSchedulerBlocked(ctx, s.blocker, acc.ID, firstAccountGroupID(ctx, acc), 0, classification.Reason, "account_monitor", until, extra)
	return true, false
}

func firstAccountGroupID(ctx context.Context, acc *Account) int64 {
	if groupIDs := schedulingProtectionGroupIDs(ctx, acc); len(groupIDs) > 0 {
		return groupIDs[0]
	}
	return 0
}

// buildErrorCheck 构造一条 error 状态的探测记录。
func (s *AccountMonitorService) buildErrorCheck(m *AccountMonitor, msg string) *AccountMonitorCheck {
	return &AccountMonitorCheck{
		AccountMonitorID: m.ID,
		Model:            m.Model,
		Status:           MonitorStatusError,
		Message:          truncateMessage(sanitizeErrorMessage(msg)),
		CheckedAt:        time.Now(),
	}
}

// persistChecks 落库探测记录并更新 last_checked_at（失败仅吞错，探测是 best-effort）。
func (s *AccountMonitorService) persistChecks(ctx context.Context, m *AccountMonitor, checks []*AccountMonitorCheck) {
	if len(checks) == 0 {
		return
	}
	_ = s.repo.InsertChecks(ctx, checks)
	_ = s.repo.UpdateLastCheckedAt(ctx, m.ID, checks[len(checks)-1].CheckedAt)
}

// resolveAccountEndpoint 取账号 base_url，缺省回退默认 OpenAI endpoint，并归一化。
func resolveAccountEndpoint(acc *Account) string {
	base := strings.TrimSpace(acc.GetCredential("base_url"))
	if base == "" {
		base = accountMonitorDefaultBaseURL
	}
	return normalizeEndpoint(base)
}

// validateAccountMonitorEndpoint 是账号监控专用的端点校验，比渠道监控的 validateEndpoint
// 放宽：**允许 http 和 https**（账号 base_url 是管理员自配的可信上游，部分上游仅 http）。
// 仍保留：必须是 origin（无 path/query/fragment）、SSRF 私网/loopback 拦截。
// 明文凭证风险由"仅管理员可配账号 + 上游可信"承担；SSRF 兜底仍由 checker 的 safeDialContext 提供。
func validateAccountMonitorEndpoint(ep string) error {
	ep = strings.TrimSpace(ep)
	if ep == "" {
		return ErrChannelMonitorInvalidEndpoint
	}
	u, err := url.Parse(ep)
	if err != nil {
		return ErrChannelMonitorInvalidEndpoint
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrChannelMonitorEndpointScheme
	}
	if u.Host == "" {
		return ErrChannelMonitorInvalidEndpoint
	}
	if u.Path != "" && u.Path != "/" {
		return ErrChannelMonitorEndpointPath
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return ErrChannelMonitorEndpointPath
	}
	hostname := u.Hostname()
	ctx, cancel := context.WithTimeout(context.Background(), monitorEndpointResolveTimeout)
	defer cancel()
	blocked, err := isPrivateOrLoopbackHost(ctx, hostname)
	if err != nil {
		return ErrChannelMonitorEndpointUnreachable
	}
	if blocked {
		return ErrChannelMonitorEndpointPrivate
	}
	return nil
}

// inferProviderFromAccount 据账号 platform 推断 provider，未知回退 openai。
func inferProviderFromAccount(acc *Account) string {
	switch strings.ToLower(strings.TrimSpace(acc.Platform)) {
	case "anthropic", "claude":
		return MonitorProviderAnthropic
	case "gemini", "google":
		return MonitorProviderGemini
	case PlatformGrok:
		return MonitorProviderGrok
	default:
		return MonitorProviderOpenAI
	}
}

func orDefaultInterval(v int) int {
	if v <= 0 {
		return accountMonitorDefaultInterval
	}
	return v
}
