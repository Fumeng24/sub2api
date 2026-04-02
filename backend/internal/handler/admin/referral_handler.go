package admin

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ReferralHandler handles admin referral management endpoints
type ReferralHandler struct {
	referralService *service.ReferralService
}

// NewReferralHandler creates a new admin ReferralHandler
func NewReferralHandler(referralService *service.ReferralService) *ReferralHandler {
	return &ReferralHandler{
		referralService: referralService,
	}
}

// GetStats handles getting platform-wide referral statistics
// GET /api/v1/admin/referral/stats
func (h *ReferralHandler) GetStats(c *gin.Context) {
	stats, err := h.referralService.GetPlatformStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// List handles getting platform-wide referral statistics (alias for stats)
// GET /api/v1/admin/referral/list
func (h *ReferralHandler) List(c *gin.Context) {
	h.GetStats(c)
}
