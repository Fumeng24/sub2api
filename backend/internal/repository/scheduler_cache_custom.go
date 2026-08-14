package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

var setOutboxWatermarkScriptCustom = redis.NewScript(`
local current = redis.call('GET', KEYS[1])
local nextValue = tonumber(ARGV[1])

if nextValue == nil then
	return redis.error_reply('invalid outbox watermark')
end

if current ~= false then
	local currentValue = tonumber(current)
	if currentValue ~= nil and currentValue > nextValue then
		return 0
	end
end

redis.call('SET', KEYS[1], ARGV[1])
return 1
`)

func (c *schedulerCache) setAccountCustom(ctx context.Context, account *service.Account) (bool, error) {
	id := strconv.FormatInt(account.ID, 10)
	payload, err := marshalSchedulerFullAccount(*account)
	if err != nil {
		// A stale valid payload must not survive when the latest account cannot
		// be serialized. Treat this as a cache eviction, matching the bulk
		// snapshot writer's behavior.
		return true, c.DeleteAccount(ctx, account.ID)
	}

	existing, err := c.rdb.Get(ctx, schedulerAccountKey(id)).Bytes()
	if err != nil || !bytes.Equal(existing, payload) {
		return false, nil
	}

	metaExists, err := c.rdb.Exists(ctx, schedulerAccountMetaKey(id)).Result()
	if err != nil || metaExists != 1 {
		return false, nil
	}
	return true, nil
}

func (c *schedulerCache) setOutboxWatermarkCustom(ctx context.Context, id int64) error {
	_, err := setOutboxWatermarkScriptCustom.Run(
		ctx,
		c.rdb,
		[]string{schedulerOutboxWatermarkKey},
		strconv.FormatInt(id, 10),
	).Result()
	return err
}

func (c *schedulerCache) SetBucketMembers(ctx context.Context, bucket service.SchedulerBucket, accountIDs []int64) error {
	activeKey := schedulerBucketKey(schedulerActivePrefix, bucket)
	oldActive, _ := c.rdb.Get(ctx, activeKey).Result()

	version, err := c.rdb.Incr(ctx, schedulerBucketKey(schedulerVersionPrefix, bucket)).Result()
	if err != nil {
		return err
	}

	versionStr := strconv.FormatInt(version, 10)
	snapshotKey := schedulerSnapshotKey(bucket, versionStr)
	pipe := c.rdb.Pipeline()
	if len(accountIDs) > 0 {
		members := make([]redis.Z, 0, len(accountIDs))
		for idx, id := range accountIDs {
			members = append(members, redis.Z{Score: float64(idx), Member: strconv.FormatInt(id, 10)})
		}
		pipe.ZAdd(ctx, snapshotKey, members...)
	} else {
		pipe.Del(ctx, snapshotKey)
	}
	pipe.Set(ctx, activeKey, versionStr, 0)
	pipe.Set(ctx, schedulerBucketKey(schedulerReadyPrefix, bucket), "1", 0)
	pipe.SAdd(ctx, schedulerBucketSetKey, bucket.String())
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	if oldActive != "" && oldActive != versionStr {
		_ = c.rdb.Del(ctx, schedulerSnapshotKey(bucket, oldActive)).Err()
	}
	return nil
}

func (c *schedulerCache) RemoveAccountFromBuckets(ctx context.Context, accountID int64) error {
	if accountID <= 0 {
		return nil
	}

	buckets, err := c.rdb.SMembers(ctx, schedulerBucketSetKey).Result()
	if err != nil {
		return err
	}
	if len(buckets) == 0 {
		return nil
	}

	accountIDStr := strconv.FormatInt(accountID, 10)
	pipe := c.rdb.Pipeline()
	for _, bucketStr := range buckets {
		bucket, ok := service.ParseSchedulerBucket(bucketStr)
		if !ok {
			continue
		}
		activeVer, err := c.rdb.Get(ctx, schedulerBucketKey(schedulerActivePrefix, bucket)).Result()
		if err != nil || activeVer == "" {
			continue
		}
		pipe.ZRem(ctx, schedulerSnapshotKey(bucket, activeVer), accountIDStr)
	}

	_, err = pipe.Exec(ctx)
	return err
}

var schedulerCacheCredentialDenyList = []string{"id_token"}

func marshalSchedulerFullAccount(account service.Account) ([]byte, error) {
	cp := account
	if len(cp.Credentials) > 0 {
		filtered := make(map[string]any, len(cp.Credentials))
		for key, value := range cp.Credentials {
			if schedulerCacheCredentialDeniedCustom(key) {
				continue
			}
			filtered[key] = value
		}
		cp.Credentials = filtered
	}
	return json.Marshal(&cp)
}

func schedulerCacheCredentialDeniedCustom(key string) bool {
	for _, denied := range schedulerCacheCredentialDenyList {
		if key == denied {
			return true
		}
	}
	return false
}

func applySchedulerAccountGroupCustom(target *service.AccountGroup, source service.AccountGroup) {
	target.Role = source.NormalizedRole()
	target.Weight = source.EffectiveWeight()
	target.SortOrder = source.EffectiveSortOrder()
	target.SchedulingConfigured = true
}

func appendSchedulerExtraKeysCustom(keys []string) []string {
	return append(keys,
		"openai_compact_mode",
		"openai_compact_supported",
	)
}
