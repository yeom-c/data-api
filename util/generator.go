package util

import (
	"errors"
	"fmt"
	"mime/multipart"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/goccy/go-json"
	"github.com/xuri/excelize/v2"
	"github.com/yeom-c/data-api/database"
)

type Sheet struct {
	StartRow      int
	StartCol      int
	Name          string
	TableName     string
	Schemas       []Schema
	DataMap       map[string]interface{}
	Errors        []string
	GenJson       []byte
	GenQuery      string
	DataTableId   int32
	DataVersionId int32
	Version       int32
}

type Schema struct {
	ColumnIndex    int
	Column         string
	DataGroup      string
	DataType       string
	DataLength     int
	DataKey        bool
	Enum           string
	ReferenceSheet string
}

type DataGenerator struct {
	Excel  *excelize.File
	sheets []Sheet
}

func (g *DataGenerator) SetSheet(sheetName string, startRow, startCol int, dataTableId, dataVersionId, version int32) {
	whiteSpaceRegex := regexp.MustCompile(`\s`)
	tableName := strings.ToLower(whiteSpaceRegex.ReplaceAllString(strings.TrimSpace(sheetName), "_"))
	sheet := Sheet{
		StartRow:      startRow,
		StartCol:      startCol,
		Name:          sheetName,
		TableName:     tableName,
		DataTableId:   dataTableId,
		DataVersionId: dataVersionId,
		Version:       version,
	}

	g.sheets = append(g.sheets, sheet)
}

func (g *DataGenerator) GetSheets() []Sheet {
	return g.sheets
}

func (g *DataGenerator) ReadSheets() error {
	if len(g.sheets) > 0 {
		var wg sync.WaitGroup
		wg.Add(len(g.sheets))

		for i := range g.sheets {
			go func(sheet *Sheet) {
				defer wg.Done()

				sheetRows, err := g.Excel.GetRows(sheet.Name)
				if err != nil {
					sheet.Errors = append(sheet.Errors, fmt.Sprintf("%s 시트 없음", sheet.Name))
					return
				}
				sheetHeader := sheetRows[sheet.StartRow-1]

				schemaRows, err := g.Excel.GetRows("Schema")
				if err != nil {
					sheet.Errors = append(sheet.Errors, "Schema 시트 없음")
					return
				}

				// schema 생성.
				for rowIndex, row := range schemaRows {
					if row == nil {
						continue
					}

					schemaSheetName := ""
					if len(row) > 0 {
						schemaSheetName = row[0]
					}
					if schemaSheetName == sheet.Name {
						columnIndex := 0
						column := ""
						if len(row) > 1 {
							column = strings.TrimSpace(row[1])
							for i, col := range sheetHeader {
								if col == column {
									columnIndex = i
									break
								}
							}
						}
						dataGroup := ""
						if len(row) > 2 {
							dataGroup = strings.TrimSpace(row[2])
						}
						dataType := ""
						dataLength := 0
						if len(row) > 3 {
							dataType = strings.TrimSpace(row[3])
							if dataType != "string" &&
								dataType != "text" &&
								dataType != "uint" &&
								dataType != "int" &&
								dataType != "float" &&
								dataType != "bool" {
								sheet.Errors = append(sheet.Errors, fmt.Sprintf("sheet: Schema, row: %d, value: %v, err: DATA_TYPE 변환 오류", rowIndex+1, row[3]))
							}

							// 기본 length 설정.
							if dataType == "string" {
								dataLength = 255
							} else if dataType == "uint" {
								dataLength = 10
							} else if dataType == "int" {
								dataLength = 10
							}
						}
						if len(row) > 4 {
							if row[4] != "" {
								if dataLength != 0 {
									dataLength, err = strconv.Atoi(strings.TrimSpace(row[4]))
									if err != nil {
										sheet.Errors = append(sheet.Errors, fmt.Sprintf("sheet: Schema, row: %d, value: %v, err: DATA_LENGTH 변환 오류", rowIndex+1, row[4]))
									}
								}
							}
						}
						dataKey := false
						if len(row) > 5 {
							if row[5] != "" {
								dataKey, err = strconv.ParseBool(strings.TrimSpace(row[5]))
								if err != nil {
									sheet.Errors = append(sheet.Errors, fmt.Sprintf("sheet: Schema, row: %d, value: %v, err: DATA_KEY 변환 오류", rowIndex+1, row[5]))
								}
							}
						}
						enum := ""
						if len(row) > 6 {
							enum = strings.TrimSpace(row[6])
						}
						referenceSheet := ""
						if len(row) > 7 {
							referenceSheet = strings.TrimSpace(row[7])
						}

						// 필수 스키마 체크.
						if column == "" || dataType == "" {
							continue
						}
						sheet.Schemas = append(sheet.Schemas, Schema{
							ColumnIndex:    columnIndex,
							Column:         column,
							DataGroup:      dataGroup,
							DataType:       dataType,
							DataLength:     dataLength,
							DataKey:        dataKey,
							Enum:           enum,
							ReferenceSheet: referenceSheet,
						})

						// DataMap SCHEMA 추가
						if sheet.DataMap == nil {
							sheet.DataMap = map[string]interface{}{
								"SCHEMA": map[string]interface{}{},
							}
						}
						if dataGroup == "" {
							sheet.DataMap["SCHEMA"].(map[string]interface{})[column] = map[string]interface{}{
								"DATA_TYPE":       dataType,
								"DATA_LENGTH":     dataLength,
								"DATA_KEY":        dataKey,
								"ENUM":            enum,
								"REFERENCE_SHEET": referenceSheet,
							}
						} else {
							if _, exists := sheet.DataMap["SCHEMA"].(map[string]interface{})[dataGroup]; !exists {
								sheet.DataMap["SCHEMA"].(map[string]interface{})[dataGroup] = map[string]interface{}{
									"IS_ARRAY":        true,
									"DATA_TYPE":       dataType,
									"DATA_LENGTH":     1,
									"DATA_KEY":        dataKey,
									"ENUM":            enum,
									"REFERENCE_SHEET": referenceSheet,
								}
							} else {
								sheet.DataMap["SCHEMA"].(map[string]interface{})[dataGroup].(map[string]interface{})["DATA_LENGTH"] =
									sheet.DataMap["SCHEMA"].(map[string]interface{})[dataGroup].(map[string]interface{})["DATA_LENGTH"].(int) + 1
							}
						}
					} else {
						if len(sheet.Schemas) > 0 {
							break
						}
					}
				}

				// data 생성.
				for rowIndex, row := range sheetRows {
					if rowIndex < sheet.StartRow {
						continue
					}

					dataGroupMap := map[string][]interface{}{}
					rowMap := map[string]interface{}{}
					for _, schema := range sheet.Schemas {
						var value interface{}
						var err error
						if len(row) > schema.ColumnIndex {
							value = row[schema.ColumnIndex]
						} else {
							value = ""
						}
						originValue := value

						switch schema.DataType {
						case "uint":
							if value == "" {
								value = 0
							} else {
								value, err = strconv.ParseUint(value.(string), 0, 64)
							}
						case "int":
							if value == "" {
								value = 0
							} else {
								value, err = strconv.Atoi(value.(string))
							}
						case "float":
							if value == "" {
								value = 0
							} else {
								value, err = strconv.ParseFloat(value.(string), 64)
							}
						case "bool":
							if value == "" {
								value = false
							} else {
								value, err = strconv.ParseBool(value.(string))
							}
						}
						if err != nil {
							sheet.Errors = append(sheet.Errors, fmt.Sprintf("row: %d, col: %v, value: %v, err: %s", rowIndex+1, schema.Column, originValue, err.Error()))
						}

						// group 데이터인 경우 dataGroupMap 에 array 로 별도 저장.
						if schema.DataGroup != "" {
							dataGroupMap[schema.DataGroup] = append(dataGroupMap[schema.DataGroup], value)
						} else {
							rowMap[schema.Column] = value
						}
					}
					// 별도 저장한 group 데이터들 rowMap 에 추가.
					for groupColumn, dataGroupValue := range dataGroupMap {
						rowMap[groupColumn] = dataGroupValue
					}

					sheet.DataMap[rowMap["Enum_Id"].(string)] = rowMap
				}
			}(&g.sheets[i])
		}

		wg.Wait()
	} else {
		return errors.New("sheet 정보 없음")
	}

	return nil
}

func (g *DataGenerator) GenJson() error {
	if len(g.sheets) > 0 {
		var wg sync.WaitGroup
		wg.Add(len(g.sheets))
		for i := range g.sheets {
			go func(sheet *Sheet) {
				defer wg.Done()

				dataJson, err := json.Marshal(sheet.DataMap)
				if err != nil {
					sheet.Errors = append(sheet.Errors, fmt.Sprintf("sheet: %s, err: json 변환 오류 %s", sheet.Name, err.Error()))
				}
				sheet.GenJson = dataJson
			}(&g.sheets[i])
		}

		wg.Wait()
	} else {
		return errors.New("sheet 정보 없음")
	}

	return nil
}

func (g *DataGenerator) GenDb() error {
	if len(g.sheets) > 0 {
		var wg sync.WaitGroup
		wg.Add(len(g.sheets))
		for i := range g.sheets {
			go func(sheet *Sheet) {
				defer wg.Done()

				if len(sheet.Errors) > 0 {
					return
				}

				// 테이블 생성.
				if len(sheet.Schemas) == 0 {
					sheet.Errors = append(sheet.Errors, fmt.Sprintf("sheet: %s, err: db 변환 오류 스키마 정보 없음", sheet.Name))
					return
				}

				dataGroupMap := map[string]bool{}
				schemasQuery := []string{}
				for _, schema := range sheet.Schemas {
					query := ""
					if schema.DataGroup == "" {
						switch schema.DataType {
						case "string":
							query = fmt.Sprintf("`%s` varchar(%d) NOT NULL", strings.ToLower(schema.Column), schema.DataLength)
						case "text":
							query = fmt.Sprintf("`%s` text NOT NULL", strings.ToLower(schema.Column))
						case "uint":
							query = fmt.Sprintf("`%s` int(%d) unsigned NOT NULL", strings.ToLower(schema.Column), schema.DataLength)
						case "int":
							query = fmt.Sprintf("`%s` int(%d) NOT NULL", strings.ToLower(schema.Column), schema.DataLength)
						case "float":
							query = fmt.Sprintf("`%s` double NOT NULL", strings.ToLower(schema.Column))
						case "bool":
							query = fmt.Sprintf("`%s` tinyint(1) NOT NULL", strings.ToLower(schema.Column))
						default:
							sheet.Errors = append(sheet.Errors, fmt.Sprintf("sheet: %s, column: %s, data_type: %s, err: 데이터 타입 오류", sheet.Name, schema.Column, schema.DataType))
							return
						}
					} else {
						if _, exists := dataGroupMap[schema.DataGroup]; !exists {
							dataGroupMap[schema.DataGroup] = true
							query = fmt.Sprintf("`%s` text NOT NULL", strings.ToLower(schema.DataGroup))
						}
					}
					if query != "" {
						schemasQuery = append(schemasQuery, query)
					}
				}

				genTableName := fmt.Sprintf("%s_v%d", sheet.TableName, sheet.Version)
				_, err := database.Database().StaticDataGenConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`;", genTableName))
				if err != nil {
					sheet.Errors = append(sheet.Errors, fmt.Sprintf("sheet: %s, table: %s, err: db 테이블 삭제 오류 %s", sheet.Name, genTableName, err.Error()))
					return
				}

				schemaQuery := strings.Join(schemasQuery, ",")
				query := "CREATE TABLE `" + genTableName + "` (" +
					schemaQuery +
					"  , `created_at` timestamp NOT NULL DEFAULT current_timestamp()," +
					"  PRIMARY KEY (`id`)," +
					"  UNIQUE KEY `uniq_enum_id` (`enum_id`) USING BTREE" +
					") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;"
				_, err = database.Database().StaticDataGenConn.Exec(query)
				if err != nil {
					sheet.Errors = append(sheet.Errors, fmt.Sprintf("sheet: %s, table: %s, err: db 테이블 생성 오류 %s", sheet.Name, genTableName, err.Error()))
					return
				}

				var columns []string
				var columnsQuery []string
				schema := sheet.DataMap["SCHEMA"]
				for column := range schema.(map[string]interface{}) {
					columns = append(columns, column)
					columnsQuery = append(columnsQuery, fmt.Sprintf("`%s`", strings.ToLower(column)))
				}

				// 테이블 데이터 추가.
				inserts := []string{}
				for key, data := range sheet.DataMap {
					if key != "SCHEMA" {
						insert := ""
						for _, column := range columns {
							value := data.(map[string]interface{})[column]
							if reflect.TypeOf(value).Kind().String() == "slice" {
								valueJson, _ := json.Marshal(value)
								value = string(valueJson)
							}

							if reflect.TypeOf(value).Kind().String() == "string" {
								value = strings.Replace(value.(string), "'", "\\'", -1)
								value = fmt.Sprintf("'%s'", value)
							}

							if insert == "" {
								insert = fmt.Sprintf("%v", value)
							} else {
								insert = fmt.Sprintf("%v,%v", insert, value)
							}
						}
						inserts = append(inserts, fmt.Sprintf("(%s)", insert))
					}
				}

				if len(inserts) > 0 {
					query = fmt.Sprintf("INSERT INTO `%s`(%s) VALUES ", genTableName, strings.Join(columnsQuery, ","))
					query += strings.Join(inserts, ",")
					_, err = database.Database().StaticDataGenConn.Exec(query)
					if err != nil {
						sheet.Errors = append(sheet.Errors, fmt.Sprintf("sheet: %s, table: %s, err: db 테이블 데이터 생성 오류 %s", sheet.Name, genTableName, err.Error()))
						return
					}
				}
			}(&g.sheets[i])
		}

		wg.Wait()
	} else {
		return errors.New("sheet 정보 없음")
	}

	return nil
}

type Enum struct {
	Name    string
	Values  []string
	BitFlag bool
}

type EnumGenerator struct {
	File      *multipart.File
	Enums     map[string]Enum
	EnumNames []string
	Errors    []string
}

func (g *EnumGenerator) ReadSheet(sheetName string, startRow, startCol int) {
	excel, err := excelize.OpenReader(*g.File)
	if err != nil {
		g.Errors = append(g.Errors, err.Error())
		return
	}
	defer excel.Close()

	rows, err := excel.GetRows(sheetName)
	if err != nil {
		g.Errors = append(g.Errors, err.Error())
		return
	}

	g.Enums = map[string]Enum{}
	g.EnumNames = []string{}
	for rowIndex, row := range rows {
		if rowIndex+1 == startRow {
			// header

		} else if rowIndex+1 > startRow {
			// data
			startColIndex := startCol - 1
			enum := Enum{}
			for colIndex, colCell := range row {
				if colIndex < startColIndex {
					continue
				} else if colIndex == startColIndex {
					enum.Name = colCell
				} else if colIndex == startColIndex+1 {
					if colCell == "true" {
						enum.BitFlag = true
					}
				} else if colIndex > startColIndex+1 {
					enum.Values = append(enum.Values, colCell)
				}
			}

			if enum.BitFlag {
				if len(enum.Values) > 32 {
					g.Errors = append(g.Errors, fmt.Sprintf("%s Bit Enum 32개 이하로 가능", enum.Name))
				}
			}

			if _, exist := g.Enums[enum.Name]; exist {
				g.Errors = append(g.Errors, fmt.Sprintf("%s 중복 이넘", enum.Name))
			} else {
				g.Enums[enum.Name] = enum
				g.EnumNames = append(g.EnumNames, enum.Name)
			}
		}
	}
	sort.Strings(g.EnumNames)
}

func (g *EnumGenerator) GenCSharp() string {
	var gen string

	template := `using System;

public static partial class Data {
	public static class Enum {
		%s
	}
}`
	enums := []string{}
	for _, enumName := range g.EnumNames {
		enum := g.Enums[enumName]
		enumTemplate := `public enum %s {
			%s
		}`

		values := []string{}
		for i, value := range enum.Values {
			values = append(values, fmt.Sprintf("%s = %d", value, i))
		}
		enumString := fmt.Sprintf(enumTemplate, enum.Name, strings.Join(values, ",\n\t\t\t"))
		enums = append(enums, enumString)

		if enum.BitFlag {
			enumTemplate := `[Flags]
		public enum %s_Flags {
			EMPTY_FLAG = 0,
			%s
		}`
			values := []string{}
			for i, value := range enum.Values {
				values = append(values, fmt.Sprintf("%s = 1 << %d", value, i))
			}
			enumString := fmt.Sprintf(enumTemplate, enum.Name, strings.Join(values, ",\n\t\t\t"))
			enums = append(enums, enumString)
		}
	}

	gen = fmt.Sprintf(template, strings.Join(enums, "\n\n\t\t"))

	return gen
}

func (g *EnumGenerator) GenGolang() string {
	var gen string

	template := `package enum

%s
`
	enums := []string{}
	for _, enumName := range g.EnumNames {
		enum := g.Enums[enumName]
		enumTemplate := `type %s int32
const (
%s
)

var (
	%s
)

func (e %s) String() string {
	return %sName[e]
}

func Get%s(s string) %s {
	return %sValue[s]
}`
		enumName := enum.Name
		consts := []string{}
		varNames := []string{}
		varValues := []string{}
		for i, value := range enum.Values {
			constName := fmt.Sprintf("%s_%s", enumName, value)
			if i == 0 {
				consts = append(consts, fmt.Sprintf("\t%s %s = iota", constName, enumName))
			} else {
				consts = append(consts, fmt.Sprintf("\t%s", constName))
			}

			if i == len(enum.Values)-1 {
				varNames = append(varNames, fmt.Sprintf("\t\t%s: \"%s\",", constName, value))
				varValues = append(varValues, fmt.Sprintf("\t\t\"%s\": %s,", value, constName))
			} else {
				varNames = append(varNames, fmt.Sprintf("\t\t%s: \"%s\"", constName, value))
				varValues = append(varValues, fmt.Sprintf("\t\t\"%s\": %s", value, constName))
			}
		}

		varTemplate := `%sName = map[%s]string{
%s
	}

	%sValue = map[string]%s{
%s
	}`
		varString := fmt.Sprintf(varTemplate, enumName, enumName, strings.Join(varNames, ",\n"), enumName, enumName, strings.Join(varValues, ",\n"))
		enumString := fmt.Sprintf(enumTemplate, enumName, strings.Join(consts, "\n"), varString, enumName, enumName, enumName, enumName, enumName)
		enums = append(enums, enumString)

		if enum.BitFlag {
			enumName := enum.Name + "Bit"
			consts := []string{}
			varNames := []string{}
			varValues := []string{}
			for i, value := range enum.Values {
				constName := fmt.Sprintf("%s_%s", enumName, value)
				if i == 0 {
					consts = append(consts, fmt.Sprintf("\t%s %s = 1 << iota", constName, enumName))
				} else {
					consts = append(consts, fmt.Sprintf("\t%s", constName))
				}

				if i == len(enum.Values)-1 {
					varNames = append(varNames, fmt.Sprintf("\t\t%s: \"%s\",", constName, value))
					varValues = append(varValues, fmt.Sprintf("\t\t\"%s\": %s,", value, constName))
				} else {
					varNames = append(varNames, fmt.Sprintf("\t\t%s: \"%s\"", constName, value))
					varValues = append(varValues, fmt.Sprintf("\t\t\"%s\": %s", value, constName))
				}
			}

			varTemplate := `%sName = map[%s]string{
%s
	}

	%sValue = map[string]%s{
%s
	}`
			varString := fmt.Sprintf(varTemplate, enumName, enumName, strings.Join(varNames, ",\n"), enumName, enumName, strings.Join(varValues, ",\n"))
			enumString := fmt.Sprintf(enumTemplate, enumName, strings.Join(consts, "\n"), varString, enumName, enumName, enumName, enumName, enumName)
			enums = append(enums, enumString)
		}
	}

	gen = fmt.Sprintf(template, strings.Join(enums, "\n\n"))

	return gen
}
