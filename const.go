package goblog

import "net/http"

// MIME types
const (
	MIMEApplicationJSON = "application/json"
	MIMEApplicationJSONCharsetUTF8 = "application/json; charset=utf-8"
	MIMEApplicationXML = "application/xml"
	MIMEApplicationForm = "application/x-www-form-urlencoded"
	MIMETextHTMLCharsetUTF8 = "text/html; charset=utf-8"
)

// HTTP Header Fields
const (
	HeaderAcceptEncoding = "Accept-Encoding"
	HeaderContentLength = "Content-Length"
	HeaderContentType = "Content-Type"
	HeaderUserAgent = "User-Agent"

	HeaderAllow = "Allow"
	HeaderContentEncoding = "Content-Encoding"
	HeaderServer = "Server"
	HeaderVary = "Vary"

	HeaderXForwardedFor = "X-Forwarded-For"
	HeaderXRealIP = "X-Real-IP"
)

// Predefine
var (
	Err = &Error{Code: http.StatusInternalServerError, Err: "Error"}

	ErrBadRequest = Err.WithCode(http.StatusBadRequest)
	ErrMethodNotAllowed = Err.WithCode(http.StatusMethodNotAllowed)
	ErrUnsupportedMediaType = Err.WithCode(http.StatusUnsupportedMediaType)
	ErrInternalServerError = Err.WithCode(http.StatusInternalServerError)
	ErrNotImplemented = Err.WithCode(http.StatusNotImplemented)
)
