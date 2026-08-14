//go:build unit

package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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
		channelMonitorServiceCustomFields: channelMonitorServiceCustomFields{apiKeyRepo: &channelMonitorAPIKeyRepoStub{
			byKey: map[string]*APIKey{
				"shared-key": {
					ID:     42,
					Key:    "shared-key",
					Status: StatusActive,
				},
			},
		}},
	}
	existing := &ChannelMonitor{APIKeyID: int64Ptr(99), APIKey: "ENC:old"}
	plain, updated, err := svc.applyAPIKeyUpdateCustom(context.Background(), existing, ChannelMonitorUpdateParams{
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
		channelMonitorServiceCustomFields: channelMonitorServiceCustomFields{apiKeyRepo: &channelMonitorAPIKeyRepoStub{
			byKey: map[string]*APIKey{
				"shared-key": {
					ID:     42,
					Key:    "shared-key",
					Status: StatusActive,
				},
			},
		}},
	}
	existing := &ChannelMonitor{}
	plain, updated, err := svc.applyAPIKeyUpdateCustom(context.Background(), existing, ChannelMonitorUpdateParams{
		APIKey: stringPtr("shared-key"),
	})

	require.NoError(t, err)
	require.True(t, updated)
	require.Equal(t, "shared-key", plain)
	require.NotNil(t, existing.APIKeyID)
	require.Equal(t, int64(42), *existing.APIKeyID)
	require.Equal(t, "ENC:shared-key", existing.APIKey)
}

func TestValidateCreateParamsRejectsInvalidSortOrder(t *testing.T) {
	t.Parallel()

	err := validateCreateParams(ChannelMonitorCreateParams{
		Name:            "monitor",
		Provider:        MonitorProviderAnthropic,
		Endpoint:        "https://example.com",
		APIKey:          "sk-test",
		PrimaryModel:    "claude-sonnet",
		IntervalSeconds: monitorMinIntervalSeconds,
		SortOrder:       -1,
	})

	require.ErrorIs(t, err, ErrChannelMonitorInvalidSortOrder)
}

func TestApplyMonitorUpdateRejectsInvalidSortOrder(t *testing.T) {
	t.Parallel()

	existing := &ChannelMonitor{
		Provider:         MonitorProviderAnthropic,
		APIMode:          MonitorAPIModeChatCompletions,
		SortOrder:        10,
		IntervalSeconds:  monitorMinIntervalSeconds,
		BodyOverrideMode: MonitorBodyOverrideModeOff,
	}
	err := applyMonitorUpdate(existing, ChannelMonitorUpdateParams{
		SortOrder: channelMonitorIntPtr(monitorMaxSortOrder + 1),
	})

	require.ErrorIs(t, err, ErrChannelMonitorInvalidSortOrder)
	require.Equal(t, 10, existing.SortOrder)
}

func TestLatestUserMonitorCheckedAtUsesNewestTimelinePoint(t *testing.T) {
	t.Parallel()

	older := time.Date(2026, 6, 28, 1, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 28, 2, 30, 0, 0, time.UTC)
	latest := latestUserMonitorCheckedAt([]*UserMonitorView{
		{
			ID: 1,
			Timeline: []UserMonitorTimelinePoint{
				{CheckedAt: older},
			},
		},
		{
			ID: 2,
			Timeline: []UserMonitorTimelinePoint{
				{CheckedAt: newer},
				{CheckedAt: time.Time{}},
			},
		},
	})

	require.NotNil(t, latest)
	require.Equal(t, newer, *latest)
}

func channelMonitorIntPtr(v int) *int {
	return &v
}

type failingChannelMonitorEncryptor struct{}

func (f failingChannelMonitorEncryptor) Encrypt(string) (string, error) {
	return "", errors.New("encrypt failed")
}

func (f failingChannelMonitorEncryptor) Decrypt(string) (string, error) {
	return "", errors.New("decrypt: cipher: message authentication failed")
}

type channelMonitorRepoRunCheckStub struct {
	monitor      *ChannelMonitor
	historyRows  []*ChannelMonitorHistoryRow
	markChecked  int
	insertCalled int
}

func (s *channelMonitorRepoRunCheckStub) Create(context.Context, *ChannelMonitor) error {
	panic("unexpected Create call")
}

func (s *channelMonitorRepoRunCheckStub) GetByID(context.Context, int64) (*ChannelMonitor, error) {
	if s.monitor == nil {
		return nil, ErrChannelMonitorNotFound
	}
	clone := *s.monitor
	if s.monitor.ExtraModels != nil {
		clone.ExtraModels = append([]string(nil), s.monitor.ExtraModels...)
	}
	return &clone, nil
}

func (s *channelMonitorRepoRunCheckStub) Update(context.Context, *ChannelMonitor) error {
	panic("unexpected Update call")
}

func (s *channelMonitorRepoRunCheckStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (s *channelMonitorRepoRunCheckStub) List(context.Context, ChannelMonitorListParams) ([]*ChannelMonitor, int64, error) {
	panic("unexpected List call")
}

func (s *channelMonitorRepoRunCheckStub) UpdateSortOrders(context.Context, []ChannelMonitorSortOrderUpdate) error {
	panic("unexpected UpdateSortOrders call")
}

func (s *channelMonitorRepoRunCheckStub) FindByDuplicateOperationID(context.Context, string) (*ChannelMonitor, error) {
	return nil, ErrChannelMonitorNotFound
}

func (s *channelMonitorRepoRunCheckStub) ListEnabled(context.Context) ([]*ChannelMonitor, error) {
	panic("unexpected ListEnabled call")
}

func (s *channelMonitorRepoRunCheckStub) MarkChecked(context.Context, int64, time.Time) error {
	s.markChecked++
	return nil
}

func (s *channelMonitorRepoRunCheckStub) InsertHistoryBatch(_ context.Context, rows []*ChannelMonitorHistoryRow) error {
	s.insertCalled++
	s.historyRows = append(s.historyRows, rows...)
	return nil
}

func (s *channelMonitorRepoRunCheckStub) DeleteHistoryBefore(context.Context, time.Time) (int64, error) {
	panic("unexpected DeleteHistoryBefore call")
}

func (s *channelMonitorRepoRunCheckStub) ListHistory(context.Context, int64, string, int) ([]*ChannelMonitorHistoryEntry, error) {
	panic("unexpected ListHistory call")
}

func (s *channelMonitorRepoRunCheckStub) ListLatestPerModel(context.Context, int64) ([]*ChannelMonitorLatest, error) {
	panic("unexpected ListLatestPerModel call")
}

func (s *channelMonitorRepoRunCheckStub) ComputeAvailability(context.Context, int64, int) ([]*ChannelMonitorAvailability, error) {
	panic("unexpected ComputeAvailability call")
}

func (s *channelMonitorRepoRunCheckStub) ListLatestForMonitorIDs(context.Context, []int64) (map[int64][]*ChannelMonitorLatest, error) {
	panic("unexpected ListLatestForMonitorIDs call")
}

func (s *channelMonitorRepoRunCheckStub) ComputeAvailabilityForMonitors(context.Context, []int64, int) (map[int64][]*ChannelMonitorAvailability, error) {
	panic("unexpected ComputeAvailabilityForMonitors call")
}

func (s *channelMonitorRepoRunCheckStub) ListRecentHistoryForMonitors(context.Context, []int64, map[int64]string, int) (map[int64][]*ChannelMonitorHistoryEntry, error) {
	panic("unexpected ListRecentHistoryForMonitors call")
}

func (s *channelMonitorRepoRunCheckStub) UpsertDailyRollupsFor(context.Context, time.Time) (int64, error) {
	panic("unexpected UpsertDailyRollupsFor call")
}

func (s *channelMonitorRepoRunCheckStub) DeleteRollupsBefore(context.Context, time.Time) (int64, error) {
	panic("unexpected DeleteRollupsBefore call")
}

func (s *channelMonitorRepoRunCheckStub) LoadAggregationWatermark(context.Context) (*time.Time, error) {
	panic("unexpected LoadAggregationWatermark call")
}

func (s *channelMonitorRepoRunCheckStub) UpdateAggregationWatermark(context.Context, time.Time) error {
	panic("unexpected UpdateAggregationWatermark call")
}

func TestChannelMonitorServiceRunCheckPersistsDecryptFailureHistory(t *testing.T) {
	t.Parallel()

	repo := &channelMonitorRepoRunCheckStub{
		monitor: &ChannelMonitor{
			ID:           7,
			Name:         "broken monitor",
			Provider:     "openai",
			APIKey:       "ciphertext",
			PrimaryModel: "gpt-5.5",
			ExtraModels:  []string{"gpt-5.4-mini"},
			Enabled:      true,
		},
	}
	svc := NewChannelMonitorService(repo, failingChannelMonitorEncryptor{})

	results, err := svc.RunCheck(context.Background(), 7)

	require.ErrorIs(t, err, ErrChannelMonitorAPIKeyDecryptFailed)
	require.Nil(t, results)
	require.Equal(t, 1, repo.insertCalled)
	require.Equal(t, 1, repo.markChecked)
	require.Len(t, repo.historyRows, 2)
	require.Equal(t, []string{"gpt-5.5", "gpt-5.4-mini"}, []string{repo.historyRows[0].Model, repo.historyRows[1].Model})
	for _, row := range repo.historyRows {
		require.Equal(t, MonitorStatusError, row.Status)
		require.True(t, strings.Contains(row.Message, "decrypt_failed"), row.Message)
	}
}

func TestChannelMonitorRuntimeKeyErrorKind(t *testing.T) {
	require.Equal(t, "decrypt_failed", channelMonitorRuntimeKeyErrorKind(ErrChannelMonitorAPIKeyDecryptFailed))
	require.Equal(t, "api_key_missing", channelMonitorRuntimeKeyErrorKind(ErrChannelMonitorMissingAPIKey))
	require.Equal(t, "linked_api_key_unavailable", channelMonitorRuntimeKeyErrorKind(ErrAPIKeyExpired))
	require.Equal(t, "api_key_unavailable", channelMonitorRuntimeKeyErrorKind(errors.New("unexpected key resolver error")))
}
