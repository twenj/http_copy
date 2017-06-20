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