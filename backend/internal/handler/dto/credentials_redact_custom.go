package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

// RedactSensitiveMap recursively removes credential-like keys from response data.
func RedactSensitiveMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		if service.IsSensitiveCredentialKey(key) {
			continue
		}
		out[key] = redactSensitiveValue(value)
	}
	return out
}

func redactSensitiveValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return RedactSensitiveMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = redactSensitiveValue(typed[i])
		}
		return out
	default:
		return value
	}
}
