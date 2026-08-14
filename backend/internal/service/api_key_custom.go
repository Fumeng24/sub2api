package service

import "github.com/Wei-Shaw/sub2api/internal/domain"

const (
	APIKeyCategoryOpenAI    = domain.APIKeyCategoryOpenAI
	APIKeyCategoryAnthropic = domain.APIKeyCategoryAnthropic
	APIKeyCategoryOther     = domain.APIKeyCategoryOther
)

func NormalizeAPIKeyCategory(category string) (string, bool) {
	switch category {
	case "", APIKeyCategoryOther:
		return APIKeyCategoryOther, true
	case APIKeyCategoryOpenAI:
		return APIKeyCategoryOpenAI, true
	case APIKeyCategoryAnthropic:
		return APIKeyCategoryAnthropic, true
	default:
		return "", false
	}
}
