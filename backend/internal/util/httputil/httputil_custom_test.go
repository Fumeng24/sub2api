package httputil

import (
	"net/http"
	"testing"
)

func TestIsCloudflareChallengeResponseCustom(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		headers http.Header
		body    string
		want    bool
	}{
		{name: "error 1010", status: http.StatusForbidden, body: "Error code: 1010", want: true},
		{name: "error 1020", status: http.StatusForbidden, body: "Error code 1020", want: true},
		{name: "browser signature", status: http.StatusForbidden, body: "The browser's signature did not match", want: true},
		{name: "cf ray html", status: http.StatusTooManyRequests, headers: http.Header{"Cf-Ray": {"ray-id"}, "Content-Type": {"text/html"}}, body: "<html>Cloudflare</html>", want: true},
		{name: "cf ray json", status: http.StatusForbidden, headers: http.Header{"Cf-Ray": {"ray-id"}, "Content-Type": {"application/json"}}, body: `{"error":"cloudflare"}`, want: false},
		{name: "non challenge status", status: http.StatusBadGateway, body: "Error code: 1020", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCloudflareChallengeResponse(tt.status, tt.headers, []byte(tt.body)); got != tt.want {
				t.Fatalf("IsCloudflareChallengeResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}
