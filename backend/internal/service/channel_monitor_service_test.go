//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type channelMonitorAPIKeyRepoStub struct {
	apiKeyRepoStub
	byID  map[int64]*APIKey
	byKey map[string]*APIKey
}

func (s *channelMonitorAPIKeyRepoStub) GetByID(_ context.Context, id int64) (*APIKey, error) {
	if s.byID == nil {
		return nil, ErrAPIKeyNotFound
	}
	apiKey, ok := s.byID[id]
	if !ok || apiKey == nil {
		return nil, ErrAPIKeyNotFound
	}
	clone := *apiKey
	return &clone, nil
}

func (s *channelMonitorAPIKeyRepoStub) GetByKey(_ context.Context, key string) (*APIKey, error) {
	if s.byKey == nil {
		return nil, ErrAPIKeyNotFound
	}
	apiKey, ok := s.byKey[key]
	if !ok || apiKey == nil {
		return nil, ErrAPIKeyNotFound
	}
	clone := *apiKey
	return &clone, nil
}

func TestChannelMonitorServiceApplyAPIKeyUpdateClearsLinkedKeyWhenManualKeyDoesNotMatch(t *testing.T) {
	t.Parallel()

	svc := &ChannelMonitorService{
		encryptor: &plainEncryptor{},
		apiKeyRepo: &channelMonitorAPIKeyRepoStub{
			byKey: map[string]*APIKey{
				"shared-key": {
					ID:     42,
					Key:    "shared-key",
					Status: StatusActive,
				},
			},
		},
	}
	existing := &ChannelMonitor{APIKeyID: int64Ptr(99), APIKey: "ENC:old"}
	plain, updated, err := svc.applyAPIKeyUpdate(context.Background(), existing, ChannelMonitorUpdateParams{
		APIKey: stringPtr("manual-key"),
	})

	require.NoError(t, err)
	require.True(t, updated)
	require.Equal(t, "manual-key", plain)
	require.Nil(t, existing.APIKeyID)
	require.Equal(t, "ENC:manual-key", existing.APIKey)
}

func TestChannelMonitorServiceApplyAPIKeyUpdateBindsMatchingManualKey(t *testing.T) {
	t.Parallel()

	svc := &ChannelMonitorService{
		encryptor: &plainEncryptor{},
		apiKeyRepo: &channelMonitorAPIKeyRepoStub{
			byKey: map[string]*APIKey{
				"shared-key": {
					ID:     42,
					Key:    "shared-key",
					Status: StatusActive,
				},
			},
		},
	}
	existing := &ChannelMonitor{}
	plain, updated, err := svc.applyAPIKeyUpdate(context.Background(), existing, ChannelMonitorUpdateParams{
		APIKey: stringPtr("shared-key"),
	})

	require.NoError(t, err)
	require.True(t, updated)
	require.Equal(t, "shared-key", plain)
	require.NotNil(t, existing.APIKeyID)
	require.Equal(t, int64(42), *existing.APIKeyID)
	require.Equal(t, "ENC:shared-key", existing.APIKey)
}
