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

func TestBuildSchedulerMetadataAccount_KeepsOpenAIWSFlags(t *testing.T) {
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

func TestBuildSchedulerMetadataAccount_KeepsSlimGroupMembership(t *testing.T) {
	account := service.Account{
		ID:       42,
		Platform: service.PlatformAnthropic,
		GroupIDs: []int64{7, 9, 7, 0},
		AccountGroups: []service.AccountGroup{
			{
				AccountID: 42,
				GroupID:   7,
				Priority:  2,
				Account:   &service.Account{ID: 42, Name: "drop-from-metadata"},
				Group:     &service.Group{ID: 7, Name: "drop-from-metadata"},
			},
			{
				AccountID: 42,
				GroupID:   11,
				Priority:  3,
				Group:     &service.Group{ID: 11, Name: "drop-from-metadata"},
			},
			{
				AccountID: 42,
				GroupID:   0,
				Priority:  4,
			},
		},
	}

	got := buildSchedulerMetadataAccount(account)

	require.Equal(t, []int64{7, 9, 11}, got.GroupIDs)
	require.Len(t, got.AccountGroups, 2)
	require.Equal(t, int64(42), got.AccountGroups[0].AccountID)
	require.Equal(t, int64(7), got.AccountGroups[0].GroupID)
	require.Equal(t, 2, got.AccountGroups[0].Priority)
	require.Nil(t, got.AccountGroups[0].Account)
	require.Nil(t, got.AccountGroups[0].Group)
	require.Equal(t, int64(11), got.AccountGroups[1].GroupID)
	require.Nil(t, got.Groups)
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

func TestBuildSchedulerMetadataAccount_KeepsQuotaAutoPauseFields(t *testing.T) {
	account := service.Account{
		ID: 88,
		Extra: map[string]any{
			"codex_5h_used_percent":        12.34,
			"codex_7d_used_percent":        56.78,
			"codex_5h_reset_at":            "2026-05-29T10:00:00Z",
			"codex_7d_reset_at":            "2026-06-01T10:00:00Z",
			"codex_5h_reset_after_seconds": 300,
			"codex_7d_reset_after_seconds": 600,
			"codex_usage_updated_at":       "2026-05-29T09:00:00Z",
			"auto_pause_5h_threshold":      0.95,
			"auto_pause_7d_threshold":      0.96,
			"auto_pause_5h_disabled":       true,
			"auto_pause_7d_disabled":       false,
		},
	}

	got := buildSchedulerMetadataAccount(account)

	require.Equal(t, 12.34, got.Extra["codex_5h_used_percent"])
	require.Equal(t, 56.78, got.Extra["codex_7d_used_percent"])
	require.Equal(t, "2026-05-29T10:00:00Z", got.Extra["codex_5h_reset_at"])
	require.Equal(t, "2026-06-01T10:00:00Z", got.Extra["codex_7d_reset_at"])
	require.Equal(t, 300, got.Extra["codex_5h_reset_after_seconds"])
	require.Equal(t, 600, got.Extra["codex_7d_reset_after_seconds"])
	require.Equal(t, "2026-05-29T09:00:00Z", got.Extra["codex_usage_updated_at"])
	require.Equal(t, 0.95, got.Extra["auto_pause_5h_threshold"])
	require.Equal(t, 0.96, got.Extra["auto_pause_7d_threshold"])
	require.Equal(t, true, got.Extra["auto_pause_5h_disabled"])
	require.Equal(t, false, got.Extra["auto_pause_7d_disabled"])
}

func TestBuildSchedulerMetadataAccount_KeepsModelRateLimits(t *testing.T) {
	account := service.Account{
		ID:       90,
		Platform: service.PlatformAntigravity,
		Extra: map[string]any{
			"model_rate_limits": map[string]any{
				"gemini-3-flash": map[string]any{
					"rate_limit_reset_at": "2026-05-30T10:10:00Z",
				},
				"antigravity:gemini": map[string]any{
					"rate_limit_reset_at": "2026-05-30T10:10:00Z",
				},
			},
			"unused_large_field": "drop-me",
		},
	}

	got := buildSchedulerMetadataAccount(account)

	limits, ok := got.Extra["model_rate_limits"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, limits, "gemini-3-flash")
	require.Contains(t, limits, "antigravity:gemini")
	require.Nil(t, got.Extra["unused_large_field"])
}
