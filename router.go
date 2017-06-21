package goblog

import (
	"net/http"
	"strings"
	"github.com/teambition/trie-mux"
)

type Router struct {
	root       string
	rt         string
	trie       *trie.Trie
	otherwise  Middleware
	middleware Middleware
	mds        []Middleware
}

type RouterOptions struct {
	Root string
	IgnoreCase bool
	FixedPathRedirect bool
	TrailingSlashRedirect bool
}

var defaultRouterOptions = RouterOptions{
	Root:                  "/",
	IgnoreCase:            true,
	FixedPathRedirect:     true,
	TrailingSlashRedirect: true,
}

func NewRouter(routerOptions ...RouterOptions) *Router {
	opts := defaultRouterOptions
	if len(routerOptions) > 0 {
		opts = routerOptions[0]
	}
	if opts.Root == "" || opts.Root[len(opts.Root) - 1] != '/' {
		opts.Root += "/"
	}

	return &Router{
		root: opts.Root,
		rt: opts.Root[0 : len(opts.Root)-1],
		mds: make([]Middleware, 0),
		trie: trie.New(trie.Options{
			IgnoreCase:            opts.IgnoreCase,
			FixedPathRedirect:     opts.FixedPathRedirect,
			TrailingSlashRedirect: opts.TrailingSlashRedirect,
		}),
	}
}

func (r *Router) Handle(method, pattern string, handlers ...Middleware) {
	if method == "" {
		panic(Err.WithMsg("invalid method"))
	}
	if len(handlers) == 0 {
		panic(Err.WithMsg("invalid middleware"))
	}
	r.trie.Define(pattern).Handle(strings.ToUpper(method), Compose(handlers...))
}

func (r *Router) Get(pattern string, handlers ...Middleware) {
	r.Handle(http.MethodHead, pattern, handlers...)
}

func (r *Router) Otherwise(handlers ...Middleware) {
	if len(handlers) == 0 {
		panic(Err.WithMsg("invalid middleware"))
	}
	r.otherwise = Compose(handlers...)
}

func (r *Router) Serve(ctx *Context) error {
	path := ctx.Path
	method := ctx.Method
	var handler Middleware

	if !strings.HasPrefix(path, r.root) && path != r.rt {
		return nil
	}

	if path == r.rt {
		path = "/"
	} else if l := len(r.rt); l > 0 {
		path = path[l:]
	}

	matched := r.trie.Match(path)
	if matched.Node == nil {
		//FixedPathRedirect or TrailingSlashRedirect
		if matched.TSR != "" || matched.FPR != "" {
			ctx.Req.URL.Path = matched.TSR
			if matched.FPR != "" {
				ctx.Req.URL.Path = matched.FPR
			}
			if len(r.root) > 1 {
				ctx.Req.URL.Path = r.root + ctx.Req.URL.Path[1:]
			}

			code := http.StatusMovedPermanently
			if method != "GET" {
				code = http.StatusTemporaryRedirect
			}
			ctx.Status(code)
			return ctx.Redirect(ctx.Req.URL.String())
		}

		if r.otherwise == nil {
			return ErrNotImplemented.WithMsgf(`"%s" is not implemented`, ctx.Path)
		}
		handler = r.otherwise
	} else {
		ok := false
		if handler, ok = matched.Node.GetHandler(method).(Middleware); !ok {
			// OPTIONS support
			if method == http.MethodOptions {
				ctx.Set(HeaderAllow, matched.Node.GetAllow())
				return ctx.End(http.StatusNoContent)
			}

			if r.otherwise == nil {
				// If no route handler is returned, it's a 405 error
				ctx.Set(HeaderAllow, matched.Node.GetAllow())
				return ErrMethodNotAllowed.WithMsgf(`"%s" is not allowed in "%s"`, method, ctx.Path)
			}
			handler = r.otherwise
		}
	}

	ctx.SetAny(paramsKey, matched.Params)
	if len(r.mds) > 0 {
		handler = Compose(r.middleware, handler)
	}
	return nil
}