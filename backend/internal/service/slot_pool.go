package service

import (
	"context"
	"time"
)

// SlotPoolCache 提供候选账号索引池的 Redis 操作。
// 池内成员语义是"可能可用"，不是"强一致可用"。
type SlotPoolCache interface {
	// Pop 弹出最优账号（最高优先级 + 最近使用 + 最早创建）
	// 返回 0 表示池空
	Pop(ctx context.Context, bucket SchedulerBucket) (int64, error)

	// Add 将账号加入池
	Add(ctx context.Context, bucket SchedulerBucket, accountID int64, priority int, lastUsed time.Time) error

	// Remove 从池中移除账号
	Remove(ctx context.Context, bucket SchedulerBucket, accountID int64) error

	// Rebuild 重建整个池
	Rebuild(ctx context.Context, bucket SchedulerBucket, entries []SlotPoolEntry) error

	// Size 获取池大小（监控用）
	Size(ctx context.Context, bucket SchedulerBucket) (int64, error)
}

// SlotPoolEntry 池条目
type SlotPoolEntry struct {
	AccountID int64
	Priority  int
	LastUsed  time.Time
}

// SlotPoolService 池管理服务
type SlotPoolService interface {
	// AcquireFromPool 从池获取账号并尝试占槽位
	// candidates: 当前请求的有效候选账号ID集合（用于校验池成员）
	// requestID: 用于并发槽位标识
	// 返回 (accountID, acquired, error)
	AcquireFromPool(ctx context.Context, bucket SchedulerBucket, candidates map[int64]*Account, requestID string) (accountID int64, acquired bool, err error)

	// OnSlotReleased 槽位释放回调，决定是否回填
	OnSlotReleased(ctx context.Context, accountID int64) error

	// RebuildBucketPool 重建指定 bucket 的池（由 SchedulerSnapshotService 调用）
	RebuildBucketPool(ctx context.Context, bucket SchedulerBucket, accounts []*Account) error

	// Start 启动后台 worker（仅做兜底重建）
	Start()

	// Stop 停止后台 worker
	Stop()

	// IsEnabled 是否启用
	IsEnabled() bool
}
