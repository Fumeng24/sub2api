package service

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type openAIRuntimeCooldowns struct {
	oAuth429Fallback               time.Duration
	requestError                   time.Duration
	transientPersistMinInterval    time.Duration
	stopSchedulingBridge           time.Duration
	accountCircuitHalfOpenProbeTTL time.Duration
}

func defaultOpenAIRuntimeCooldowns() openAIRuntimeCooldowns {
	return openAIRuntimeCooldowns{
		oAuth429Fallback:               openAIOAuth429FallbackCooldown,
		requestError:                   openAIRequestErrorCooldown,
		transientPersistMinInterval:    openAITransientCooldownPersistMinInterval,
		stopSchedulingBridge:           openAIStopSchedulingBridgeCooldown,
		accountCircuitHalfOpenProbeTTL: openAIAccountCircuitHalfOpenProbeTTL,
	}
}

func openAIRuntimeCooldownsFromConfig(cfg *config.Config) openAIRuntimeCooldowns {
	cooldowns := defaultOpenAIRuntimeCooldowns()
	if cfg == nil {
		return cooldowns
	}
	runtime := cfg.Gateway.OpenAIScheduler.RuntimeCooldowns
	if runtime.OAuth429FallbackCooldownSeconds > 0 {
		cooldowns.oAuth429Fallback = time.Duration(runtime.OAuth429FallbackCooldownSeconds) * time.Second
	}
	if runtime.RequestErrorCooldownSeconds > 0 {
		cooldowns.requestError = time.Duration(runtime.RequestErrorCooldownSeconds) * time.Second
	}
	if runtime.TransientCooldownPersistMinIntervalSeconds > 0 {
		cooldowns.transientPersistMinInterval = time.Duration(runtime.TransientCooldownPersistMinIntervalSeconds) * time.Second
	}
	if runtime.StopSchedulingBridgeCooldownSeconds > 0 {
		cooldowns.stopSchedulingBridge = time.Duration(runtime.StopSchedulingBridgeCooldownSeconds) * time.Second
	}
	if runtime.AccountCircuitHalfOpenProbeTTLSeconds > 0 {
		cooldowns.accountCircuitHalfOpenProbeTTL = time.Duration(runtime.AccountCircuitHalfOpenProbeTTLSeconds) * time.Second
	}
	return cooldowns
}

func (s *OpenAIGatewayService) openAIRuntimeCooldowns() openAIRuntimeCooldowns {
	if s == nil {
		return defaultOpenAIRuntimeCooldowns()
	}
	return openAIRuntimeCooldownsFromConfig(s.cfg)
}

func (s *OpenAIGatewayService) openAISchedulerCooldownForCategory(category string, headers http.Header) time.Duration {
	return schedulerCooldownForCategoryWithOpenAIRequestError(category, headers, s.openAIRuntimeCooldowns().requestError)
}
