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

func writeClientGoogleError(c *gin.Context, status int, message string) {
	message = ClientFacingErrorMessage(status, "", message)
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    status,
			"message": message,
			"status":  googleHTTPStatus(status),
		},
	})
}

func googleHTTPStatus(status int) string {
	switch status {
	case 400:
		return "INVALID_ARGUMENT"
	case 401:
		return "UNAUTHENTICATED"
	case 403:
		return "PERMISSION_DENIED"
	case 404:
		return "NOT_FOUND"
	case 409:
		return "ABORTED"
	case 429:
		return "RESOURCE_EXHAUSTED"
	case 499:
		return "CANCELLED"
	case 500:
		return "INTERNAL"
	case 501:
		return "UNIMPLEMENTED"
	case 503:
		return "UNAVAILABLE"
	case 504:
		return "DEADLINE_EXCEEDED"
	default:
		return "UNKNOWN"
	}
}
