package main

import (
	"goblog"
	"goblog/logging"
)

func main() {
	app := goblog.New()

	// Add logging middleware
	app.UseHandler(logging.Default(true))

	// Add router middleware
	router := goblog.NewRouter()

	// try: http://127.0.0.1:3000/hello
	router.Get("/hello", func(ctx *goblog.Context) error {
		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
	})

	// try: http://127.0.0.1:3000/test?query=hello
	/*router.Otherwise(func(ctx *goblog.Context) error {
		return ctx.JSON(200, map[string]interface{}{
			"Host": ctx.Host,
			"Method": ctx.Method,
			"Path": ctx.Path,
			"URL": ctx.Req.URL.String(),
			"Headers": ctx.Req.Header,
		})
	})*/
	app.UseHandler(router)
	app.Error(app.Listen(":3000"))
}
