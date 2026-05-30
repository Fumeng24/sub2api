package handler

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const failoverAccountSelectionTimeout = 5 * time.Second

func failoverAccountSelectionContext(parent context.Context, schedulerEndpoint string, detachFromCancel bool) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if detachFromCancel {
		parent = context.WithoutCancel(parent)
	}
	ctx, cancel := context.WithTimeout(parent, failoverAccountSelectionTimeout)
	return service.WithSchedulerEndpoint(ctx, schedulerEndpoint), cancel
}

func openAIAccountSelectionContext(parent context.Context, schedulerEndpoint string, detachFromCancel bool) (context.Context, context.CancelFunc) {
	return failoverAccountSelectionContext(parent, schedulerEndpoint, detachFromCancel)
}
