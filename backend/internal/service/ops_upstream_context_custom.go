package service

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func normalizeOpsUpstreamErrorEventCustom(c *gin.Context, ev *OpsUpstreamErrorEvent) {
	if c == nil || ev == nil {
		return
	}
	ev.UpstreamRequestBody = strings.TrimSpace(ev.UpstreamRequestBody)
	if !ev.CooldownApplied && ev.CooldownReason == "" &&
		strings.TrimSpace(ev.Platform) == PlatformOpenAI && strings.TrimSpace(ev.Kind) == "failover" && isOpenAITransient5xxStatus(ev.UpstreamStatusCode) {
		ev.CooldownApplied = true
		ev.CooldownReason = "openai_transient_5xx"
	}
	ev.CooldownReason = strings.TrimSpace(ev.CooldownReason)
	if ev.CooldownReason != "" {
		ev.CooldownReason = truncateString(sanitizeUpstreamErrorMessage(ev.CooldownReason), 128)
	}
	if ev.UpstreamRequestBody != "" {
		return
	}
	if value, ok := c.Get(OpsUpstreamRequestBodyKey); ok {
		switch raw := value.(type) {
		case string:
			ev.UpstreamRequestBody = strings.TrimSpace(raw)
		case []byte:
			ev.UpstreamRequestBody = strings.TrimSpace(string(raw))
		}
	}
}
