package service

import (
	"strings"
	"testing"
)

func TestSanitizeAndTrimJSONPayload_RedactsCommonCredentialKeys(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"apiKey":"sk-testsecret123456789","clientSecret":"client-secret-value","headers":{"Authorization":"Bearer ya29.a0AfH6SMDUMMYTOKEN","Cookie":"sessionid=secret-cookie"},"message":"safe"}`)
	out, _, _ := sanitizeAndTrimJSONPayload(raw, 10*1024)
	if out == "" {
		t.Fatalf("expected non-empty sanitized output")
	}
	for _, leaked := range []string{"sk-testsecret", "client-secret-value", "ya29.a0AfH6", "secret-cookie"} {
		if strings.Contains(out, leaked) {
			t.Fatalf("sanitized JSON leaked %q: %s", leaked, out)
		}
	}
}

func TestSanitizeErrorBodyForStorage_RedactsNonJSONText(t *testing.T) {
	t.Parallel()

	raw := `request failed api_key=sk-testsecret123456789, Authorization: Bearer ya29.a0AfH6SMDUMMYTOKEN, Cookie: sessionid=secret-cookie`
	out, _ := sanitizeErrorBodyForStorage(raw, 10*1024)
	for _, leaked := range []string{"sk-testsecret", "ya29.a0AfH6", "secret-cookie"} {
		if strings.Contains(out, leaked) {
			t.Fatalf("sanitized text leaked %q: %s", leaked, out)
		}
	}
}
