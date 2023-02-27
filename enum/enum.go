package enum

type DataTableType int32

const (
	DataTableTypeData DataTableType = iota
	DataTableTypeEnum
	DataTableTypeMap
)

var (
	DataTableTypeName = map[DataTableType]string{
		DataTableTypeData: "Data",
		DataTableTypeEnum: "Enum",
		DataTableTypeMap:  "Map",
	}
	DataTableTypeValue = map[string]DataTableType{
		"Data": DataTableTypeData,
		"Enum": DataTableTypeEnum,
		"Map":  DataTableTypeMap,
	}
)

func (e DataTableType) String() string {
	return DataTableTypeName[e]
}

func GetDataTableType(s string) DataTableType {
	return DataTableType(DataTableTypeValue[s])
}

type DataVersionStatus int32

const (
	DataVersionStatusNone DataVersionStatus = iota
	DataVersionStatusProcessing
	DataVersionStatusComplete
	DataVersionStatusError
)

var (
	DataVersionStatusName = map[DataVersionStatus]string{
		DataVersionStatusNone:       "None",
		DataVersionStatusProcessing: "Processing",
		DataVersionStatusComplete:   "Complete",
		DataVersionStatusError:      "Error",
	}
	DataVersionStatusValue = map[string]DataVersionStatus{
		"None":       DataVersionStatusNone,
		"Processing": DataVersionStatusProcessing,
		"Complete":   DataVersionStatusComplete,
		"Error":      DataVersionStatusError,
	}
)

func (e DataVersionStatus) String() string {
	return DataVersionStatusName[e]
}

func GetDataVersionStatus(s string) DataVersionStatus {
	return DataVersionStatus(DataVersionStatusValue[s])
}

type TrueFalse int32

const (
	TrueFalseFalse TrueFalse = iota
	TrueFalseTrue
)

var (
	TrueFalseName = map[TrueFalse]string{
		TrueFalseFalse: "False",
		TrueFalseTrue:  "True",
	}
	TrueFalseValue = map[string]TrueFalse{
		"False": TrueFalseFalse,
		"True":  TrueFalseTrue,
	}
)

func (e TrueFalse) String() string {
	return TrueFalseName[e]
}

func GetTrueFalse(s string) TrueFalse {
	return TrueFalse(TrueFalseValue[s])
}
