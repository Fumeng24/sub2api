package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

type slotPoolService struct {
	cache            SlotPoolCache
	schedulerCache   SchedulerCache
	concurrencyCache ConcurrencyCache
	cfg              *config.Config

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// NewSlotPoolService 创建候选账号索引池服务
func NewSlotPoolService(
	cache SlotPoolCache,
	schedulerCache SchedulerCache,
	concurrencyCache ConcurrencyCache,
	cfg *config.Config,
) SlotPoolService {
	return &slotPoolService{
		cache:            cache,
		schedulerCache:   schedulerCache,
		concurrencyCache: concurrencyCache,
		cfg:              cfg,
		stopCh:           make(chan struct{}),
	}
}

// IsEnabled 是否启用池
func (s *slotPoolService) IsEnabled() bool {
	if s == nil || s.cache == nil || s.cfg == nil {
		return false
	}
	return s.cfg.Gateway.Scheduling.SlotPool.Enabled
}

// AcquireFromPool 从池获取账号并尝试占槽位
// candidates: 当前请求的有效候选账号ID集合（用于校验池成员）
// requestID: 用于并发槽位标识
// 返回 (accountID, acquired, error)
func (s *slotPoolService) AcquireFromPool(
	ctx context.Context,
	bucket SchedulerBucket,
	candidates map[int64]*Account,
	requestID string,
) (int64, bool, error) {
	if !s.IsEnabled() {
		return 0, false, nil
	}

	maxRetries := s.cfg.Gateway.Scheduling.SlotPool.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 10
	}

	// 追踪本次 retry loop 内已 pop 过的非候选账号，避免重复 pop 饿死有效候选
	skippedInLoop := make(map[int64]struct{})
	// 延迟批量放回：loop 结束后再将非候选账号放回池，避免同一 loop 内重复 pop
	var deferredPutBacks []SlotPoolEntry

	for i := 0; i < maxRetries; i++ {
		// 从池弹出
		accountID, err := s.cache.Pop(ctx, bucket)
		if err != nil {
			logger.FromContext(ctx).Warn("slot pool pop failed",
				zap.String("bucket", bucket.String()),
				zap.Error(err))
			// 放回已跳过的账号
			for _, entry := range deferredPutBacks {
				_ = s.cache.Add(ctx, bucket, entry.AccountID, entry.Priority, entry.LastUsed)
			}
			return 0, false, err
		}
		if accountID == 0 {
			// 池空，放回已跳过的账号
			for _, entry := range deferredPutBacks {
				_ = s.cache.Add(ctx, bucket, entry.AccountID, entry.Priority, entry.LastUsed)
			}
			return 0, false, nil
		}

		// 本次 loop 已跳过过，直接跳过
		// 注意：不做 i--，重复 pop 的账号仍计入 retry 次数，防止并发 reinsert 导致无限循环
		if _, seen := skippedInLoop[accountID]; seen {
			continue
		}

		// 检查是否在当前候选集中（防止池与快照偏差）
		account, ok := candidates[accountID]
		if !ok {
			// 不在当前请求的候选集（如模型过滤）
			// 记录以避免同一 loop 内重复处理
			skippedInLoop[accountID] = struct{}{}

			// 验证账号是否仍属于此 bucket 且有可用容量，再决定是否延迟放回
			// 如果账号已不属于此 bucket（group/platform/mode 漂移），不放回，让 rebuild 清理
			if cachedAccount, err := s.schedulerCache.GetAccount(ctx, accountID); err == nil && cachedAccount != nil {
				// 验证账号仍属于当前 bucket
				if s.accountBelongsToBucket(cachedAccount, bucket) {
					// 检查容量后再决定是否放回
					if entry, ok := s.slotPoolEntryForBucket(ctx, cachedAccount, bucket); ok {
						deferredPutBacks = append(deferredPutBacks, entry)
					}
				}
			}
			continue
		}

		// 尝试获取槽位（无限并发账号直接视为已获取，匹配 ConcurrencyService 短路语义）
		var acquired bool
		if account.Concurrency <= 0 {
			acquired = true
		} else {
			var acqErr error
			acquired, acqErr = s.concurrencyCache.AcquireAccountSlot(
				ctx, accountID, account.Concurrency, requestID)
			if acqErr != nil {
				err = acqErr
			}
		}
		if err != nil {
			logger.FromContext(ctx).Warn("slot pool acquire slot failed",
				zap.Int64("account_id", accountID),
				zap.Error(err))
			// 出错时把账号放回池
			_ = s.cache.Add(ctx, bucket, accountID, account.Priority, s.getLastUsed(account))
			// 放回已跳过的账号
			for _, entry := range deferredPutBacks {
				_ = s.cache.Add(ctx, bucket, entry.AccountID, entry.Priority, entry.LastUsed)
			}
			return 0, false, err
		}

		if acquired {
			// 成功获取，检查是否还有空闲槽位
			currentLoad, _ := s.concurrencyCache.GetAccountConcurrency(ctx, accountID)
			if hasAvailableSlot(currentLoad, account.Concurrency) {
				// 还有空闲，放回池（使用当前时间作为 lastUsed，确保热账号优先）
				_ = s.cache.Add(ctx, bucket, accountID, account.Priority, time.Now())
			}
			// 放回已跳过的账号
			for _, entry := range deferredPutBacks {
				_ = s.cache.Add(ctx, bucket, entry.AccountID, entry.Priority, entry.LastUsed)
			}
			return accountID, true, nil
		}

		// 获取失败，账号已满，不放回池
		// 继续尝试下一个
	}

	// 重试次数耗尽，放回已跳过的账号
	for _, entry := range deferredPutBacks {
		_ = s.cache.Add(ctx, bucket, entry.AccountID, entry.Priority, entry.LastUsed)
	}
	return 0, false, nil
}

// OnSlotReleased 槽位释放回调，决定是否回填
func (s *slotPoolService) OnSlotReleased(ctx context.Context, accountID int64) error {
	if !s.IsEnabled() {
		return nil
	}

	// 获取账号信息
	account, err := s.schedulerCache.GetAccount(ctx, accountID)
	if err != nil || account == nil {
		return err
	}

	// 如果有空闲槽位，回填到相关池
	// 直接用 time.Now() 作为排序时间，确保刚释放的账号优先被复用
	// 不用 cache 里的 LastUsedAt，因为它是延迟批量更新的，可能是旧值
	if baseEntry, ok := s.slotPoolEntryForAccount(ctx, account); ok {
		baseEntry.LastUsed = time.Now()
		buckets := s.getAccountBuckets(account)
		for _, bucket := range buckets {
			entry := baseEntry
			entry.Priority = account.SchedulingPriorityForGroup(bucket.GroupID)
			_ = s.cache.Add(ctx, bucket, entry.AccountID, entry.Priority, entry.LastUsed)
		}
	}

	return nil
}

// RebuildBucketPool 重建指定 bucket 的池（由 SchedulerSnapshotService 调用）
func (s *slotPoolService) RebuildBucketPool(ctx context.Context, bucket SchedulerBucket, accounts []*Account) error {
	if !s.IsEnabled() {
		return nil
	}

	// 构造池条目（只包含可调度且有空闲槽位的账号）
	var entries []SlotPoolEntry
	for _, acc := range accounts {
		if entry, ok := s.slotPoolEntryForBucket(ctx, acc, bucket); ok {
			entries = append(entries, entry)
		}
	}

	return s.cache.Rebuild(ctx, bucket, entries)
}

// Start 启动后台 worker（仅做兜底重建）
func (s *slotPoolService) Start() {
	if !s.IsEnabled() {
		return
	}

	log := logger.L()
	log.Info("starting slot pool service")

	// 初始化池
	ctx := context.Background()
	if err := s.rebuildAllPools(ctx); err != nil {
		log.Error("failed to rebuild slot pools", zap.Error(err))
	}

	// 启动周期性重建
	interval := s.cfg.Gateway.Scheduling.SlotPool.RebuildInterval
	if interval > 0 {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.runRebuildWorker(interval)
		}()
	}
}

func (s *slotPoolService) rebuildAllPools(ctx context.Context) error {
	buckets, err := s.schedulerCache.ListBuckets(ctx)
	if err != nil {
		return err
	}

	for _, bucket := range buckets {
		accounts, hit, err := s.schedulerCache.GetSnapshot(ctx, bucket)
		if err != nil || !hit {
			continue
		}

		// 构造池条目（只包含可调度且有空闲槽位的账号）
		var entries []SlotPoolEntry
		for _, acc := range accounts {
			if entry, ok := s.slotPoolEntryForBucket(ctx, acc, bucket); ok {
				entries = append(entries, entry)
			}
		}

		if err := s.cache.Rebuild(ctx, bucket, entries); err != nil {
			logger.L().Error("failed to rebuild slot pool",
				zap.String("bucket", bucket.String()),
				zap.Error(err))
		}
	}

	return nil
}

func (s *slotPoolService) slotPoolEntryForAccount(ctx context.Context, account *Account) (SlotPoolEntry, bool) {
	if account == nil || account.SchedulingBlockReasonAt(time.Now()) != AccountSchedulingBlockNone {
		return SlotPoolEntry{}, false
	}
	currentLoad, _ := s.concurrencyCache.GetAccountConcurrency(ctx, account.ID)
	if !hasAvailableSlot(currentLoad, account.Concurrency) {
		return SlotPoolEntry{}, false
	}
	return SlotPoolEntry{
		AccountID: account.ID,
		Priority:  account.Priority,
		LastUsed:  s.getLastUsed(account),
	}, true
}

func (s *slotPoolService) slotPoolEntryForBucket(ctx context.Context, account *Account, bucket SchedulerBucket) (SlotPoolEntry, bool) {
	entry, ok := s.slotPoolEntryForAccount(ctx, account)
	if !ok {
		return SlotPoolEntry{}, false
	}
	entry.Priority = account.SchedulingPriorityForGroup(bucket.GroupID)
	return entry, true
}

func (s *slotPoolService) runRebuildWorker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			ctx := context.Background()
			if err := s.rebuildAllPools(ctx); err != nil {
				logger.L().Error("slot pool rebuild failed", zap.Error(err))
			}
		}
	}
}

// Stop 停止后台 worker
func (s *slotPoolService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

// getAccountBuckets 获取账号所属的所有 bucket
// 账号可能同时属于多个 bucket：
// - single: 按 platform + groupID
// - forced: 按 platform + groupID（强制平台模式）
// - mixed: 对于 anthropic/gemini 平台或启用了 mixed_scheduling 的 antigravity 账号
func (s *slotPoolService) getAccountBuckets(account *Account) []SchedulerBucket {
	if account == nil {
		return nil
	}

	// 处理 groupIDs：简单模式统一归 0，普通模式为空则归 ungrouped (groupID=0)
	groupIDs := s.normalizeGroupIDs(account.GroupIDs)

	var buckets []SchedulerBucket
	for _, groupID := range groupIDs {
		// single 和 forced bucket
		buckets = append(buckets, SchedulerBucket{
			GroupID:  groupID,
			Platform: account.Platform,
			Mode:     SchedulerModeSingle,
		})
		buckets = append(buckets, SchedulerBucket{
			GroupID:  groupID,
			Platform: account.Platform,
			Mode:     SchedulerModeForced,
		})

		// mixed bucket: 仅 anthropic/gemini 平台有 mixed 调度
		if account.Platform == PlatformAnthropic || account.Platform == PlatformGemini {
			buckets = append(buckets, SchedulerBucket{
				GroupID:  groupID,
				Platform: account.Platform,
				Mode:     SchedulerModeMixed,
			})
		}

		// antigravity 账号启用 mixed_scheduling 时，会被加入 anthropic 和 gemini 的 mixed bucket
		if account.Platform == PlatformAntigravity && account.IsMixedSchedulingEnabled() {
			buckets = append(buckets, SchedulerBucket{
				GroupID:  groupID,
				Platform: PlatformAnthropic,
				Mode:     SchedulerModeMixed,
			})
			buckets = append(buckets, SchedulerBucket{
				GroupID:  groupID,
				Platform: PlatformGemini,
				Mode:     SchedulerModeMixed,
			})
		}
	}
	return buckets
}

// getLastUsed 获取账号最后使用时间
// 对于从未使用过的账号返回 Unix 纪元时间，使其在同优先级下基于 ID 排序
func (s *slotPoolService) getLastUsed(account *Account) time.Time {
	if account.LastUsedAt != nil {
		return *account.LastUsedAt
	}
	// 从未使用过的账号，返回最早时间，在同优先级下依靠 ID 做 tie-breaker
	return time.Unix(0, 0)
}

// isRunModeSimple 检查是否为简单模式
func (s *slotPoolService) isRunModeSimple() bool {
	return s.cfg != nil && s.cfg.RunMode == config.RunModeSimple
}

// normalizeGroupIDs 处理 groupIDs，简单模式下统一归到 groupID=0
func (s *slotPoolService) normalizeGroupIDs(groupIDs []int64) []int64 {
	if s.isRunModeSimple() {
		return []int64{0}
	}
	if len(groupIDs) == 0 {
		return []int64{0}
	}
	return groupIDs
}

// hasAvailableSlot 检查账号是否有可用槽位
// maxConcurrency <= 0 表示无限并发，始终返回 true
func hasAvailableSlot(currentLoad, maxConcurrency int) bool {
	if maxConcurrency <= 0 {
		return true // 无限并发
	}
	return currentLoad < maxConcurrency
}

// accountBelongsToBucket 检查账号是否属于指定 bucket
// 用于验证从池中弹出的账号是否仍属于该 bucket，防止 group/platform/mode 漂移导致错误放回
func (s *slotPoolService) accountBelongsToBucket(account *Account, bucket SchedulerBucket) bool {
	if account == nil {
		return false
	}
	validBuckets := s.getAccountBuckets(account)
	for _, b := range validBuckets {
		if b.GroupID == bucket.GroupID && b.Platform == bucket.Platform && b.Mode == bucket.Mode {
			return true
		}
	}
	return false
}
