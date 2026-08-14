package service

import "time"

const (
	openAICompactBadOutputCode               = "compact_bad_output"
	openAICompactContextWindowFallbackCode   = "compact_context_window_fallback"
	openAICompactMinOutputRunes              = 8
	openAICompactLargeInputTokenThreshold    = 4096
	openAICompactLargeInputMinOutputTokens   = 16
	openAICompactLargeInputMinOutputRunes    = 80
	openAIEmptyOutputCode                    = "empty_effective_output"
	openAICodexImageBridgeAutoDisableTimeout = 5 * time.Second
)
