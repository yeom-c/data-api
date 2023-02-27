package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/yeom-c/data-api/auth_token"
	"github.com/yeom-c/data-api/database"
	"github.com/yeom-c/data-api/middleware"
	"github.com/yeom-c/data-api/util"
)

func (h *handler) DataVersionList(c *fiber.Ctx) error {
	payload := c.UserContext().Value(middleware.AuthorizationPayloadKey).(*auth_token.Payload)
	userId := int32(payload.Id)
	if userId == 0 {
		return h.errRes(c, fiber.StatusBadRequest, "유저 정보 없음")
	}

	var req listReq
	err := c.BodyParser(&req)
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, "요청 데이터 오류")
	}

	countDbSession := database.Database().DataConn.
		Join("LEFT", "user", "user.id = data_version.user_id").
		Desc("data_version.id")
	dbSession := database.Database().DataConn.
		Join("LEFT", "user", "user.id = data_version.user_id").
		Desc("data_version.id")

	// filters
	util.SetFilter(req.Filter, countDbSession)
	util.SetFilter(req.Filter, dbSession)

	total, err := countDbSession.Count(database.DataVersionUser{})
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	dataVersionUserList := []database.DataVersionUser{}
	if req.Page > 0 {
		dbSession = dbSession.Limit(int(req.Limit), int((req.Page-1)*req.Limit))
	}
	err = dbSession.Find(&dataVersionUserList)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	dataVersionIdList := []int32{}
	for _, dataVersionUser := range dataVersionUserList {
		dataVersionIdList = append(dataVersionIdList, dataVersionUser.DataVersion.Id)
	}

	uploadRefList := []database.UploadRef{}
	err = database.Database().DataConn.
		Where("ref_table = 'data_version'").
		In("ref_id", dataVersionIdList).
		Find(&uploadRefList)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	uploadIdList := []int32{}
	mapDataVersionIdUploadIdList := map[int32][]int32{}
	for _, uploadRef := range uploadRefList {
		uploadIdList = append(uploadIdList, uploadRef.UploadId)
		mapDataVersionIdUploadIdList[uploadRef.RefId] = append(mapDataVersionIdUploadIdList[uploadRef.RefId], uploadRef.UploadId)
	}

	mapUploadIdUpload := make(map[int32]database.Upload)
	err = database.Database().DataConn.
		In("id", uploadIdList).
		Find(&mapUploadIdUpload)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	var dataTableId int32
	dataVersionList := []dataVersionWithUpload{}
	for _, dataVersionUser := range dataVersionUserList {
		if dataTableId == 0 {
			dataTableId = dataVersionUser.DataTableId
		}
		uploadList := []database.Upload{}
		for _, uploadId := range mapDataVersionIdUploadIdList[dataVersionUser.DataVersion.Id] {
			uploadList = append(uploadList, mapUploadIdUpload[uploadId])
		}
		dataVersionList = append(dataVersionList, dataVersionWithUpload{
			DataVersion: dataVersionUser.DataVersion,
			User: simpleUserRes{
				Id:         dataVersionUser.User.Id,
				EmployeeId: dataVersionUser.User.EmployeeId.Int32,
				Email:      dataVersionUser.User.Email,
				Name:       dataVersionUser.User.Name,
				Position:   dataVersionUser.User.Position,
				Color:      dataVersionUser.User.Color,
			},
			UploadList: uploadList,
		})
	}

	// data_table_uploader 제거.
	database.Database().DataConn.Table("data_table_uploader").Where("data_table_id = ?", dataTableId).Where("user_id = ?", userId).Delete()

	return h.okRes(c, dataVersionListRes{
		DataVersionList: dataVersionList,
		Total:           total,
	})
}

func (h *handler) UpdateDataVersion(c *fiber.Ctx) error {
	var req storeDataVersionReq
	if err := c.BodyParser(&req); err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("요청 파라미터 오류: %s", err.Error()))
	}

	dataVersion := database.DataVersion{Id: req.Id}
	has, err := database.Database().DataConn.Get(&dataVersion)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if !has {
		return h.errRes(c, fiber.StatusNotFound, "데이터 버전 정보 없음")
	}

	// 버프 수정.
	dataVersion.MemoTitle = req.MemoTitle
	dataVersion.Memo = req.Memo
	_, err = database.Database().DataConn.Cols("memo_title", "memo").ID(dataVersion.Id).Update(&dataVersion)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	return h.okRes(c, fiber.Map{"result": "ok"})
}
