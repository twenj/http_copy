package main

import (
	"goblog"
	"goblog/logging"
	"os"
	"html/template"
	"io"
)

type RenderTest struct{
	tpl *template.Template
}

func (t *RenderTest) Render(ctx *goblog.Context, w io.Writer, name string, data interface{}) (err error) {
	if err = t.tpl.ExecuteTemplate(w, name, data); err != nil {
		err = goblog.ErrNotFound.From(err)
	}
	return
}

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

	router.Get("/home", func(ctx *goblog.Context) error{
		file, err := os.Open("template/home.html")
		if err != nil {
			return nil
		}
		return ctx.Stream(200, goblog.MIMETextHTMLCharsetUTF8, file)
	})

	router.Get("/index", func(ctx *goblog.Context) error {
		app.Set(goblog.SetRenderer, &RenderTest{
			tpl: template.Must(template.ParseFiles("template/home.html")),
		})
		return ctx.Render(200, "home", "Gear")
	})

	// try: http://127.0.0.1:3000/test?query=hello
	router.Otherwise(func(ctx *goblog.Context) error {
		return ctx.JSON(200, map[string]interface{}{
			"Host": ctx.Host,
			"Method": ctx.Method,
			"Path": ctx.Path,
			"URL": ctx.Req.URL.String(),
			"Headers": ctx.Req.Header,
		})
	})
	app.UseHandler(router)
	app.Error(app.Listen(":3000"))
}
