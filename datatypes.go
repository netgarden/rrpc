package rrpc_gen_ts

import (
	"slices"
	"strings"
)

var numberRrpcTypes = []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float", "float32", "float64", "byte"}

func rrpcDataTypeToTSType(dataType string) string {

	if dataType == "bool" {
		return "boolean"
	}
	if dataType == "string" {
		return "string"
	}
	if slices.Contains(numberRrpcTypes, dataType) {
		return "number"
	}

	if dataType == "bytes" {
		return "Uint8Array"
	}

	if strings.HasPrefix(dataType, "[]") {
		return rrpcDataTypeToTSType(dataType[2:]) + "[]"
	}

	if strings.HasPrefix(dataType, "map[") {
		return "any"
	}

	panic("unsupported rrpc type: " + dataType)
}
