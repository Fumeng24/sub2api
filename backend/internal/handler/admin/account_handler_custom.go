package admin

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func accountSchedulerGroupScoreLessCustom(left, right AccountSchedulerGroupScore) bool {
	if left.GroupID == nil {
		return right.GroupID != nil
	}
	if right.GroupID == nil {
		return false
	}
	return *left.GroupID < *right.GroupID
}

func sortedMappedModelIDs(mapping map[string]string) []string {
	if len(mapping) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(mapping))
	modelIDs := make([]string, 0, len(mapping))
	for modelID := range mapping {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}
		seen[modelID] = struct{}{}
		modelIDs = append(modelIDs, modelID)
	}
	sort.Strings(modelIDs)
	return modelIDs
}

// UpdateAccountSchedulerConfigRequest is a narrow update surface for the
// scheduler page. It intentionally excludes credentials and full extra payloads.
type UpdateAccountSchedulerConfigRequest struct {
	Concurrency    *int     `json:"concurrency"`
	RateMultiplier *float64 `json:"rate_multiplier"`
	LoadFactor     *int     `json:"load_factor"`
	ManualRate     *float64 `json:"manual_rate"`
	RateScale      *float64 `json:"rate_scale"`
}

// UpdateSchedulerConfig handles updating scheduler-page account config only.
// PUT /api/v1/admin/accounts/:id/scheduler-config
func (h *AccountHandler) UpdateSchedulerConfig(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	var req UpdateAccountSchedulerConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.RateMultiplier != nil && *req.RateMultiplier < 0 {
		response.BadRequest(c, "rate_multiplier must be >= 0")
		return
	}
	if req.ManualRate != nil && *req.ManualRate < 0 {
		response.BadRequest(c, "manual_rate must be >= 0")
		return
	}
	if req.RateScale != nil && *req.RateScale < 0 {
		response.BadRequest(c, "rate_scale must be >= 0")
		return
	}

	account, err := h.adminService.UpdateAccount(c.Request.Context(), accountID, &service.UpdateAccountInput{
		Concurrency:    req.Concurrency,
		RateMultiplier: req.RateMultiplier,
		LoadFactor:     req.LoadFactor,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	extraUpdates := map[string]any{}
	if _, ok := raw["manual_rate"]; ok {
		if req.ManualRate == nil {
			extraUpdates["manual_rate"] = nil
		} else {
			extraUpdates["manual_rate"] = *req.ManualRate
		}
	}
	if _, ok := raw["rate_scale"]; ok {
		if req.RateScale == nil {
			extraUpdates["rate_scale"] = nil
		} else {
			extraUpdates["rate_scale"] = *req.RateScale
		}
	}
	if len(extraUpdates) > 0 {
		if err := h.adminService.UpdateAccountExtra(c.Request.Context(), accountID, extraUpdates); err != nil {
			response.ErrorFrom(c, err)
			return
		}
		account, err = h.adminService.GetAccount(c.Request.Context(), accountID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}

	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), account))
}

type BulkTestModelsRequest struct {
	AccountIDs  []int64  `json:"account_ids"`
	ModelIDs    []string `json:"model_ids"`
	Prompt      string   `json:"prompt"`
	Mode        string   `json:"mode"`
	Concurrency int      `json:"concurrency"`
}

type BulkTestModelResult struct {
	AccountID int64  `json:"account_id"`
	ModelID   string `json:"model_id"`
	Success   bool   `json:"success"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	LatencyMs int64  `json:"latency_ms"`
}

type BulkTestModelsResponse struct {
	Total   int                   `json:"total"`
	Success int                   `json:"success"`
	Failed  int                   `json:"failed"`
	Results []BulkTestModelResult `json:"results"`
}

// BulkTestModels tests selected account/model pairs in parallel.
// POST /api/v1/admin/accounts/bulk-test-models
func (h *AccountHandler) BulkTestModels(c *gin.Context) {
	if h.accountTestService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Account test service unavailable")
		return
	}

	var req BulkTestModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	accountIDs := normalizeBulkTestAccountIDs(req.AccountIDs)
	modelIDs := normalizeBulkTestModelIDs(req.ModelIDs)
	if len(accountIDs) == 0 {
		response.BadRequest(c, "account_ids is required")
		return
	}
	if len(modelIDs) == 0 {
		response.BadRequest(c, "model_ids is required")
		return
	}
	if len(accountIDs) > 100 {
		response.BadRequest(c, "Too many accounts selected, maximum is 100")
		return
	}
	if len(modelIDs) > 20 {
		response.BadRequest(c, "Too many models selected, maximum is 20")
		return
	}

	total := len(accountIDs) * len(modelIDs)
	if total > 500 {
		response.BadRequest(c, "Too many test tasks, maximum is 500")
		return
	}

	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 6
	}
	if concurrency > 20 {
		concurrency = 20
	}
	if concurrency > total {
		concurrency = total
	}

	type bulkTestTask struct {
		index     int
		accountID int64
		modelID   string
	}

	tasks := make([]bulkTestTask, 0, total)
	for _, accountID := range accountIDs {
		for _, modelID := range modelIDs {
			tasks = append(tasks, bulkTestTask{
				index:     len(tasks),
				accountID: accountID,
				modelID:   modelID,
			})
		}
	}

	ctx := c.Request.Context()
	results := make([]BulkTestModelResult, total)
	taskCh := make(chan bulkTestTask)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				result := BulkTestModelResult{
					AccountID: task.accountID,
					ModelID:   task.modelID,
					Status:    "failed",
					Message:   "test failed",
				}

				testResult, err := h.accountTestService.RunTestBackgroundWithOptions(ctx, task.accountID, task.modelID, req.Prompt, req.Mode)
				if testResult != nil {
					result.LatencyMs = testResult.LatencyMs
					result.Status = testResult.Status
					if testResult.Status == "success" {
						result.Success = true
						result.Message = "success"
						if strings.TrimSpace(testResult.ResponseText) != "" {
							result.Message = strings.TrimSpace(testResult.ResponseText)
						}
					} else if strings.TrimSpace(testResult.ErrorMessage) != "" {
						result.Message = strings.TrimSpace(testResult.ErrorMessage)
					}
				}
				if err != nil {
					result.Success = false
					result.Status = "failed"
					result.Message = err.Error()
				}
				if result.Success && h.rateLimitService != nil {
					if _, recoverErr := h.rateLimitService.RecoverAccountAfterSuccessfulTest(ctx, task.accountID); recoverErr != nil {
						log.Printf("failed to recover account after bulk test success: account_id=%d err=%v", task.accountID, recoverErr)
					}
				}

				results[task.index] = result
			}
		}()
	}

	for _, task := range tasks {
		select {
		case <-ctx.Done():
			close(taskCh)
			wg.Wait()
			response.Error(c, http.StatusRequestTimeout, "Request canceled")
			return
		case taskCh <- task:
		}
	}
	close(taskCh)
	wg.Wait()

	payload := BulkTestModelsResponse{
		Total:   total,
		Results: results,
	}
	for _, result := range results {
		if result.Success {
			payload.Success++
		} else {
			payload.Failed++
		}
	}

	response.Success(c, payload)
}

func normalizeBulkTestAccountIDs(raw []int64) []int64 {
	seen := make(map[int64]struct{}, len(raw))
	result := make([]int64, 0, len(raw))
	for _, id := range raw {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func normalizeBulkTestModelIDs(raw []string) []string {
	seen := make(map[string]struct{}, len(raw))
	result := make([]string, 0, len(raw))
	for _, modelID := range raw {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}
		seen[modelID] = struct{}{}
		result = append(result, modelID)
	}
	return result
}

var oauthCredentialSettingsToPreserve = []string{
	"model_mapping",
	"compact_model_mapping",
}

func preserveOAuthCredentialSettings(existing, incoming map[string]any) map[string]any {
	out := make(map[string]any, len(incoming)+len(oauthCredentialSettingsToPreserve))
	for key, value := range incoming {
		out[key] = value
	}

	for _, key := range oauthCredentialSettingsToPreserve {
		if _, hasIncoming := out[key]; hasIncoming {
			continue
		}
		if value, ok := existing[key]; ok {
			out[key] = value
		}
	}

	return out
}
