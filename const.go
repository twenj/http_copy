package goblog

import "net/http"

// MIME types
const (
	MIMEApplicationJSON = "application/json"
)

// HTTP Header Fields
const (
	HeaderUserAgent = "User-Agent"

	HeaderXForwardedFor = "X-Forwarded-For"
	HeaderXRealIP = "X-Real-IP"
)

// Predefine
var (
	Err = &Error{Code: http.StatusInternalServerError, Err: "Error"}

	ErrBadRequest = Err.WithCode(http.StatusBadRequest)
	ErrInternalServerError = Err.WithCode(http.StatusInternalServerError)
)
