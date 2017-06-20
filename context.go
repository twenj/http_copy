package goblog

import (
	"net/http"
	"github.com/go-http-utils/cookie"
	"net/url"
	"context"
	"net"
	"strings"
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

func (ctx *Context) Get(key string) string {
	return ctx.Req.Header.Get(key)
}

func (ctx *Context) OnEnd(hook func()) {
	if ctx.Res.ended.isTrue() {
		panic(Err.WithMsg(`can't add "end hook" after middleware process ended`))
	}
	ctx.Res.endHooks = append(ctx.Res.endHooks, hook)
}