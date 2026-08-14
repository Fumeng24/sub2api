package handler

import (
	"bytes"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const (
	openAIResponsesRequestShapeLargeBodyBytes  = 256 * 1024
	openAIResponsesRequestShapeLargeInputItems = 100
)

type openAIResponsesRequestShape struct {
	BodyBytes                             int
	InputType                             string
	InputRawBytes                         int
	InputItemCount                        int
	HasPreviousResponseID                 bool
	PreviousResponseIDKind                string
	PreviousResponseIDLen                 int
	HasFunctionCallOutput                 bool
	FunctionCallOutputCount               int
	HasFunctionCallOutputMissingCallID    bool
	HasToolCallContext                    bool
	ToolCallContextCount                  int
	HasItemReference                      bool
	ItemReferenceCount                    int
	HasItemReferenceForAllFunctionCallIDs bool
	HasTools                              bool
	ToolsCount                            int
	HasToolChoice                         bool
	DiagnosticReasons                     []string
}

func analyzeOpenAIResponsesRequestShape(body []byte) openAIResponsesRequestShape {
	shape := openAIResponsesRequestShape{
		BodyBytes:              len(body),
		PreviousResponseIDKind: service.OpenAIPreviousResponseIDKindEmpty,
	}
	if len(body) == 0 || !gjson.ValidBytes(body) {
		shape.InputType = "invalid"
		shape.addDiagnosticReason("invalid_json")
		return shape
	}

	if prev := strings.TrimSpace(gjson.GetBytes(body, "previous_response_id").String()); prev != "" {
		shape.HasPreviousResponseID = true
		shape.PreviousResponseIDKind = service.ClassifyOpenAIPreviousResponseIDKind(prev)
		shape.PreviousResponseIDLen = len(prev)
		shape.addDiagnosticReason("previous_response_id")
	}

	input := gjson.GetBytes(body, "input")
	shape.InputType = classifyOpenAIResponsesInputType(input)
	if input.Exists() {
		shape.InputRawBytes = len(input.Raw)
	}
	if !shouldDeepAnalyzeOpenAIResponsesRequestShape(body, shape.HasPreviousResponseID) {
		return shape
	}
	if input.IsArray() {
		callIDs := make(map[string]struct{})
		referenceIDs := make(map[string]struct{})
		input.ForEach(func(_, item gjson.Result) bool {
			shape.InputItemCount++
			if !item.IsObject() {
				return true
			}
			itemType := item.Get("type").String()
			switch {
			case isOpenAIResponsesToolCallOutputItemType(itemType):
				shape.HasFunctionCallOutput = true
				shape.FunctionCallOutputCount++
				callID := strings.TrimSpace(item.Get("call_id").String())
				if callID == "" {
					shape.HasFunctionCallOutputMissingCallID = true
					return true
				}
				callIDs[callID] = struct{}{}
			case isOpenAIResponsesToolCallContextItemType(itemType):
				shape.HasToolCallContext = true
				shape.ToolCallContextCount++
			case itemType == "item_reference":
				shape.HasItemReference = true
				shape.ItemReferenceCount++
				if idValue := strings.TrimSpace(item.Get("id").String()); idValue != "" {
					referenceIDs[idValue] = struct{}{}
				}
			}
			return true
		})
		if len(callIDs) > 0 && len(referenceIDs) > 0 {
			shape.HasItemReferenceForAllFunctionCallIDs = true
			for callID := range callIDs {
				if _, ok := referenceIDs[callID]; !ok {
					shape.HasItemReferenceForAllFunctionCallIDs = false
					break
				}
			}
		}
	}

	tools := gjson.GetBytes(body, "tools")
	if tools.IsArray() {
		shape.ToolsCount = int(tools.Get("#").Int())
		shape.HasTools = shape.ToolsCount > 0
	} else {
		shape.HasTools = tools.Exists() && tools.Type != gjson.Null
	}
	toolChoice := gjson.GetBytes(body, "tool_choice")
	shape.HasToolChoice = toolChoice.Exists() && toolChoice.Type != gjson.Null

	if shape.BodyBytes >= openAIResponsesRequestShapeLargeBodyBytes {
		shape.addDiagnosticReason("large_body")
	}
	if shape.InputItemCount >= openAIResponsesRequestShapeLargeInputItems {
		shape.addDiagnosticReason("large_input")
	}
	if shape.HasFunctionCallOutput {
		shape.addDiagnosticReason("function_call_output")
	}
	return shape
}

func shouldDeepAnalyzeOpenAIResponsesRequestShape(body []byte, hasPreviousResponseID bool) bool {
	if len(body) >= openAIResponsesRequestShapeLargeBodyBytes || hasPreviousResponseID {
		return true
	}
	for _, marker := range [][]byte{
		[]byte(`"function_call_output"`),
		[]byte(`"tool_search_output"`),
		[]byte(`"custom_tool_call_output"`),
		[]byte(`"mcp_tool_call_output"`),
		[]byte(`"item_reference"`),
	} {
		if bytes.Contains(body, marker) {
			return true
		}
	}
	return false
}

func classifyOpenAIResponsesInputType(input gjson.Result) string {
	if !input.Exists() {
		return "missing"
	}
	if input.IsArray() {
		return "array"
	}
	if input.IsObject() {
		return "object"
	}
	switch input.Type {
	case gjson.String:
		return "string"
	case gjson.Number:
		return "number"
	case gjson.True, gjson.False:
		return "bool"
	case gjson.Null:
		return "null"
	default:
		return "unknown"
	}
}

func isOpenAIResponsesToolCallContextItemType(typ string) bool {
	switch strings.TrimSpace(typ) {
	case "tool_call",
		"function_call",
		"local_shell_call",
		"tool_search_call",
		"custom_tool_call",
		"mcp_tool_call":
		return true
	default:
		return false
	}
}

func isOpenAIResponsesToolCallOutputItemType(typ string) bool {
	switch strings.TrimSpace(typ) {
	case "function_call_output",
		"tool_search_output",
		"custom_tool_call_output",
		"mcp_tool_call_output":
		return true
	default:
		return false
	}
}

func (s *openAIResponsesRequestShape) addDiagnosticReason(reason string) {
	for _, existing := range s.DiagnosticReasons {
		if existing == reason {
			return
		}
	}
	s.DiagnosticReasons = append(s.DiagnosticReasons, reason)
}

func (s openAIResponsesRequestShape) ShouldLog() bool {
	return len(s.DiagnosticReasons) > 0
}

func logOpenAIResponsesRequestShape(reqLog *zap.Logger, body []byte) {
	if reqLog == nil {
		return
	}
	shape := analyzeOpenAIResponsesRequestShape(body)
	if !shape.ShouldLog() {
		return
	}
	reqLog.Info("openai.responses_request_shape",
		zap.Int("shape_body_bytes", shape.BodyBytes),
		zap.String("shape_input_type", shape.InputType),
		zap.Int("shape_input_raw_bytes", shape.InputRawBytes),
		zap.Int("shape_input_item_count", shape.InputItemCount),
		zap.Bool("shape_has_previous_response_id", shape.HasPreviousResponseID),
		zap.String("shape_previous_response_id_kind", shape.PreviousResponseIDKind),
		zap.Int("shape_previous_response_id_len", shape.PreviousResponseIDLen),
		zap.Bool("shape_has_function_call_output", shape.HasFunctionCallOutput),
		zap.Int("shape_function_call_output_count", shape.FunctionCallOutputCount),
		zap.Bool("shape_has_function_call_output_missing_call_id", shape.HasFunctionCallOutputMissingCallID),
		zap.Bool("shape_has_tool_call_context", shape.HasToolCallContext),
		zap.Int("shape_tool_call_context_count", shape.ToolCallContextCount),
		zap.Bool("shape_has_item_reference", shape.HasItemReference),
		zap.Int("shape_item_reference_count", shape.ItemReferenceCount),
		zap.Bool("shape_has_item_reference_for_all_function_call_ids", shape.HasItemReferenceForAllFunctionCallIDs),
		zap.Bool("shape_has_tools", shape.HasTools),
		zap.Int("shape_tools_count", shape.ToolsCount),
		zap.Bool("shape_has_tool_choice", shape.HasToolChoice),
		zap.Strings("shape_diagnostic_reasons", shape.DiagnosticReasons),
	)
}
