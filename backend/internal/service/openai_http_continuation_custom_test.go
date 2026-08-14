package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestApplyOpenAIHTTPPreviousResponseRecoveryCustomResetsRequestState(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","previous_response_id":"resp_old","input":"hello"}`)
	requestView := newOpenAIRequestView(body)
	reqBody := map[string]any{"stale": true}
	bodyModified := true

	recovered := applyOpenAIHTTPPreviousResponseRecoveryCustom(
		&Account{Name: "test"},
		&body,
		&requestView,
		&reqBody,
		&bodyModified,
		http.StatusNotFound,
		"previous response not found",
		[]byte(`{"error":{"code":"previous_response_not_found"}}`),
	)

	require.True(t, recovered)
	require.False(t, gjson.GetBytes(body, "previous_response_id").Exists())
	require.Equal(t, "gpt-5.5", requestView.Model)
	require.Empty(t, requestView.PreviousResponseID)
	require.Nil(t, reqBody)
	require.False(t, bodyModified)
}
