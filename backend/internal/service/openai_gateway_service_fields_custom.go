package service

import (
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type openAIGatewayServiceCustomFields struct {
	openaiTransientCooldownThrottle *accountWriteThrottle
	openaiSlowReserveOnce           sync.Once
	openaiSlowReserve               *openAIAccountSlowReserveState
}

func configureOpenAIGatewayServiceCustom(svc *OpenAIGatewayService, cfg *config.Config) {
	if svc == nil {
		return
	}
	svc.openaiTransientCooldownThrottle = newAccountWriteThrottle(
		openAIRuntimeCooldownsFromConfig(cfg).transientPersistMinInterval,
	)
}
