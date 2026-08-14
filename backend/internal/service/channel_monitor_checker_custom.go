package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func init() {
	chat := providerOpenAIChatAdapter
	chat.buildBody = buildOpenAIChatMonitorBodyCustom
	providerOpenAIChatAdapter = chat
	providerAdapters[MonitorProviderOpenAI] = chat

	responses := providerOpenAIResponsesAdapter
	responses.buildBody = buildOpenAIResponsesMonitorBodyCustom
	providerOpenAIResponsesAdapter = responses

	anthropic := providerAdapters[MonitorProviderAnthropic]
	anthropic.buildBody = buildAnthropicMonitorBodyCustom
	providerAdapters[MonitorProviderAnthropic] = anthropic

	gemini := providerAdapters[MonitorProviderGemini]
	gemini.buildBody = buildGeminiMonitorBodyCustom
	providerAdapters[MonitorProviderGemini] = gemini
}

func buildOpenAIChatMonitorBodyCustom(model, prompt string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "."},
			{"role": "user", "content": prompt},
		},
		"max_tokens": monitorChallengeMaxTokens,
		"stream":     false,
	})
}

func buildOpenAIResponsesMonitorBodyCustom(model, prompt string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"model":             model,
		"instructions":      ".",
		"input":             prompt,
		"max_output_tokens": monitorChallengeMaxTokens,
		"stream":            false,
	})
}

func buildAnthropicMonitorBodyCustom(model, prompt string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"model":      model,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens": monitorProviderChallengeMaxTokens,
	})
}

func buildGeminiMonitorBodyCustom(_ string, prompt string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]any{{"text": prompt}}},
		},
		"generationConfig": map[string]any{"maxOutputTokens": monitorProviderChallengeMaxTokens},
	})
}

func checkOptionsWithRuntimePlaceholdersCustom(opts *CheckOptions) *CheckOptions {
	bodyOverride := monitorBodyOverrideWithRuntimePlaceholders(opts)
	if len(bodyOverride) == 0 {
		return opts
	}
	cloned := *opts
	cloned.BodyOverride = bodyOverride
	return &cloned
}

// extractAnthropicMessagesText aggregates text blocks because thinking/tool
// blocks can precede the actual challenge answer.
func extractAnthropicMessagesText(respBytes []byte) string {
	var texts []string
	content := gjson.GetBytes(respBytes, "content")
	if content.IsArray() {
		content.ForEach(func(_, block gjson.Result) bool {
			blockType := block.Get("type").String()
			if blockType != "" && blockType != "text" {
				return true
			}
			if text := block.Get("text").String(); strings.TrimSpace(text) != "" {
				texts = append(texts, text)
			}
			return true
		})
	}
	if len(texts) > 0 {
		return strings.Join(texts, "")
	}
	return gjson.GetBytes(respBytes, providerAdapters[MonitorProviderAnthropic].textPath).String()
}

func monitorBodyOverrideWithRuntimePlaceholders(opts *CheckOptions) map[string]any {
	if opts == nil || len(opts.BodyOverride) == 0 {
		return nil
	}
	values := map[string]string{
		monitorPlaceholderSessionID: newMonitorSessionID(),
		monitorPlaceholderDeviceID:  newMonitorDeviceID(),
	}
	out := make(map[string]any, len(opts.BodyOverride))
	for k, v := range opts.BodyOverride {
		out[k] = replaceMonitorRuntimePlaceholders(v, values)
	}
	return out
}

func replaceMonitorRuntimePlaceholders(v any, values map[string]string) any {
	switch x := v.(type) {
	case string:
		out := x
		for placeholder, value := range values {
			out = strings.ReplaceAll(out, placeholder, value)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = replaceMonitorRuntimePlaceholders(item, values)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, item := range x {
			out[k] = replaceMonitorRuntimePlaceholders(item, values)
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(x))
		for i, item := range x {
			next := make(map[string]any, len(item))
			for k, value := range item {
				next[k] = replaceMonitorRuntimePlaceholders(value, values)
			}
			out[i] = next
		}
		return out
	default:
		return v
	}
}

func newMonitorSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return formatMonitorSessionID(fmt.Sprintf("%032x", time.Now().UnixNano()))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return formatMonitorSessionID(hex.EncodeToString(b[:]))
}

func formatMonitorSessionID(raw string) string {
	if len(raw) < 32 {
		raw = fmt.Sprintf("%032s", raw)
	}
	raw = raw[:32]
	return raw[:8] + "-" + raw[8:12] + "-" + raw[12:16] + "-" + raw[16:20] + "-" + raw[20:]
}

func newMonitorDeviceID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%064x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
