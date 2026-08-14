package service

import (
	"context"
	"log/slog"
)

func (s *RateLimitService) handleOpenAI403Policy(ctx context.Context, account *Account, upstreamMsg string, responseBody []byte) bool {
	msg := buildForbiddenErrorMessage(
		"Access forbidden (403):",
		upstreamMsg,
		responseBody,
		"account may be suspended or lack permissions",
	)
	if !isOpenAI403ProbeCircuitBody(account, responseBody) {
		s.handleAuthError(ctx, account, msg)
		return true
	}
	if s.openAI403CounterCache == nil {
		s.handleAuthError(ctx, account, msg)
		return true
	}

	count, err := s.openAI403CounterCache.IncrementOpenAI403Count(ctx, account.ID, openAI403CounterWindowMinutes)
	if err != nil {
		slog.Warn("openai_403_increment_failed", "account_id", account.ID, "error", err)
	}

	slog.Warn(
		"openai_403_probe_circuit",
		"account_id", account.ID,
		"count", count,
		"message", msg,
	)
	return true
}
