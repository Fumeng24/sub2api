package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type concurrencyCacheCustom struct {
	// Keep inline cleanup enabled when the background cleanup worker is disabled;
	// otherwise stale slots can make scheduler load remain artificially high.
	inlineCleanupOnRead bool
}

func (c *concurrencyCache) getAccountConcurrencyBatchCustom(ctx context.Context, accountIDs []int64) (map[int64]int, error) {
	if len(accountIDs) == 0 {
		return map[int64]int{}, nil
	}

	// 热路径默认只发只读命令（ZCARD），把 N=5000 时的命令数从 2N 降到 N，
	// 且去掉所有写命令（AOF/主从压力同步下降）。过期槽位的清理由后台 worker 统一负责：
	// service/concurrency_service.go:StartSlotCleanupWorker，周期由
	// gateway.scheduling.slot_cleanup_interval 控制（默认 30s）。
	// 真正的入队路径（AcquireAccountSlot → acquireScript）在 ZADD 前仍会做原子清理，
	// 因此槽位不会无限累积；此处读到的计数最多滞后一个 worker 周期。
	//
	// 边界：当 slot_cleanup_interval <= 0（worker 被显式禁用）时，wire 层会把
	// inlineCleanupOnRead 置 true，此时退化为原始的 2N 命令行为做正确性兜底，
	// 防止 stale 计数把账号锁死在虚高负载状态（这是一个自强化的 livelock：
	// ZCARD 虚高 → 调度避开 → 不走 acquire → stale 永远不清 → 继续 ZCARD 虚高）。
	var cutoffTime int64
	if c.inlineCleanupOnRead {
		now, err := c.rdb.Time(ctx).Result()
		if err != nil {
			return nil, fmt.Errorf("redis TIME: %w", err)
		}
		cutoffTime = now.Unix() - int64(c.slotTTLSeconds)
	}

	pipe := c.rdb.Pipeline()
	type accountCmd struct {
		accountID int64
		zcardCmd  *redis.IntCmd
	}
	cmds := make([]accountCmd, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		slotKey := accountSlotKeyPrefix + strconv.FormatInt(accountID, 10)
		if c.inlineCleanupOnRead {
			pipe.ZRemRangeByScore(ctx, slotKey, "-inf", strconv.FormatInt(cutoffTime, 10))
		}
		cmds = append(cmds, accountCmd{
			accountID: accountID,
			zcardCmd:  pipe.ZCard(ctx, slotKey),
		})
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("pipeline exec: %w", err)
	}

	result := make(map[int64]int, len(accountIDs))
	for _, cmd := range cmds {
		result[cmd.accountID] = int(cmd.zcardCmd.Val())
	}
	return result, nil
}

func (c *concurrencyCache) tryGetAccountConcurrencyBatchCustom(ctx context.Context, accountIDs []int64) (map[int64]int, error, bool) {
	result, err := c.getAccountConcurrencyBatchCustom(ctx, accountIDs)
	return result, err, true
}

func (c *concurrencyCache) getAccountsLoadBatchCustom(ctx context.Context, accounts []service.AccountWithConcurrency) (map[int64]*service.AccountLoadInfo, error) {
	if len(accounts) == 0 {
		return map[int64]*service.AccountLoadInfo{}, nil
	}

	// 使用 Pipeline 替代 Lua 脚本，兼容 Redis Cluster（Lua 内动态拼 key 会 CROSSSLOT）。
	// 热路径默认只发只读命令：ZCARD（并发数）+ GET（等待数）。
	// 过期槽位的清理（ZREMRANGEBYSCORE）已移至后台 worker 统一负责：
	// service/concurrency_service.go:StartSlotCleanupWorker，周期由
	// gateway.scheduling.slot_cleanup_interval 控制（默认 30s）。
	// 真正的入队路径（AcquireAccountSlot → acquireScript）在 ZADD 前仍会做原子清理，
	// 因此槽位不会无限累积；此处读到的计数最多滞后一个 worker 周期，对负载打分影响可忽略。
	// 收益：N=5000 账号时，从每请求 3N=15000 条 Redis 命令降到 2N=10000 条，
	// 且剥离了所有写命令（AOF/主从复制压力同步下降）。
	//
	// 边界：当 slot_cleanup_interval <= 0（worker 被显式禁用）时，wire 层会把
	// inlineCleanupOnRead 置 true，此时退化为原始的 3N 命令行为做正确性兜底，
	// 防止 stale 计数把账号锁死在虚高负载状态。
	var cutoffTime int64
	if c.inlineCleanupOnRead {
		now, err := c.rdb.Time(ctx).Result()
		if err != nil {
			return nil, fmt.Errorf("redis TIME: %w", err)
		}
		cutoffTime = now.Unix() - int64(c.slotTTLSeconds)
	}

	pipe := c.rdb.Pipeline()

	type accountCmds struct {
		id             int64
		maxConcurrency int
		zcardCmd       *redis.IntCmd
		getCmd         *redis.StringCmd
	}
	cmds := make([]accountCmds, 0, len(accounts))
	for _, acc := range accounts {
		slotKey := accountSlotKeyPrefix + strconv.FormatInt(acc.ID, 10)
		waitKey := accountWaitKeyPrefix + strconv.FormatInt(acc.ID, 10)
		if c.inlineCleanupOnRead {
			pipe.ZRemRangeByScore(ctx, slotKey, "-inf", strconv.FormatInt(cutoffTime, 10))
		}
		ac := accountCmds{
			id:             acc.ID,
			maxConcurrency: acc.MaxConcurrency,
			zcardCmd:       pipe.ZCard(ctx, slotKey),
			getCmd:         pipe.Get(ctx, waitKey),
		}
		cmds = append(cmds, ac)
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("pipeline exec: %w", err)
	}

	loadMap := make(map[int64]*service.AccountLoadInfo, len(accounts))
	for _, ac := range cmds {
		currentConcurrency := int(ac.zcardCmd.Val())
		waitingCount := 0
		if v, err := ac.getCmd.Int(); err == nil {
			waitingCount = v
		}
		loadRate := 0
		if ac.maxConcurrency > 0 {
			loadRate = (currentConcurrency + waitingCount) * 100 / ac.maxConcurrency
		}
		loadMap[ac.id] = &service.AccountLoadInfo{
			AccountID:          ac.id,
			CurrentConcurrency: currentConcurrency,
			WaitingCount:       waitingCount,
			LoadRate:           loadRate,
		}
	}

	return loadMap, nil
}

func (c *concurrencyCache) tryGetAccountsLoadBatchCustom(ctx context.Context, accounts []service.AccountWithConcurrency) (map[int64]*service.AccountLoadInfo, error, bool) {
	result, err := c.getAccountsLoadBatchCustom(ctx, accounts)
	return result, err, true
}

// SetInlineCleanupOnRead configures the fallback cleanup mode when the
// background slot cleanup worker is disabled.
func (c *concurrencyCache) SetInlineCleanupOnRead(enabled bool) {
	c.inlineCleanupOnRead = enabled
}
