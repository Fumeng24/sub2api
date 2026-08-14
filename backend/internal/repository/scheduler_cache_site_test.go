//go:build unit

package repository

import (
	"context"
	"encoding/json"
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

func TestBuildSchedulerMetadataAccount_KeepsOpenAIWSFlags_SiteExtension(t *testing.T) {
	account := service.Account{
		ID:       42,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"gpt-5.5": "gpt-5.5",
			},
			"compact_model_mapping": map[string]any{
				"gpt-5.5": "gpt-5.5-openai-compact",
			},
			"id_token": "drop-me",
		},
		Extra: map[string]any{
			"openai_oauth_responses_websockets_v2_enabled": true,
			"openai_oauth_responses_websockets_v2_mode":    service.OpenAIWSIngressModePassthrough,
			"openai_ws_force_http":                         true,
			"openai_responses_mode":                        "force_chat_completions",
			"openai_responses_supported":                   false,
			"openai_compact_mode":                          service.OpenAICompactModeForceOff,
			"openai_compact_supported":                     false,
			"mixed_scheduling":                             true,
			"unused_large_field":                           "drop-me",
		},
	}

	got := buildSchedulerMetadataAccount(account)

	require.NotNil(t, got.Credentials["model_mapping"])
	require.NotNil(t, got.Credentials["compact_model_mapping"])
	require.Nil(t, got.Credentials["id_token"])
	require.Equal(t, true, got.Extra["openai_oauth_responses_websockets_v2_enabled"])
	require.Equal(t, service.OpenAIWSIngressModePassthrough, got.Extra["openai_oauth_responses_websockets_v2_mode"])
	require.Equal(t, true, got.Extra["openai_ws_force_http"])
	require.Equal(t, "force_chat_completions", got.Extra["openai_responses_mode"])
	require.Equal(t, false, got.Extra["openai_responses_supported"])
	require.Equal(t, service.OpenAICompactModeForceOff, got.Extra["openai_compact_mode"])
	require.Equal(t, false, got.Extra["openai_compact_supported"])
	require.Equal(t, true, got.Extra["mixed_scheduling"])
	require.Nil(t, got.Extra["unused_large_field"])
}

func TestMarshalSchedulerFullAccount_KeepsFullGroupPayload(t *testing.T) {
	account := service.Account{
		ID:       42,
		Platform: service.PlatformAnthropic,
		Credentials: map[string]any{
			"access_token": "keep-access-token",
			"id_token":     "drop-id-token",
		},
		AccountGroups: []service.AccountGroup{
			{
				AccountID: 42,
				GroupID:   7,
				Group:     &service.Group{ID: 7, Name: "keep-nested-group"},
			},
		},
		Groups: []*service.Group{{ID: 7, Name: "keep-group-list"}},
	}

	payload, err := marshalSchedulerFullAccount(account)
	require.NoError(t, err)

	var got service.Account
	require.NoError(t, json.Unmarshal(payload, &got))
	require.Equal(t, "keep-access-token", got.GetCredential("access_token"))
	require.Empty(t, got.GetCredential("id_token"))
	require.Len(t, got.AccountGroups, 1)
	require.NotNil(t, got.AccountGroups[0].Group)
	require.Equal(t, "keep-nested-group", got.AccountGroups[0].Group.Name)
	require.Len(t, got.Groups, 1)
	require.Equal(t, "keep-group-list", got.Groups[0].Name)
}
