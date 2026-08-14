package service

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	openAIRequestErrorCooldown                = 10 * time.Second
	openAITransientCooldownPersistMinInterval = 5 * time.Second
	openAIAccountCircuitHalfOpenProbeTTL      = 60 * time.Second
)

func schedulerCooldownForCategoryWithOpenAIRequestError(category string, headers http.Header, requestErrorCooldown time.Duration) time.Duration {
	_ = headers
	_ = category
	return requestErrorCooldown
}

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

func openAIRuntimeCooldownSeconds(seconds int, fallback time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}
	maxSeconds := int64(time.Duration(1<<63-1) / time.Second)
	if int64(seconds) > maxSeconds {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func openAIRuntimeCooldownsFromConfig(cfg *config.Config) openAIRuntimeCooldowns {
	cooldowns := defaultOpenAIRuntimeCooldowns()
	if cfg == nil {
		return cooldowns
	}
	runtime := cfg.Gateway.OpenAIScheduler.RuntimeCooldowns
	cooldowns.oAuth429Fallback = openAIRuntimeCooldownSeconds(runtime.OAuth429FallbackCooldownSeconds, cooldowns.oAuth429Fallback)
	cooldowns.requestError = openAIRuntimeCooldownSeconds(runtime.RequestErrorCooldownSeconds, cooldowns.requestError)
	cooldowns.transientPersistMinInterval = openAIRuntimeCooldownSeconds(runtime.TransientCooldownPersistMinIntervalSeconds, cooldowns.transientPersistMinInterval)
	cooldowns.stopSchedulingBridge = openAIRuntimeCooldownSeconds(runtime.StopSchedulingBridgeCooldownSeconds, cooldowns.stopSchedulingBridge)
	cooldowns.accountCircuitHalfOpenProbeTTL = openAIRuntimeCooldownSeconds(runtime.AccountCircuitHalfOpenProbeTTLSeconds, cooldowns.accountCircuitHalfOpenProbeTTL)
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
