package usecase

import (
	"strings"
)

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "NO_SUCH_NOTE") ||
		strings.Contains(msg, "NO_SUCH_FILE") ||
		strings.Contains(msg, "(HTTP 404)")
}

func isNonPublicRenoteError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "renderAnnounce: cannot render non-public note")
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "(HTTP 401)") || strings.Contains(msg, "(HTTP 403)")
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "(HTTP 429)")
}

func isServerError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "(HTTP 5")
}
