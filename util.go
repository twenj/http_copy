package goblog

import (
	"strings"
	"reflect"
	"net/http"
	"unicode/utf8"
	"fmt"
	"net/textproto"
	"runtime"
)

type middlewares []Middlewares

type atomicBool int32

type Error struct {
	Code  int          `json:"-"`
	Err   string       `json:"error"`
	Msg   string 	   `json:"message"`
	Data  interface{} `json:"data,omitempty"`
	Stack string       `json:"-"`
}

func IsNil(val interface{}) bool {
	if val == nil {
		return true
	}

	value := reflect.ValueOf(val)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func (err *Error) Error() string {
	return fmt.Sprint("%s: %s", err.Err, err.Msg)
}

func (err Error) String() string {
	return err.GoString()
}

func (err Error) GoString() string {
	if v, ok := err.Data.([]byte); ok && utf8.Valid(v) {
		err.Data = string(v)
	}
	return fmt.Sprintf(`Error{Code:%d, Err:"%s", Msg:"%s", Data:%#v, Stack:"%s"}`,
		err.Code, err.Err, err.Msg, err.Data, err.Stack)
}

func (err Error) WithMsg(msgs ...string) *Error {
	if len(msgs) > 0{
		err.Msg = strings.Join(msgs, ", ")
	}
	return &err
}

func (err Error) WithMsgf(format string, args ...interface{}) *Error {
	return err.WithMsg(fmt.Sprintf(format, args...))
}

func (err Error) WithCode(code int) *Error {
	err.Code = code
	if text := http.StatusText(code); text != "" {
		err.Err = text
	}
	return &err
}

func (err Error) From(e error) *Error {
	if IsNil(e) {
		return nil
	}

	switch v := e.(type) {
	case *Error:
		return v
	case HTTPError:
		err.Code = v.Status()
		err.Msg = v.Error()
	case *textproto.Error:
		err.Code = v.Code
		err.Msg = v.Msg
	default:
		err.Msg = e.Error()
	}

	if err.Err == "" {
		err.Err = http.StatusText(err.Code)
	}
	return &err
}

func ErrorWithStack(val interface{}, skip ...int) *Error {
	if IsNil(val) {
		return nil
	}

	var err *Error
	switch v := val.(type) {
	case *Error:
		err = v.WithMsg() // must clone, should not change the origin *Error instance
	case error:
		err = ErrInternalServerError.From(v)
	case string:
		err = ErrInternalServerError.WithMsg(v)
	default:
		err = ErrInternalServerError.WithMsgf("%#v", v)
	}

	if err.Stack == "" {
		buf := make([]byte, 2048)
		buf = buf[:runtime.Stack(buf, false)]
		s := 1
		if len(skip) != 0 {
			s = skip[0]
		}
		err.Stack = pruneStack(buf, s)
	}
	return err
}

func pruneStack(stack []byte, skip int) string {
	lines := strings.Split(string(stack), "\n")[1:]
	newLines := make([]string, 0, len(lines)/2)

	num := 0
	for idx, line := range lines {
		if idx%2 == 0 {
			continue
		}
		skip--
		if skip >= 0 {
			continue
		}
		num++

		loc := strings.Split(line, " ")[0]
		loc = strings.Replace(loc, "\t", "\\t", -1)
		// only need odd line
		newLines = append(newLines, loc)
		if num == 10 {
			break
		}

	}
	return strings.Join(newLines, "\\n")
}