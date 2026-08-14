package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const slotPoolKeyPrefix = "slot:pool:"

type slotPoolCache struct {
	rdb *redis.Client
}

// NewSlotPoolCache 创建候选账号索引池 Redis 缓存
func NewSlotPoolCache(rdb *redis.Client) service.SlotPoolCache {
	return &slotPoolCache{rdb: rdb}
}

func slotPoolKey(bucket service.SchedulerBucket) string {
	return fmt.Sprintf("%s%d:%s:%s", slotPoolKeyPrefix, bucket.GroupID, bucket.Platform, bucket.Mode)
}

// Pop 弹出最优账号（最高优先级 + 最近使用 + 最早创建）
// score = priority * 1e13 - lastUsed_ms，值越小越优先
// member = zero-padded accountID，同分时按字典序（即 ID 升序）tie-break
// 返回 0 表示池空
func (c *slotPoolCache) Pop(ctx context.Context, bucket service.SchedulerBucket) (int64, error) {
	key := slotPoolKey(bucket)
	result, err := c.rdb.ZPopMin(ctx, key, 1).Result()
	if err != nil {
		return 0, err
	}
	if len(result) == 0 {
		return 0, nil // 池空
	}
	// member 是 zero-padded 的 accountID，直接 ParseInt 会忽略前导零
	memberStr, ok := result[0].Member.(string)
	if !ok {
		return 0, fmt.Errorf("invalid member type: %T", result[0].Member)
	}
	id, err := strconv.ParseInt(memberStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid member format: %w", err)
	}
	return id, nil
}

// Add 将账号加入池
// score = priority * 1e13 - lastUsed_ms
// 优先级越小越优先，使用时间越近越优先
// member = zero-padded accountID，同分时按字典序 tie-break（ID 小 = 创建早 = 优先）
func (c *slotPoolCache) Add(ctx context.Context, bucket service.SchedulerBucket, accountID int64, priority int, lastUsed time.Time) error {
	key := slotPoolKey(bucket)
	// 1e13 显著大于可预期的 Unix milliseconds 范围（~1.7e12），确保 priority 是绝对主排序键
	// priority <= 100 时 score ≈ 1e15，远在 float64 精度安全区（2^53 ≈ 9e15）
	score := float64(priority)*1e13 - float64(lastUsed.UnixMilli())
	// member 用 19 位固定宽度，确保字典序 = 数值序（覆盖 int64 正数范围）
	return c.rdb.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: fmt.Sprintf("%019d", accountID),
	}).Err()
}

// Remove 从池中移除账号
func (c *slotPoolCache) Remove(ctx context.Context, bucket service.SchedulerBucket, accountID int64) error {
	key := slotPoolKey(bucket)
	return c.rdb.ZRem(ctx, key, fmt.Sprintf("%019d", accountID)).Err()
}

// Rebuild 重建整个池
// 使用 pipeline 批量操作：先删除旧池，再批量添加新成员
func (c *slotPoolCache) Rebuild(ctx context.Context, bucket service.SchedulerBucket, entries []service.SlotPoolEntry) error {
	key := slotPoolKey(bucket)

	// 使用 pipeline 批量操作
	pipe := c.rdb.Pipeline()
	pipe.Del(ctx, key)

	if len(entries) > 0 {
		members := make([]redis.Z, len(entries))
		for i, e := range entries {
			// 1e13 >> Unix ms，priority 主排序；member 做同分 tie-break
			score := float64(e.Priority)*1e13 - float64(e.LastUsed.UnixMilli())
			members[i] = redis.Z{
				Score:  score,
				Member: fmt.Sprintf("%019d", e.AccountID),
			}
		}
		pipe.ZAdd(ctx, key, members...)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Size 获取池大小（监控用）
func (c *slotPoolCache) Size(ctx context.Context, bucket service.SchedulerBucket) (int64, error) {
	return c.rdb.ZCard(ctx, slotPoolKey(bucket)).Result()
}
