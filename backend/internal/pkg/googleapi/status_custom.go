package googleapi

import "net/http"

// HTTPStatusToGoogleStatusWithExtensions adds site-specific statuses while
// preserving the upstream mapping for all standard cases.
func HTTPStatusToGoogleStatusWithExtensions(status int) string {
	if status == http.StatusPaymentRequired {
		return "FAILED_PRECONDITION"
	}
	return HTTPStatusToGoogleStatus(status)
}
