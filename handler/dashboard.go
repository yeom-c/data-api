package handler

import (
	"github.com/gofiber/fiber/v2"
)

func (h *handler) Dashboard(c *fiber.Ctx) error {
	return h.okRes(c, dashboardRes{})
}
