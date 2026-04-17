//go:build unit

package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSchedulerCache_SetBucketMembers(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	cache := NewSchedulerCache(rdb).(*schedulerCache)
	ctx := context.Background()

	bucket := service.SchedulerBucket{GroupID: 1, Platform: "anthropic", Mode: "single"}
	accountIDs := []int64{100, 200, 300}

	mock.ExpectGet("sched:active:1:anthropic:single").SetVal("4")
	mock.ExpectIncr("sched:ver:1:anthropic:single").SetVal(5)
	mock.ExpectZAdd("sched:1:anthropic:single:v5",
		redis.Z{Score: 0, Member: "100"},
		redis.Z{Score: 1, Member: "200"},
		redis.Z{Score: 2, Member: "300"},
	).SetVal(3)
	mock.ExpectSet("sched:active:1:anthropic:single", "5", 0).SetVal("OK")
	mock.ExpectSet("sched:ready:1:anthropic:single", "1", 0).SetVal("OK")
	mock.ExpectSAdd("sched:buckets", "1:anthropic:single").SetVal(1)
	mock.ExpectDel("sched:1:anthropic:single:v4").SetVal(1)

	err := cache.SetBucketMembers(ctx, bucket, accountIDs)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerCache_RemoveAccountFromBuckets(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	cache := NewSchedulerCache(rdb).(*schedulerCache)
	ctx := context.Background()

	mock.ExpectSMembers("sched:buckets").SetVal([]string{"1:anthropic:single", "0:openai:single"})
	mock.ExpectGet("sched:active:1:anthropic:single").SetVal("5")
	mock.ExpectGet("sched:active:0:openai:single").SetVal("2")
	mock.ExpectZRem("sched:1:anthropic:single:v5", "123").SetVal(1)
	mock.ExpectZRem("sched:0:openai:single:v2", "123").SetVal(0)

	err := cache.RemoveAccountFromBuckets(ctx, 123)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildSchedulerMetadataAccount_KeepsOpenAIWSFlags(t *testing.T) {
	account := service.Account{
		ID:       42,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Extra: map[string]any{
			"openai_oauth_responses_websockets_v2_enabled": true,
			"openai_oauth_responses_websockets_v2_mode":    service.OpenAIWSIngressModePassthrough,
			"openai_ws_force_http":                         true,
			"mixed_scheduling":                             true,
			"unused_large_field":                           "drop-me",
		},
	}

	got := buildSchedulerMetadataAccount(account)

	require.Equal(t, true, got.Extra["openai_oauth_responses_websockets_v2_enabled"])
	require.Equal(t, service.OpenAIWSIngressModePassthrough, got.Extra["openai_oauth_responses_websockets_v2_mode"])
	require.Equal(t, true, got.Extra["openai_ws_force_http"])
	require.Equal(t, true, got.Extra["mixed_scheduling"])
	require.Nil(t, got.Extra["unused_large_field"])
}
