package database

import (
	"database/sql"
	"time"
)

type User struct {
	Id                int32         `json:"id" xorm:"pk autoincr"`
	EmployeeId        sql.NullInt32 `json:"employee_id"`
	Email             string        `json:"email"`
	Name              string        `json:"name"`
	HashedPassword    string        `json:"hashedPassword"`
	PasswordChangedAt sql.NullTime  `json:"password_changed_at"`
	Position          string        `json:"position"`
	Color             string        `json:"color"`
	JoinedAt          sql.NullTime  `json:"joined_at"`
	RetiredAt         sql.NullTime  `json:"retired_at"`
	CreatedAt         time.Time     `json:"created_at"`
}

type Server struct {
	Id          int32        `json:"id" xorm:"pk autoincr"`
	Env         string       `json:"env"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	UpdatedAt   sql.NullTime `json:"updated_at"`
	CreatedAt   time.Time    `json:"created_at"`
}

type DataSchema struct {
	Id         int32        `json:"id" xorm:"pk autoincr"`
	Type       int32        `json:"type"`
	ServerId   int32        `json:"server_id"`
	Version    string       `json:"version"`
	UpdateLock int32        `json:"update_lock"`
	UpdatedAt  sql.NullTime `json:"updated_at"`
	CreatedAt  time.Time    `json:"created_at"`
}

type DataSchemaServer struct {
	DataSchema `xorm:"extends"`
	Server     `xorm:"extends"`
}

func (DataSchemaServer) TableName() string {
	return "data_schema"
}

type DataTable struct {
	Id            int32  `json:"id" xorm:"pk autoincr"`
	Type          int32  `json:"type"`
	Name          string `json:"name"`
	SheetName     string `json:"sheet_name"`
	LatestVersion int32  `json:"latest_version"`
}

type DataVersion struct {
	Id          int32     `json:"id" xorm:"pk autoincr"`
	Version     int32     `json:"version"`
	Status      int32     `json:"status"`
	Error       string    `json:"error"`
	MemoTitle   string    `json:"memo_title"`
	Memo        string    `json:"memo"`
	DataTableId int32     `json:"data_table_id"`
	UserId      int32     `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type DataVersionUser struct {
	DataVersion `xorm:"extends"`
	User        `xorm:"extends"`
}

func (DataVersionUser) TableName() string {
	return "data_version"
}

type DataVersionDataTable struct {
	DataVersion `xorm:"extends"`
	DataTable   `xorm:"extends"`
}

func (DataVersionDataTable) TableName() string {
	return "data_version"
}

type MapVersion struct {
	Id          int32     `json:"id" xorm:"pk autoincr"`
	Version     int32     `json:"version"`
	Status      int32     `json:"status"`
	Error       string    `json:"error"`
	Data        string    `json:"data"`
	MemoTitle   string    `json:"memo_title"`
	Memo        string    `json:"memo"`
	DataTableId int32     `json:"data_table_id"`
	UserId      int32     `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type MapVersionUser struct {
	MapVersion `xorm:"extends"`
	User       `xorm:"extends"`
}

func (MapVersionUser) TableName() string {
	return "map_version"
}

type MapVersionDataTable struct {
	MapVersion `xorm:"extends"`
	DataTable  `xorm:"extends"`
}

func (MapVersionDataTable) TableName() string {
	return "map_version"
}

type DataTableUploader struct {
	Id          int32 `json:"id" xorm:"pk autoincr"`
	DataTableId int32 `json:"data_table_id"`
	UserId      int32 `json:"user_id"`
}

type DataTableUploaderUser struct {
	DataTableUploader `xorm:"extends"`
	User              `xorm:"extends"`
}

func (DataTableUploaderUser) TableName() string {
	return "data_table_uploader"
}

type Upload struct {
	Id       int32  `json:"id" xorm:"pk autoincr"`
	FileSize int32  `json:"file_size"`
	FileType string `json:"file_type"`
	FileName string `json:"file_name"`
	Url      string `json:"url"`
}

type UploadRef struct {
	Id       int32 `xorm:"pk autoincr"`
	UploadId int32
	RefTable string
	RefId    int32
}
