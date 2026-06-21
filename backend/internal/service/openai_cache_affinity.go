package service

import (
	"context"
	"fmt"
	"strings"
)

type openAICacheAffinityContextKey struct{}

var openAICacheAffinityKey = openAICacheAffinityContextKey{}

// WithOpenAICacheAffinityHash binds a stable account-affinity key to the request.
// It is separate from the client session hash: sessionHash can change per turn,
// while this key keeps long-context cache traffic on the last healthy account.
func WithOpenAICacheAffinityHash(ctx context.Context, affinityHash string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	affinityHash = strings.TrimSpace(affinityHash)
	if affinityHash == "" {
		return ctx
	}
	return context.WithValue(ctx, openAICacheAffinityKey, affinityHash)
}

func openAICacheAffinityHashFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(openAICacheAffinityKey).(string)
	return strings.TrimSpace(value)
}

// BuildOpenAICacheAffinityHash returns a stable hash used only for account
// affinity. It intentionally does not replace GenerateSessionHash, because the
// existing session hash is also used for request/session limits.
func BuildOpenAICacheAffinityHash(userID int64, apiKeyID int64, groupID *int64, model string, endpoint string, body []byte) string {
	if userID <= 0 || apiKeyID <= 0 {
		return ""
	}
	model = strings.TrimSpace(model)
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = "/v1/responses"
	}
	if model == "" {
		return ""
	}

	seed := deriveOpenAIContentSessionSeed(body)
	if seed == "" {
		seed = "model=" + model
	}
	return DeriveSessionHashFromSeed(fmt.Sprintf(
		"openai_cache_affinity:%d:%d:%d:%s:%s:%s",
		derefGroupID(groupID),
		userID,
		apiKeyID,
		model,
		endpoint,
		seed,
	))
}
