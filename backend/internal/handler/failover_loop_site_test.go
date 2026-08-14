package handler

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestHandleFailoverError_SingleAccountRetrySkipsTempUnschedule(t *testing.T) {
	mock := &mockTempUnscheduler{}
	fs := NewFailoverState(3, false)
	fs.SameAccountRetryCount[100] = maxSameAccountRetries
	ctx := service.WithSingleAccountRetry(context.Background(), true, false)
	err := newTestFailoverErr(502, true, false)

	action := fs.HandleFailoverError(ctx, mock, 100, service.PlatformAnthropic, maxSameAccountRetries, err)

	require.Equal(t, FailoverContinue, action)
	require.Equal(t, 1, fs.SwitchCount)
	require.Contains(t, fs.FailedAccountIDs, int64(100))
	require.Empty(t, mock.calls, "single-candidate retry must not temp-unschedule the only account")
}

func TestHandleSelectionExhausted_RetryOnSelectionExhausted(t *testing.T) {
	fs := NewFailoverState(3, false)
	fs.LastFailoverErr = &service.UpstreamFailoverError{
		StatusCode:                400,
		RetryOnSelectionExhausted: true,
	}
	fs.SwitchCount = 1
	fs.FailedAccountIDs[100] = struct{}{}

	start := time.Now()
	action := fs.HandleSelectionExhausted(context.Background())
	elapsed := time.Since(start)

	require.Equal(t, FailoverContinue, action)
	require.Empty(t, fs.FailedAccountIDs)
	require.Equal(t, 2, fs.SwitchCount, "耗尽后重试也应消耗切换预算，避免单账号无限循环")
	require.GreaterOrEqual(t, elapsed, singleAccountBackoffDelay-100*time.Millisecond)
	require.Less(t, elapsed, singleAccountBackoffDelay+2*time.Second)
}
