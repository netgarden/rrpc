package rrpc_gen_ts

import (
	"strings"
	"text/template"

	def "github.com/netgarden/rrpc/def"
)

func getTemplatesFuncs() template.FuncMap {

	funcs := def.GetTemplateFuncMap()
	funcs["rrpcDataTypeToTSType"] = rrpcDataTypeToTSType
	funcs["rrpcDataTypeToTSInputType"] = rrpcDataTypeToTSInputType
	funcs["methodPathParams"] = methodPathParams
	funcs["methodPathToTS"] = methodPathToTS
	funcs["methodPathHasPathParam"] = methodPathHasPathParam

	return funcs
}

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

// methodPathToTS converts a path string to a TypeScript expression.
// "string"-typed params (the default) are escaped as a single path segment
// via encodeURIComponent; "path"-typed params may themselves contain "/"
// and are escaped per-segment via escapePathValue so the separators
// survive.
// E.g. "/upload/{filename:string}" -> "/upload/" + encodeURIComponent(filename)
func methodPathToTS(path string) string {
	var parts []string
	rest := path
	for {
		start := strings.Index(rest, "{")
		if start == -1 {
			if rest != "" {
				parts = append(parts, `"`+rest+`"`)
			}
			break
		}
		if start > 0 {
			parts = append(parts, `"`+rest[:start]+`"`)
		}
		end := strings.Index(rest[start:], "}")
		if end == -1 {
			break
		}
		param := rest[start+1 : start+end]
		paramType := "string"
		if colonIdx := strings.Index(param, ":"); colonIdx != -1 {
			paramType = param[colonIdx+1:]
			param = param[:colonIdx]
		}
		parts = append(parts, pathParamEscapeExprTS(paramType, param))
		rest = rest[start+end+1:]
	}
	if len(parts) == 0 {
		return `"` + path + `"`
	}
	return strings.Join(parts, " + ")
}

// pathParamEscapeExprTS returns the TypeScript expression that escapes a
// path parameter's value for inclusion in a URL, per its declared type.
func pathParamEscapeExprTS(paramType, param string) string {
	switch paramType {
	case "string":
		return "encodeURIComponent(" + param + ")"
	case "path":
		return "escapePathValue(" + param + ")"
	default:
		panic("unknown path parameter type \"" + paramType + "\" for parameter \"" + param + "\" (expected \"string\" or \"path\")")
	}
}

// methodPathHasPathParam reports whether path declares at least one
// "path"-typed (catch-all) path parameter — used to decide whether the
// generated file needs to import escapePathValue.
func methodPathHasPathParam(path string) bool {
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
		if paramType == "path" {
			return true
		}
		rest = rest[start+end+1:]
	}
}
