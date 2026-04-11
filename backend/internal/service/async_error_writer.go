package service

import (
	"context"
	"strconv"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"golang.org/x/sync/singleflight"
)

// AsyncErrorWriter provides non-blocking, deduplicated SetError calls.
// Multiple concurrent SetError calls for the same account are collapsed
// into a single database write via singleflight.
type AsyncErrorWriter struct {
	accountRepo AccountRepository
	group       singleflight.Group
	wg          sync.WaitGroup
}

// NewAsyncErrorWriter creates a new AsyncErrorWriter.
func NewAsyncErrorWriter(accountRepo AccountRepository) *AsyncErrorWriter {
	return &AsyncErrorWriter{
		accountRepo: accountRepo,
	}
}

// SetError schedules an async SetError call. The call is deduplicated
// using singleflight: concurrent calls for the same accountID are collapsed.
// Returns immediately without waiting for the database write.
func (w *AsyncErrorWriter) SetError(ctx context.Context, accountID int64, errorMsg string) {
	if w == nil || w.accountRepo == nil || accountID <= 0 {
		return
	}

	key := strconv.FormatInt(accountID, 10)

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		// Use background context to avoid cancellation from request context
		bgCtx := context.Background()

		_, _, _ = w.group.Do(key, func() (any, error) {
			if err := w.accountRepo.SetError(bgCtx, accountID, errorMsg); err != nil {
				logger.LegacyPrintf("service.async_error_writer", "[AsyncSetError] failed: account=%d err=%v", accountID, err)
				return nil, err
			}
			return nil, nil
		})
	}()
}

// Wait blocks until all pending SetError calls have completed.
// Useful for graceful shutdown.
func (w *AsyncErrorWriter) Wait() {
	if w == nil {
		return
	}
	w.wg.Wait()
}
