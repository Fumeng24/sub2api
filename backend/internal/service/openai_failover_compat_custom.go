package service

import (
	"context"
	"net/http"
	"strings"
)

func (s *OpenAIGatewayService) shouldFailoverOpenAIUpstreamResponseCustom(statusCode int, upstreamBody []byte) bool {
	return isUpstreamModelNotFoundError(statusCode, upstreamBody) ||
		isOpenAIEndpointMigrationError(statusCode, extractUpstreamErrorMessage(upstreamBody), upstreamBody)
}

// Simplified failover methods (scheduler health policy removed).

func (s *OpenAIGatewayService) shouldFailoverOpenAIUpstreamResponseForAccount(ctx context.Context, account *Account, statusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if isOpenAIContextWindowError(upstreamMsg, upstreamBody) {
		return false
	}
	if statusCode >= 400 && isOpenAIThinkingSignatureInvalidError(upstreamBody, upstreamMsg) {
		return false
	}
	class := classifyOpenAIUpstreamError(statusCode, upstreamMsg, upstreamBody)
	return openAIUpstreamErrorClassShouldFailover(class)
}

func (s *OpenAIGatewayService) retryableOnSameOpenAIAccount(ctx context.Context, account *Account, statusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if isOpenAIEndpointMigrationError(statusCode, upstreamMsg, upstreamBody) {
		return false
	}
	if retryableOpenAIUpstreamErrorOnSameAccount(statusCode, upstreamMsg, upstreamBody) {
		return true
	}
	if statusCode == http.StatusPaymentRequired || statusCode == http.StatusForbidden || statusCode == http.StatusTooManyRequests {
		return false
	}
	return account.IsPoolMode() &&
		(account.IsPoolModeRetryableStatus(statusCode) || isOpenAITransientProcessingError(statusCode, upstreamMsg, upstreamBody))
}

func (s *OpenAIGatewayService) retryableOnSameOpenAIAccountStatus(ctx context.Context, account *Account, statusCode int) bool {
	// Status-only callers do not have the provider body. A 410 is still a
	// deterministic endpoint retirement signal and must never consume the
	// same-account retry budget.
	if statusCode == http.StatusGone {
		return false
	}
	if retryableOpenAIUpstreamErrorOnSameAccount(statusCode, "", nil) {
		return true
	}
	if statusCode == http.StatusPaymentRequired || statusCode == http.StatusForbidden || statusCode == http.StatusTooManyRequests {
		return false
	}
	return account.IsPoolMode() && account.IsPoolModeRetryableStatus(statusCode)
}

func (s *OpenAIGatewayService) schedulerCategoryOverrideForOpenAIUpstreamError(_ context.Context, account *Account, statusCode int, responseBody []byte) string {
	if s == nil || account == nil || account.Platform != PlatformOpenAI || statusCode != http.StatusForbidden {
		return ""
	}
	class := classifyOpenAIUpstreamError(statusCode, extractUpstreamErrorMessage(responseBody), responseBody)
	if class != openAIUpstreamErrorForbidden {
		return ""
	}
	return "transient"
}

// accountsContainCurrentGroupBinding reports whether any account has an active group binding for groupID.
func accountsContainCurrentGroupBinding(accounts []*Account, groupID *int64) bool {
	if groupID == nil || len(accounts) == 0 {
		return false
	}
	for _, acc := range accounts {
		if _, ok := resolveAccountCurrentGroupOrder(acc, groupID); ok {
			return true
		}
	}
	return false
}

// isAccountBetterByCurrentGroupOrder compares two accounts by group binding priority.
func isAccountBetterByCurrentGroupOrder(candidate, current *Account, groupID *int64) bool {
	if candidate == nil || current == nil || groupID == nil {
		return false
	}
	candidateOrder, candidateInGroup := resolveAccountCurrentGroupOrder(candidate, groupID)
	currentOrder, currentInGroup := resolveAccountCurrentGroupOrder(current, groupID)
	if candidateInGroup != currentInGroup {
		return candidateInGroup
	}
	if !candidateInGroup {
		return false
	}
	if candidateOrder.sortOrder != currentOrder.sortOrder {
		return candidateOrder.sortOrder < currentOrder.sortOrder
	}
	if candidateOrder.priority != currentOrder.priority {
		return candidateOrder.priority < currentOrder.priority
	}
	return candidate.ID < current.ID
}

type accountCurrentGroupOrder struct {
	sortOrder int
	priority  int
}

func resolveAccountCurrentGroupOrder(account *Account, groupID *int64) (accountCurrentGroupOrder, bool) {
	if account == nil || groupID == nil {
		return accountCurrentGroupOrder{}, false
	}
	for _, ag := range account.AccountGroups {
		if ag.GroupID == *groupID {
			return accountCurrentGroupOrder{
				sortOrder: ag.EffectiveSortOrder(),
				priority:  ag.Priority,
			}, true
		}
	}
	for _, id := range account.GroupIDs {
		if id == *groupID {
			return accountCurrentGroupOrder{
				sortOrder: account.Priority,
				priority:  account.Priority,
			}, true
		}
	}
	for _, g := range account.Groups {
		if g != nil && g.ID == *groupID {
			return accountCurrentGroupOrder{
				sortOrder: account.Priority,
				priority:  account.Priority,
			}, true
		}
	}
	return accountCurrentGroupOrder{}, false
}

func retryableOpenAIUpstreamErrorOnSameAccount(statusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if statusCode == 0 {
		return true
	}
	class := classifyOpenAIUpstreamError(statusCode, upstreamMsg, upstreamBody)
	return class == openAIUpstreamErrorTransient
}

func (s *OpenAIGatewayService) closeOpenAIAccountIdleConnectionsForCircuit(accountID int64, statusCode int, reason string, responseBody []byte) {
	if s == nil || accountID <= 0 || !shouldCloseOpenAIIdleConnectionsForCircuit(statusCode, reason, responseBody) {
		return
	}
	closer, ok := s.httpUpstream.(HTTPUpstreamAccountIdleCloser)
	if !ok || closer == nil {
		return
	}
	closer.CloseIdleConnectionsForAccount(accountID)
}

func shouldCloseOpenAIIdleConnectionsForCircuit(statusCode int, reason string, responseBody []byte) bool {
	if statusCode != 0 {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(reason + " " + string(responseBody)))
	if text == "" {
		return false
	}
	for _, marker := range []string{"connection reset by peer", "http2:", "client connection force closed", "use of closed network connection"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func isOpenAITransient5xxStatus(statusCode int) bool {
	return statusCode >= 500 && statusCode <= 599
}
