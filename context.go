package goblog

import (
	"net/http"
	"github.com/go-http-utils/cookie"
	"net/url"
	"context"
	"net"
	"strings"
	"encoding/json"
	"github.com/go-http-utils/negotiator"
	"io"
	"bytes"
)

type contextKey int

const (
	isContext contextKey = iota
	isRecursive
	paramsKey
)

type Any interface {
	New(ctx *Context) (interface{}, error)
}

type Context struct {
	app 	*App
	Req 	*http.Request
	Res 	*Response
	Cookies *cookie.Cookies

	Host   string
	Method string
	Path   string

	query 	  url.Values
	ctx 	  context.Context
	_ctx 	  context.Context
	cancelCtx context.CancelFunc
	kv 		  map[interface{}]interface{}
}

func NewContext(app *App, w http.ResponseWriter, r *http.Request) *Context {
	ctx := Context{
		app: app,
		Req: r,
		Res: &Response{w: w, rw: w},

		Host: r.Host,
		Method: r.Method,
		Path: r.URL.Path,

		Cookies: cookie.New(w, r, app.keys...),
		kv: make(map[interface{}]interface{}),
	}

	if app.serverName != "" {
		ctx.Set(HeaderServer, app.serverName)
	}

	if app.timeout <= 0 {
		ctx.ctx, ctx.cancelCtx = context.WithCancel(r.Context())
	} else {
		ctx.ctx, ctx.cancelCtx = context.WithTimeout(r.Context(), app.timeout)
	}
	ctx.ctx = context.WithValue(ctx.ctx, isContext, isContext)

	if app.withContext != nil {
		ctx._ctx = app.withContext(r.WithContext(ctx.ctx))
		if ctx._ctx.Value(isContext) == nil {
			panic(Err.WithMsg("the context is not created from gear.Context"))
		}
	} else {
		ctx._ctx = ctx.ctx
	}
	return &ctx
}

func (ctx *Context) Done() <-chan struct{} {
	return ctx.ctx.Done()
}

func (ctx *Context) Err() error {
	return ctx.ctx.Err()
}

func (ctx *Context) Any(any interface{}) (val interface{}, err error) {
	var ok bool
	if val, ok = ctx.kv[any]; !ok {
		switch v := any.(type) {
		case Any:
			if val, err = v.New(ctx); err == nil {
				ctx.kv[any] = val
			}
		default:
			return nil, Err.WithMsg("non-existent key")
		}
	}
	return
}

func (ctx *Context) SetAny(key, val interface{}) {
	ctx.kv[key] = val
}

func (ctx *Context) IP() net.IP {
	ra := ctx.Req.RemoteAddr
	if ip := ctx.Req.Header.Get(HeaderXForwardedFor); ip != "" {
		ra = ip
	} else if ip := ctx.Req.Header.Get(HeaderXRealIP); ip != "" {
		ra = ip
	} else {
		ra, _, _ = net.SplitHostPort(ra)
	}
	if index := strings.IndexByte(ra, ','); index >= 0 {
		ra = ra[0:index]
	}
	return net.ParseIP(strings.TrimSpace(ra))
}

func (ctx *Context) AcceptEncoding(preferred ...string) string {
	return negotiator.New(ctx.Req.Header).Language(preferred...)
}

func (ctx *Context) Get(key string) string {
	return ctx.Req.Header.Get(key)
}

func (ctx *Context) Set(key, value string) {
	ctx.Res.Set(key, value)
}

func (ctx *Context) Status(code int) {
	ctx.Res.status = code
}

func (ctx *Context) Type(str string) {
	ctx.Res.Set(HeaderContentType, str)
}

func (ctx *Context) HTML(code int, str string) error {
	ctx.Type(MIMETextHTMLCharsetUTF8)
	return ctx.End(code, []byte(str))
}

func (ctx *Context) JSON(code int, val interface{}) error {
	buf, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return ctx.JSONBlob(code, buf)
}

func (ctx *Context) JSONBlob(code int, buf []byte) error {
	ctx.Type(MIMEApplicationJSONCharsetUTF8)
	return ctx.End(code, buf)
}

func (ctx *Context) Redirect(url string) (err error) {
	if ctx.Res.ended.swapTrue() {
		if !isRedirectStatus(ctx.Res.status) {
			ctx.Res.status = http.StatusFound
		}
		http.Redirect(ctx.Res, ctx.Req, url, ctx.Res.status)
	}
	return
}

func (ctx *Context) Render(code int, name string, data interface{}) (err error) {
	if ctx.app.renderer == nil {
		return Err.WithMsg("renderer not registered")
	}
	buf := new(bytes.Buffer)
	if err = ctx.app.renderer.Render(ctx, buf, name, data); err == nil {
		ctx.Type(MIMETextHTMLCharsetUTF8)
		return ctx.End(code,buf.Bytes())
	}
	return
}

func (ctx *Context) Stream(code int, contentType string, r io.Reader) (err error) {
	if ctx.Res.ended.swapTrue() {
		ctx.Status(code)
		ctx.Type(contentType)
		_, err = io.Copy(ctx.Res, r)
	}
	return
}

func (ctx *Context) Error(e error) error {
	ctx.Res.afterHooks = nil	// clear afterHooks when any error
	ctx.Res.ResetHeader()
	err := ParseError(e, ctx.Res.status)
	if err == nil {
		err = ErrInternalServerError.WithMsg("nil error")
	}
	if ctx.app.onerror != nil {
		ctx.app.onerror(ctx, err)
	}
	// try to respond error if 'OnError' does't do it
	ctx.respondError(err)
	return nil
}

func (ctx *Context) End(code int, buf ...[]byte) (err error) {
	if ctx.Res.ended.swapTrue() {
		var body []byte
		if len(buf) > 0 {
			body = buf[0]
		}
		err = ctx.Res.respond(code, body)
	}
	return
}

func (ctx *Context) OnEnd(hook func()) {
	if ctx.Res.ended.isTrue() {
		panic(Err.WithMsg(`can't add "end hook" after middleware process ended`))
	}
	ctx.Res.endHooks = append(ctx.Res.endHooks, hook)
}

func (ctx *Context) respondError(err HTTPError) {
	if !ctx.Res.wroteHeader.isTrue() {
		code := err.Status()
		// we don't need to logging 501, 4xx errors
		if code == 500 || code > 501 || code < 400 {
			ctx.app.Error(err)
		}
		// try to render error as json
		ctx.Set(HeaderContentType, MIMEApplicationJSONCharsetUTF8)
		ctx.Set(HeaderXContentTypeOptions, "nosniff")

		buf, _ := json.Marshal(err)
		ctx.Res.respond(code, buf)
	}
}

func (ctx *Context) handleCompress() (cw *compressWriter) {
	if ctx.app.compress != nil && ctx.Method != http.MethodHead && ctx.Method != http.MethodOptions {
		if cw = newCompress(ctx.Res, ctx.app.compress, ctx.AcceptEncoding("gzip", "deflate")); cw != nil {
			ctx.Res.rw = cw //override with http.ResponseWriter wrapper.
		}
	}
	return
}