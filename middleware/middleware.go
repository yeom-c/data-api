package middleware

import (
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/yeom-c/data-api/app"
)

func SetMiddleware() {
	app.Server().Fiber.Use(recover.New())
	app.Server().Fiber.Use(logger.New())
	app.Server().Fiber.Use(cors.New(cors.Config{
		AllowOrigins:     app.Config().CorsHost,
		AllowCredentials: true,
	}))
}
