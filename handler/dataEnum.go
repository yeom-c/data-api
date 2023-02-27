package handler

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/slack-go/slack"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/yeom-c/data-api/app"
	"github.com/yeom-c/data-api/auth_token"
	"github.com/yeom-c/data-api/database"
	"github.com/yeom-c/data-api/enum"
	"github.com/yeom-c/data-api/middleware"
	"github.com/yeom-c/data-api/util"
)

func (h *handler) StoreDataEnum(c *fiber.Ctx) error {
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

	// 엑셀 파일 S3 upload.
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
			UploadPath:      fmt.Sprintf("%s/%s_%d/%s", app.Config().AwsS3UploadRootPath, payload.Name, payload.Id, "enum"),
			Url:             app.Config().AwsS3UploadUrl,
		},
	}
	err = uploader.ToS3(false, true)
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("파일 저장 실패: %s", err.Error()))
	}

	// 엑셀 파일 user upload_ref 기록.
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
	sheetName := "Common_Enum"
	sheetName = strings.TrimSpace(sheetName)
	whiteSpaceRegex := regexp.MustCompile(`\s`)
	tableName := strings.ToLower(whiteSpaceRegex.ReplaceAllString(sheetName, "_"))

	dataTable := database.DataTable{}
	has, err := database.Database().DataConn.Where("type = ?", enum.DataTableTypeEnum).Where("name = ?", tableName).Get(&dataTable)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if !has {
		dataTable.Type = int32(enum.DataTableTypeEnum)
		dataTable.Name = tableName
		dataTable.SheetName = sheetName
		_, err := database.Database().DataConn.Insert(&dataTable)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	// 버전 테이블 기록.
	updateVersion := dataTable.LatestVersion + 1
	dataVersion := database.DataVersion{
		Version:     updateVersion,
		Status:      int32(enum.DataVersionStatusProcessing),
		DataTableId: dataTable.Id,
		UserId:      userId,
	}
	_, err = database.Database().DataConn.Insert(&dataVersion)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	// 엑셀 파일 data_version upload_ref 기록.
	uploadRef = []database.UploadRef{
		{
			UploadId: uploader.UploadedTableId,
			RefTable: "data_version",
			RefId:    dataVersion.Id,
		},
	}
	_, err = database.Database().DataConn.Insert(&uploadRef)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	// 변환을 위한 엑셀 파일 시트 읽기.
	excelFile, err := file.Open()
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, "파일 읽기 실패")
	}
	defer excelFile.Close()
	eg := util.EnumGenerator{
		File: &excelFile,
	}
	eg.ReadSheet(sheetName, 2, 2)

	genGoUrl := ""
	genCsUrl := ""
	errors := eg.Errors
	if len(errors) == 0 {
		// enum csharp string 변환.
		genCsharp := eg.GenCSharp()
		genCSharpBytes := []byte(genCsharp)

		// enum golang string 변환.
		genGolang := eg.GenGolang()
		genGolangBytes := []byte(genGolang)

		// 변환 파일 S3 업로드.
		uploadRef := []database.UploadRef{}
		uploadConfig := util.S3Config{
			AccessKeyId:     app.Config().AwsUserFrontendAccessKeyId,
			SecretAccessKey: app.Config().AwsUserFrontendSecretAccessKey,
			Region:          app.Config().AwsS3UploadRegion,
			Bucket:          app.Config().AwsS3UploadBucket,
			UploadPath:      fmt.Sprintf("%s/gen/enum/%s/%d", app.Config().AwsS3UploadRootPath, tableName, updateVersion),
			Url:             app.Config().AwsS3UploadUrl,
		}
		genCsUploader := util.Uploader{
			File:     genCSharpBytes,
			FileSize: int64(len(genCSharpBytes)),
			FileType: "cs",
			FileName: "Data_Enum.cs",
			S3Config: uploadConfig,
		}
		err = genCsUploader.ToS3(true, true)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Data_Enum.cs S3 업로드 실패: %s", err.Error()))
		}
		uploadRef = append(uploadRef, database.UploadRef{
			UploadId: genCsUploader.UploadedTableId,
			RefTable: "data_version",
			RefId:    dataVersion.Id,
		})
		genCsUrl = genCsUploader.Url

		genGoUploader := util.Uploader{
			File:     genGolangBytes,
			FileSize: int64(len(genGolangBytes)),
			FileType: "go",
			FileName: "enum.go",
			S3Config: uploadConfig,
		}
		err = genGoUploader.ToS3(true, true)
		if err != nil {
			errors = append(errors, fmt.Sprintf("enum.go S3 업로드 실패: %s", err.Error()))
		}
		uploadRef = append(uploadRef, database.UploadRef{
			UploadId: genGoUploader.UploadedTableId,
			RefTable: "data_version",
			RefId:    dataVersion.Id,
		})
		genGoUrl = genGoUploader.Url

		// upload_ref 테이블 기록.
		_, err = database.Database().DataConn.Insert(&uploadRef)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}

		// 변환 파일 cdn 에 업로드.
		genCsUploader.S3Config.Region = app.Config().AwsS3CdnRegion
		genCsUploader.S3Config.Bucket = app.Config().AwsS3CdnBucket
		genCsUploader.S3Config.UploadPath = "enum"
		genCsUploader.S3Config.Url = app.Config().AwsS3CdnUrl
		err = genCsUploader.ToS3(true, false)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Data_Enum.cs S3 업로드 실패: %s", err.Error()))
		}

		genGoUploader.S3Config.Region = app.Config().AwsS3CdnRegion
		genGoUploader.S3Config.Bucket = app.Config().AwsS3CdnBucket
		genGoUploader.S3Config.UploadPath = "enum"
		genGoUploader.S3Config.Url = app.Config().AwsS3CdnUrl
		err = genGoUploader.ToS3(true, false)
		if err != nil {
			errors = append(errors, fmt.Sprintf("enum.go S3 업로드 실패: %s", err.Error()))
		}
	}

	// data_version status 상태 업데이트.
	dataVersion.Status = int32(enum.DataVersionStatusComplete)
	if len(errors) > 0 {
		errJson, _ := json.Marshal(errors)
		dataVersion.Error = string(errJson)
		dataVersion.Status = int32(enum.DataVersionStatusError)
	}
	_, err = database.Database().DataConn.Cols("error", "status").ID(dataVersion.Id).Update(&dataVersion)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	// data_table latest 버전 업데이트.
	if dataVersion.Status == int32(enum.DataVersionStatusComplete) {
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
		app.Config().SlackEnumChannelId,
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
				{
					Value: "cs 파일: " + genCsUrl,
				},
				{
					Value: "go 파일: " + genGoUrl,
				},
			},
			Color: msgColor,
		}),
		slack.MsgOptionAsUser(true),
	)

	return h.okRes(c, fiber.Map{"result": "ok"})
}

func (h *handler) RefreshDataEnum(c *fiber.Ctx) error {
	payload := c.UserContext().Value(middleware.AuthorizationPayloadKey).(*auth_token.Payload)

	// cloudfront refresh cache
	cloudfront := util.CloudFront{
		AccessKeyId:     app.Config().AwsUserFrontendAccessKeyId,
		SecretAccessKey: app.Config().AwsUserFrontendSecretAccessKey,
		Region:          app.Config().AwsS3CdnRegion,
		DistributionId:  app.Config().AwsCloudfrontCdnDistributionId,
	}
	_, err := cloudfront.CreateInvalidation("/enum/*")
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("이넘 캐시 갱신 실패: %s", err.Error()))
	}

	// slack 알림.
	msgTitle := "[이넘 캐시]"
	msgColor := "#00C851"
	util.Slack().Client.PostMessage(
		app.Config().SlackEnumChannelId,
		slack.MsgOptionAttachments(slack.Attachment{
			Fields: []slack.AttachmentField{
				{
					Title: msgTitle,
					Value: payload.Name,
				},
				{
					Title: "이넘 파일 캐시 새로고침",
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
