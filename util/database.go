package util

import (
	"fmt"
	"reflect"
	"strings"
	"time"
	"xorm.io/xorm"
)

func SetFilter(filterList map[string]map[string]interface{}, dbSession *xorm.Session) {
	for key, filter := range filterList {
		qry := filter["qry"]
		if qry == "" || qry == nil {
			continue
		}

		qryType := reflect.TypeOf(qry)
		if qryType.Kind() == reflect.String {
			qry = strings.TrimSpace(qry.(string))
			if qry == "" {
				continue
			}
		} else if qryType.Kind() == reflect.Slice {
			v := reflect.ValueOf(qry)
			if v.Len() == 0 {
				continue
			}
		}

		if filter["op"] == "like" {
			dbSession.Where(fmt.Sprintf("%s %s '%%%v%%'", key, filter["op"], qry))
		} else if filter["op"] == "in" {
			dbSession.In(key, qry)
		} else {
			dbSession.Where(fmt.Sprintf("%s %s '%v'", key, filter["op"], qry))
		}
	}
}

type tableField struct {
	Field   string
	Type    string
	Null    string
	Key     string
	Default string
	Extra   string
}

func CopyTable(originDbConn *xorm.Engine, copyDbConn *xorm.Engine, originTableName string, copyTableName string) error {
	// 테이블 생성 쿼리 생성.
	rows, err := originDbConn.Query(fmt.Sprintf("DESC %s;", originTableName))
	if err != nil {
		return fmt.Errorf("테이블 스키마 읽기 실패, err: %s", err.Error())
	}

	columns := []string{}
	columnTypes := map[string]string{}
	tableFields := []tableField{}
	for _, row := range rows {
		columns = append(columns, string(row["Field"]))
		columnTypes[string(row["Field"])] = string(row["Type"])
		tableFields = append(tableFields, tableField{
			Field:   string(row["Field"]),
			Type:    string(row["Type"]),
			Null:    string(row["Null"]),
			Key:     string(row["Key"]),
			Default: string(row["Default"]),
			Extra:   string(row["Extra"]),
		})
	}

	fieldQueries := []string{}
	for _, field := range tableFields {
		null := "NULL"
		if field.Null == "NO" {
			null = "NOT NULL"
		}
		def := ""
		if field.Default != "" {
			def = fmt.Sprintf("DEFAULT %s", field.Default)
		}
		fieldQueries = append(fieldQueries, fmt.Sprintf("`%s` %s %s %s", field.Field, field.Type, null, def))
	}

	createTableQuery := "CREATE TABLE `%s` (" +
		"%s," +
		" PRIMARY KEY (`id`)," +
		" UNIQUE KEY `uniq_enum_id` (`enum_id`) USING BTREE" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;"

	// 대상 데이터베이스에 복사.
	// 이미 존재하는 테이블 삭제.
	err = copyDbConn.NewSession().DropTable(originTableName)
	if err != nil {
		return fmt.Errorf("테이블 삭제 실패, err: %s", err.Error())
	}

	// 테이블 생성.
	_, err = copyDbConn.Exec(fmt.Sprintf(createTableQuery, originTableName, strings.Join(fieldQueries, ",")))
	if err != nil {
		return fmt.Errorf("테이블 생성 실패, err: %s", err.Error())
	}

	// 데이터 복사.
	insertColumns := []string{}
	for _, column := range columns {
		insertColumns = append(insertColumns, fmt.Sprintf("`%s`", column))
	}

	insertValues := []string{}
	rows, err = originDbConn.Query(fmt.Sprintf("SELECT * FROM %s;", originTableName))
	if err != nil {
		return fmt.Errorf("테이블 데이터 읽기 실패, err: %s", err.Error())
	}
	for _, row := range rows {
		values := []string{}
		for _, column := range columns {
			value := string(row[column])
			columnType := columnTypes[column]
			if columnType == "timestamp" {
				valueTime, err := time.Parse(time.RFC3339, value)
				if err != nil {
					return fmt.Errorf("데이터 변환 실패, err: %s", err.Error())
				}
				value = valueTime.Format("2006-01-02 15:04:05")
			}
			value = strings.Replace(value, "'", "\\'", -1)
			values = append(values, fmt.Sprintf("'%s'", value))
		}
		insertValues = append(insertValues, fmt.Sprintf("(%s)", strings.Join(values, ",")))
	}

	if len(insertValues) > 0 {
		insertQuery := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES %s;", originTableName, strings.Join(insertColumns, ","), strings.Join(insertValues, ","))
		_, err = copyDbConn.Exec(insertQuery)
		if err != nil {
			return fmt.Errorf("데이터 복사 실패, err: %s", err.Error())
		}
	}

	err = copyDbConn.NewSession().DropTable(copyTableName)
	if err != nil {
		return fmt.Errorf("테이블 삭제 실패, err: %s", err.Error())
	}
	_, err = copyDbConn.Exec(fmt.Sprintf("ALTER TABLE `%s` RENAME TO `%s`;", originTableName, copyTableName))
	if err != nil {
		return fmt.Errorf("테이블 이름 변경 실패, err: %s", err.Error())
	}

	return nil
}
