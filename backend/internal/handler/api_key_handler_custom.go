package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

// GetSubscriptionCapability exposes only the site capability flag, without
// leaking subscription group details.
func (h *APIKeyHandler) GetSubscriptionCapability(c *gin.Context) {
	if _, ok := middleware.GetAuthSubjectFromContext(c); !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	hasGroups, err := h.apiKeyService.HasActiveSubscriptionGroups(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"has_subscription_groups": hasGroups})
}
