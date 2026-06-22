package service

import (
	"net/http"
	"time"
)

type schedulerResultReporter struct {
	health              *accountSchedulerHealthStats
	source              string
	cooldownForCategory func(category string, headers http.Header) time.Duration
}

type schedulerResultReport struct {
	AccountID    int64
	Model        string
	Endpoint     string
	Success      bool
	FirstTokenMs *int
	FailoverErr  *UpstreamFailoverError
}

type schedulerResultReportOutcome struct {
	SlowStreamingSuccess bool
	FailureCategory      string
	Cooldown             time.Duration
}

func (r schedulerResultReporter) report(input schedulerResultReport) schedulerResultReportOutcome {
	outcome := schedulerResultReportOutcome{}
	if r.health == nil || input.AccountID <= 0 {
		return outcome
	}
	if input.Success {
		outcome.SlowStreamingSuccess = schedulerReportStreamingSuccess(r.health, input.AccountID, input.Model, input.Endpoint, input.FirstTokenMs)
		if outcome.SlowStreamingSuccess {
			schedulerLogSlowStreamingTTFT(input.AccountID, input.Model, input.Endpoint, input.FirstTokenMs, r.source)
		}
		return outcome
	}

	statusCode, body, headers := schedulerFailureInputs(input.FailoverErr)
	category := schedulerClassifyFailure(r.source, input.FailoverErr, statusCode, body)
	cooldown := schedulerCooldownForCategory(category, headers)
	if r.cooldownForCategory != nil {
		cooldown = r.cooldownForCategory(category, headers)
	}
	r.health.reportFailure(input.AccountID, input.Model, input.Endpoint, category, cooldown)
	outcome.FailureCategory = category
	outcome.Cooldown = cooldown
	return outcome
}

func schedulerFailureInputs(failoverErr *UpstreamFailoverError) (int, []byte, http.Header) {
	if failoverErr == nil {
		return 0, nil, nil
	}
	return failoverErr.StatusCode, failoverErr.ResponseBody, failoverErr.ResponseHeaders
}

func schedulerClassifyFailure(platform string, failoverErr *UpstreamFailoverError, statusCode int, body []byte) string {
	return schedulerClassifierForPlatform(platform).Classify(statusCode, body, failoverErr)
}
