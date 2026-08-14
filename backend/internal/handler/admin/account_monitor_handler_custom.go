package admin

import (
	"strconv"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AccountMonitorHandler 账号监控 admin handler（纯管理员，无用户路由）。
type AccountMonitorHandler struct {
	monitorService *service.AccountMonitorService
}

// NewAccountMonitorHandler 创建 handler。
func NewAccountMonitorHandler(monitorService *service.AccountMonitorService) *AccountMonitorHandler {
	return &AccountMonitorHandler{monitorService: monitorService}
}

// --- Request / Response ---

type accountMonitorCreateRequest struct {
	AccountID       int64  `json:"account_id" binding:"required"`
	Provider        string `json:"provider" binding:"omitempty,oneof=openai anthropic gemini"`
	Model           string `json:"model" binding:"omitempty,max=200"`
	Enabled         *bool  `json:"enabled"`
	IntervalSeconds int    `json:"interval_seconds" binding:"omitempty,min=15,max=3600"`
	JitterSeconds   int    `json:"jitter_seconds" binding:"omitempty,min=0,max=3600"`
}

type accountMonitorUpdateRequest struct {
	Provider        *string `json:"provider" binding:"omitempty,oneof=openai anthropic gemini"`
	Model           *string `json:"model" binding:"omitempty,max=200"`
	Enabled         *bool   `json:"enabled"`
	IntervalSeconds *int    `json:"interval_seconds" binding:"omitempty,min=15,max=3600"`
	JitterSeconds   *int    `json:"jitter_seconds" binding:"omitempty,min=0,max=3600"`
}

type accountMonitorResponse struct {
	ID              int64      `json:"id"`
	AccountID       int64      `json:"account_id"`
	Provider        string     `json:"provider"`
	Model           string     `json:"model"`
	Enabled         bool       `json:"enabled"`
	IntervalSeconds int        `json:"interval_seconds"`
	JitterSeconds   int        `json:"jitter_seconds"`
	LastCheckedAt   *time.Time `json:"last_checked_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type accountMonitorTimelinePoint struct {
	Status    string    `json:"status"`
	LatencyMs *int      `json:"latency_ms"`
	CheckedAt time.Time `json:"checked_at"`
}

type accountMonitorStatusResponse struct {
	MonitorID      int64                         `json:"monitor_id"`
	AccountID      int64                         `json:"account_id"`
	Model          string                        `json:"model"`
	Enabled        bool                          `json:"enabled"`
	LatestStatus   string                        `json:"latest_status"`
	LatestLatency  *int                          `json:"latest_latency_ms"`
	PingLatencyMs  *int                          `json:"ping_latency_ms"`
	Availability1h float64                       `json:"availability_1h"`
	AvgLatency1h   *float64                      `json:"avg_latency_1h"`
	LastCheckedAt  *time.Time                    `json:"last_checked_at"`
	Timeline       []accountMonitorTimelinePoint `json:"timeline"`
}

func accountMonitorToResponse(m *service.AccountMonitor) accountMonitorResponse {
	return accountMonitorResponse{
		ID:              m.ID,
		AccountID:       m.AccountID,
		Provider:        m.Provider,
		Model:           m.Model,
		Enabled:         m.Enabled,
		IntervalSeconds: m.IntervalSeconds,
		JitterSeconds:   m.JitterSeconds,
		LastCheckedAt:   m.LastCheckedAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func parseAccountMonitorID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_MONITOR_ID", "invalid monitor id"))
		return 0, false
	}
	return id, true
}

// --- Handlers ---

// List GET /api/v1/admin/account-monitors
func (h *AccountMonitorHandler) List(c *gin.Context) {
	items, err := h.monitorService.List(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]accountMonitorResponse, 0, len(items))
	for _, m := range items {
		out = append(out, accountMonitorToResponse(m))
	}
	response.Success(c, gin.H{"items": out})
}

// Status GET /api/v1/admin/account-monitors/status
// 返回所有监控的聚合状态，按 account_id 索引（前端 SchedulerView 按行匹配）。
func (h *AccountMonitorHandler) Status(c *gin.Context) {
	statuses, err := h.monitorService.StatusByAccountID(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make(map[string]accountMonitorStatusResponse, len(statuses))
	for accountID, st := range statuses {
		timeline := make([]accountMonitorTimelinePoint, 0, len(st.Timeline))
		for _, c := range st.Timeline {
			timeline = append(timeline, accountMonitorTimelinePoint{
				Status:    c.Status,
				LatencyMs: c.LatencyMs,
				CheckedAt: c.CheckedAt,
			})
		}
		out[strconv.FormatInt(accountID, 10)] = accountMonitorStatusResponse{
			MonitorID:      st.MonitorID,
			AccountID:      st.AccountID,
			Model:          st.Model,
			Enabled:        st.Enabled,
			LatestStatus:   st.LatestStatus,
			LatestLatency:  st.LatestLatency,
			PingLatencyMs:  st.PingLatencyMs,
			Availability1h: st.Availability1h,
			AvgLatency1h:   st.AvgLatency1h,
			LastCheckedAt:  st.LastCheckedAt,
			Timeline:       timeline,
		}
	}
	response.Success(c, gin.H{"statuses": out})
}

// Create POST /api/v1/admin/account-monitors
func (h *AccountMonitorHandler) Create(c *gin.Context) {
	var req accountMonitorCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	m, err := h.monitorService.Create(c.Request.Context(), service.AccountMonitorCreateParams{
		AccountID:       req.AccountID,
		Provider:        req.Provider,
		Model:           req.Model,
		Enabled:         enabled,
		IntervalSeconds: req.IntervalSeconds,
		JitterSeconds:   req.JitterSeconds,
		CreatedBy:       subject.UserID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, accountMonitorToResponse(m))
}

// Update PUT /api/v1/admin/account-monitors/:id
func (h *AccountMonitorHandler) Update(c *gin.Context) {
	id, ok := parseAccountMonitorID(c)
	if !ok {
		return
	}
	var req accountMonitorUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	m, err := h.monitorService.Update(c.Request.Context(), id, service.AccountMonitorUpdateParams{
		Provider:        req.Provider,
		Model:           req.Model,
		Enabled:         req.Enabled,
		IntervalSeconds: req.IntervalSeconds,
		JitterSeconds:   req.JitterSeconds,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, accountMonitorToResponse(m))
}

// Delete DELETE /api/v1/admin/account-monitors/:id
func (h *AccountMonitorHandler) Delete(c *gin.Context) {
	id, ok := parseAccountMonitorID(c)
	if !ok {
		return
	}
	if err := h.monitorService.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// Run POST /api/v1/admin/account-monitors/:id/run
// 立即触发一次探测（前端开启监控后即时出状态）。
func (h *AccountMonitorHandler) Run(c *gin.Context) {
	id, ok := parseAccountMonitorID(c)
	if !ok {
		return
	}
	check, err := h.monitorService.RunCheck(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"status":     check.Status,
		"latency_ms": check.LatencyMs,
		"message":    check.Message,
		"checked_at": check.CheckedAt,
	})
}
