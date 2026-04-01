package admin

import (
	"net/http"
	"strconv"

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

// List 分页获取所有邀请关系（复用 inviter 列表，不带 inviterID 筛选暂用统计）
func (h *ReferralHandler) List(c *gin.Context) {
	_ = strconv.Atoi(c.DefaultQuery("page", "1"))
	_ = strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 简化实现：返回平台统计数据
	stats, err := h.referralService.GetPlatformStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
