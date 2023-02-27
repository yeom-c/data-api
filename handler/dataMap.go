package handler

import (
	"bytes"
	"fmt"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/slack-go/slack"
	"github.com/yeom-c/data-api/database"
	"github.com/yeom-c/data-api/enum"
	"github.com/yeom-c/data-api/util"

	"github.com/yeom-c/data-api/auth_token"
	"github.com/yeom-c/data-api/middleware"

	"github.com/yeom-c/data-api/app"
)

func (h *handler) StoreDataMap(c *fiber.Ctx) error {
	payload := c.UserContext().Value(middleware.AuthorizationPayloadKey).(*auth_token.Payload)
	userId := int32(payload.Id)
	if userId == 0 {
		return h.errRes(c, fiber.StatusBadRequest, "유저 정보 없음")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, "요청 데이터 오류")
	}

	openFile, err := file.Open()
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, "파일 읽기 실패")
	}
	defer openFile.Close()

	// JSON 파일 S3 upload.
	buffer := make([]byte, file.Size)
	openFile.Read(buffer)
	uploader := util.Uploader{
		File:     buffer,
		FileSize: file.Size,
		FileType: file.Header.Get("Content-Type"),
		FileName: file.Filename,
		S3Config: util.S3Config{
			AccessKeyId:     app.Config().AwsUserFrontendAccessKeyId,
			SecretAccessKey: app.Config().AwsUserFrontendSecretAccessKey,
			Region:          app.Config().AwsS3UploadRegion,
			Bucket:          app.Config().AwsS3UploadBucket,
			UploadPath:      fmt.Sprintf("%s/%s_%d/%s", app.Config().AwsS3UploadRootPath, payload.Name, payload.Id, "map"),
			Url:             app.Config().AwsS3UploadUrl,
		},
	}
	err = uploader.ToS3(false, true)
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("파일 저장 실패: %s", err.Error()))
	}

	// JSON 파일 user upload_ref 기록.
	uploadRef := []database.UploadRef{
		{
			UploadId: uploader.UploadedTableId,
			RefTable: "user",
			RefId:    userId,
		},
	}
	_, err = database.Database().DataConn.Insert(&uploadRef)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	// 마지막 버전 조회. 없으면 생성.
	dataTable := database.DataTable{}
	has, err := database.Database().DataConn.Where("name = ?", file.Filename).Get(&dataTable)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if !has {
		dataTable.Type = int32(enum.DataTableTypeMap)
		dataTable.Name = file.Filename
		dataTable.SheetName = file.Filename
		_, err := database.Database().DataConn.Insert(&dataTable)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	// 버전 테이블 기록.
	updateVersion := dataTable.LatestVersion + 1
	mapVersion := database.MapVersion{
		Version:     updateVersion,
		Status:      int32(enum.DataVersionStatusProcessing),
		Data:        "\"\"",
		DataTableId: dataTable.Id,
		UserId:      userId,
	}
	_, err = database.Database().DataConn.Insert(&mapVersion)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	// JSON 파일 map_version upload_ref 기록.
	uploadRef = []database.UploadRef{
		{
			UploadId: uploader.UploadedTableId,
			RefTable: "map_version",
			RefId:    mapVersion.Id,
		},
	}
	_, err = database.Database().DataConn.Insert(&uploadRef)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	// map json minify.
	errors := []string{}
	var minify bytes.Buffer
	err = json.Compact(&minify, buffer)
	if err != nil {
		errors = append(errors, fmt.Sprintf("map json minify 실패: %s", err.Error()))
	}

	// data_version status 상태 업데이트.
	mapVersion.Status = int32(enum.DataVersionStatusComplete)
	if len(errors) > 0 {
		errJson, _ := json.Marshal(errors)
		mapVersion.Error = string(errJson)
		mapVersion.Status = int32(enum.DataVersionStatusError)
	} else {
		mapVersion.Data = minify.String()
	}
	_, err = database.Database().DataConn.Cols("data", "error", "status").ID(mapVersion.Id).Update(&mapVersion)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	// data_table latest 버전 업데이트.
	if mapVersion.Status == int32(enum.DataVersionStatusComplete) {
		dataTable.LatestVersion = updateVersion
		_, err = database.Database().DataConn.Cols("latest_version").ID(dataTable.Id).Update(&dataTable)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	// slack 알림.
	msgTitle := "[파일 업로드]"
	msgColor := "#00C851"
	if len(errors) > 0 {
		msgTitle = "[파일 업로드 실패]"
		msgColor = "#ff4444"
	}
	util.Slack().Client.PostMessage(
		app.Config().SlackMapChannelId,
		slack.MsgOptionAttachments(slack.Attachment{
			Fields: []slack.AttachmentField{
				{
					Title: msgTitle,
					Value: file.Filename,
				},
				{
					Title: payload.Name,
					Value: fmt.Sprintf("버전: %d", updateVersion),
				},
				{
					Value: "업로드 파일: " + uploader.Url,
				},
			},
			Color: msgColor,
		}),
		slack.MsgOptionAsUser(true),
	)

	return h.okRes(c, fiber.Map{"result": "ok"})
}
