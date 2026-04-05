package rrpc_gen_ts

import (
	def "github.com/netgarden/rrpc-def"
	"strings"
	"text/template"
)

func getTemplatesFuncs() template.FuncMap {

	funcs := def.GetTemplateFuncMap()
	funcs["rrpcDataTypeToTSType"] = rrpcDataTypeToTSType
	funcs["methodPathParams"] = methodPathParams
	funcs["methodPathToTS"] = methodPathToTS

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
		if colonIdx := strings.Index(param, ":"); colonIdx != -1 {
			param = param[:colonIdx]
		}
		parts = append(parts, "encodeURIComponent("+param+")")
		rest = rest[start+end+1:]
	}
	if len(parts) == 0 {
		return `"` + path + `"`
	}
	return strings.Join(parts, " + ")
}