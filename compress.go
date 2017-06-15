package goblog

type Compressible interface {
	Compressible(contentType string, contentLength int) bool
}
