package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
)

// RunTestBackgroundWithOptions runs the interactive account probe path while
// allowing scheduled jobs to supply the same prompt and mode options.
func (s *AccountTestService) RunTestBackgroundWithOptions(ctx context.Context, accountID int64, modelID, prompt, mode string) (*ScheduledTestResult, error) {
	startedAt := time.Now()
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = (&http.Request{}).WithContext(ctx)

	testErr := s.TestAccountConnection(ginCtx, accountID, modelID, prompt, mode)
	finishedAt := time.Now()
	responseText, errMsg := parseTestSSEOutput(w.Body.String())

	status := "success"
	if testErr != nil || errMsg != "" {
		status = "failed"
		if errMsg == "" {
			errMsg = testErr.Error()
		}
	}

	return &ScheduledTestResult{
		Status:       status,
		ResponseText: responseText,
		ErrorMessage: errMsg,
		LatencyMs:    finishedAt.Sub(startedAt).Milliseconds(),
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
	}, nil
}
