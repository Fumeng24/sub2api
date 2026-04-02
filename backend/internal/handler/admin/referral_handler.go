package admin

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ReferralHandler struct {
	referralService *service.ReferralService
}

func NewReferralHandler(referralService *service.ReferralService) *ReferralHandler {
	return &ReferralHandler{referralService: referralService}
}

// GetStats 获取平台邀请统计
func (h *ReferralHandler) GetStats(c *gin.Context) {
	stats, err := h.referralService.GetPlatformStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// List 获取平台邀请统计数据
func (h *ReferralHandler) List(c *gin.Context) {
	stats, err := h.referralService.GetPlatformStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
