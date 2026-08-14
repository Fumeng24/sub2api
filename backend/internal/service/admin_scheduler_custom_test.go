package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type adminSchedulerBaseRepoStub struct{}

func (adminSchedulerBaseRepoStub) FirstCreatedAtAfter(context.Context, int64) (time.Time, bool, error) {
	return time.Time{}, false, nil
}

func (adminSchedulerBaseRepoStub) ListAfterAndReleaseDedup(context.Context, int64, int) ([]SchedulerOutboxEvent, error) {
	return nil, nil
}

func (adminSchedulerBaseRepoStub) MaxID(context.Context) (int64, error) {
	return 0, nil
}

func (adminSchedulerBaseRepoStub) DeleteConsumedUpTo(context.Context, int64, int) (int64, error) {
	return 0, nil
}

func (adminSchedulerBaseRepoStub) TryAcquireCleanupLock(context.Context) (SchedulerOutboxCleanupLease, bool, error) {
	return nil, false, nil
}

type adminSchedulerHistoryRepoStub struct {
	adminSchedulerBaseRepoStub
	events  []SchedulerOutboxEvent
	err     error
	groupID int64
	limit   int
}

func (r *adminSchedulerHistoryRepoStub) ListByGroup(_ context.Context, groupID int64, limit int) ([]SchedulerOutboxEvent, error) {
	r.groupID = groupID
	r.limit = limit
	return r.events, r.err
}

func TestGetGroupSchedulerHistoryWithoutRepositoryReturnsEmpty(t *testing.T) {
	service := &adminServiceImpl{}

	events, err := service.GetGroupSchedulerHistory(context.Background(), 12, 30)

	require.NoError(t, err)
	require.Empty(t, events)
}

func TestGetGroupSchedulerHistoryUsesOptionalCapability(t *testing.T) {
	want := []SchedulerOutboxEvent{{ID: 7, EventType: "account_changed"}}
	repo := &adminSchedulerHistoryRepoStub{events: want}
	service := &adminServiceImpl{AdminServiceCustomDependencies: AdminServiceCustomDependencies{schedulerOutboxRepo: repo}}

	events, err := service.GetGroupSchedulerHistory(context.Background(), 12, 30)

	require.NoError(t, err)
	require.Equal(t, want, events)
	require.Equal(t, int64(12), repo.groupID)
	require.Equal(t, 30, repo.limit)
}

func TestGetGroupSchedulerHistoryRejectsMissingCapability(t *testing.T) {
	service := &adminServiceImpl{AdminServiceCustomDependencies: AdminServiceCustomDependencies{schedulerOutboxRepo: adminSchedulerBaseRepoStub{}}}

	events, err := service.GetGroupSchedulerHistory(context.Background(), 12, 30)

	require.Nil(t, events)
	require.ErrorIs(t, err, errSchedulerHistoryReaderUnavailable)
}

func TestGetGroupSchedulerHistoryPropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("history query failed")
	repo := &adminSchedulerHistoryRepoStub{err: wantErr}
	service := &adminServiceImpl{AdminServiceCustomDependencies: AdminServiceCustomDependencies{schedulerOutboxRepo: repo}}

	events, err := service.GetGroupSchedulerHistory(context.Background(), 12, 30)

	require.Nil(t, events)
	require.ErrorIs(t, err, wantErr)
}
