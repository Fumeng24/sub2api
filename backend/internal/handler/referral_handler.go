package handler

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
)

type ReferralHandler struct {
	referralService *service.ReferralService
	settingService  *service.SettingService
}

func NewReferralHandler(referralService *service.ReferralService, settingService *service.SettingService) *ReferralHandler {
	return &ReferralHandler{
		referralService: referralService,
		settingService:  settingService,
	}
}

// GetReferralInfo 获取当前用户的邀请信息
func (h *ReferralHandler) GetReferralInfo(c *gin.Context) {
	if !h.settingService.IsReferralEnabled(c.Request.Context()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "referral feature is not enabled"})
		return
	}

	userID, _ := middleware2.GetAuthSubjectFromContext(c)

	siteBaseURL := c.Request.Header.Get("Origin")
	if siteBaseURL == "" {
		siteBaseURL = c.Request.Header.Get("Referer")
	}

	info, err := h.referralService.GetMyReferralInfo(c.Request.Context(), userID, siteBaseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

// GetInvitees 获取被邀请人列表
func (h *ReferralHandler) GetInvitees(c *gin.Context) {
	if !h.settingService.IsReferralEnabled(c.Request.Context()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "referral feature is not enabled"})
		return
	}

	userID, _ := middleware2.GetAuthSubjectFromContext(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	invitees, paginationResult, err := h.referralService.ListMyInvitees(c.Request.Context(), userID, pagination.PaginationParams{Page: page, PageSize: pageSize})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       invitees,
		"pagination": paginationResult,
	})
}

// GetRewardSummary 获取奖励汇总
func (h *ReferralHandler) GetRewardSummary(c *gin.Context) {
	if !h.settingService.IsReferralEnabled(c.Request.Context()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "referral feature is not enabled"})
		return
	}

	userID, _ := middleware2.GetAuthSubjectFromContext(c)

	totalInvitees, err := h.referralService.CountInvitees(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalReward, err := h.referralService.SumRewards(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_invitees": totalInvitees,
		"total_reward":   totalReward,
	})
}
