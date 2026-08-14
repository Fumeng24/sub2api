package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/tidwall/gjson"
)

func shouldStripInvalidOpenAIPreviousResponseID(body []byte) bool {
	previousResponseID := gjson.GetBytes(body, "previous_response_id")
	return previousResponseID.Exists() &&
		(previousResponseID.Type != gjson.String || strings.TrimSpace(previousResponseID.String()) == "")
}

func recoverOpenAIHTTPPreviousResponseIDBody(
	account *Account,
	body []byte,
	statusCode int,
	upstreamMsg string,
	respBody []byte,
) ([]byte, bool) {
	if !isOpenAICompatPreviousResponseNotFound(statusCode, upstreamMsg, respBody) &&
		!isOpenAICompatPreviousResponseUnsupported(statusCode, upstreamMsg, respBody) {
		return body, false
	}
	previousResponseID := strings.TrimSpace(gjson.GetBytes(body, "previous_response_id").String())
	if previousResponseID == "" {
		return body, false
	}

	decoded, err := getOpenAIRequestBodyMap(nil, body)
	if err != nil {
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Skip HTTP previous_response_id recovery because request body decode failed (account: %s)", openAIContinuationAccountName(account))
		return body, false
	}
	if HasFunctionCallOutput(decoded) {
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Skip HTTP previous_response_id recovery because request contains function_call_output (account: %s)", openAIContinuationAccountName(account))
		return body, false
	}
	delete(decoded, "previous_response_id")
	nextBody, err := marshalOpenAIUpstreamJSON(decoded)
	if err != nil {
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Skip HTTP previous_response_id recovery because request body serialize failed (account: %s)", openAIContinuationAccountName(account))
		return body, false
	}
	logger.LegacyPrintf(
		"service.openai_gateway",
		"[OpenAI] Retrying HTTP request once without previous_response_id after upstream rejected continuation (account: %s, previous_response_id_kind: %s)",
		openAIContinuationAccountName(account),
		ClassifyOpenAIPreviousResponseIDKind(previousResponseID),
	)
	return nextBody, true
}

func applyOpenAIHTTPPreviousResponseRecoveryCustom(
	account *Account,
	body *[]byte,
	requestView *openAIRequestView,
	reqBody *map[string]any,
	bodyModified *bool,
	statusCode int,
	upstreamMsg string,
	respBody []byte,
) bool {
	if body == nil || requestView == nil || reqBody == nil || bodyModified == nil {
		return false
	}
	recoveredBody, recovered := recoverOpenAIHTTPPreviousResponseIDBody(account, *body, statusCode, upstreamMsg, respBody)
	if !recovered {
		return false
	}
	*body = recoveredBody
	*requestView = newOpenAIRequestView(recoveredBody)
	*reqBody = nil
	*bodyModified = false
	return true
}

func openAIContinuationAccountName(account *Account) string {
	if account == nil {
		return ""
	}
	return account.Name
}
