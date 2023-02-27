package app

import (
	"fmt"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"sync"
)

type server struct {
	Fiber *fiber.App
}

var once sync.Once
var instance *server

func Server() *server {
	once.Do(func() {
		if instance == nil {
			instance = &server{}

			instance.Fiber = fiber.New(fiber.Config{
				JSONEncoder: json.Marshal,
				JSONDecoder: json.Unmarshal,
			})
		}
	})

	return instance
}

func (svr *server) Run() error {
	return svr.Fiber.Listen(fmt.Sprintf(":%d", Config().ServerPort))
}
