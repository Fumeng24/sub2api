package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type openAICodexImageBridgeAppliedContextKey struct{}

func withOpenAICodexImageBridgeApplied(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, openAICodexImageBridgeAppliedContextKey{}, true)
}

func openAICodexImageBridgeApplied(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	applied, _ := ctx.Value(openAICodexImageBridgeAppliedContextKey{}).(bool)
	return applied
}

func isOpenAICodexImageBridgeUnsupportedError(message string, payload []byte) bool {
	text := normalizeOpenAIUpstreamErrorText(message, payload)
	return containsAnyOpenAIErrorText(text, imageGenerationPermissionMessage)
}

func (s *OpenAIGatewayService) autoDisableCodexImageBridgeForUnsupportedUpstream(ctx context.Context, account *Account, message string, payload []byte) bool {
	if !openAICodexImageBridgeApplied(ctx) {
		return false
	}
	if !isOpenAICodexImageBridgeUnsupportedError(message, payload) {
		return false
	}
	if s == nil || s.accountRepo == nil || account == nil || account.ID <= 0 || account.Platform != PlatformOpenAI {
		return false
	}
	if override := account.CodexImageGenerationBridgeOverride(); override != nil && !*override {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openAICodexImageBridgeAutoDisableTimeout)
	defer cancel()

	if err := s.accountRepo.UpdateExtra(dbCtx, account.ID, map[string]any{featureKeyCodexImageGenerationBridge: false}); err != nil {
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Auto-disable Codex image_generation bridge failed: account=%d name=%s err=%v", account.ID, account.Name, err)
		return true
	}
	if account.Extra == nil {
		account.Extra = make(map[string]any)
	}
	account.Extra[featureKeyCodexImageGenerationBridge] = false
	logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Auto-disabled Codex image_generation bridge after upstream unsupported error: account=%d name=%s", account.ID, account.Name)
	return true
}
