package main

import (
	"goblog"
)

func main() {
	goblog.New()
	// Add logging middleware
	//logging.Default(true)
	//app.UseHandler(logging.Default(true))
}
