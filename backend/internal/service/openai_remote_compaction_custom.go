package service

import "github.com/gin-gonic/gin"

const (
	openAIRemoteCompactionV2Key     = "openai_remote_compaction_v2"
	openAICompactReasoningEffortKey = "openai_compact_reasoning_effort"
)

// MarkOpenAIRemoteCompactionV2 marks native remote compaction requests without
// rewriting their upstream endpoint.
func MarkOpenAIRemoteCompactionV2(c *gin.Context) {
	if c != nil {
		c.Set(openAIRemoteCompactionV2Key, true)
	}
}

func IsOpenAIRemoteCompactionV2(c *gin.Context) bool {
	if c == nil {
		return false
	}
	marked, ok := c.Get(openAIRemoteCompactionV2Key)
	isRemote, ok := marked.(bool)
	return ok && isRemote
}

func isOpenAILogicalCompactRequest(c *gin.Context) bool {
	return isOpenAIResponsesCompactPath(c) || IsOpenAIRemoteCompactionV2(c)
}

func setOpenAICompactReasoningEffort(c *gin.Context, effort string) {
	if c == nil {
		return
	}
	c.Set(openAICompactReasoningEffortKey, effort)
}

func openAICompactReasoningEffort(c *gin.Context) string {
	if c == nil {
		return ""
	}
	value, ok := c.Get(openAICompactReasoningEffortKey)
	if !ok {
		return ""
	}
	effort, _ := value.(string)
	return effort
}
