package service

import (
	"log/slog"
	"strings"
	"time"
)

const schedulerSlowTTFTCategory = "transient_timeout"

var schedulerHealthyTTFTThreshold = 15 * time.Second

func schedulerTTFTWithinHealthyThreshold(ttftMs int) bool {
	if ttftMs <= 0 {
		return false
	}
	if schedulerHealthyTTFTThreshold <= 0 {
		return true
	}
	return time.Duration(ttftMs)*time.Millisecond <= schedulerHealthyTTFTThreshold
}

func schedulerStreamingTTFTIsSlow(endpoint string, firstTokenMs *int) bool {
	if firstTokenMs == nil || *firstTokenMs <= 0 || schedulerTTFTWithinHealthyThreshold(*firstTokenMs) {
		return false
	}
	return schedulerTTFTPolicyEndpointAllowed(endpoint)
}

func schedulerTTFTPolicyEndpointAllowed(endpoint string) bool {
	endpoint = normalizeSchedulerDimension(endpoint, defaultSchedulerEndpoint)
	if isOpenAIImageGenerationSchedulerEndpoint(endpoint) || strings.HasPrefix(endpoint, "images:") {
		return false
	}
	switch endpoint {
	case "/v1/chat/completions", "/v1/responses", "/v1/responses/compact", "/v1/messages", PlatformAnthropic, PlatformGemini, defaultSchedulerEndpoint:
		return true
	default:
		if strings.HasPrefix(endpoint, "gemini:") || strings.HasPrefix(endpoint, "anthropic:") {
			return true
		}
		return false
	}
}

func schedulerReportStreamingSuccess(
	health *accountSchedulerHealthStats,
	accountID int64,
	model string,
	endpoint string,
	firstTokenMs *int,
) bool {
	if health == nil || accountID <= 0 {
		return false
	}
	if schedulerStreamingTTFTIsSlow(endpoint, firstTokenMs) {
		health.reportFailure(accountID, model, endpoint, schedulerSlowTTFTCategory, schedulerCooldownForCategory(schedulerSlowTTFTCategory, nil))
		return true
	}
	health.reportSuccess(accountID, model, endpoint, firstTokenMs)
	return false
}

func schedulerLogSlowStreamingTTFT(accountID int64, model, endpoint string, firstTokenMs *int, source string) {
	ttftMs := 0
	if firstTokenMs != nil {
		ttftMs = *firstTokenMs
	}
	slog.Warn("account_slow_stream_ttft_circuit_open",
		"account_id", accountID,
		"model", model,
		"endpoint", endpoint,
		"ttft_ms", ttftMs,
		"healthy_ttft_ms", schedulerHealthyTTFTThreshold.Milliseconds(),
		"source", strings.TrimSpace(source),
	)
}
