package goblog

import (
	"strings"
	"reflect"
	"net/http"
	"unicode/utf8"
	"fmt"
	"net/textproto"
	"runtime"
	"strconv"
	"encoding"
	"encoding/json"
	"sync/atomic"
)

type middlewares []Middleware

func (m middlewares) run(ctx *Context) (err error) {
	for _, fn := range m {
		if err = fn(ctx); !IsNil(err) || ctx.Res.ended.isTrue() {
			return
		}
	}
	return
}

func Compose(mds ...Middleware) Middleware {
	switch len(mds) {
	case 0:
		return noOp
	case 1:
		return mds[0]
	default:
		return middlewares(mds).run
	}
}

var noOp Middleware = func(ctx *Context) error { return nil }

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
	return fmt.Sprintf("%s: %s", err.Err, err.Msg)
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

func ValuesToStruct(values map[string][]string, target interface{}, tag string) (err error) {
	if values == nil {
		return fmt.Errorf("invalid struct: %v", values)
	}
	if len(values) == 0 {
		return
	}
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("invalid struct: %v", rv)
	}

	rv = rv.Elem()
	rt := rv.Type()
	n := rv.NumField()

	for i := 0; i < n; i++ {
		fv := rv.Field(i)
		if !fv.CanSet() {
			continue
		}

		fk := rt.Field(i).Tag.Get(tag)
		if fk == "" {
			continue
		}

		if vals, ok := values[fk]; ok {
			if fv.Kind() == reflect.Slice {
				err = setRefSlice(fv, vals)
			} else if len(vals) > 0 {
				err = setRefField(fv, vals[0])
			}
			if err != nil {
				return
			}
		}
	}

	return
}

func shouldDeref(k reflect.Kind) bool {
	switch k {
	case reflect.String, reflect.Bool, reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func setRefSlice(v reflect.Value, vals []string) error {
	l := len(vals)
	slice := reflect.MakeSlice(v.Type(), l, l)

	for i := 0; i < l; i++ {
		if err := setRefField(slice.Index(i), vals[i]); err != nil {
			return err
		}
	}

	v.Set(slice)
	return nil
}

func setRefField(v reflect.Value, str string) error {
	if v.Kind() == reflect.Ptr && shouldDeref(v.Type().Elem().Kind()) {
		v.Set(reflect.New(v.Type().Elem()))
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(str)
		return nil
	case reflect.Bool:
		return setRefBool(v, str)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return setRefInt(v, str, v.Type().Bits())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return setRefUint(v, str, v.Type().Bits())
	case reflect.Float32, reflect.Float64:
		return setRefFloat(v, str, v.Type().Bits())
	default:
		return tryUnmarshalValue(v, str)
	}
}

func setRefBool(v reflect.Value, str string) error {
	val, err := strconv.ParseBool(str)
	if err == nil {
		v.SetBool(val)
	}
	return err
}

func setRefInt(v reflect.Value, str string, size int) error {
	val, err := strconv.ParseInt(str, 10, size)
	if err == nil {
		v.SetInt(val)
	}
	return err
}

func setRefUint(v reflect.Value, str string, size int) error {
	val, err := strconv.ParseUint(str, 10, size)
	if err == nil {
		v.SetUint(val)
	}
	return err
}

func setRefFloat(v reflect.Value, str string, size int) error {
	val, err := strconv.ParseFloat(str, size)
	if err == nil {
		v.SetFloat(val)
	}
	return err
}

func tryUnmarshalValue(v reflect.Value, str string) error {
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}

	if v.Type().NumMethod() > 0 {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		i := v.Interface()
		if u, ok := i.(encoding.TextUnmarshaler); ok {
			return u.UnmarshalText([]byte(str))
		}
		if u, ok := i.(json.Unmarshaler); ok {
			return u.UnmarshalJSON([]byte(str))
		}
	}
	return fmt.Errorf("unknown field type: %v", v.Type())
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

func (b *atomicBool) isTrue() bool {
	return atomic.LoadInt32((*int32)(b)) == 1
}