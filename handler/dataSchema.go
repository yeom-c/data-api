package handler

import (
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/slack-go/slack"
	"github.com/yeom-c/data-api/app"
	"github.com/yeom-c/data-api/auth_token"
	"github.com/yeom-c/data-api/database"
	"github.com/yeom-c/data-api/enum"
	"github.com/yeom-c/data-api/middleware"
	"github.com/yeom-c/data-api/util"
	"xorm.io/xorm"
)

func (h *handler) DataSchemaList(c *fiber.Ctx) error {
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

	total, err := countDbSession.Count(database.DataSchema{})
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	dataList := []database.DataSchema{}
	if req.Page > 0 {
		dbSession = dbSession.Limit(int(req.Limit), int((req.Page-1)*req.Limit))
	}
	err = dbSession.Find(&dataList)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	return h.okRes(c, dataSchemaListRes{
		DataSchemaList: dataList,
		Total:          total,
	})
}

func (h *handler) DataSchema(c *fiber.Ctx) error {
	var req dataSchemaReq
	err := c.QueryParser(&req)
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, "요청 데이터 오류")
	}

	dataSchema := database.DataSchema{}
	has, err := database.Database().DataConn.
		Where("type = ?", req.Type).
		Where("server_id = ?", req.ServerId).
		Get(&dataSchema)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	res := dataSchemaRes{}
	if has {
		res.DataSchema = &dataSchema
	}

	return h.okRes(c, res)
}

func (h *handler) StoreDataSchema(c *fiber.Ctx) error {
	var req storeDataSchemaReq
	if err := c.BodyParser(&req); err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("요청 파라미터 오류: %s", err.Error()))
	}

	server := database.Server{
		Env:         req.Env,
		Name:        req.Name,
		Description: req.Description,
	}
	_, err := database.Database().DataConn.Insert(&server)
	if err != nil {
		if err.(*mysql.MySQLError).Number == 1062 {
			server = database.Server{}
			_, err = database.Database().DataConn.Where("env = ?", req.Env).Get(&server)
			if err != nil {
				return h.errRes(c, fiber.StatusInternalServerError, err.Error())
			}
		} else {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	dataSchema := []database.DataSchema{
		{
			ServerId: server.Id,
			Type:     int32(enum.DataTableTypeData),
		},
		{
			ServerId: server.Id,
			Type:     int32(enum.DataTableTypeMap),
		},
	}
	_, err = database.Database().DataConn.Insert(&dataSchema)
	if err != nil {
		if err.(*mysql.MySQLError).Number != 1062 {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	return h.okRes(c, fiber.Map{
		"result": "ok",
	})
}

func (h *handler) UpdateDataSchema(c *fiber.Ctx) error {
	payload := c.UserContext().Value(middleware.AuthorizationPayloadKey).(*auth_token.Payload)
	var req updateDataSchemaReq
	if err := c.BodyParser(&req); err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("요청 파라미터 오류: %s", err.Error()))
	}

	dataSchema := database.DataSchema{
		UpdateLock: req.UpdateLock,
	}
	_, err := database.Database().DataConn.Cols("update_lock").Where("server_id = ?", req.ServerId).Update(&dataSchema)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	server := database.Server{}
	_, err = database.Database().DataConn.
		ID(req.ServerId).
		Get(&server)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	// slack 알림.
	msgTitle := "[데이터 버전 변경 가능]"
	msgColor := "#00C851"
	if req.UpdateLock == int32(enum.TrueFalseTrue) {
		msgTitle = "[데이터 버전 변경 불가능]"
		msgColor = "#ff4444"
	}
	util.Slack().Client.PostMessage(
		app.Config().SlackDataChannelId,
		slack.MsgOptionAttachments(slack.Attachment{
			Fields: []slack.AttachmentField{
				{
					Title: msgTitle,
					Value: payload.Name,
				},
				{
					Title: "환경",
					Value: server.Name,
				},
			},
			Color: msgColor,
		}),
		slack.MsgOptionAsUser(true),
	)

	return h.okRes(c, fiber.Map{
		"result": "ok",
	})
}

func (h *handler) DeleteDataSchema(c *fiber.Ctx) error {
	var req deleteDataSchemaReq
	if err := c.BodyParser(&req); err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("요청 파라미터 오류: %s", err.Error()))
	}

	_, err := database.Database().DataConn.Table("data_schema").Where("server_id = ?", req.ServerId).Delete()
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	_, err = database.Database().DataConn.Table("server").Where("id = ?", req.ServerId).Delete()
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	return h.okRes(c, fiber.Map{
		"result": "ok",
	})
}

func (h *handler) ApplyDataVersion(c *fiber.Ctx) error {
	var req applyDataVersionReq
	if err := c.BodyParser(&req); err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("요청 파라메터 오류: %s", err.Error()))
	}

	// 데이터 스키마 체크.
	dataSchemaServer := database.DataSchemaServer{}
	has, err := database.Database().DataConn.
		Join("LEFT", "server", "server.id = data_schema.server_id").
		Where("data_schema.type = ?", enum.DataTableTypeData).
		Where("data_schema.server_id = ?", req.ServerId).
		Get(&dataSchemaServer)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if !has {
		return h.errRes(c, fiber.StatusNotFound, "데이터 스키마 정보 없음")
	}

	if dataSchemaServer.UpdateLock == int32(enum.TrueFalseTrue) {
		return h.errRes(c, fiber.StatusNotFound, "데이터 스키마 변경 불가")
	}

	dataVersionDataTable := database.DataVersionDataTable{}
	has, err = database.Database().DataConn.
		Join("LEFT", "data_table", "data_table.id = data_version.data_table_id").
		Where("data_version.id = ?", req.VersionId).
		Get(&dataVersionDataTable)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if !has {
		return h.errRes(c, fiber.StatusNotFound, "데이터 버전 정보 없음")
	}

	// data_schema version 데이터 수정.
	versionMap := map[string]int32{}
	if dataSchemaServer.Version != "" {
		err = json.Unmarshal([]byte(dataSchemaServer.Version), &versionMap)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	versionMap[dataVersionDataTable.DataTable.Name] = dataVersionDataTable.DataVersion.Version

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
		// DB static_data 에 적용 데이터 테이블 덮어쓰기.
		var staticDataConn *xorm.Engine
		if conn, ok := database.Database().StaticDataConn[dataSchemaServer.Env]; ok {
			staticDataConn = conn
		} else {
			return h.errRes(c, fiber.StatusInternalServerError, fmt.Sprintf("static_data %s 연결 정보 없음", dataSchemaServer.Env))
		}

		genTableName := fmt.Sprintf("%s_v%d", dataVersionDataTable.DataTable.Name, dataVersionDataTable.Version)
		err = util.CopyTable(database.Database().StaticDataGenConn, staticDataConn, genTableName, dataVersionDataTable.DataTable.Name)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}

		// 스키마 json s3 업로드.
		schemaMap := map[string]interface{}{
			"id":      dataSchemaServer.DataSchema.Id,
			"version": versionMap,
			"server": map[string]interface{}{
				"id":   dataSchemaServer.Server.Id,
				"env":  dataSchemaServer.Server.Env,
				"name": dataSchemaServer.Server.Name,
			},
		}
		schemaJson, err := json.Marshal(schemaMap)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}

		uploader := util.Uploader{
			File:     schemaJson,
			FileSize: int64(len(schemaJson)),
			FileType: "json",
			FileName: fmt.Sprintf("%s.json", dataSchemaServer.Server.Env),
			S3Config: util.S3Config{
				AccessKeyId:     app.Config().AwsUserFrontendAccessKeyId,
				SecretAccessKey: app.Config().AwsUserFrontendSecretAccessKey,
				Region:          app.Config().AwsS3CdnRegion,
				Bucket:          app.Config().AwsS3CdnBucket,
				UploadPath:      "schema",
				Url:             app.Config().AwsS3CdnUrl,
			},
		}
		err = uploader.ToS3(true, false)
		if err != nil {
			return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("파일 저장 실패: %s", err.Error()))
		}
	}

	return h.okRes(c, fiber.Map{
		"result": "ok",
	})
}

func (h *handler) UnapplyDataVersion(c *fiber.Ctx) error {
	var req unapplyDataVersionReq
	if err := c.BodyParser(&req); err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("요청 파라미터 오류: %s", err.Error()))
	}

	dataSchemaServer := database.DataSchemaServer{}
	has, err := database.Database().DataConn.
		Join("LEFT", "server", "server.id = data_schema.server_id").
		Where("data_schema.type = ?", enum.DataTableTypeData).
		Where("data_schema.server_id = ?", req.ServerId).
		Get(&dataSchemaServer)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if !has {
		return h.errRes(c, fiber.StatusNotFound, "데이터 스키마 정보 없음")
	}

	if dataSchemaServer.UpdateLock == int32(enum.TrueFalseTrue) {
		return h.errRes(c, fiber.StatusNotFound, "데이터 스키마 변경 불가")
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
		// DB static_data 에서 미적용 데이터 테이블 삭제.
		var staticDataConn *xorm.Engine
		if conn, ok := database.Database().StaticDataConn[dataSchemaServer.Env]; ok {
			staticDataConn = conn
		} else {
			return h.errRes(c, fiber.StatusInternalServerError, fmt.Sprintf("static_data %s 연결 정보 없음", dataSchemaServer.Env))
		}

		err = staticDataConn.NewSession().DropTable(req.TableName)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}

		// 스키마 json s3 업로드.
		schemaMap := map[string]interface{}{
			"id":      dataSchemaServer.DataSchema.Id,
			"version": versionMap,
			"server": map[string]interface{}{
				"id":   dataSchemaServer.Server.Id,
				"env":  dataSchemaServer.Server.Env,
				"name": dataSchemaServer.Server.Name,
			},
		}
		schemaJson, err := json.Marshal(schemaMap)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}

		uploader := util.Uploader{
			File:     schemaJson,
			FileSize: int64(len(schemaJson)),
			FileType: "json",
			FileName: fmt.Sprintf("%s.json", dataSchemaServer.Server.Env),
			S3Config: util.S3Config{
				AccessKeyId:     app.Config().AwsUserFrontendAccessKeyId,
				SecretAccessKey: app.Config().AwsUserFrontendSecretAccessKey,
				Region:          app.Config().AwsS3CdnRegion,
				Bucket:          app.Config().AwsS3CdnBucket,
				UploadPath:      "schema",
				Url:             app.Config().AwsS3CdnUrl,
			},
		}
		err = uploader.ToS3(true, false)
		if err != nil {
			return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("파일 저장 실패: %s", err.Error()))
		}
	}

	return h.okRes(c, fiber.Map{
		"result": "ok",
	})
}

func (h *handler) RefreshDataSchema(c *fiber.Ctx) error {
	payload := c.UserContext().Value(middleware.AuthorizationPayloadKey).(*auth_token.Payload)

	// cloudfront refresh cache
	cloudfront := util.CloudFront{
		AccessKeyId:     app.Config().AwsUserFrontendAccessKeyId,
		SecretAccessKey: app.Config().AwsUserFrontendSecretAccessKey,
		Region:          app.Config().AwsS3CdnRegion,
		DistributionId:  app.Config().AwsCloudfrontCdnDistributionId,
	}
	_, err := cloudfront.CreateInvalidation("/schema/*")
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("스키마 캐시 갱신 실패: %s", err.Error()))
	}

	// slack 알림.
	msgTitle := "[데이터 캐시]"
	msgColor := "#00C851"
	util.Slack().Client.PostMessage(
		app.Config().SlackDataChannelId,
		slack.MsgOptionAttachments(slack.Attachment{
			Fields: []slack.AttachmentField{
				{
					Title: msgTitle,
					Value: payload.Name,
				},
				{
					Title: "스키마 파일 캐시 새로고침",
				},
			},
			Color: msgColor,
		}),
		slack.MsgOptionAsUser(true),
	)

	return h.okRes(c, fiber.Map{
		"result": "ok",
	})
}
