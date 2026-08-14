package middleware

import "strings"

func onlyRequestBodyLimitErrors(errorsText string) bool {
	errorsText = strings.ToLower(strings.TrimSpace(errorsText))
	if errorsText == "" {
		return false
	}
	for _, part := range strings.Split(errorsText, "\n") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.Contains(part, "http: request body too large") &&
			!strings.Contains(part, "request body too large") {
			return false
		}
	}
	return true
}
