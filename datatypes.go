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

	panic("unsupported rrpc type: " + dataType)
}

func rrpcDataTypeToTSInputType(dataType string) string {

	if dataType == "bytes" {
		return "string | Uint8Array"
	}

	return rrpcDataTypeToTSType(dataType)
}
