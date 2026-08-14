package handler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGatewayHandlerSubmitUsageRecordTask_DropPolicyDoesNotSyncFallback(t *testing.T) {
	pool := service.NewUsageRecordWorkerPoolWithOptions(service.UsageRecordWorkerPoolOptions{
		WorkerCount:           1,
		QueueSize:             1,
		TaskTimeout:           time.Second,
		OverflowPolicy:        "drop",
		OverflowSamplePercent: 0,
		AutoScaleEnabled:      false,
	})
	t.Cleanup(pool.Stop)
	h := &GatewayHandler{usageRecordWorkerPool: pool}

	block := make(chan struct{})
	release := make(chan struct{})
	pool.Submit(func(context.Context) {
		close(block)
		<-release
	})
	<-block
	pool.Submit(func(context.Context) {})

	var called atomic.Bool
	h.submitUsageRecordTask(context.Background(), func(context.Context) {
		called.Store(true)
	})
	close(release)

	require.False(t, called.Load(), "explicit drop policy must not be overridden by synchronous fallback")
}

func TestOpenAIGatewayHandlerSubmitUsageRecordTask_DropPolicyDoesNotSyncFallback(t *testing.T) {
	pool := service.NewUsageRecordWorkerPoolWithOptions(service.UsageRecordWorkerPoolOptions{
		WorkerCount:           1,
		QueueSize:             1,
		TaskTimeout:           time.Second,
		OverflowPolicy:        "drop",
		OverflowSamplePercent: 0,
		AutoScaleEnabled:      false,
	})
	t.Cleanup(pool.Stop)
	h := &OpenAIGatewayHandler{usageRecordWorkerPool: pool}

	block := make(chan struct{})
	release := make(chan struct{})
	pool.Submit(func(context.Context) {
		close(block)
		<-release
	})
	<-block
	pool.Submit(func(context.Context) {})

	var called atomic.Bool
	h.submitUsageRecordTask(context.Background(), func(context.Context) {
		called.Store(true)
	})
	close(release)

	require.False(t, called.Load(), "explicit drop policy must not be overridden by synchronous fallback")
}
