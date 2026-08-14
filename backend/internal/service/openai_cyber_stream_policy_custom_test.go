package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func bindCyberMatchingPassthroughRule(c *gin.Context) {
	code := http.StatusTeapot
	message := "custom rule must not replace cyber failure"
	ruleService := &ErrorPassthroughService{}
	ruleService.setLocalCache([]*model.ErrorPassthroughRule{{
		ID:              991,
		Enabled:         true,
		Platforms:       []string{PlatformOpenAI},
		Keywords:        []string{"cyber_policy"},
		MatchMode:       model.MatchModeAny,
		ResponseCode:    &code,
		CustomMessage:   &message,
		PassthroughBody: true,
	}})
	BindErrorPassthroughService(c, ruleService)
}

func newCyberFailedStreamResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(
			"data: {\"type\":\"response.failed\",\"response\":{\"id\":\"resp_cyber_rule\",\"error\":{\"code\":\"cyber_policy\",\"message\":\"blocked by cyber_policy\"},\"usage\":{\"input_tokens\":9,\"output_tokens\":1}}}\n\n",
		)),
	}
}

func TestOpenAIStreamingCyberPolicyBypassesPassthroughRule(t *testing.T) {
	for _, tc := range []struct {
		name        string
		passthrough bool
	}{
		{name: "normalized"},
		{name: "passthrough", passthrough: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			bindCyberMatchingPassthroughRule(c)
			svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}}
			account := &Account{ID: 1, Platform: PlatformOpenAI, Name: "cyber-account"}

			var err error
			if tc.passthrough {
				_, err = svc.handleStreamingResponsePassthrough(context.Background(), newCyberFailedStreamResponse(), c, account, time.Now(), "", "")
			} else {
				_, err = svc.handleStreamingResponse(context.Background(), newCyberFailedStreamResponse(), c, account, time.Now(), "gpt-5.5", "gpt-5.5")
			}

			require.Error(t, err)
			var failoverErr *UpstreamFailoverError
			require.NotErrorAs(t, err, &failoverErr)
			require.NotNil(t, GetOpsCyberPolicy(c))
			require.NotEqual(t, http.StatusTeapot, rec.Code)
			require.Contains(t, rec.Body.String(), "response.failed")
			require.NotContains(t, rec.Body.String(), "custom rule must not replace cyber failure")
		})
	}
}

func TestOpenAISSEToJSONCyberPolicyMarksUsageAndDoesNotFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	body := []byte("data: {\"type\":\"response.failed\",\"response\":{\"id\":\"resp_sse_cyber\",\"error\":{\"code\":\"cyber_policy\",\"message\":\"blocked by cyber_policy\"},\"usage\":{\"input_tokens\":12,\"output_tokens\":3}}}\n\n")
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}}
	svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}}

	result, err := svc.handleSSEToJSONWithContext(context.Background(), resp, c, &Account{ID: 1, Platform: PlatformOpenAI}, body, "gpt-5.5", "gpt-5.5")

	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.NotErrorAs(t, err, &failoverErr)
	require.NotNil(t, GetOpsCyberPolicy(c))
	require.Equal(t, 12, GetOpsCyberPolicy(c).UpstreamInTok)
	require.Equal(t, 3, GetOpsCyberPolicy(c).UpstreamOutTok)
	require.True(t, c.Writer.Written())
}
