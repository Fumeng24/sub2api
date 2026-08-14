package service

import (
	"strconv"
	"strings"
)

// InvalidateGroup removes every cached rate for a group regardless of the
// user's fixed or relative configuration.
func (r *userGroupRateResolver) InvalidateGroup(groupID int64) {
	if r == nil || r.cache == nil || groupID <= 0 {
		return
	}
	for key := range r.cache.Items() {
		parts := strings.SplitN(key, ":", 3)
		if len(parts) != 3 {
			continue
		}
		cachedGroupID, err := strconv.ParseInt(parts[1], 10, 64)
		if err == nil && cachedGroupID == groupID {
			r.cache.Delete(key)
		}
	}
}

// InvalidateUserGroupRateCache makes pricing coefficient changes take effect
// immediately in the non-OpenAI gateway.
func (s *GatewayService) InvalidateUserGroupRateCache(groupID int64) {
	if s == nil {
		return
	}
	s.userGroupRateResolver.InvalidateGroup(groupID)
}

// InvalidateUserGroupRateCache makes pricing coefficient changes take effect
// immediately in the OpenAI gateway.
func (s *OpenAIGatewayService) InvalidateUserGroupRateCache(groupID int64) {
	if s == nil {
		return
	}
	s.userGroupRateResolver.InvalidateGroup(groupID)
}
