package service

func (s *RateLimitService) SetTransientErrorCounterCache(cache TransientErrorCounterCache) {
	s.transientErrorCounter = cache
}

func (s *RateLimitService) SetPolicyExtension(extension RateLimitPolicyExtension) {
	s.policyExtension = extension
}
