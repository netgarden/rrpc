package rrpc_gen_ts

import (
	"slices"
	"strings"
)

var numberRrpcTypes = []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float", "float32", "float64", "byte"}

func rrpcDataTypeToTSType(dataType string) string {

	if strings.HasPrefix(dataType, "[]") {
		return rrpcDataTypeToTSType(dataType[2:]) + "[]"
	}

	if strings.HasPrefix(dataType, "map[") {
		// Mirrors def/parser/processor.go's checkDataType parsing of the
		// same "map[K]V" syntax, which has already validated K/V by the
		// time codegen runs.
		parts := strings.SplitN(dataType[4:], "]", 2)
		if len(parts) == 2 {
			return "Record<" + rrpcDataTypeToTSType(parts[0]) + ", " + rrpcDataTypeToTSType(parts[1]) + ">"
		}
		return "any"
	}

	if slices.Contains(numberRrpcTypes, dataType) {
		return "number"
	}

	switch dataType {
	case "string":
		return "string"
	case "bool":
		return "boolean"
	case "bytes":
		return "Uint8Array"
	}

	// Not a primitive: the parser has already validated this as a declared
	// struct name, so treat it as a reference to the generated TS class.
	return dataType
}

func rrpcDataTypeToTSInputType(dataType string) string {

	if dataType == "bytes" {
		return "string | Uint8Array"
	}

	return rrpcDataTypeToTSType(dataType)
}
