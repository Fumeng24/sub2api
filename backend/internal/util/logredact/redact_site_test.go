package logredact

import (
	"strings"
	"testing"
)

func TestRedactText_APIKeyAuthorizationAndCookie(t *testing.T) {
	in := `api_key=sk-testsecret123456789, Authorization: Bearer ya29.a0AfH6SMDUMMYTOKEN, Cookie: sessionid=secret-cookie`
	out := RedactText(in)
	for _, leaked := range []string{"sk-testsecret", "ya29.a0AfH6", "secret-cookie"} {
		if strings.Contains(out, leaked) {
			t.Fatalf("expected %q redacted, got %q", leaked, out)
		}
	}
	for _, want := range []string{"api_key=***", "Authorization: ***", "Cookie: ***"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
}
