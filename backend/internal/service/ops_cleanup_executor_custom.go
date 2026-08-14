package service

import "fmt"

type opsCleanupDeletedCountsCustom struct {
	schedulerOutbox  int64
	schedulerHistory int64
}

func (c opsCleanupDeletedCounts) customSuffix() string {
	return fmt.Sprintf(" scheduler_outbox=%d scheduler_history=%d", c.schedulerOutbox, c.schedulerHistory)
}
