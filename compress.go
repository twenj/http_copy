package goblog

import (
	"io"
	"net/http"
	"compress/gzip"
	"compress/flate"
)

type Compressible interface {
	Compressible(contentType string, contentLength int) bool
}

type compressWriter struct {
	compress Compressible
	encoding string
	writer io.WriteCloser
	res *Response
	rw http.ResponseWriter
}

func newCompress(res *Response, c Compressible, encoding string) *compressWriter {
	switch encoding {
	case "gzip", "deflate":
		return &compressWriter{
			compress: c,
			res:      res,
			rw:       res.rw,
			encoding: encoding,
		}
	default:
		return nil
	}
}

func (cw *compressWriter) WriteHeader(code int) {
	defer cw.rw.WriteHeader(code)

	if !isEmptyStatus(code) &&
		cw.compress.Compressible(cw.res.Get(HeaderContentType), len(cw.res.body)) {
		var w io.WriteCloser

		switch cw.encoding {
		case "gzip":
			w, _ = gzip.NewWriterLevel(cw.rw, gzip.DefaultCompression)
		case "deflate":
			w, _ = flate.NewWriter(cw.rw, flate.DefaultCompression)
		}

		if w != nil {
			cw.writer = w
			cw.res.Del(HeaderContentLength)
			cw.res.Set(HeaderContentEncoding, cw.encoding)
			cw.res.Vary(HeaderAcceptEncoding)
		}
	}
}

func (cw *compressWriter) Header() http.Header {
	return cw.rw.Header()
}

func (cw *compressWriter) Write(b []byte) (int, error) {
	if cw.writer != nil {
		return cw.writer.Write(b)
	}
	return cw.rw.Write(b)
}

func (cw *compressWriter) Close() error {
	if cw.writer != nil {
		return cw.writer.Close()
	}
	return nil
}