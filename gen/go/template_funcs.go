package rrpc_gen_go

import (
	"strings"

	def "github.com/netgarden/rrpc/def"
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

// methodPathToGo converts a path string to a Go expression producing the
// path. "string"-typed params (the default when no type is given) are
// escaped as a single path segment via url.PathEscape; "path"-typed params
// may themselves contain "/" and are escaped per-segment via
// rrpc.EscapePathValue so the separators survive.
// E.g. "/upload/{filename:string}" -> `fmt.Sprintf("/upload/%s", url.PathEscape(filename))`
func methodPathToGo(path string) string {
	var args []string
	literal := ""
	rest := path
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
		paramType := "string"
		if colonIdx := strings.Index(param, ":"); colonIdx != -1 {
			paramType = param[colonIdx+1:]
			param = param[:colonIdx]
		}
		literal += "%s"
		args = append(args, pathParamEscapeExprGo(paramType, param))
		rest = rest[start+end+1:]
	}
	if len(args) == 0 {
		return `"` + literal + `"`
	}
	return `fmt.Sprintf("` + literal + `", ` + strings.Join(args, ", ") + `)`
}

// pathParamEscapeExprGo returns the Go expression that escapes a path
// parameter's value for inclusion in a URL, per its declared type.
func pathParamEscapeExprGo(paramType, param string) string {
	switch paramType {
	case "string":
		return "url.PathEscape(" + param + ")"
	case "path":
		return "rrpc.EscapePathValue(" + param + ")"
	default:
		panic("unknown path parameter type \"" + paramType + "\" for parameter \"" + param + "\" (expected \"string\" or \"path\")")
	}
}

// methodPathHasStringParam reports whether path declares at least one
// "string"-typed (the default) path parameter — used to decide whether the
// generated file needs to import "net/url" for url.PathEscape.
func methodPathHasStringParam(path string) bool {
	rest := path
	for {
		start := strings.Index(rest, "{")
		if start == -1 {
			return false
		}
		end := strings.Index(rest[start:], "}")
		if end == -1 {
			return false
		}
		param := rest[start+1 : start+end]
		paramType := "string"
		if colonIdx := strings.Index(param, ":"); colonIdx != -1 {
			paramType = param[colonIdx+1:]
		}
		if paramType == "string" {
			return true
		}
		rest = rest[start+end+1:]
	}
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
