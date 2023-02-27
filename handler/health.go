package handler

import "github.com/gofiber/fiber/v2"

func (h *handler) Health(c *fiber.Ctx) error {
	return h.okRes(c, fiber.Map{
		"result": "ok",
	})
}
