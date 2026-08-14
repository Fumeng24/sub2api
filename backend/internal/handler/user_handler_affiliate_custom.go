package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

type BindAffiliateInviterRequest struct {
	Code string `json:"code"`
}

// BindAffiliateInviter binds the current user to an affiliate inviter by code.
// POST /api/v1/user/aff/bind
func (h *UserHandler) BindAffiliateInviter(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req BindAffiliateInviterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		response.BadRequest(c, "Affiliate code is required")
		return
	}

	if err := h.affiliateService.BindInviterByCode(c.Request.Context(), subject.UserID, code); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	detail, err := h.affiliateService.GetAffiliateDetail(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, detail)
}

// ClaimAffiliateBindBonus lets an eligible newly registered invitee claim the
// configured bind bonus manually after their inviter has been bound.
// POST /api/v1/user/aff/bind-bonus/claim
func (h *UserHandler) ClaimAffiliateBindBonus(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	balance, err := h.affiliateService.ClaimBindBonus(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	detail, err := h.affiliateService.GetAffiliateDetail(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"balance": balance,
		"detail":  detail,
	})
}
