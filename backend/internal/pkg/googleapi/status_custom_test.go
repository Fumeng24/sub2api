package googleapi

import (
	"net/http"
	"testing"
)

func TestHTTPStatusToGoogleStatusWithExtensions(t *testing.T) {
	if got := HTTPStatusToGoogleStatusWithExtensions(http.StatusPaymentRequired); got != "FAILED_PRECONDITION" {
		t.Fatalf("402 status=%q", got)
	}
	if got := HTTPStatusToGoogleStatusWithExtensions(http.StatusUnauthorized); got != "UNAUTHENTICATED" {
		t.Fatalf("401 status=%q", got)
	}
}
