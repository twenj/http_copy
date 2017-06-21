package goblog

import "net/http"

type Response struct {
	status 		int	// response Status Code
	body 		[]byte	// the response content
	afterHooks 	[]func()
	endHooks 	[]func()
	ended 		atomicBool
	wroteHeader atomicBool
	w 			http.ResponseWriter
	rw 			http.ResponseWriter
}

func (r *Response) Status() int {
	return r.status
}

func (r *Response) Body() []byte {
	return r.body
}

func (r *Response) Set(key, value string) {
	r.Header().Set(key, value)
}

func (r *Response) Header() http.Header {
	return r.rw.Header()
}

func (r *Response) Write(buf []byte) (int, error) {
	if !r.wroteHeader.isTrue() {
		if r.status == 0 {
			r.status = 200
		}
		r.WriteHeader(0)
	}
	return r.rw.Write(buf)
}

func (r *Response) WriteHeader(code int) {
	if !r.wroteHeader.swapTrue() {
		return
	}
	// ensure that ended is true
	r.ended.setTrue()

	// set status before afterHooks
	if code > 0 {
		r.status = code
	}

	// execute "after hooks" with LIFO order before Response.WriteHeader
	runHooks(r.afterHooks)

	// check status, r.status maybe changed in afterHooks
	if !IsStatusCode(r.status) {
		if r.body != nil {
			r.status = http.StatusOK
		} else {
			r.status = 421
		}
	} else if isEmptyStatus(r.status) {
		r.body = nil
	}

	r.rw.WriteHeader(r.status)

	if len(r.endHooks) > 0 {
		go runHooks(r.endHooks)
	}
}

func (r *Response) respond(status int, body []byte) (err error) {
	r.body = body
	r.WriteHeader(status)
	if r.body != nil {
		_, err = r.Write(r.body)
	}
	return
}

func runHooks(hooks []func()) {
	// run hooks in LIFO order
	for i := len(hooks) - 1; i >= 0; i-- {
		hooks[i]()
	}
}