package rrpc_def

var SupportedPrimitiveTypes = []string{
	"bool",
	"string",
	"int", "int8", "int16", "int32", "int64",
	"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
	"float32", "float64",
	"byte",
}

var SupportedComplexTypes = []string{
	"datetime",
}

func GetSupportedPrimitiveTypes() []string {
	return SupportedPrimitiveTypes
}

func GetSupportedComplexTypes() []string {
	return SupportedComplexTypes
}
