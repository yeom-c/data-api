package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yeom-c/data-api/database"
	"github.com/yeom-c/data-api/util"
)

func (h *handler) ServerList(c *fiber.Ctx) error {
	var req listReq
	err := c.BodyParser(&req)
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, "요청 데이터 오류")
	}

	countDbSession := database.Database().DataConn.
		Desc("created_at")
	dbSession := database.Database().DataConn.
		Desc("created_at")

	// filters
	util.SetFilter(req.Filter, countDbSession)
	util.SetFilter(req.Filter, dbSession)

	total, err := countDbSession.Count(database.Server{})
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	serverList := []database.Server{}
	if req.Page > 0 {
		dbSession = dbSession.Limit(int(req.Limit), int((req.Page-1)*req.Limit))
	}
	err = dbSession.Find(&serverList)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	return h.okRes(c, serverListRes{
		ServerList: serverList,
		Total:      total,
	})
}
