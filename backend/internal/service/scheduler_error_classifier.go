package service

import "strings"

type schedulerErrorClassifier interface {
	Classify(statusCode int, body []byte, failoverErr *UpstreamFailoverError) string
}

type defaultSchedulerErrorClassifier struct{}

func (defaultSchedulerErrorClassifier) Classify(statusCode int, body []byte, failoverErr *UpstreamFailoverError) string {
	if failoverErr != nil && strings.TrimSpace(failoverErr.SchedulerCategory) != "" {
		return strings.TrimSpace(failoverErr.SchedulerCategory)
	}
	return schedulerFailureCategory(statusCode, body)
}

func schedulerClassifierForPlatform(platform string) schedulerErrorClassifier {
	switch strings.TrimSpace(strings.ToLower(platform)) {
	case PlatformOpenAI, PlatformAnthropic, PlatformGemini:
		return defaultSchedulerErrorClassifier{}
	default:
		return defaultSchedulerErrorClassifier{}
	}
}
