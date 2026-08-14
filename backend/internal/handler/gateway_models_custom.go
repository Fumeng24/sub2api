package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func writeConfiguredGatewayModelsCustom(c *gin.Context, apiKey *service.APIKey, platform string) bool {
	if apiKey == nil || apiKey.Group == nil || !apiKey.Group.CustomModelsListEnabled() {
		return false
	}
	writeCustomModelsList(c, platform, mergeModelIDs(apiKey.Group.ModelsListConfig.Models, nil))
	return true
}
