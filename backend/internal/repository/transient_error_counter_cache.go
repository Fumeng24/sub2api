package repository

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const transientErrorCounterPrefix = "transient_5xx_count:account:"
const transientErrorRequestCounterPrefix = "transient_5xx_request:account:"

var transientErrorCounterIncrScript = redis.NewScript(`
	local key = KEYS[1]
	local ttl = tonumber(ARGV[1])
	local count = redis.call('INCR', key)
	if count == 1 then
		redis.call('EXPIRE', key, ttl)
	end
	return count
`)

var transientErrorCounterIncrOnceScript = redis.NewScript(`
	local countKey = KEYS[1]
	local requestKey = KEYS[2]
	local ttl = tonumber(ARGV[1])
	if redis.call('SET', requestKey, '1', 'NX', 'EX', ttl) then
		local count = redis.call('INCR', countKey)
		if count == 1 then
			redis.call('EXPIRE', countKey, ttl)
		end
		return { count, 1 }
	end
	local current = redis.call('GET', countKey)
	if not current then
		current = '0'
	end
	return { tonumber(current), 0 }
`)

type transientErrorCounterCache struct {
	rdb *redis.Client
}

func NewTransientErrorCounterCache(rdb *redis.Client) service.TransientErrorCounterCache {
	return &transientErrorCounterCache{rdb: rdb}
}

func (c *transientErrorCounterCache) IncrementTransientErrorCount(ctx context.Context, accountID int64, windowMinutes int) (int64, error) {
	key := fmt.Sprintf("%s%d", transientErrorCounterPrefix, accountID)
	ttlSeconds := windowMinutes * 60
	if ttlSeconds < 60 {
		ttlSeconds = 60
	}
	count, err := transientErrorCounterIncrScript.Run(ctx, c.rdb, []string{key}, ttlSeconds).Int64()
	if err != nil {
		return 0, fmt.Errorf("increment transient error count: %w", err)
	}
	return count, nil
}

func (c *transientErrorCounterCache) IncrementTransientErrorCountOnce(ctx context.Context, accountID int64, requestID string, windowMinutes int) (int64, bool, error) {
	key := fmt.Sprintf("%s%d", transientErrorCounterPrefix, accountID)
	ttlSeconds := windowMinutes * 60
	if ttlSeconds < 60 {
		ttlSeconds = 60
	}
	sum := sha256.Sum256([]byte(requestID))
	requestKey := fmt.Sprintf("%s%d:%x", transientErrorRequestCounterPrefix, accountID, sum[:])
	values, err := transientErrorCounterIncrOnceScript.Run(ctx, c.rdb, []string{key, requestKey}, ttlSeconds).Int64Slice()
	if err != nil {
		return 0, false, fmt.Errorf("increment transient error count once: %w", err)
	}
	if len(values) != 2 {
		return 0, false, fmt.Errorf("increment transient error count once: unexpected result length %d", len(values))
	}
	return values[0], values[1] == 1, nil
}

func (c *transientErrorCounterCache) ResetTransientErrorCount(ctx context.Context, accountID int64) error {
	key := fmt.Sprintf("%s%d", transientErrorCounterPrefix, accountID)
	return c.rdb.Del(ctx, key).Err()
}
