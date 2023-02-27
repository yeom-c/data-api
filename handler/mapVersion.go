package handler

import (
	"fmt"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/yeom-c/data-api/auth_token"
	"github.com/yeom-c/data-api/database"
	"github.com/yeom-c/data-api/enum"
	"github.com/yeom-c/data-api/middleware"
	"github.com/yeom-c/data-api/util"
	"xorm.io/xorm"
)

func (h *handler) MapVersionList(c *fiber.Ctx) error {
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
		Join("LEFT", "user", "user.id = map_version.user_id").
		Desc("map_version.id")
	dbSession := database.Database().DataConn.
		Join("LEFT", "user", "user.id = map_version.user_id").
		Desc("map_version.id")

	// filters
	util.SetFilter(req.Filter, countDbSession)
	util.SetFilter(req.Filter, dbSession)

	total, err := countDbSession.Count(database.MapVersionUser{})
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	mapVersionUserList := []database.MapVersionUser{}
	if req.Page > 0 {
		dbSession = dbSession.Limit(int(req.Limit), int((req.Page-1)*req.Limit))
	}
	err = dbSession.Find(&mapVersionUserList)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	mapVersionIdList := []int32{}
	for _, mapVersionUser := range mapVersionUserList {
		mapVersionIdList = append(mapVersionIdList, mapVersionUser.MapVersion.Id)
	}

	uploadRefList := []database.UploadRef{}
	err = database.Database().DataConn.
		Where("ref_table = 'map_version'").
		In("ref_id", mapVersionIdList).
		Find(&uploadRefList)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	uploadIdList := []int32{}
	mapMapVersionIdUploadIdList := map[int32][]int32{}
	for _, uploadRef := range uploadRefList {
		uploadIdList = append(uploadIdList, uploadRef.UploadId)
		mapMapVersionIdUploadIdList[uploadRef.RefId] = append(mapMapVersionIdUploadIdList[uploadRef.RefId], uploadRef.UploadId)
	}

	mapUploadIdUpload := make(map[int32]database.Upload)
	err = database.Database().DataConn.
		In("id", uploadIdList).
		Find(&mapUploadIdUpload)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	var dataTableId int32
	mapVersionList := []mapVersionWithUpload{}
	for _, mapVersionUser := range mapVersionUserList {
		if dataTableId == 0 {
			dataTableId = mapVersionUser.DataTableId
		}
		uploadList := []database.Upload{}
		for _, uploadId := range mapMapVersionIdUploadIdList[mapVersionUser.MapVersion.Id] {
			uploadList = append(uploadList, mapUploadIdUpload[uploadId])
		}
		mapVersionList = append(mapVersionList, mapVersionWithUpload{
			MapVersion: mapVersionUser.MapVersion,
			User: simpleUserRes{
				Id:         mapVersionUser.User.Id,
				EmployeeId: mapVersionUser.User.EmployeeId.Int32,
				Email:      mapVersionUser.User.Email,
				Name:       mapVersionUser.User.Name,
				Position:   mapVersionUser.User.Position,
				Color:      mapVersionUser.User.Color,
			},
			UploadList: uploadList,
		})
	}

	// data_table_uploader 제거.
	database.Database().DataConn.Table("data_table_uploader").Where("data_table_id = ?", dataTableId).Where("user_id = ?", userId).Delete()

	return h.okRes(c, mapVersionListRes{
		MapVersionList: mapVersionList,
		Total:          total,
	})
}

func (h *handler) UpdateMapVersion(c *fiber.Ctx) error {
	var req storeMapVersionReq
	if err := c.BodyParser(&req); err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("요청 파라미터 오류: %s", err.Error()))
	}

	mapVersion := database.MapVersion{Id: req.Id}
	has, err := database.Database().DataConn.Get(&mapVersion)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if !has {
		return h.errRes(c, fiber.StatusNotFound, "맵 버전 정보 없음")
	}

	// 버프 수정.
	mapVersion.MemoTitle = req.MemoTitle
	mapVersion.Memo = req.Memo
	_, err = database.Database().DataConn.Cols("memo_title", "memo").ID(mapVersion.Id).Update(&mapVersion)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	return h.okRes(c, fiber.Map{"result": "ok"})
}

func (h *handler) ApplyMapVersion(c *fiber.Ctx) error {
	var req applyDataVersionReq
	if err := c.BodyParser(&req); err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("요청 파라메터 오류: %s", err.Error()))
	}

	// 데이터 스키마 체크.
	dataSchemaServer := database.DataSchemaServer{}
	has, err := database.Database().DataConn.
		Join("LEFT", "server", "server.id = data_schema.server_id").
		Where("data_schema.type = ?", enum.DataTableTypeMap).
		Where("data_schema.server_id = ?", req.ServerId).
		Get(&dataSchemaServer)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if !has {
		return h.errRes(c, fiber.StatusNotFound, "맵 스키마 정보 없음")
	}

	mapVersionDataTable := database.MapVersionDataTable{}
	has, err = database.Database().DataConn.
		Join("LEFT", "data_table", "data_table.id = map_version.data_table_id").
		Where("map_version.id = ?", req.VersionId).
		Get(&mapVersionDataTable)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if !has {
		return h.errRes(c, fiber.StatusNotFound, "맵 버전 정보 없음")
	}

	// data_schema version 데이터 수정.
	versionMap := map[string]int32{}
	if dataSchemaServer.Version != "" {
		err = json.Unmarshal([]byte(dataSchemaServer.Version), &versionMap)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	versionMap[mapVersionDataTable.DataTable.Name] = mapVersionDataTable.MapVersion.Version

	version, err := json.Marshal(versionMap)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	// data_schema version 정보 갱신.
	updateDataSchema := database.DataSchema{Version: string(version)}
	affectedRows, err := database.Database().DataConn.ID(dataSchemaServer.DataSchema.Id).Cols("version").Update(&updateDataSchema)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	if affectedRows > 0 {
		// DB static_data 에 적용.
		var staticDataConn *xorm.Engine
		if conn, ok := database.Database().StaticDataConn[dataSchemaServer.Env]; ok {
			staticDataConn = conn
		} else {
			return h.errRes(c, fiber.StatusInternalServerError, fmt.Sprintf("static_data %s 연결 정보 없음", dataSchemaServer.Env))
		}

		// 테이블이 없으면 생성.
		tableName := "map_abyss"
		createTableQuery := "CREATE TABLE IF NOT EXISTS `" + tableName + "` (\n" +
			"`id` int(10) unsigned NOT NULL AUTO_INCREMENT,\n" +
			"`name` varchar(255) NOT NULL,\n" +
			"`data` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`data`)),\n" +
			"`created_at` timestamp NOT NULL DEFAULT current_timestamp(),\n" +
			"PRIMARY KEY (`id`) USING BTREE,\n" +
			"UNIQUE KEY `uniq_name` (`name`) USING BTREE\n" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;"
		_, err = staticDataConn.Exec(createTableQuery)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, fmt.Sprintf("static_data 테이블 생성 실패, err: %s", err.Error()))
		}

		// map data 추가.
		insertQuery := fmt.Sprintf("INSERT INTO `%s` (`name`, `data`) VALUES  ('%s', '%s') ON DUPLICATE KEY UPDATE `data` = '%s';", tableName, mapVersionDataTable.DataTable.Name, mapVersionDataTable.MapVersion.Data, mapVersionDataTable.MapVersion.Data)
		_, err = staticDataConn.Exec(insertQuery)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, fmt.Sprintf("맵 데이터 추가 실패, err: %s", err.Error()))
		}
	}

	return h.okRes(c, fiber.Map{
		"result": "ok",
	})
}

func (h *handler) UnapplyMapVersion(c *fiber.Ctx) error {
	var req unapplyDataVersionReq
	if err := c.BodyParser(&req); err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("요청 파라미터 오류: %s", err.Error()))
	}

	dataSchemaServer := database.DataSchemaServer{}
	has, err := database.Database().DataConn.
		Join("LEFT", "server", "server.id = data_schema.server_id").
		Where("data_schema.type = ?", enum.DataTableTypeMap).
		Where("data_schema.server_id = ?", req.ServerId).
		Get(&dataSchemaServer)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if !has {
		return h.errRes(c, fiber.StatusNotFound, "맵 스키마 정보 없음")
	}

	var versionMap map[string]int32
	err = json.Unmarshal([]byte(dataSchemaServer.Version), &versionMap)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	delete(versionMap, req.TableName)

	version, err := json.Marshal(versionMap)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	// data_schema version 정보 갱신.
	updateDataSchema := database.DataSchema{Version: string(version)}
	affectedRows, err := database.Database().DataConn.ID(dataSchemaServer.DataSchema.Id).Cols("version").Update(&updateDataSchema)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	if affectedRows > 0 {
		// DB static_data 에서 미적용 데이터 삭제.
		var staticDataConn *xorm.Engine
		if conn, ok := database.Database().StaticDataConn[dataSchemaServer.Env]; ok {
			staticDataConn = conn
		} else {
			return h.errRes(c, fiber.StatusInternalServerError, fmt.Sprintf("static_data %s 연결 정보 없음", dataSchemaServer.Env))
		}
		tableName := "map_abyss"
		deleteQuery := fmt.Sprintf("DELETE FROM `%s` WHERE `name` = '%s';", tableName, req.TableName)
		_, err = staticDataConn.Exec(deleteQuery)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, fmt.Sprintf("맵 데이터 삭제 실패, err: %s", err.Error()))
		}
	}

	return h.okRes(c, fiber.Map{
		"result": "ok",
	})
}
