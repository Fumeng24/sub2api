package handler

import "github.com/Wei-Shaw/sub2api/internal/service"

func openAIForwardNeedsCyberErrorUsage(result *service.OpenAIForwardResult, err error) bool {
	if err == nil {
		return false
	}
	return result == nil || result.ImageCount <= 0
}
