package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

// WithSchedulerEndpoint stores the logical endpoint used by runtime scheduling
// health. It is intentionally separate from usage logging because account
// selection happens before the final upstream platform is known.
func WithSchedulerEndpoint(ctx context.Context, endpoint string) context.Context {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxkey.SchedulerEndpoint, endpoint)
}

func schedulerEndpointFromContext(ctx context.Context, fallback string) string {
	if ctx != nil {
		if endpoint, ok := ctx.Value(ctxkey.SchedulerEndpoint).(string); ok && strings.TrimSpace(endpoint) != "" {
			return strings.TrimSpace(endpoint)
		}
	}
	return strings.TrimSpace(fallback)
}
