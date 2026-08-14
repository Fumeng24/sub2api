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

func reportAccountScheduleFailureCustom(ctx context.Context, gatewayService TempUnscheduler, accountID int64, model, endpoint string, failoverErr *service.UpstreamFailoverError) {
	if reporter, ok := gatewayService.(AccountScheduleFailureReporter); ok {
		reporter.ReportAccountScheduleFailure(ctx, accountID, model, endpoint, failoverErr)
	}
}

func shouldTempUnscheduleFailoverCustom(ctx context.Context, statusCode int) bool {
	return !shouldSkipSingleCandidateTempUnschedule(ctx, statusCode)
}

func shouldRetrySelectionExhaustedCustom(s *FailoverState) bool {
	return s != nil && s.LastFailoverErr != nil && s.LastFailoverErr.RetryOnSelectionExhausted
}

func prepareSelectionExhaustedRetryCustom(s *FailoverState) {
	if s != nil {
		s.SwitchCount++
	}
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
	sameAccountRetry := failoverErr != nil && failoverErr.RetryableOnSameAccount && s.SameAccountRetryCount[accountID] < maxSameAccountRetries
	if needForceCacheBilling(s.hasBoundSession, failoverErr, sameAccountRetry) {
		s.ForceCacheBilling = true
	}

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

	reportAccountScheduleFailureCustom(ctx, gatewayService, accountID, model, endpoint, failoverErr)
	if failoverErr.RetryableOnSameAccount && shouldTempUnscheduleFailoverCustom(ctx, failoverErr.StatusCode) {
		gatewayService.TempUnscheduleRetryableError(ctx, accountID, failoverErr)
	}

	s.FailedAccountIDs[accountID] = struct{}{}
	if s.SwitchCount >= s.MaxSwitches {
		return FailoverExhausted
	}

	s.SwitchCount++
	fields := s.failoverLogFields(accountID, model, endpoint, failoverErr, extraFields...)
	fields = append(fields,
		zap.Int("switch_count", s.SwitchCount),
		zap.Int("max_switches", s.MaxSwitches),
	)
	logger.FromContext(ctx).Warn("gateway.failover_switch_account", fields...)

	if platform == service.PlatformAntigravity {
		delay := time.Duration(s.SwitchCount-1) * time.Second
		if !sleepWithContext(ctx, delay) {
			return FailoverCanceled
		}
	}
	return FailoverContinue
}

func shouldSingleCandidateRetryStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusBadRequest,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		529:
		return true
	default:
		return false
	}
}

func shouldSkipSingleCandidateTempUnschedule(ctx context.Context, statusCode int) bool {
	v, _ := service.SingleAccountRetryFromContext(ctx)
	return v && shouldSingleCandidateRetryStatus(statusCode)
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
	return append(fields, extraFields...)
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

const gatewayUpstreamResponseBudget = 5 * time.Minute

func gatewayFailoverBudgetExceeded(start time.Time) bool {
	return !start.IsZero() && time.Since(start) >= gatewayUpstreamResponseBudget
}

func logGatewayFailoverBudgetExceeded(reqLog *zap.Logger, event string, accountID int64, platform string, failoverErr *service.UpstreamFailoverError, switchCount, maxSwitches int, model, endpoint string) {
	if reqLog == nil {
		return
	}
	statusCode := 0
	if failoverErr != nil {
		statusCode = failoverErr.StatusCode
	}
	reqLog.Warn(event,
		zap.Int64("account_id", accountID),
		zap.String("platform", platform),
		zap.Int("upstream_status", statusCode),
		zap.Int("switch_count", switchCount),
		zap.Int("max_switches", maxSwitches),
		zap.Duration("failover_budget", gatewayUpstreamResponseBudget),
		zap.String("model", model),
		zap.String("endpoint", endpoint),
	)
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

func manualFailoverSwitchFields(accountID int64, upstreamStatus, switchCount, maxSwitches int, failedAccountIDs map[int64]struct{}, model, endpoint string, currentSize, writerSizeBeforeForward int) []zap.Field {
	fields := []zap.Field{
		zap.Int64("account_id", accountID),
		zap.Int64("failed_account", accountID),
		zap.Int64s("tried_accounts", sortedInt64SetKeys(failedAccountIDs)),
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
	return append(fields, failoverWriterFields(currentSize, writerSizeBeforeForward)...)
}

func upstreamClientErrorForFailoverStatus(statusCode int) UpstreamClientError {
	return upstreamClientErrorForFailoverStatusWithPolicy(statusCode, upstreamFailoverClientPolicy{})
}

func upstreamClientErrorForFailoverStatusWithPolicy(statusCode int, policy upstreamFailoverClientPolicy) UpstreamClientError {
	overloadedType := strings.TrimSpace(policy.OverloadedType)
	if overloadedType == "" {
		overloadedType = "overloaded_error"
	}

	switch statusCode {
	case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden:
		return UpstreamClientError{Status: http.StatusBadGateway, Type: "upstream_error", Message: "Service temporarily unavailable, please retry later"}
	case http.StatusTooManyRequests:
		return UpstreamClientError{Status: http.StatusServiceUnavailable, Type: "upstream_error", Message: "Service temporarily unavailable, please retry later"}
	case 529:
		return UpstreamClientError{Status: http.StatusServiceUnavailable, Type: overloadedType, Message: "Service temporarily unavailable, please retry later"}
	default:
		return UpstreamClientError{Status: http.StatusBadGateway, Type: "upstream_error", Message: "Service temporarily unavailable, please retry later"}
	}
}

func applyUpstreamFailoverClientPolicy(err UpstreamClientError) UpstreamClientError {
	normalized := service.NormalizeUpstreamClientError(err.Status, err.Type, err.Message)
	return UpstreamClientError{Status: normalized.Status, Type: normalized.Type, Message: normalized.Message}
}

func upstreamClientErrorForPassthroughFailover(statusCode int, errType, message string) UpstreamClientError {
	return applyUpstreamFailoverClientPolicy(UpstreamClientError{Status: statusCode, Type: errType, Message: message})
}
