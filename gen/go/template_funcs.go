package rrpc_gen_go

import (
	"strings"

	def "github.com/netgarden/rrpc-def"
)

// methodPathParams extracts path parameter names from a path string.
// E.g. "/upload/{filename:string}" -> ["filename"]
func methodPathParams(path string) []string {
	var params []string
	rest := path
	for {
		start := strings.Index(rest, "{")
		if start == -1 {
			break
		}
		end := strings.Index(rest[start:], "}")
		if end == -1 {
			break
		}
		param := rest[start+1 : start+end]
		if colonIdx := strings.Index(param, ":"); colonIdx != -1 {
			param = param[:colonIdx]
		}
		params = append(params, param)
		rest = rest[start+end+1:]
	}
	return params
}

// methodPathToGo converts a path string to a Go expression producing the path.
// E.g. "/upload/{filename:string}" -> `fmt.Sprintf("/upload/%s", url.PathEscape(filename))`
func methodPathToGo(path string) string {
	var segments []string
	var args []string
	rest := path
	literal := ""
	for {
		start := strings.Index(rest, "{")
		if start == -1 {
			literal += rest
			break
		}
		literal += rest[:start]
		end := strings.Index(rest[start:], "}")
		if end == -1 {
			literal += rest[start:]
			break
		}
		param := rest[start+1 : start+end]
		if colonIdx := strings.Index(param, ":"); colonIdx != -1 {
			param = param[:colonIdx]
		}
		literal += "%s"
		args = append(args, "url.PathEscape("+param+")")
		rest = rest[start+end+1:]
	}
	if len(args) == 0 {
		return `"` + literal + `"`
	}
	segments = append(segments, `"`+literal+`"`)
	_ = segments
	return `fmt.Sprintf("` + literal + `", ` + strings.Join(args, ", ") + `)`
}

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
