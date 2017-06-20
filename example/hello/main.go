package main

import (
	"goblog"
	"goblog/logging"
)

func main() {
	app := goblog.New()
	// Add logging middleware
	app.UseHandler(logging.Default(true))
}
