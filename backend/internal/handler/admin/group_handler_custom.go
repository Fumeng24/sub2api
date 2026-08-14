package admin

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type groupHandlerCustom struct {
	concurrencyService *service.ConcurrencyService
}

type createGroupRequestCustom struct {
	ForceOpenAIPriority bool                        `json:"force_openai_priority"`
	OpenAIStableLowTTFT bool                        `json:"openai_stable_low_ttft"`
	AutoSortConfig      service.GroupAutoSortConfig `json:"auto_sort_config"`
}

type updateGroupRequestCustom struct {
	ForceOpenAIPriority *bool                        `json:"force_openai_priority"`
	OpenAIStableLowTTFT *bool                        `json:"openai_stable_low_ttft"`
	AutoSortConfig      *service.GroupAutoSortConfig `json:"auto_sort_config"`
}

func applyCreateGroupInputCustom(req *CreateGroupRequest, input *service.CreateGroupInput) {
	input.ForceOpenAIPriority = req.ForceOpenAIPriority
	input.OpenAIStableLowTTFT = req.OpenAIStableLowTTFT
	input.AutoSortConfig = req.AutoSortConfig
}

func applyUpdateGroupInputCustom(req *UpdateGroupRequest, input *service.UpdateGroupInput) {
	input.ForceOpenAIPriority = req.ForceOpenAIPriority
	input.OpenAIStableLowTTFT = req.OpenAIStableLowTTFT
	input.AutoSortConfig = req.AutoSortConfig
}

type groupAccountSchedulingService interface {
	GetGroupAccountScheduling(ctx context.Context, groupID int64) ([]service.AccountSchedulingEntry, error)
	UpdateGroupAccountScheduling(ctx context.Context, groupID int64, configs []service.AccountSchedulingConfig) error
}

type groupSchedulerHistoryService interface {
	GetGroupSchedulerHistory(ctx context.Context, groupID int64, limit int) ([]service.SchedulerOutboxEvent, error)
}

type groupRelativeRateService interface {
	GetGroupRelativeRateMultipliers(ctx context.Context, groupID int64) ([]service.GroupRelativeRateMultiplierEntry, error)
	SyncGroupRelativeRateMultipliers(ctx context.Context, groupID int64, entries []service.GroupRelativeRateMultiplierInput) error
}

type groupRateChangeNotificationService interface {
	PreviewGroupRateChangeNotification(ctx context.Context, groupID int64, input service.GroupRateChangeNotificationInput) (*service.GroupRateChangeNotificationPreview, error)
	SendGroupRateChangeNotification(ctx context.Context, groupID int64, input service.GroupRateChangeNotificationInput) (*service.GroupRateChangeNotificationSendResult, error)
}

type groupRelativeRateMultipliersRequest struct {
	Entries []service.GroupRelativeRateMultiplierInput `json:"entries"`
}

type GroupRateChangeNotificationRequest struct {
	NewRateMultiplier float64    `json:"new_rate_multiplier"`
	WindowMinutes     int        `json:"window_minutes"`
	EffectiveAt       *time.Time `json:"effective_at"`
	Message           string     `json:"message"`
}

type AccountSchedulingConfigRequest struct {
	AccountID            int64  `json:"account_id" binding:"required"`
	Role                 string `json:"role" binding:"omitempty,oneof=primary backup"`
	Weight               int    `json:"weight"`
	SortOrder            int    `json:"sort_order"`
	SchedulingConfigured *bool  `json:"scheduling_configured"`
}

type UpdateAccountSchedulingRequest struct {
	Accounts []AccountSchedulingConfigRequest `json:"accounts" binding:"required"`
}

type groupSchedulerHistoryItemResponse struct {
	ID        int64          `json:"id"`
	EventType string         `json:"event_type"`
	AccountID *int64         `json:"account_id,omitempty"`
	GroupID   *int64         `json:"group_id,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// GetAccountScheduling returns per-group account role/weight/sort configuration.
func (h *GroupHandler) GetAccountScheduling(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	schedulingSvc, ok := h.adminService.(groupAccountSchedulingService)
	if !ok {
		response.Error(c, 500, "Account scheduling service is not configured")
		return
	}

	entries, err := schedulingSvc.GetGroupAccountScheduling(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"accounts": h.accountSchedulingResponse(c.Request.Context(), entries)})
}

// UpdateAccountScheduling updates per-group account role/weight/sort configuration.
func (h *GroupHandler) UpdateAccountScheduling(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	var req UpdateAccountSchedulingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	configs := make([]service.AccountSchedulingConfig, 0, len(req.Accounts))
	for _, item := range req.Accounts {
		configs = append(configs, service.AccountSchedulingConfig{
			AccountID:            item.AccountID,
			Role:                 item.Role,
			Weight:               item.Weight,
			SortOrder:            item.SortOrder,
			SchedulingConfigured: item.SchedulingConfigured == nil || *item.SchedulingConfigured,
		})
	}

	schedulingSvc, ok := h.adminService.(groupAccountSchedulingService)
	if !ok {
		response.Error(c, 500, "Account scheduling service is not configured")
		return
	}
	if err := schedulingSvc.UpdateGroupAccountScheduling(c.Request.Context(), groupID, configs); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	entries, err := schedulingSvc.GetGroupAccountScheduling(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"accounts": h.accountSchedulingResponse(c.Request.Context(), entries)})
}

func (h *GroupHandler) accountSchedulingResponse(ctx context.Context, entries []service.AccountSchedulingEntry) []dto.AccountSchedulingEntry {
	accountIDs := make([]int64, 0, len(entries))
	for i := range entries {
		if entries[i].Account != nil {
			accountIDs = append(accountIDs, entries[i].Account.ID)
		}
	}

	concurrencyCounts := make(map[int64]int)
	if h.concurrencyService != nil && len(accountIDs) > 0 {
		if counts, err := h.concurrencyService.GetAccountConcurrencyBatch(ctx, accountIDs); err == nil && counts != nil {
			concurrencyCounts = counts
		}
	}

	out := make([]dto.AccountSchedulingEntry, 0, len(entries))
	for i := range entries {
		item := dto.AccountSchedulingEntryFromService(&entries[i])
		if item == nil {
			continue
		}
		if item.Account != nil {
			item.Account.CurrentConcurrency = concurrencyCounts[entries[i].Account.ID]
		}
		out = append(out, *item)
	}
	return out
}

// History returns recent scheduler events for a group.
func (h *GroupHandler) History(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || groupID <= 0 {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	limit := 30
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if value, parseErr := strconv.Atoi(raw); parseErr == nil && value > 0 {
			if value > 100 {
				value = 100
			}
			limit = value
		}
	}

	svc, ok := h.adminService.(groupSchedulerHistoryService)
	if !ok {
		response.Success(c, gin.H{"items": []groupSchedulerHistoryItemResponse{}})
		return
	}
	events, err := svc.GetGroupSchedulerHistory(c.Request.Context(), groupID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]groupSchedulerHistoryItemResponse, 0, len(events))
	for _, event := range events {
		out = append(out, groupSchedulerHistoryItemResponse{
			ID:        event.ID,
			EventType: event.EventType,
			AccountID: event.AccountID,
			GroupID:   event.GroupID,
			Payload:   event.Payload,
			CreatedAt: event.CreatedAt,
		})
	}
	response.Success(c, gin.H{"items": out})
}

// GetRelativeRateMultipliers returns the relative user pricing coefficients
// for a group. Fixed final rates are included only as conflict markers.
func (h *GroupHandler) GetRelativeRateMultipliers(c *gin.Context) {
	groupID, ok := parseRelativeRateGroupID(c)
	if !ok {
		return
	}
	svc, ok := h.adminService.(groupRelativeRateService)
	if !ok {
		response.Error(c, 500, "Relative rate service is not configured")
		return
	}
	entries, err := svc.GetGroupRelativeRateMultipliers(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if entries == nil {
		entries = []service.GroupRelativeRateMultiplierEntry{}
	}
	response.Success(c, entries)
}

// UpdateRelativeRateMultipliers fully replaces relative coefficients for a
// group. It intentionally does not change fixed rates or RPM overrides.
func (h *GroupHandler) UpdateRelativeRateMultipliers(c *gin.Context) {
	groupID, ok := parseRelativeRateGroupID(c)
	if !ok {
		return
	}
	var req groupRelativeRateMultipliersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	svc, ok := h.adminService.(groupRelativeRateService)
	if !ok {
		response.Error(c, 500, "Relative rate service is not configured")
		return
	}
	if err := svc.SyncGroupRelativeRateMultipliers(c.Request.Context(), groupID, req.Entries); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Relative rate multipliers updated successfully"})
}

// ClearRelativeRateMultipliers removes only relative coefficients for a group.
func (h *GroupHandler) ClearRelativeRateMultipliers(c *gin.Context) {
	groupID, ok := parseRelativeRateGroupID(c)
	if !ok {
		return
	}
	svc, ok := h.adminService.(groupRelativeRateService)
	if !ok {
		response.Error(c, 500, "Relative rate service is not configured")
		return
	}
	if err := svc.SyncGroupRelativeRateMultipliers(c.Request.Context(), groupID, nil); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Relative rate multipliers cleared successfully"})
}

func parseRelativeRateGroupID(c *gin.Context) (int64, bool) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || groupID <= 0 {
		response.BadRequest(c, "Invalid group ID")
		return 0, false
	}
	return groupID, true
}

// PreviewGroupRateChangeNotification previews recent users affected by a group rate change.
func (h *GroupHandler) PreviewGroupRateChangeNotification(c *gin.Context) {
	groupID, req, ok := bindGroupRateChangeNotification(c)
	if !ok {
		return
	}
	svc, ok := h.adminService.(groupRateChangeNotificationService)
	if !ok {
		response.Error(c, 500, "Group rate notification service is not configured")
		return
	}
	preview, err := svc.PreviewGroupRateChangeNotification(c.Request.Context(), groupID, req.serviceInput())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, preview)
}

// SendGroupRateChangeNotification sends advance notice emails to recent group users.
func (h *GroupHandler) SendGroupRateChangeNotification(c *gin.Context) {
	groupID, req, ok := bindGroupRateChangeNotification(c)
	if !ok {
		return
	}
	svc, ok := h.adminService.(groupRateChangeNotificationService)
	if !ok {
		response.Error(c, 500, "Group rate notification service is not configured")
		return
	}
	result, err := svc.SendGroupRateChangeNotification(c.Request.Context(), groupID, req.serviceInput())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func bindGroupRateChangeNotification(c *gin.Context) (int64, GroupRateChangeNotificationRequest, bool) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return 0, GroupRateChangeNotificationRequest{}, false
	}

	var req GroupRateChangeNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return 0, req, false
	}
	if req.NewRateMultiplier <= 0 {
		response.BadRequest(c, "new_rate_multiplier must be > 0")
		return 0, req, false
	}
	if req.WindowMinutes > 24*60 {
		response.BadRequest(c, "window_minutes must be <= 1440")
		return 0, req, false
	}
	return groupID, req, true
}

func (r GroupRateChangeNotificationRequest) serviceInput() service.GroupRateChangeNotificationInput {
	return service.GroupRateChangeNotificationInput{
		NewRateMultiplier: r.NewRateMultiplier,
		WindowMinutes:     r.WindowMinutes,
		EffectiveAt:       r.EffectiveAt,
		Message:           r.Message,
	}
}
