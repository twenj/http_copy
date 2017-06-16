package goblog

import "net/http"

// HTTP Header Fields
const (
	HeaderUserAgent = "User-Agent"

	HeaderXForwardedFor = "X-Forwarded-For"
	HeaderXRealIP = "X-Real-IP"
)

// Predefine
var (
	Err = &Error{Code: http.StatusInternalServerError, Err: "Error"}

	ErrInternalServerError = Err.WithCode(http.StatusInternalServerError)
)
