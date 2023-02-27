package handler

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yeom-c/data-api/auth_token"
	"github.com/yeom-c/data-api/database"
	"github.com/yeom-c/data-api/middleware"
	"github.com/yeom-c/data-api/util"
)

func (h *handler) Profile(c *fiber.Ctx) error {
	payload := c.UserContext().Value(middleware.AuthorizationPayloadKey).(*auth_token.Payload)

	user := database.User{
		Id: int32(payload.Id),
	}
	has, err := database.Database().DataConn.Get(&user)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if !has {
		return h.errRes(c, fiber.StatusNotFound, "유저 정보 없음")
	}

	var passwordChangedAt, joinedAt, retiredAt string
	if user.PasswordChangedAt.Valid {
		passwordChangedAt = user.PasswordChangedAt.Time.Format(time.RFC3339)
	}
	if user.JoinedAt.Valid {
		joinedAt = user.JoinedAt.Time.Format("2006-01-02")
	}
	if user.RetiredAt.Valid {
		retiredAt = user.RetiredAt.Time.Format("2006-01-02")
	}

	return h.okRes(c, profileRes{
		User: userRes{
			Id:                user.Id,
			EmployeeId:        user.EmployeeId.Int32,
			Email:             user.Email,
			Name:              user.Name,
			Position:          user.Position,
			Color:             user.Color,
			PasswordChangedAt: passwordChangedAt,
			JoinedAt:          joinedAt,
			RetiredAt:         retiredAt,
			CreatedAt:         user.CreatedAt,
		},
	})
}

func (h *handler) StoreProfile(c *fiber.Ctx) error {
	payload := c.UserContext().Value(middleware.AuthorizationPayloadKey).(*auth_token.Payload)

	var req storeProfileReq
	if err := c.BodyParser(&req); err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("요청 파라미터 오류: %s", err.Error()))
	}

	updateUser := map[string]interface{}{
		"name":     req.Name,
		"position": req.Position,
		"color":    req.Color,
	}
	if req.Password != "" {
		hPassword, err := util.HashPassword(req.Password)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}

		updateUser["hashed_password"] = hPassword
		updateUser["password_changed_at"] = time.Now()
	}

	_, err := database.Database().DataConn.Table(new(database.User)).ID(payload.Id).Update(updateUser)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	return h.okRes(c, fiber.Map{"result": "ok"})
}
