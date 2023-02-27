package handler

import (
	"time"

	"github.com/yeom-c/data-api/database"
)

type signInReq struct {
	AuthProvider string `json:"authProvider"`
	AuthCode     string `json:"authCode"`
	Email        string `json:"email"`
	Password     string `json:"password"`
}

type signInRes struct {
	AccessToken string  `json:"access_token"`
	Profile     userRes `json:"profile"`
}

type dashboardRes struct {
}

type simpleUserRes struct {
	Id         int32  `json:"id"`
	EmployeeId int32  `json:"employee_id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Position   string `json:"position"`
	Color      string `json:"color"`
}

type userRes struct {
	Id                int32     `json:"id"`
	EmployeeId        int32     `json:"employee_id"`
	Email             string    `json:"email"`
	Name              string    `json:"name"`
	Position          string    `json:"position"`
	Color             string    `json:"color"`
	PasswordChangedAt string    `json:"password_changed_at"`
	JoinedAt          string    `json:"joined_at"`
	RetiredAt         string    `json:"retired_at"`
	CreatedAt         time.Time `json:"created_at"`
}

type profileRes struct {
	User userRes `json:"user"`
}

type storeProfileReq struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Position string `json:"position"`
	Color    string `json:"color"`
}

type listReq struct {
	Filter map[string]map[string]interface{} `json:"filter"`
	Page   int32                             `json:"page"`
	Limit  int32                             `json:"limit"`
}

type serverListRes struct {
	ServerList []database.Server `json:"server_list"`
	Total      int64             `json:"total"`
}

type dataSchemaListRes struct {
	DataSchemaList []database.DataSchema `json:"data_schema_list"`
	Total          int64                 `json:"total"`
}

type dataSchemaReq struct {
	Type     int32 `query:"type"`
	ServerId int32 `query:"server_id"`
}

type dataSchemaRes struct {
	DataSchema *database.DataSchema `json:"data_schema"`
}

type storeDataSchemaReq struct {
	Env         string `json:"env"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateDataSchemaReq struct {
	Id         int32 `json:"id"`
	ServerId   int32 `json:"server_id"`
	UpdateLock int32 `json:"update_lock"`
}

type deleteDataSchemaReq struct {
	ServerId int32 `json:"server_id"`
}

type unapplyDataVersionReq struct {
	ServerId  int32  `json:"server_id"`
	TableName string `json:"table_name"`
}

type applyDataVersionReq struct {
	ServerId  int32 `json:"server_id"`
	VersionId int32 `json:"version_id"`
}

type dataTableWithUploader struct {
	DataTable    database.DataTable `json:"data_table"`
	UploaderList []simpleUserRes    `json:"uploader_list"`
}

type dataTableListRes struct {
	DataTableList []dataTableWithUploader `json:"data_table_list"`
	Total         int64                   `json:"total"`
}

type storeDataTableReq struct {
	Sheet string `form:"sheet"`
}

type dataVersionWithUpload struct {
	DataVersion database.DataVersion `json:"data_version"`
	User        simpleUserRes        `json:"user"`
	UploadList  []database.Upload    `json:"upload_list"`
}

type dataVersionListRes struct {
	DataVersionList []dataVersionWithUpload `json:"data_version_list"`
	Total           int64                   `json:"total"`
}

type storeDataVersionReq struct {
	Id        int32  `json:"id"`
	MemoTitle string `json:"memo_title"`
	Memo      string `json:"memo"`
}

type mapVersionWithUpload struct {
	MapVersion database.MapVersion `json:"map_version"`
	User       simpleUserRes       `json:"user"`
	UploadList []database.Upload   `json:"upload_list"`
}

type mapVersionListRes struct {
	MapVersionList []mapVersionWithUpload `json:"map_version_list"`
	Total          int64                  `json:"total"`
}

type storeMapVersionReq struct {
	Id        int32  `json:"id"`
	MemoTitle string `json:"memo_title"`
	Memo      string `json:"memo"`
}
