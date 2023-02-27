package handler

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/slack-go/slack"
	"github.com/xuri/excelize/v2"
	"github.com/yeom-c/data-api/app"
	"github.com/yeom-c/data-api/auth_token"
	"github.com/yeom-c/data-api/database"
	"github.com/yeom-c/data-api/enum"
	"github.com/yeom-c/data-api/middleware"
	"github.com/yeom-c/data-api/util"
)

func (h *handler) DataTableList(c *fiber.Ctx) error {
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
		Asc("name")
	dbSession := database.Database().DataConn.
		Asc("name")

	// filters
	util.SetFilter(req.Filter, countDbSession)
	util.SetFilter(req.Filter, dbSession)

	total, err := countDbSession.Count(database.DataTable{})
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	dataList := []database.DataTable{}
	if req.Page > 0 {
		dbSession = dbSession.Limit(int(req.Limit), int((req.Page-1)*req.Limit))
	}
	err = dbSession.Find(&dataList)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	// data_table_uploader
	dataTableIdList := []int32{}
	for _, dataTable := range dataList {
		dataTableIdList = append(dataTableIdList, dataTable.Id)
	}

	dataTableUploaderUserList := []database.DataTableUploaderUser{}
	err = database.Database().DataConn.
		Join("LEFT", "user", "user.id = data_table_uploader.user_id").
		In("data_table_uploader.data_table_id", dataTableIdList).
		Where("data_table_uploader.user_id = ?", userId).
		Find(&dataTableUploaderUserList)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	dataTableUploaderMap := map[int32][]simpleUserRes{}
	for _, dataTableUploaderUser := range dataTableUploaderUserList {
		dataTableUploaderMap[dataTableUploaderUser.DataTableId] = append(dataTableUploaderMap[dataTableUploaderUser.DataTableId], simpleUserRes{
			Id:         dataTableUploaderUser.User.Id,
			EmployeeId: dataTableUploaderUser.User.EmployeeId.Int32,
			Email:      dataTableUploaderUser.User.Email,
			Name:       dataTableUploaderUser.User.Name,
			Position:   dataTableUploaderUser.User.Position,
			Color:      dataTableUploaderUser.User.Color,
		})
	}

	dataTableWithUploaderList := []dataTableWithUploader{}
	for _, dataTable := range dataList {
		uploaderList := []simpleUserRes{}
		if len(dataTableUploaderMap[dataTable.Id]) > 0 {
			uploaderList = append(uploaderList, dataTableUploaderMap[dataTable.Id][0])
		}
		dataTableWithUploaderList = append(dataTableWithUploaderList, dataTableWithUploader{
			DataTable:    dataTable,
			UploaderList: uploaderList,
		})
	}

	return h.okRes(c, dataTableListRes{
		DataTableList: dataTableWithUploaderList,
		Total:         total,
	})
}

func (h *handler) DataTable(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, "요청 데이터 오류")
	}

	dataTable := database.DataTable{
		Id: int32(id),
	}
	has, err := database.Database().DataConn.Get(&dataTable)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if !has {
		return h.errRes(c, fiber.StatusNotFound, "데이터 테이블 정보 없음")
	}

	return h.okRes(c, fiber.Map{
		"data_table": dataTable,
	})
}

func (h *handler) StoreDataTable(c *fiber.Ctx) error {
	payload := c.UserContext().Value(middleware.AuthorizationPayloadKey).(*auth_token.Payload)
	userId := int32(payload.Id)
	if userId == 0 {
		return h.errRes(c, fiber.StatusBadRequest, "유저 정보 없음")
	}

	var req storeDataTableReq
	err := c.BodyParser(&req)
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, "요청 데이터 오류")
	}
	uploadSheetList := strings.Split(req.Sheet, ",")

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
	excelUploader := util.Uploader{
		File:     buffer,
		FileSize: file.Size,
		FileType: file.Header.Get("Content-Type"),
		FileName: file.Filename,
		S3Config: util.S3Config{
			AccessKeyId:     app.Config().AwsUserFrontendAccessKeyId,
			SecretAccessKey: app.Config().AwsUserFrontendSecretAccessKey,
			Region:          app.Config().AwsS3UploadRegion,
			Bucket:          app.Config().AwsS3UploadBucket,
			UploadPath:      fmt.Sprintf("%s/%s_%d/%s", app.Config().AwsS3UploadRootPath, payload.Name, payload.Id, "data"),
			Url:             app.Config().AwsS3UploadUrl,
		},
	}
	err = excelUploader.ToS3(false, true)
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, fmt.Sprintf("파일 저장 실패: %s", err.Error()))
	}

	// 엑셀 파일 user upload_ref 기록.
	if excelUploader.UploadedTableId > 0 {
		uploadRef := []database.UploadRef{
			{
				UploadId: excelUploader.UploadedTableId,
				RefTable: "user",
				RefId:    userId,
			},
		}
		_, err = database.Database().DataConn.Insert(&uploadRef)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	// 마지막 버전 조회. 없으면 생성.
	uploadTableList := []string{}
	tableNameSheetNameMap := map[string]string{}
	for _, sheet := range uploadSheetList {
		sheetName := strings.TrimSpace(sheet)
		whiteSpaceRegex := regexp.MustCompile(`\s`)
		tableName := strings.ToLower(whiteSpaceRegex.ReplaceAllString(sheetName, "_"))
		uploadTableList = append(uploadTableList, tableName)
		tableNameSheetNameMap[tableName] = sheet
	}

	uploadDataTableList := []database.DataTable{}
	err = database.Database().DataConn.Where("type = ?", enum.DataTableTypeData).In("name", uploadTableList).Find(&uploadDataTableList)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	existDataTableMap := map[string]database.DataTable{}
	for _, dataTable := range uploadDataTableList {
		existDataTableMap[dataTable.Name] = dataTable
	}

	insertDataTableList := []database.DataTable{}
	for _, tableName := range uploadTableList {
		if _, exist := existDataTableMap[tableName]; !exist {
			insertDataTable := database.DataTable{
				Type:      int32(enum.DataTableTypeData),
				Name:      tableName,
				SheetName: tableNameSheetNameMap[tableName],
			}
			insertDataTableList = append(insertDataTableList, insertDataTable)
		}
	}

	if len(insertDataTableList) > 0 {
		_, err = database.Database().DataConn.Insert(&insertDataTableList)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}

		uploadDataTableList = []database.DataTable{}
		err = database.Database().DataConn.Where("type = ?", enum.DataTableTypeData).In("name", uploadTableList).Find(&uploadDataTableList)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	// data_version table 기록.
	uploadRef := []database.UploadRef{}
	dataTableIdDataVersionId := map[int32]int32{}
	for _, dataTable := range uploadDataTableList {
		dataVersion := database.DataVersion{
			Version:     dataTable.LatestVersion + 1,
			Status:      int32(enum.DataVersionStatusProcessing),
			DataTableId: dataTable.Id,
			UserId:      userId,
		}
		_, err = database.Database().DataConn.Insert(&dataVersion)
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}
		dataTableIdDataVersionId[dataTable.Id] = dataVersion.Id
		uploadRef = append(uploadRef, database.UploadRef{
			UploadId: excelUploader.UploadedTableId,
			RefTable: "data_version",
			RefId:    dataVersion.Id,
		})
	}

	// 엑셀 파일 data_version upload_ref 기록.
	_, err = database.Database().DataConn.Insert(&uploadRef)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	// 엑셀 변환 준비.
	excelFile, err := file.Open()
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, "파일 읽기 실패")
	}
	defer excelFile.Close()
	excel, err := excelize.OpenReader(excelFile)
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, "파일 읽기 실패")
	}
	defer excel.Close()

	dataGenerator := util.DataGenerator{
		Excel: excel,
	}
	for _, dataTable := range uploadDataTableList {
		dataGenerator.SetSheet(dataTable.SheetName, 2, 2, dataTable.Id, dataTableIdDataVersionId[dataTable.Id], dataTable.LatestVersion+1)
	}
	// 세팅된 sheet 읽어서 Schemas, DataMap 생성.
	err = dataGenerator.ReadSheets()
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, err.Error())
	}

	// 변환.
	sheetUploadUrl := map[string]string{}
	sheets := dataGenerator.GetSheets()
	err = dataGenerator.GenJson()
	if err == nil {
		// json s3 upload.
		uploadRef = []database.UploadRef{}
		for i := range sheets {
			sheet := sheets[i]
			if len(sheet.Errors) > 0 {
				continue
			}
			uploadConfig := util.S3Config{
				AccessKeyId:     app.Config().AwsUserFrontendAccessKeyId,
				SecretAccessKey: app.Config().AwsUserFrontendSecretAccessKey,
				Region:          app.Config().AwsS3CdnRegion,
				Bucket:          app.Config().AwsS3CdnBucket,
				UploadPath:      fmt.Sprintf("data/%s/%d", sheet.TableName, sheet.Version),
				Url:             app.Config().AwsS3CdnUrl,
			}
			uploader := util.Uploader{
				File:     sheet.GenJson,
				FileSize: int64(len(sheet.GenJson)),
				FileType: "json",
				FileName: "data.json",
				S3Config: uploadConfig,
			}
			err = uploader.ToS3(true, true)
			if err != nil {
				sheet.Errors = append(sheet.Errors, fmt.Sprintf("sheet: %s, version: %v, err: data.json S3 업로드 실패 %s", sheet.Name, sheet.Version, err.Error()))
			}
			uploadRef = append(uploadRef, database.UploadRef{
				UploadId: uploader.UploadedTableId,
				RefTable: "data_version",
				RefId:    sheet.DataVersionId,
			})

			sheetUploadUrl[sheet.Name] = uploader.Url
		}

		database.Database().DataConn.Insert(&uploadRef)
	}

	err = dataGenerator.GenDb()

	// 결과 업데이트.
	dataTableUploaders := []database.DataTableUploader{}
	for _, sheet := range sheets {
		// data_version status 상태 업데이트.
		dataVersion := database.DataVersion{
			Id:     sheet.DataVersionId,
			Status: int32(enum.DataVersionStatusComplete),
		}
		if len(sheet.Errors) > 0 {
			errJson, _ := json.Marshal(sheet.Errors)
			dataVersion.Error = string(errJson)
			dataVersion.Status = int32(enum.DataVersionStatusError)
		}
		database.Database().DataConn.Cols("error", "status").ID(dataVersion.Id).Update(&dataVersion)

		// data_table latest 버전 업데이트.
		if dataVersion.Status == int32(enum.DataVersionStatusComplete) {
			dataTable := database.DataTable{
				Id:            sheet.DataTableId,
				LatestVersion: sheet.Version,
			}
			database.Database().DataConn.Cols("latest_version").ID(dataTable.Id).Update(&dataTable)
		}

		// upload user 기록.
		dataTableUploaders = append(dataTableUploaders, database.DataTableUploader{
			DataTableId: sheet.DataTableId,
			UserId:      userId,
		})

		// slack 알림.
		msgTitle := "[데이터 업로드]"
		msgColor := "#00C851"
		if len(sheet.Errors) > 0 {
			msgTitle = "[데이터 업로드 실패]"
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
						Title: sheet.Name,
						Value: fmt.Sprintf("버전: %d", sheet.Version),
					},
					{
						Value: "excel 파일: " + excelUploader.Url,
					},
					{
						Value: "json 파일: " + sheetUploadUrl[sheet.Name],
					},
				},
				Color: msgColor,
			}),
			slack.MsgOptionAsUser(true),
		)
	}
	if len(dataTableUploaders) > 0 {
		database.Database().DataConn.Insert(&dataTableUploaders)
	}

	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, err.Error())
	}
	return h.okRes(c, fiber.Map{"result": "ok"})
}

func (h *handler) DeleteDataTable(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.errRes(c, fiber.StatusBadRequest, "요청 데이터 오류")
	}

	// data_table 조회.
	dataTable := database.DataTable{
		Id: int32(id),
	}
	has, err := database.Database().DataConn.Get(&dataTable)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if !has {
		return h.errRes(c, fiber.StatusNotFound, "데이터 테이블 정보 없음")
	}

	// 적용된 스키마 존재하면 해제 필요.
	has, err = database.Database().DataConn.Where("type = ?", dataTable.Type).Where(fmt.Sprintf("JSON_EXISTS(version, '$.\"%s\"')", dataTable.Name)).Exist(&database.DataSchema{})
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	if has {
		return h.errRes(c, fiber.StatusInternalServerError, "적용서버 해제 필요")
	}

	// data_table_uploader 에서 삭제.
	_, err = database.Database().DataConn.Table("data_table_uploader").Where("data_table_id = ?", dataTable.Id).Delete()
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	versionIdList := []int32{}
	versionTableName := ""
	if dataTable.Type == int32(enum.DataTableTypeData) {
		versionTableName = "data_version"
	} else if dataTable.Type == int32(enum.DataTableTypeMap) {
		versionTableName = "map_version"
	}
	// data_version 관련 데이터 삭제.
	// upload_ref 관련 데이터 삭제.
	err = database.Database().DataConn.Table(versionTableName).Cols("id").Where("data_table_id = ?", dataTable.Id).Find(&versionIdList)
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	_, err = database.Database().DataConn.Table("upload_ref").Where(fmt.Sprintf("ref_table = '%s'", versionTableName)).In("ref_id", versionIdList).Delete()
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}
	_, err = database.Database().DataConn.Table(versionTableName).Where("data_table_id = ?", dataTable.Id).Delete()
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	if dataTable.Type == int32(enum.DataTableTypeData) {
		// static_data_gen db 에서 테이블 삭제.
		results, err := database.Database().StaticDataGenConn.Query(fmt.Sprintf("SHOW TABLES WHERE Tables_in_static_data_gen LIKE '%s_v%%';", dataTable.Name))
		if err != nil {
			return h.errRes(c, fiber.StatusInternalServerError, err.Error())
		}

		dropTables := []string{}
		for _, row := range results {
			for _, data := range row {
				dropTables = append(dropTables, string(data))
			}
		}
		if len(dropTables) > 0 {
			_, err = database.Database().StaticDataGenConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", strings.Join(dropTables, ",")))
			if err != nil {
				return h.errRes(c, fiber.StatusInternalServerError, err.Error())
			}
		}
	}

	// data_table 에서 삭제.
	_, err = database.Database().DataConn.Table("data_table").Where("id = ?", dataTable.Id).Delete()
	if err != nil {
		return h.errRes(c, fiber.StatusInternalServerError, err.Error())
	}

	return h.okRes(c, fiber.Map{"result": "ok"})
}
