package service

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

func (s *GatewayService) handleUpstreamErrorForScheduling(ctx context.Context, account *Account, statusCode int, headers http.Header, body []byte, requestedModel ...string) bool {
	if s == nil || s.rateLimitService == nil || account == nil {
		return false
	}
	if len(requestedModel) > 0 {
		return s.rateLimitService.HandleUpstreamError(ctx, account, statusCode, headers, body, requestedModel[0])
	}
	return s.rateLimitService.HandleUpstreamError(ctx, account, statusCode, headers, body)
}

func (s *GatewayService) shouldSkipSchedulingBlockForLastGroupAccount(ctx context.Context, accountID int64, account *Account, statusCode int, source string) bool {
	if s == nil || s.accountRepo == nil || accountID <= 0 {
		return false
	}
	account = s.accountForSchedulingProtection(ctx, accountID, account)
	groupIDs := schedulingProtectionGroupIDs(ctx, account)
	for _, groupID := range groupIDs {
		accounts, err := listSchedulablePeersForProtection(ctx, s.accountRepo, groupID, accountID, account)
		if err != nil {
			slog.Warn("gateway_last_group_account_check_failed", "account_id", accountID, "group_id", groupID, "error", err)
			continue
		}
		count := 0
		contains := false
		for i := range accounts {
			count++
			if accounts[i].ID == accountID {
				contains = true
			}
		}
		if contains && count <= 1 {
			recordSchedulerBlockSkipped(ctx, s.accountRepo, accountID, groupID, statusCode, "last_group_account", source)
			return true
		}
	}
	return false
}

func (s *GatewayService) accountForSchedulingProtection(ctx context.Context, accountID int64, account *Account) *Account {
	if account != nil && strings.TrimSpace(account.Platform) != "" && len(collectAccountGroupIDs(ctx, account)) > 0 {
		return account
	}
	if s == nil || s.accountRepo == nil || accountID <= 0 {
		return account
	}
	loaded, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		slog.Warn("gateway_scheduling_protection_account_load_failed", "account_id", accountID, "error", err)
		return account
	}
	return loaded
}

func (s *GatewayService) recordSchedulingBlockSkipped(ctx context.Context, accountID int64, account *Account, statusCode int, reason string, source string) {
	if s == nil || s.accountRepo == nil || accountID <= 0 {
		return
	}
	account = s.accountForSchedulingProtection(ctx, accountID, account)
	groupID := int64(0)
	if groupIDs := schedulingProtectionGroupIDs(ctx, account); len(groupIDs) > 0 {
		groupID = groupIDs[0]
	}
	recordSchedulerBlockSkipped(ctx, s.accountRepo, accountID, groupID, statusCode, reason, source)
}

type schedulingProtectionAccountLister interface {
	ListSchedulableByGroupID(ctx context.Context, groupID int64) ([]Account, error)
	ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error)
}

func listSchedulablePeersForProtection(ctx context.Context, repo schedulingProtectionAccountLister, groupID int64, accountID int64, account *Account) ([]Account, error) {
	if repo == nil {
		return nil, nil
	}
	platform := ""
	if account != nil {
		platform = strings.TrimSpace(account.Platform)
	}
	if platform != "" {
		accounts, err := repo.ListSchedulableByGroupIDAndPlatform(ctx, groupID, platform)
		if err != nil {
			return nil, err
		}
		if accountListContainsID(accounts, accountID) {
			return accounts, nil
		}
		allAccounts, fallbackErr := repo.ListSchedulableByGroupID(ctx, groupID)
		if fallbackErr != nil {
			return accounts, nil
		}
		return filterProtectionPeers(allAccounts, accountID, platform), nil
	}
	return repo.ListSchedulableByGroupID(ctx, groupID)
}

func accountListContainsID(accounts []Account, accountID int64) bool {
	for i := range accounts {
		if accounts[i].ID == accountID {
			return true
		}
	}
	return false
}

func filterProtectionPeers(accounts []Account, accountID int64, platform string) []Account {
	if platform == "" {
		return accounts
	}
	filtered := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.ID == accountID || strings.TrimSpace(account.Platform) == platform {
			filtered = append(filtered, account)
		}
	}
	return filtered
}

func splitGroupsForSchedulingBlock(ctx context.Context, repo schedulingProtectionAccountLister, account *Account, groupIDs []int64) ([]int64, []int64) {
	if account == nil || account.ID <= 0 {
		return nil, nil
	}
	blockableGroups := make([]int64, 0, len(groupIDs))
	skippedGroups := make([]int64, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		if groupID <= 0 {
			continue
		}
		accounts, err := listSchedulablePeersForProtection(ctx, repo, groupID, account.ID, account)
		if err != nil {
			slog.Warn("last_group_account_check_failed", "account_id", account.ID, "group_id", groupID, "error", err)
			blockableGroups = append(blockableGroups, groupID)
			continue
		}
		if accountListContainsID(accounts, account.ID) && len(accounts) <= 1 {
			skippedGroups = append(skippedGroups, groupID)
			continue
		}
		blockableGroups = append(blockableGroups, groupID)
	}
	return blockableGroups, skippedGroups
}

type schedulerOutboxAppender interface {
	AppendSchedulerOutboxEvent(ctx context.Context, eventType string, accountID *int64, groupID *int64, payload map[string]any) error
}

func recordSchedulerBlockSkipped(ctx context.Context, recorder any, accountID, groupID int64, statusCode int, reason string, source string) {
	appender, ok := recorder.(schedulerOutboxAppender)
	if !ok || appender == nil || accountID <= 0 {
		return
	}
	payload := map[string]any{
		"reason":      reason,
		"source":      source,
		"status_code": statusCode,
	}
	if groupID > 0 {
		payload["group_ids"] = []int64{groupID}
	}
	var groupIDPtr *int64
	if groupID > 0 {
		groupIDPtr = &groupID
	}
	if err := appender.AppendSchedulerOutboxEvent(ctx, SchedulerOutboxEventSchedulingBlockSkipped, &accountID, groupIDPtr, payload); err != nil {
		slog.Warn("scheduler_block_skipped_outbox_failed", "account_id", accountID, "group_id", groupID, "status_code", statusCode, "error", err)
	}
}

func recordSchedulerBlocked(ctx context.Context, recorder any, accountID, groupID int64, statusCode int, reason string, source string, until time.Time, extra map[string]any) {
	appender, ok := recorder.(schedulerOutboxAppender)
	if !ok || appender == nil || accountID <= 0 {
		return
	}
	payload := map[string]any{
		"reason":      reason,
		"source":      source,
		"status_code": statusCode,
	}
	if !until.IsZero() {
		payload["until"] = until.Format(time.RFC3339)
	}
	if groupID > 0 {
		payload["group_ids"] = []int64{groupID}
	}
	for key, value := range extra {
		if strings.TrimSpace(key) != "" {
			payload[key] = value
		}
	}
	var groupIDPtr *int64
	if groupID > 0 {
		groupIDPtr = &groupID
	}
	if err := appender.AppendSchedulerOutboxEvent(ctx, SchedulerOutboxEventSchedulingBlocked, &accountID, groupIDPtr, payload); err != nil {
		slog.Warn("scheduler_blocked_outbox_failed", "account_id", accountID, "group_id", groupID, "status_code", statusCode, "error", err)
	}
}

func schedulingProtectionGroupIDs(ctx context.Context, account *Account) []int64 {
	if group, ok := ctx.Value(ctxkey.Group).(*Group); ok && IsGroupContextValid(group) {
		return []int64{group.ID}
	}
	return collectAccountGroupIDs(ctx, account)
}

func hasExplicitSchedulingProtectionGroup(ctx context.Context) bool {
	group, ok := ctx.Value(ctxkey.Group).(*Group)
	return ok && IsGroupContextValid(group)
}

func accountHasMultipleBoundGroups(account *Account) bool {
	return len(collectAccountGroupIDs(context.Background(), account)) > 1
}

func collectAccountGroupIDs(ctx context.Context, account *Account) []int64 {
	seen := make(map[int64]struct{})
	groupIDs := make([]int64, 0, 2)
	add := func(groupID int64) {
		if groupID <= 0 {
			return
		}
		if _, ok := seen[groupID]; ok {
			return
		}
		seen[groupID] = struct{}{}
		groupIDs = append(groupIDs, groupID)
	}
	if group, ok := ctx.Value(ctxkey.Group).(*Group); ok && IsGroupContextValid(group) {
		add(group.ID)
	}
	if account != nil {
		for _, groupID := range account.GroupIDs {
			add(groupID)
		}
		for _, accountGroup := range account.AccountGroups {
			add(accountGroup.GroupID)
		}
		for _, group := range account.Groups {
			if group != nil && IsGroupContextValid(group) {
				add(group.ID)
			}
		}
	}
	return groupIDs
}
