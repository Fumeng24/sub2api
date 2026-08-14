package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *GatewayHandler) handleFailoverExhaustedNilCustom(c *gin.Context, failoverErr *service.UpstreamFailoverError, streamStarted bool) bool {
	if failoverErr != nil {
		return false
	}
	h.handleFailoverExhaustedSimple(c, http.StatusBadGateway, streamStarted)
	return true
}

func runUsageRecordTaskSync(component, panicMessage string, task service.UsageRecordTask) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.L().With(zap.String("component", component), zap.Any("panic", recovered)).Error(panicMessage)
		}
	}()
	task(ctx)
}

func (h *GatewayHandler) submitUsageRecordTaskCustom(task service.UsageRecordTask) bool {
	if h.usageRecordWorkerPool == nil {
		return false
	}
	if mode := h.usageRecordWorkerPool.Submit(task); mode != service.UsageRecordSubmitModeDropped {
		return true
	}
	logger.L().With(zap.String("component", "handler.gateway.messages")).Warn("gateway.usage_record_task_dropped_sync_fallback")
	runUsageRecordTaskSync("handler.gateway.messages", "gateway.usage_record_task_panic_recovered", task)
	return true
}

func usagePricingUnavailableMessage(model string) string {
	if model == "" {
		return "The requested model has no billing price configured for the current group."
	}
	return "The requested model is not priced for the current group: " + model
}

func gatewayAccountSelectionMessageCustom(classification noAccountErrorClassification, _ string) string {
	return classification.Message
}
