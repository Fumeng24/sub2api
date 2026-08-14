package service

type gatewayServiceCustomDependencies struct {
	slotPoolService          SlotPoolService
	geminiTokenProvider      *GeminiTokenProvider
	antigravityTokenProvider *AntigravityTokenProvider
}

type accountSelectionResultCustom struct {
	WeakFallback         bool
	WeakFallbackReason   string
	BypassOpenAIHeaderTO bool
	GroupReserve         bool
	GroupReserveReason   string
}

func shouldClearStickySessionCustom(account *Account, requestedModel string) (bool, bool) {
	if stickySessionClearReason(account, requestedModel) == "" {
		return false, false
	}
	return true, true
}

func (s *GatewayService) SetAntigravityTokenProvider(provider *AntigravityTokenProvider) {
	if s != nil {
		s.antigravityTokenProvider = provider
	}
}
