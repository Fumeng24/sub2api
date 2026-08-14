//go:build unit

package service

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestParseWhamResponse_TaggedWindows(t *testing.T) {
	body := []byte(`{
		"plan_type": "plus",
		"rate_limit": {
			"limit_reached": false,
			"allowed": true,
			"primary_window":  {"used_percent": 42.0, "reset_after_seconds": 3600, "limit_window_seconds": 18000},
			"secondary_window": {"used_percent": 88.0, "reset_after_seconds": 172800, "limit_window_seconds": 604800}
		}
	}`)
	snap, status := parseWhamResponse(body)
	if status != whamStatusLow {
		t.Fatalf("expected %q for 88%% 7d, got %q", whamStatusLow, status)
	}
	if snap == nil || snap.PrimaryUsedPercent == nil || *snap.PrimaryUsedPercent != 42.0 {
		t.Fatalf("expected 5h=42%%, got %+v", snap)
	}
	if snap.SecondaryUsedPercent == nil || *snap.SecondaryUsedPercent != 88.0 {
		t.Fatalf("expected 7d=88%%, got %+v", snap.SecondaryUsedPercent)
	}
}

func TestParseWhamResponse_SwappedTags(t *testing.T) {
	// Upstream swaps which slot is 5h vs 7d — we must classify by limit_window_seconds, not position.
	body := []byte(`{
		"rate_limit": {
			"primary_window":  {"used_percent": 10.0, "limit_window_seconds": 604800},
			"secondary_window": {"used_percent": 95.0, "limit_window_seconds": 18000}
		}
	}`)
	snap, status := parseWhamResponse(body)
	if status != whamStatusHigh {
		t.Fatalf("expected high (10%% 7d), got %q", status)
	}
	if snap.SecondaryUsedPercent == nil || *snap.SecondaryUsedPercent != 10.0 {
		t.Fatalf("expected 7d=10%%, got %+v", snap.SecondaryUsedPercent)
	}
	if snap.PrimaryUsedPercent == nil || *snap.PrimaryUsedPercent != 95.0 {
		t.Fatalf("expected 5h=95%%, got %+v", snap.PrimaryUsedPercent)
	}
}

func TestParseWhamResponse_LimitReached(t *testing.T) {
	body := []byte(`{
		"rate_limit": {
			"limit_reached": true,
			"primary_window":  {"used_percent": 100.0, "limit_window_seconds": 18000},
			"secondary_window": {"used_percent": 85.0, "limit_window_seconds": 604800}
		}
	}`)
	_, status := parseWhamResponse(body)
	if status != whamStatusExhausted {
		t.Fatalf("limit_reached=true should classify as exhausted, got %q", status)
	}
}

func TestParseWhamResponse_Thresholds(t *testing.T) {
	cases := []struct {
		pct  float64
		want string
	}{
		{0, whamStatusHigh},
		{29.99, whamStatusHigh},
		{30, whamStatusMedium},
		{69.99, whamStatusMedium},
		{70, whamStatusLow},
		{99.99, whamStatusLow},
		{100, whamStatusExhausted},
		{105.5, whamStatusExhausted},
	}
	for _, tc := range cases {
		body := []byte(`{"rate_limit":{"primary_window":{"used_percent":` + floatToString(tc.pct) + `,"limit_window_seconds":604800}}}`)
		_, got := parseWhamResponse(body)
		if got != tc.want {
			t.Errorf("used_percent=%v: expected %q, got %q", tc.pct, tc.want, got)
		}
	}
}

func TestParseWhamResponse_OnlyFiveH(t *testing.T) {
	// Only 5h window present → fall back to classifying from it.
	body := []byte(`{"rate_limit":{"primary_window":{"used_percent":55.0,"limit_window_seconds":18000}}}`)
	_, status := parseWhamResponse(body)
	if status != whamStatusMedium {
		t.Fatalf("5h-only 55%% should be medium, got %q", status)
	}
}

func TestParseWhamResponse_UntaggedFallback(t *testing.T) {
	// No limit_window_seconds → default to primary=7d, secondary=5h (matches
	// ParseCodexRateLimitHeaders convention).
	body := []byte(`{"rate_limit":{"primary_window":{"used_percent":45.0},"secondary_window":{"used_percent":10.0}}}`)
	snap, status := parseWhamResponse(body)
	if status != whamStatusMedium {
		t.Fatalf("untagged primary=45%% should classify as 7d medium, got %q", status)
	}
	if snap.SecondaryUsedPercent == nil || *snap.SecondaryUsedPercent != 45.0 {
		t.Fatalf("expected secondary(7d)=45%%, got %+v", snap.SecondaryUsedPercent)
	}
	if snap.PrimaryUsedPercent == nil || *snap.PrimaryUsedPercent != 10.0 {
		t.Fatalf("expected primary(5h)=10%%, got %+v", snap.PrimaryUsedPercent)
	}
}

func TestParseWhamResponse_MissingUsedPercent(t *testing.T) {
	body := []byte(`{"rate_limit":{"primary_window":{"limit_window_seconds":604800}}}`)
	_, status := parseWhamResponse(body)
	if status != whamStatusUnknown {
		t.Fatalf("missing used_percent should classify as unknown, got %q", status)
	}
}

func TestParseWhamResponse_MalformedJSON(t *testing.T) {
	_, status := parseWhamResponse([]byte(`{not valid json}`))
	if status != whamStatusError {
		t.Fatalf("malformed JSON should classify as error, got %q", status)
	}
}

func TestWhamJWTExpiry_Valid(t *testing.T) {
	exp := time.Now().Add(1 * time.Hour).Unix()
	payload := map[string]any{"exp": exp}
	buf, _ := json.Marshal(payload)
	token := "header." + base64.RawURLEncoding.EncodeToString(buf) + ".sig"
	got, ok := whamJWTExpiry(token)
	if !ok {
		t.Fatalf("expected ok=true for valid JWT, got false")
	}
	if got.Unix() != exp {
		t.Errorf("expected exp=%d, got %d", exp, got.Unix())
	}
}

func TestWhamJWTExpiry_PaddedBase64(t *testing.T) {
	payload := []byte(`{"exp":1700000000}`)
	// StdEncoding always pads — RawURLEncoding doesn't. We should accept both.
	token := "header." + base64.StdEncoding.EncodeToString(payload) + ".sig"
	got, ok := whamJWTExpiry(token)
	if !ok {
		t.Fatalf("expected ok=true for padded JWT, got false")
	}
	if got.Unix() != 1700000000 {
		t.Errorf("expected exp=1700000000, got %d", got.Unix())
	}
}

func TestWhamJWTExpiry_MissingExp(t *testing.T) {
	token := "header." + base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"x"}`)) + ".sig"
	_, ok := whamJWTExpiry(token)
	if ok {
		t.Errorf("expected ok=false when exp is missing")
	}
}

func TestWhamJWTExpiry_InvalidFormat(t *testing.T) {
	_, ok := whamJWTExpiry("not-a-jwt")
	if ok {
		t.Errorf("expected ok=false for non-JWT string")
	}
	_, ok = whamJWTExpiry("a.b.c")
	if ok {
		t.Errorf("expected ok=false for unparseable base64 payload")
	}
}

func TestWhamTruncate_UTF8Boundary(t *testing.T) {
	// 3-byte UTF-8 rune "中" — truncating to 2 bytes must not leave a half rune.
	s := "ab中cd"
	got := whamTruncate(s, 3)
	if got != "ab" {
		t.Errorf("expected 'ab' (back off mid-rune), got %q", got)
	}
	// Exact boundary fits.
	if whamTruncate(s, 5) != "ab中" {
		t.Errorf("expected 'ab中' at boundary 5")
	}
	// Empty truncation.
	if whamTruncate("hello", 10) != "hello" {
		t.Errorf("expected pass-through when len<n")
	}
}

// floatToString renders a float without trailing zeros for test body building.
func floatToString(f float64) string {
	buf, _ := json.Marshal(f)
	return string(buf)
}
