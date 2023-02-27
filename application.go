package main

import (
	"log"
	"time"

	"github.com/yeom-c/data-api/app"
	"github.com/yeom-c/data-api/middleware"
	"github.com/yeom-c/data-api/route"
)

func main() {
	loc, err := time.LoadLocation("UTC")
	time.Local = loc

	middleware.SetMiddleware()
	route.SetRoutes()

	err = app.Server().Run()
	if err != nil {
		log.Fatal("fatal to start server: ", err)
	}
}
