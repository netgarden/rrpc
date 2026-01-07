package rrpc_gen_go

import (
	def "github.com/netgarden/rrpc-def"
)

func createFieldDefaultValue(field def.Field) string {
	if field.IsStruct {
		return createStructFieldDefaultValue(field)
	} else {
		return createPrimitiveFieldDefaultValue(field)
	}
}

func createStructFieldDefaultValue(field def.Field) string {
	return field.Type + "{}"
}

func createPrimitiveFieldDefaultValue(field def.Field) string {

	/*
		var SupportedPrimitiveTypes = []string{
			"bool",
			"string",
			"int", "int32", "int64",
			"uint", "uint32", "uint64",
			"float32", "float64",
			"byte",
		}
		var SupportedComplexTypes = []string{
			"datetime",
		}
	*/

	switch field.Type {
	case "bool":
		return "false"
	case "string":
		return "\"\""
	case "int":
		return "0"
	case "int8":
		return "0"
	case "int16":
		return "0"
	case "int32":
		return "0"
	case "int64":
		return "0"
	case "uint":
		return "0"
	case "uint8":
		return "0"
	case "uint16":
		return "0"
	case "uint32":
		return "0"
	case "uint64":
		return "0"
	case "float32":
		return "0.0"
	case "float64":
		return "0.0"
	case "byte":
		return "0"
	case "datetime":
		return "time.Time"
	}

	panic("Unknown primitive field type " + field.Type)
}
