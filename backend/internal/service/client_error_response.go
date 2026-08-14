package service

import "github.com/gin-gonic/gin"

func writeClientClaudeError(c *gin.Context, status int, errType, message string) {
	message = ClientFacingErrorMessage(status, errType, message)
	c.JSON(status, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}

func writeClientOpenAIError(c *gin.Context, status int, errType, message string) {
	message = ClientFacingErrorMessage(status, errType, message)
	c.JSON(status, gin.H{
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}
