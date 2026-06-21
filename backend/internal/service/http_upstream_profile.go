package service

import "context"

// HTTPUpstreamProfile marks HTTP upstream requests that need provider-specific
// transport policy.
type HTTPUpstreamProfile string

const (
	HTTPUpstreamProfileDefault               HTTPUpstreamProfile = ""
	HTTPUpstreamProfileOpenAI                HTTPUpstreamProfile = "openai"
	HTTPUpstreamProfileOpenAIWeakFallback    HTTPUpstreamProfile = "openai_weak_fallback"
	HTTPUpstreamProfileOpenAINoHeaderTimeout HTTPUpstreamProfile = "openai_no_header_timeout"
)

type httpUpstreamProfileContextKey struct{}
type openAIWeakFallbackUpstreamContextKey struct{}
type openAINoHeaderTimeoutUpstreamContextKey struct{}

// WithHTTPUpstreamProfile injects an upstream transport profile into ctx.
func WithHTTPUpstreamProfile(ctx context.Context, profile HTTPUpstreamProfile) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if profile == HTTPUpstreamProfileDefault {
		return ctx
	}
	return context.WithValue(ctx, httpUpstreamProfileContextKey{}, profile)
}

// HTTPUpstreamProfileFromContext resolves the upstream transport profile from ctx.
func HTTPUpstreamProfileFromContext(ctx context.Context) HTTPUpstreamProfile {
	if ctx == nil {
		return HTTPUpstreamProfileDefault
	}
	profile, ok := ctx.Value(httpUpstreamProfileContextKey{}).(HTTPUpstreamProfile)
	if !ok {
		return HTTPUpstreamProfileDefault
	}
	switch profile {
	case HTTPUpstreamProfileOpenAI, HTTPUpstreamProfileOpenAIWeakFallback, HTTPUpstreamProfileOpenAINoHeaderTimeout:
		return profile
	default:
		return HTTPUpstreamProfileDefault
	}
}

func HTTPUpstreamProfileForOpenAI(weakFallback bool, noHeaderTimeout bool) HTTPUpstreamProfile {
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
