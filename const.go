package goblog

import "net/http"

// MIME types
const (
	MIMEApplicationJSON = "application/json"
	MIMEApplicationXML = "application/xml"
	MIMEApplicationForm = "application/x-www-form-urlencoded"
	MIMETextHTMLCharsetUTF8 = "text/html; charset=utf-8"
)

// HTTP Header Fields
const (
	HeaderContentType = "Content-Type"
	HeaderUserAgent = "User-Agent"

	HeaderXForwardedFor = "X-Forwarded-For"
	HeaderXRealIP = "X-Real-IP"
)

// Predefine
var (
	Err = &Error{Code: http.StatusInternalServerError, Err: "Error"}

	ErrBadRequest = Err.WithCode(http.StatusBadRequest)
	ErrInternalServerError = Err.WithCode(http.StatusInternalServerError)
	ErrUnsupportedMediaType = Err.WithCode(http.StatusUnsupportedMediaType)
)
