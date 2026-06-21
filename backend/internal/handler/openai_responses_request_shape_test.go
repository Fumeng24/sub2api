package handler

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestAnalyzeOpenAIResponsesRequestShape_LargeContinuationOnlyReportsShape(t *testing.T) {
	body := []byte(`{
		"model":"gpt-5.5",
		"stream":true,
		"previous_response_id":"resp_secret_value",
		"tools":[{"type":"function"}],
		"tool_choice":"auto",
		"input":[
			{"type":"function_call","call_id":"call_1","name":"shell","arguments":"{}"},
			{"type":"function_call_output","call_id":"call_1","output":"secret output must not be logged"},
			{"type":"item_reference","id":"call_1"},
			{"type":"input_text","text":"secret prompt must not be logged"}
		]
	}`)

	shape := analyzeOpenAIResponsesRequestShape(body)

	require.Equal(t, len(body), shape.BodyBytes)
	require.Equal(t, "array", shape.InputType)
	require.Equal(t, 4, shape.InputItemCount)
	require.True(t, shape.HasPreviousResponseID)
	require.Equal(t, service.OpenAIPreviousResponseIDKindResponseID, shape.PreviousResponseIDKind)
	require.Equal(t, len("resp_secret_value"), shape.PreviousResponseIDLen)
	require.True(t, shape.HasFunctionCallOutput)
	require.Equal(t, 1, shape.FunctionCallOutputCount)
	require.True(t, shape.HasToolCallContext)
	require.True(t, shape.HasItemReference)
	require.True(t, shape.HasItemReferenceForAllFunctionCallIDs)
	require.True(t, shape.HasTools)
	require.Equal(t, 1, shape.ToolsCount)
	require.True(t, shape.HasToolChoice)
	require.True(t, shape.ShouldLog())
	require.Contains(t, shape.DiagnosticReasons, "previous_response_id")
	require.Contains(t, shape.DiagnosticReasons, "function_call_output")
}

func TestAnalyzeOpenAIResponsesRequestShape_SmallPlainRequestDoesNotLog(t *testing.T) {
	body := []byte(`{"model":"gpt-5.4-mini","stream":false,"input":"hello"}`)

	shape := analyzeOpenAIResponsesRequestShape(body)

	require.Equal(t, "string", shape.InputType)
	require.False(t, shape.HasPreviousResponseID)
	require.False(t, shape.HasFunctionCallOutput)
	require.False(t, shape.ShouldLog())
}

func TestLogOpenAIResponsesRequestShape_DoesNotLeakContentOrIDs(t *testing.T) {
	var buf bytes.Buffer
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = ""
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)
	reqLog := zap.New(core)
	body := []byte(`{
		"model":"gpt-5.5",
		"previous_response_id":"resp_do_not_log_this_value",
		"input":[
			{"type":"function_call_output","call_id":"call_sensitive","output":"sensitive output"},
			{"type":"input_text","text":"sensitive prompt"}
		]
	}`)

	logOpenAIResponsesRequestShape(reqLog, body)

	logged := buf.String()
	require.Contains(t, logged, "openai.responses_request_shape")
	require.Contains(t, logged, `"shape_has_previous_response_id":true`)
	require.Contains(t, logged, `"shape_previous_response_id_kind":"response_id"`)
	require.Contains(t, logged, `"shape_previous_response_id_len":26`)
	require.NotContains(t, logged, "resp_do_not_log_this_value")
	require.NotContains(t, logged, "call_sensitive")
	require.NotContains(t, logged, "sensitive output")
	require.NotContains(t, logged, "sensitive prompt")
}

func TestAnalyzeOpenAIResponsesRequestShape_LargeInputTriggersDiagnostic(t *testing.T) {
	items := make([]string, 0, openAIResponsesRequestShapeLargeInputItems)
	largeText := strings.Repeat("x", 3000)
	for i := 0; i < openAIResponsesRequestShapeLargeInputItems; i++ {
		items = append(items, `{"type":"input_text","text":"`+largeText+`"}`)
	}
	body := []byte(`{"model":"gpt-5.4","input":[` + strings.Join(items, ",") + `]}`)

	shape := analyzeOpenAIResponsesRequestShape(body)

	require.GreaterOrEqual(t, len(body), openAIResponsesRequestShapeLargeBodyBytes)
	require.Equal(t, openAIResponsesRequestShapeLargeInputItems, shape.InputItemCount)
	require.True(t, shape.ShouldLog())
	require.Contains(t, shape.DiagnosticReasons, "large_body")
	require.Contains(t, shape.DiagnosticReasons, "large_input")
}
