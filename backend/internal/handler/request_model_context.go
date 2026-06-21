package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func requestModelFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if model, ok := c.Get(opsModelKey); ok {
		if value, ok := model.(string); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
