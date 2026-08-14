package service

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// stickySessionClearReason returns the reason a sticky binding must be cleared.
// Empty means the binding can be kept.
func stickySessionClearReason(account *Account, requestedModel string) string {
	if account == nil {
		return ""
	}
	if reason := account.SchedulingBlockReasonAt(time.Now()).String(); reason != "" {
		return reason
	}
	if remaining := account.GetRateLimitRemainingTimeWithContext(context.Background(), requestedModel); remaining > 0 {
		return "model_rate_limited"
	}
	return ""
}

func (s *GatewayService) clearStickySessionBinding(ctx context.Context, groupID *int64, sessionHash string, accountID int64, reason string) {
	if s == nil || s.cache == nil || strings.TrimSpace(sessionHash) == "" || accountID <= 0 {
		return
	}
	slog.Debug("sticky.binding_clear",
		"group_id", derefGroupID(groupID),
		"account_id", accountID,
		"session", shortSessionHash(sessionHash),
		"reason", reason,
	)
	_ = s.cache.DeleteSessionAccountID(ctx, derefGroupID(groupID), sessionHash)
}
