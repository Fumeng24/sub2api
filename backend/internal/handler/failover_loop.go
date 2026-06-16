package handler

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"
)

// TempUnscheduler 用于 HandleFailoverError 中同账号重试耗尽后的临时封禁。
// GatewayService 隐式实现此接口。
type TempUnscheduler interface {
	TempUnscheduleRetryableError(ctx context.Context, accountID int64, failoverErr *service.UpstreamFailoverError)
}

type AccountScheduleFailureReporter interface {
	ReportAccountScheduleFailure(ctx context.Context, accountID int64, model, endpoint string, failoverErr *service.UpstreamFailoverError)
}

// UpstreamClientError describes the client-facing result after all upstream
// account failover candidates are exhausted. It intentionally models provider
// account failures, not local user/API-key quota failures.
type UpstreamClientError struct {
	Status  int
	Type    string
	Message string
}

type upstreamFailoverClientPolicy struct {
	ForbiddenStatus int
	ForbiddenType   string
	OverloadedType  string
}

// FailoverAction 表示 failover 错误处理后的下一步动作
type FailoverAction int

const (
	// FailoverContinue 继续循环（同账号重试或切换账号，调用方统一 continue）
	FailoverContinue FailoverAction = iota
	// FailoverExhausted 切换次数耗尽（调用方应返回错误响应）
	FailoverExhausted
	// FailoverCanceled context 已取消（调用方应直接 return）
	FailoverCanceled
)

const (
	// maxSameAccountRetries 同账号重试次数上限（针对 RetryableOnSameAccount 错误）
	maxSameAccountRetries = 3
	// sameAccountRetryDelay 同账号重试间隔
	sameAccountRetryDelay = 500 * time.Millisecond
	// singleAccountBackoffDelay 单账号分组 503 退避重试固定延时。
	// Service 层在 SingleAccountRetry 模式下已做充分原地重试（最多 3 次、总等待 30s），
	// Handler 层只需短暂间隔后重新进入 Service 层即可。
	singleAccountBackoffDelay = 2 * time.Second
)

// FailoverState 跨循环迭代共享的 failover 状态
type FailoverState struct {
	SwitchCount           int
	MaxSwitches           int
	FailedAccountIDs      map[int64]struct{}
	SameAccountRetryCount map[int64]int
	LastFailoverErr       *service.UpstreamFailoverError
	ForceCacheBilling     bool
	hasBoundSession       bool
}

// NewFailoverState 创建 failover 状态
func NewFailoverState(maxSwitches int, hasBoundSession bool) *FailoverState {
	return &FailoverState{
		MaxSwitches:           maxSwitches,
		FailedAccountIDs:      make(map[int64]struct{}),
		SameAccountRetryCount: make(map[int64]int),
		hasBoundSession:       hasBoundSession,
	}
}

// HandleFailoverError 处理 UpstreamFailoverError，返回下一步动作。
// 包含：缓存计费判断、同账号重试、临时封禁、切换计数、Antigravity 延时。
func (s *FailoverState) HandleFailoverError(
	ctx context.Context,
	gatewayService TempUnscheduler,
	accountID int64,
	platform string,
	failoverErr *service.UpstreamFailoverError,
) FailoverAction {
	return s.HandleFailoverErrorForRequest(ctx, gatewayService, accountID, platform, "", "", failoverErr)
}

func (s *FailoverState) HandleFailoverErrorForRequest(
	ctx context.Context,
	gatewayService TempUnscheduler,
	accountID int64,
	platform string,
	model string,
	endpoint string,
	failoverErr *service.UpstreamFailoverError,
	extraFields ...zap.Field,
) FailoverAction {
	s.LastFailoverErr = failoverErr
	if reporter, ok := gatewayService.(AccountScheduleFailureReporter); ok {
		reporter.ReportAccountScheduleFailure(ctx, accountID, model, endpoint, failoverErr)
	}

	// 缓存计费判断
	if needForceCacheBilling(s.hasBoundSession, failoverErr) {
		s.ForceCacheBilling = true
	}

	// 同账号重试：对 RetryableOnSameAccount 的临时性错误，先在同一账号上重试
	if failoverErr.RetryableOnSameAccount && s.SameAccountRetryCount[accountID] < maxSameAccountRetries {
		s.SameAccountRetryCount[accountID]++
		fields := s.failoverLogFields(accountID, model, endpoint, failoverErr, extraFields...)
		fields = append(fields,
			zap.Int("same_account_retry_count", s.SameAccountRetryCount[accountID]),
			zap.Int("same_account_retry_max", maxSameAccountRetries),
		)
		logger.FromContext(ctx).Warn("gateway.failover_same_account_retry", fields...)
		if !sleepWithContext(ctx, sameAccountRetryDelay) {
			return FailoverCanceled
		}
		return FailoverContinue
	}

	// 同账号重试用尽，执行临时封禁
	if failoverErr.RetryableOnSameAccount {
		gatewayService.TempUnscheduleRetryableError(ctx, accountID, failoverErr)
	}

	// 加入失败列表
	s.FailedAccountIDs[accountID] = struct{}{}

	// 检查是否耗尽
	if s.SwitchCount >= s.MaxSwitches {
		return FailoverExhausted
	}

	// 递增切换计数
	s.SwitchCount++
	fields := s.failoverLogFields(accountID, model, endpoint, failoverErr, extraFields...)
	fields = append(fields,
		zap.Int("switch_count", s.SwitchCount),
		zap.Int("max_switches", s.MaxSwitches),
	)
	logger.FromContext(ctx).Warn("gateway.failover_switch_account", fields...)

	// Antigravity 平台换号线性递增延时
	if platform == service.PlatformAntigravity {
		delay := time.Duration(s.SwitchCount-1) * time.Second
		if !sleepWithContext(ctx, delay) {
			return FailoverCanceled
		}
	}

	return FailoverContinue
}

// HandleSelectionExhausted 处理选号失败（所有候选账号都在排除列表中）时的退避重试决策。
// 针对 Antigravity 单账号分组的 503 (MODEL_CAPACITY_EXHAUSTED) 场景：
// 清除排除列表、等待退避后重新选号。
//
// 返回 FailoverContinue 时，调用方应设置 SingleAccountRetry context 并 continue。
// 返回 FailoverExhausted 时，调用方应返回错误响应。
// 返回 FailoverCanceled 时，调用方应直接 return。
func (s *FailoverState) HandleSelectionExhausted(ctx context.Context) FailoverAction {
	if s.LastFailoverErr != nil &&
		s.LastFailoverErr.StatusCode == http.StatusServiceUnavailable &&
		s.SwitchCount <= s.MaxSwitches {

		logger.FromContext(ctx).Warn("gateway.failover_single_account_backoff",
			zap.Int64s("tried_accounts", s.FailedAccountIDList()),
			zap.Int("upstream_status", s.LastFailoverErr.StatusCode),
			zap.Duration("backoff_delay", singleAccountBackoffDelay),
			zap.Int("switch_count", s.SwitchCount),
			zap.Int("max_switches", s.MaxSwitches),
		)
		if !sleepWithContext(ctx, singleAccountBackoffDelay) {
			return FailoverCanceled
		}
		logger.FromContext(ctx).Warn("gateway.failover_single_account_retry",
			zap.Int64s("tried_accounts", s.FailedAccountIDList()),
			zap.Int("upstream_status", s.LastFailoverErr.StatusCode),
			zap.Int("switch_count", s.SwitchCount),
			zap.Int("max_switches", s.MaxSwitches),
		)
		s.FailedAccountIDs = make(map[int64]struct{})
		return FailoverContinue
	}
	return FailoverExhausted
}

func (s *FailoverState) FailedAccountIDList() []int64 {
	if s == nil {
		return nil
	}
	return sortedInt64SetKeys(s.FailedAccountIDs)
}

func (s *FailoverState) triedAccountIDsWith(accountID int64) []int64 {
	ids := s.FailedAccountIDList()
	for _, id := range ids {
		if id == accountID {
			return ids
		}
	}
	ids = append(ids, accountID)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func (s *FailoverState) failoverLogFields(accountID int64, model, endpoint string, failoverErr *service.UpstreamFailoverError, extraFields ...zap.Field) []zap.Field {
	statusCode := 0
	if failoverErr != nil {
		statusCode = failoverErr.StatusCode
	}
	fields := []zap.Field{
		zap.Int64("account_id", accountID),
		zap.Int64("failed_account", accountID),
		zap.Int64s("tried_accounts", s.triedAccountIDsWith(accountID)),
		zap.Int("upstream_status", statusCode),
		zap.String("circuit_scope", failoverCircuitScope(model, endpoint)),
	}
	if strings.TrimSpace(model) != "" {
		fields = append(fields, zap.String("model", model))
	}
	if strings.TrimSpace(endpoint) != "" {
		fields = append(fields, zap.String("endpoint", endpoint))
	}
	fields = append(fields, extraFields...)
	return fields
}

func failoverCircuitScope(model, endpoint string) string {
	model = strings.TrimSpace(model)
	endpoint = strings.TrimSpace(endpoint)
	switch {
	case model != "" && endpoint != "":
		return "account+model+endpoint"
	case model != "":
		return "account+model"
	case endpoint != "":
		return "account+endpoint"
	default:
		return "account"
	}
}

func sortedInt64SetKeys(values map[int64]struct{}) []int64 {
	if len(values) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func failoverWriterFields(currentSize, writerSizeBeforeForward int) []zap.Field {
	bytesWritten := failoverBytesWritten(currentSize, writerSizeBeforeForward)
	return []zap.Field{
		zap.Int("bytes_written", bytesWritten),
		zap.Bool("response_started", bytesWritten > 0),
		zap.Int("writer_size_before_forward", writerSizeBeforeForward),
		zap.Int("writer_size_after_forward", currentSize),
	}
}

func failoverBytesWritten(currentSize, writerSizeBeforeForward int) int {
	if currentSize <= writerSizeBeforeForward {
		return 0
	}
	if writerSizeBeforeForward < 0 {
		return currentSize
	}
	return currentSize - writerSizeBeforeForward
}

func manualFailoverSwitchFields(
	accountID int64,
	upstreamStatus int,
	switchCount int,
	maxSwitches int,
	failedAccountIDs map[int64]struct{},
	model string,
	endpoint string,
	currentSize int,
	writerSizeBeforeForward int,
) []zap.Field {
	triedAccounts := sortedInt64SetKeys(failedAccountIDs)
	fields := []zap.Field{
		zap.Int64("account_id", accountID),
		zap.Int64("failed_account", accountID),
		zap.Int64s("tried_accounts", triedAccounts),
		zap.Int("upstream_status", upstreamStatus),
		zap.Int("switch_count", switchCount),
		zap.Int("max_switches", maxSwitches),
		zap.String("circuit_scope", failoverCircuitScope(model, endpoint)),
	}
	if model != "" {
		fields = append(fields, zap.String("model", model))
	}
	if endpoint != "" {
		fields = append(fields, zap.String("endpoint", endpoint))
	}
	fields = append(fields, failoverWriterFields(currentSize, writerSizeBeforeForward)...)
	return fields
}

func upstreamClientErrorForFailoverStatus(statusCode int) UpstreamClientError {
	return upstreamClientErrorForFailoverStatusWithPolicy(statusCode, upstreamFailoverClientPolicy{})
}

func upstreamClientErrorForFailoverStatusWithPolicy(statusCode int, policy upstreamFailoverClientPolicy) UpstreamClientError {
	forbiddenStatus := policy.ForbiddenStatus
	if forbiddenStatus == 0 {
		forbiddenStatus = http.StatusBadGateway
	}
	forbiddenType := strings.TrimSpace(policy.ForbiddenType)
	if forbiddenType == "" {
		forbiddenType = "upstream_error"
	}
	overloadedType := strings.TrimSpace(policy.OverloadedType)
	if overloadedType == "" {
		overloadedType = "overloaded_error"
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return UpstreamClientError{Status: http.StatusBadGateway, Type: "upstream_error", Message: "Upstream authentication failed, please contact administrator"}
	case http.StatusPaymentRequired:
		return UpstreamClientError{Status: http.StatusBadGateway, Type: "upstream_error", Message: "Upstream service temporarily unavailable"}
	case http.StatusForbidden:
		return UpstreamClientError{Status: forbiddenStatus, Type: forbiddenType, Message: "Upstream access forbidden, please contact administrator"}
	case http.StatusTooManyRequests:
		return UpstreamClientError{Status: http.StatusServiceUnavailable, Type: "upstream_error", Message: "Service temporarily unavailable, please retry later"}
	case 529:
		return UpstreamClientError{Status: http.StatusServiceUnavailable, Type: overloadedType, Message: "Upstream service overloaded, please retry later"}
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return UpstreamClientError{Status: http.StatusBadGateway, Type: "upstream_error", Message: "Upstream service temporarily unavailable"}
	default:
		return UpstreamClientError{Status: http.StatusBadGateway, Type: "upstream_error", Message: "Upstream request failed"}
	}
}

func applyUpstreamFailoverClientPolicy(err UpstreamClientError) UpstreamClientError {
	if err.Status == http.StatusTooManyRequests {
		err.Status = http.StatusServiceUnavailable
		err.Type = "upstream_error"
		err.Message = "Service temporarily unavailable, please retry later"
		return err
	}
	if strings.TrimSpace(err.Type) == "" {
		err.Type = "upstream_error"
	}
	if strings.TrimSpace(err.Message) == "" {
		err.Message = "Upstream request failed"
	}
	return err
}

func upstreamClientErrorForPassthroughFailover(statusCode int, errType, message string) UpstreamClientError {
	return applyUpstreamFailoverClientPolicy(UpstreamClientError{
		Status:  statusCode,
		Type:    errType,
		Message: message,
	})
}

// needForceCacheBilling 判断 failover 时是否需要强制缓存计费。
// 粘性会话切换账号、或上游明确标记时，将 input_tokens 转为 cache_read 计费。
func needForceCacheBilling(hasBoundSession bool, failoverErr *service.UpstreamFailoverError) bool {
	return hasBoundSession || (failoverErr != nil && failoverErr.ForceCacheBilling)
}

// sleepWithContext 等待指定时长，返回 false 表示 context 已取消。
func sleepWithContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
