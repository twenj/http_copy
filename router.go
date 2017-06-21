package goblog

import (
	"github.com/teambition/trie-mux"
	"net/http"
	"strings"
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
