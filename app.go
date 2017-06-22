package goblog

import (
	"net/http"
	"io"
	"time"
	"log"
	"context"
	"os"
	"encoding/json"
	"encoding/xml"
	"net/url"
)

type Middleware func(ctx *Context) error

type Handler interface {
	Serve(ctx *Context) error
}

type Renderer interface {
	Render(ctx *Context, w io.Writer, name string, data interface{}) error
}

type URLParser interface {
	Parse(val map[string][]string, body interface{}, tag string) error
}

type DefaultURLParser struct {}

func (d DefaultURLParser) Parse(val map[string][]string, body interface{}, tag string) error {
	return ValuesToStruct(val, body, tag)
}

type BodyParser interface {
	MaxBytes() int64
	Parse(buf []byte, body interface{}, mediaType, charset string) error
}

func (d DefaultBodyParser) MaxBytes() int64 {
	return int64(d)
}

func (d DefaultBodyParser) Parse(buf []byte, body interface{}, mediaType, charset string) error {
	if len(buf) == 0 {
		return ErrBadRequest.WithMsg("request entity empty")
	}
	switch mediaType {
	case MIMEApplicationJSON:
		return json.Unmarshal(buf, body)
	case MIMEApplicationXML:
		return xml.Unmarshal(buf, body)
	case MIMEApplicationForm:
		val, err := url.ParseQuery(string(buf))
		if err == nil {
			err = ValuesToStruct(val, body, "form")
		}
		return err
	}
	return ErrUnsupportedMediaType.WithMsg("unsupported media type")
}

type DefaultBodyParser int64

type HTTPError interface {
	Error() string
	Status() int
}

type App struct {
	Server *http.Server
	mds middlewares

	keys []string
	renderer Renderer
	bodyParser BodyParser
	urlParser URLParser
	compress Compressible
	timeout time.Duration
	serverName string
	logger *log.Logger
	onerror func(*Context, HTTPError)
	withContext func(*http.Request) context.Context
	settings map[interface{}]interface{}
}

func New() *App {
	app := new(App)
	app.Server = new(http.Server)
	app.mds = make(middlewares, 0)
	app.settings = make(map[interface{}]interface{})

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	app.Set(SetEnv, env)
	app.Set(SetServerName, "Gear/" + Version)
	app.Set(SetBodyParse, DefaultBodyParser(2 << 20))	//2MB
	app.Set(SetURLParser, DefaultURLParser{})
	app.Set(SetLogger, log.New(os.Stderr, "", log.LstdFlags))
	return app
}

func (app *App) UseHandler(h Handler) {
	app.mds = append(app.mds, h.Serve)
}

type appSetting uint8

const (
	SetBodyParse appSetting = iota

	SetURLParser

	SetCompress

	SetKeys

	SetLogger

	SetOnError

	SetRenderer

	SetTimeout

	SetWithContext

	SetEnv

	SetServerName
)

func (app *App) Set(key, val interface{}) {
	if k, ok := key.(appSetting); ok {
		switch key {
		case SetBodyParse:
			if bodyParser, ok := val.(BodyParser); !ok {
				panic(Err.WithMsg("SetBodyParse setting must implemented gear.BodyParser interface"))
			} else {
				app.bodyParser = bodyParser
			}
		case SetURLParser:
			if urlParser, ok := val.(URLParser); !ok {
				panic(Err.WithMsg("SetURLParser setting must implemented gear.URLParser interface"))
			} else {
				app.urlParser = urlParser
			}
		case SetCompress:
			if compress, ok := val.(Compressible); !ok {
				panic(Err.WithMsg("SetCompress setting must implemented gear.Compressible interface"))
			} else {
				app.compress = compress
			}
		case SetKeys:
			if keys, ok := val.([]string); !ok {
				panic(Err.WithMsg("SetKeys setting must be []string"))
			} else {
				app.keys = keys
			}
		case SetLogger:
			if logger, ok := val.(*log.Logger); !ok {
				panic(Err.WithMsg("SetLogger setting must be *log.Logger instance"))
			} else {
				app.logger = logger
			}
		case SetOnError:
			if onerror, ok := val.(func(ctx *Context, err HTTPError)); !ok {
				panic(Err.WithMsg("SetOnError setting must be func(ctx *Context, err *Error)"))
			} else {
				app.onerror = onerror
			}
		case SetRenderer:
			if renderer, ok := val.(Renderer); !ok {
				panic(Err.WithMsg("SetRenderer setting must be gear.Renderer interface"))
			} else {
				app.renderer = renderer
			}
		case SetTimeout:
			if timeout, ok := val.(time.Duration); !ok {
				panic(Err.WithMsg("SetTimeout setting must be time.Duration instance"))
			} else {
				app.timeout = timeout
			}
		case SetWithContext:
			if withContext, ok := val.(func(*http.Request) context.Context); !ok {
				panic(Err.WithMsg("SetWithContext setting must be func(*http.Request) context.Context"))
			} else {
				app.withContext = withContext
			}
		case SetEnv:
			if _, ok := val.(string); !ok {
				panic(Err.WithMsg("SetEnv setting must be string"))
			}
		case SetServerName:
			if name, ok := val.(string); !ok {
				panic(Err.WithMsg("SetServerName setting must be string"))
			} else {
				app.serverName = name
			}
		}
		app.settings[k] = val
		return
	}
	app.settings[key] = val
}

func (app *App) Listen(addr string) error {
	app.Server.Addr = addr
	app.Server.ErrorLog = app.logger
	app.Server.Handler = app
	return app.Server.ListenAndServe()
}

func (app *App) Error(err error) {
	if err := ErrorWithStack(err, 4); err != nil {
		app.logger.Println(err.String())
	}
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(app, w, r)

	if compressWriter := ctx.handleCompress(); compressWriter != nil {
		defer compressWriter.Close()
	}

	// recover panic error
	defer func() {
		if err := recover(); err != nil && err != http.ErrAbortHandler {
			ctx.Res.afterHooks = nil
			ctx.Res.ResetHeader()
			ctx.respondError(ErrorWithStack(err))
		}
	}()

	go func() {
		<-ctx.Done()
		ctx.Res.ended.setTrue()
	}()

	// process app middleware
	err := app.mds.run(ctx)
	if ctx.Res.wroteHeader.isTrue() {
		if !IsNil(err) {
			app.Error(err)
		}
		return
	}

	// if context canceled abnormally...
	if e := ctx.Err(); e != nil {
		if e == context.Canceled {
			ctx.Res.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = ErrGatewayTimeout.WithMsg(e.Error())
	}

	if !IsNil(err) {
		ctx.Error(err)
	} else {
		ctx.Res.WriteHeader(0)
	}
}