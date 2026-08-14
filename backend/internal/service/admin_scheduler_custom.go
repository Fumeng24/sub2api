package service

import (
	"context"
	"errors"
)

var errSchedulerHistoryReaderUnavailable = errors.New("scheduler history repository capability is unavailable")

type schedulerHistoryReader interface {
	ListByGroup(ctx context.Context, groupID int64, limit int) ([]SchedulerOutboxEvent, error)
}

func (s *adminServiceImpl) GetGroupSchedulerHistory(ctx context.Context, groupID int64, limit int) ([]SchedulerOutboxEvent, error) {
	if s.schedulerOutboxRepo == nil {
		return []SchedulerOutboxEvent{}, nil
	}
	historyRepo, ok := s.schedulerOutboxRepo.(schedulerHistoryReader)
	if !ok {
		return nil, errSchedulerHistoryReaderUnavailable
	}
	return historyRepo.ListByGroup(ctx, groupID, limit)
}
