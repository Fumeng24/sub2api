package service

import "context"

type openAIWeakFallbackUpstreamContextKey struct{}
type openAINoHeaderTimeoutUpstreamContextKey struct{}

func HTTPUpstreamProfileForOpenAI(weakFallback, noHeaderTimeout bool) HTTPUpstreamProfile {
	if noHeaderTimeout {
		return HTTPUpstreamProfileOpenAINoHeaderTimeout
	}
	if weakFallback {
		return HTTPUpstreamProfileOpenAIWeakFallback
	}
	return HTTPUpstreamProfileOpenAI
}

func HTTPUpstreamProfileForOpenAIWeakFallback(weakFallback bool) HTTPUpstreamProfile {
	return HTTPUpstreamProfileForOpenAI(weakFallback, false)
}

func WithOpenAIWeakFallbackUpstream(ctx context.Context, enabled bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if !enabled {
		return ctx
	}
	return context.WithValue(ctx, openAIWeakFallbackUpstreamContextKey{}, true)
}

func OpenAIWeakFallbackUpstreamFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	enabled, _ := ctx.Value(openAIWeakFallbackUpstreamContextKey{}).(bool)
	return enabled
}

func WithOpenAINoHeaderTimeoutUpstream(ctx context.Context, enabled bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if !enabled {
		return ctx
	}
	return context.WithValue(ctx, openAINoHeaderTimeoutUpstreamContextKey{}, true)
}

func OpenAINoHeaderTimeoutUpstreamFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	enabled, _ := ctx.Value(openAINoHeaderTimeoutUpstreamContextKey{}).(bool)
	return enabled
}

func WithOpenAIHTTPUpstreamProfile(ctx context.Context) context.Context {
	return WithHTTPUpstreamProfile(ctx, HTTPUpstreamProfileForOpenAI(
		OpenAIWeakFallbackUpstreamFromContext(ctx),
		OpenAINoHeaderTimeoutUpstreamFromContext(ctx),
	))
}
