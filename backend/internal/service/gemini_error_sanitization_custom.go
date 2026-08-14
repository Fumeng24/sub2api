package service

import "github.com/Wei-Shaw/sub2api/internal/util/logredact"

func sanitizeGeminiUpstreamErrorCustom(message string) string {
	return logredact.RedactText(message)
}
