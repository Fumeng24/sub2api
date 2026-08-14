package repository

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (s *httpUpstreamService) CloseIdleConnectionsForAccount(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	isolation := s.getIsolationMode()
	s.mu.Lock()
	defer s.mu.Unlock()
	matched := false
	for key, entry := range s.clients {
		if cacheKeyMatchesAccount(key, accountID) {
			s.removeClientLocked(key, entry)
			matched = true
		}
	}
	if matched || isolation != config.ConnectionPoolIsolationProxy {
		return
	}
	for key, entry := range s.clients {
		s.removeClientLocked(key, entry)
	}
}

func cacheKeyMatchesAccount(key string, accountID int64) bool {
	if accountID <= 0 {
		return false
	}
	accountKey := fmt.Sprintf("account:%d", accountID)
	tlsAccountKey := "tls:" + accountKey
	return key == accountKey || strings.HasPrefix(key, accountKey+"|") ||
		key == tlsAccountKey || strings.HasPrefix(key, tlsAccountKey+"|")
}

func isOpenAIHTTPUpstreamProfile(profile service.HTTPUpstreamProfile) bool {
	switch profile {
	case service.HTTPUpstreamProfileOpenAI,
		service.HTTPUpstreamProfileOpenAIWeakFallback,
		service.HTTPUpstreamProfileOpenAINoHeaderTimeout:
		return true
	default:
		return false
	}
}
