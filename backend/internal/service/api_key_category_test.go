package service

import "testing"

func TestNormalizeAPIKeyCategory(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		wantOkay bool
	}{
		{name: "empty defaults to other", input: "", want: APIKeyCategoryOther, wantOkay: true},
		{name: "openai", input: APIKeyCategoryOpenAI, want: APIKeyCategoryOpenAI, wantOkay: true},
		{name: "anthropic", input: APIKeyCategoryAnthropic, want: APIKeyCategoryAnthropic, wantOkay: true},
		{name: "other", input: APIKeyCategoryOther, want: APIKeyCategoryOther, wantOkay: true},
		{name: "invalid", input: "gemini", want: "", wantOkay: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NormalizeAPIKeyCategory(tt.input)
			if ok != tt.wantOkay {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOkay)
			}
			if got != tt.want {
				t.Fatalf("category = %q, want %q", got, tt.want)
			}
		})
	}
}
