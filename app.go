package goblog

import (
	"net/http"
	"io"
	"time"
	"log"
	"context"
	"os"
)

type Middlewares func(ctx *Context) error

type Renderer interface {
	Render(ctx *Context, w io.Writer, name string, data interface{}) error
}

type URLParser interface {
	Parse(val map[string][]string, body interface{}, tag string) error
}

type BodyParser interface {
	MaxBytes() int64
	Parse(buf []byte, body interface{}, mediaType, charset string) error
}

type HTTPError interface {
	Error() string
	Status() int
}

type App struct {
	Server *http.Server
	mds middlewares

	keys []string
	renderer Renderer
	bodyParse BodyParser
	urlParse URLParser
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
	app.Set(SetEnv, env)
	app.Set(SetEnv, env)
	app.Set(SetEnv, env)
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