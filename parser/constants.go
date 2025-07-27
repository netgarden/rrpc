package parser

var SupportedPrimitiveTypes = []string{
	"bool",
	"string",
	"int", "int32", "int64",
	"uint", "uint32", "uint64",
	"float32", "float64",
	"byte",
}

var SupportedComplexTypes = []string{
	// "datetime",
}

func GetSupportedPrimitiveTypes() []string {
	return SupportedPrimitiveTypes
}

func GetSupportedComplexTypes() []string {
	return SupportedComplexTypes
}
