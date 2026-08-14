package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// Keep the optional admin scheduling API and the background auto-sort worker
// wired together even when upstream reorganizes the main admin service files.
var _ groupAutoSortAdmin = (*adminServiceImpl)(nil)

func (s *adminServiceImpl) GetGroupAccountScheduling(ctx context.Context, groupID int64) ([]AccountSchedulingEntry, error) {
	if groupID <= 0 {
		return nil, ErrGroupNotFound
	}
	if _, err := s.groupRepo.GetByIDLite(ctx, groupID); err != nil {
		return nil, err
	}
	repo, ok := s.groupRepo.(AccountSchedulingConfigRepository)
	if !ok {
		return nil, errors.New("account scheduling config repository not configured")
	}
	entries, err := repo.ListAccountSchedulingConfigs(ctx, groupID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range entries {
		account := entries[i].Account
		if account == nil || !account.IsGroupReserveEligibleAt(now) {
			continue
		}
		entries[i].GroupReserve = true
		entries[i].GroupReserveUntil = account.TempUnschedulableUntil
		entries[i].GroupReserveReason = account.TempUnschedulableReason
	}

	if s.usageLogRepo == nil || len(entries) == 0 {
		return entries, nil
	}
	accountIDs := make([]int64, 0, len(entries))
	for i := range entries {
		if entries[i].AccountID > 0 {
			accountIDs = append(accountIDs, entries[i].AccountID)
		}
	}
	if len(accountIDs) == 0 {
		return entries, nil
	}
	stats, err := getRecentAccountFirstTokenStats(s.usageLogRepo, ctx, accountIDs, groupID, now.Add(-5*time.Minute))
	if err != nil {
		logger.LegacyPrintf("service.admin", "load recent scheduler first-token stats failed: group=%d err=%v", groupID, err)
		return entries, nil
	}
	for i := range entries {
		stat, ok := stats[entries[i].AccountID]
		if !ok || stat.SampleCount <= 0 {
			continue
		}
		avg := stat.AvgFirstTokenMs
		entries[i].RecentUserAvgFirstTokenMs = &avg
		entries[i].RecentUserFirstTokenSampleCnt = stat.SampleCount
	}
	return entries, nil
}

func (s *adminServiceImpl) UpdateGroupAccountScheduling(ctx context.Context, groupID int64, configs []AccountSchedulingConfig) error {
	if groupID <= 0 {
		return ErrGroupNotFound
	}
	if _, err := s.groupRepo.GetByIDLite(ctx, groupID); err != nil {
		return err
	}
	repo, ok := s.groupRepo.(AccountSchedulingConfigRepository)
	if !ok {
		return errors.New("account scheduling config repository not configured")
	}
	normalized := make([]AccountSchedulingConfig, 0, len(configs))
	for i, cfg := range configs {
		if cfg.AccountID <= 0 {
			return fmt.Errorf("account_id is required")
		}
		switch cfg.Role {
		case "", AccountGroupRolePrimary:
			cfg.Role = AccountGroupRolePrimary
		case AccountGroupRoleBackup:
			cfg.Role = AccountGroupRoleBackup
		default:
			return fmt.Errorf("invalid role %q", cfg.Role)
		}
		if cfg.Weight <= 0 {
			return fmt.Errorf("weight must be > 0")
		}
		if cfg.SortOrder == 0 {
			cfg.SortOrder = i + 1
		}
		// priority 作为旧调度链路的兼容镜像，与当前分组顺序保持一致。
		cfg.Priority = cfg.SortOrder
		normalized = append(normalized, cfg)
	}
	if err := repo.UpdateAccountSchedulingConfigs(ctx, groupID, normalized); err != nil {
		return err
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByGroupID(ctx, groupID)
	}
	return nil
}
