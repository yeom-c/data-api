package route

import (
	"github.com/yeom-c/data-api/app"
	"github.com/yeom-c/data-api/handler"
	"github.com/yeom-c/data-api/middleware"
)

func SetRoutes() {
	h := handler.Handler()

	app.Server().Fiber.Get("/health", h.Health)
	app.Server().Fiber.Post("/sign-in", h.SignIn)

	app.Server().Fiber.Use(middleware.NewAuthMiddleware())

	app.Server().Fiber.Get("/dashboard", h.Dashboard)

	app.Server().Fiber.Get("/profile", h.Profile)
	app.Server().Fiber.Post("/profile", h.StoreProfile)

	app.Server().Fiber.Post("/server/list", h.ServerList)

	dataSchemaRoute := app.Server().Fiber.Group("/data-schema")
	dataSchemaRoute.Post("/list", h.DataSchemaList)
	dataSchemaRoute.Get("/", h.DataSchema)
	dataSchemaRoute.Post("/", h.StoreDataSchema)
	dataSchemaRoute.Patch("/", h.UpdateDataSchema)
	dataSchemaRoute.Post("/apply", h.ApplyDataVersion)
	dataSchemaRoute.Post("/unapply", h.UnapplyDataVersion)
	dataSchemaRoute.Post("/refresh", h.RefreshDataSchema)
	dataSchemaRoute.Delete("/", h.DeleteDataSchema)

	dataTableRoute := app.Server().Fiber.Group("/data-table")
	dataTableRoute.Post("/list", h.DataTableList)
	dataTableRoute.Get("/:id", h.DataTable)
	dataTableRoute.Post("/", h.StoreDataTable)
	dataTableRoute.Delete("/:id", h.DeleteDataTable)

	dataEnumRoute := app.Server().Fiber.Group("/data-enum")
	dataEnumRoute.Post("/", h.StoreDataEnum)
	dataEnumRoute.Post("/refresh", h.RefreshDataEnum)

	app.Server().Fiber.Post("/data-map", h.StoreDataMap)

	dataVersionRoute := app.Server().Fiber.Group("/data-version")
	dataVersionRoute.Post("/list", h.DataVersionList)
	dataVersionRoute.Patch("/", h.UpdateDataVersion)

	mapVersionRoute := app.Server().Fiber.Group("/map-version")
	mapVersionRoute.Post("/list", h.MapVersionList)
	mapVersionRoute.Patch("/", h.UpdateMapVersion)
	mapVersionRoute.Post("/apply", h.ApplyMapVersion)
	mapVersionRoute.Post("/unapply", h.UnapplyMapVersion)
}
