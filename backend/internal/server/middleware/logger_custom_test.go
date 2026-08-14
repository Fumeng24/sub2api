//go:build unit

package middleware

import "testing"

func TestOnlyRequestBodyLimitErrors(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "empty", text: "", want: false},
		{name: "standard", text: "http: request body too large", want: true},
		{name: "gin prefix", text: "Error #01: request body too large", want: true},
		{name: "multiple limits", text: "request body too large\nhttp: request body too large", want: true},
		{name: "mixed", text: "request body too large\ndatabase unavailable", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := onlyRequestBodyLimitErrors(tt.text); got != tt.want {
				t.Fatalf("onlyRequestBodyLimitErrors(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}
