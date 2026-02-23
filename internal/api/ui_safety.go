package api

import (
	"fmt"
	"net/http"
	"strings"
)

// Requests coming from the bundled web UI include this marker.
const webUIClientHeader = "X-Conductor-Client"

func isWebUIRequest(r *http.Request) bool {
	if r == nil {
		return false
	}

	client := strings.TrimSpace(strings.ToLower(r.Header.Get(webUIClientHeader)))
	if client == "web-ui" {
		return true
	}

	// Browsers set fetch metadata and/or origin/referrer headers for UI calls.
	if strings.TrimSpace(r.Header.Get("Sec-Fetch-Mode")) != "" {
		return true
	}
	if strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")) != "" {
		return true
	}
	if strings.TrimSpace(r.Header.Get("Origin")) != "" {
		return true
	}
	if strings.TrimSpace(r.Header.Get("Referer")) != "" {
		return true
	}
	return false
}

func rejectUIDestructiveAction(r *http.Request, action string) *apiError {
	if !isWebUIRequest(r) {
		return nil
	}
	return apiErrorForbidden(fmt.Sprintf("%s is disabled in the web UI", action))
}
