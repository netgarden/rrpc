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

// streamGoType returns the rrpc runtime server-side stream type (with
// type parameters, if any) for a streaming method - e.g.
// "Stream[ChatMessage, ChatReply]", "InStream[LogLine]", "RawOutStream".
// The validator guarantees the input/output counts implied by
// method.StreamDirection and that "both"-direction methods aren't a
// mixed struct/bytes combination, so this doesn't re-check those.
func streamGoType(method *def.Method) string {
	switch method.StreamDirection {
	case "both":
		in, out := method.Inputs[0], method.Outputs[0]
		if in.Type == "bytes" {
			return "RawStream"
		}
		return "Stream[" + in.Type + ", " + out.Type + "]"
	case "in":
		in := method.Inputs[0]
		if in.Type == "bytes" {
			return "RawInStream"
		}
		return "InStream[" + in.Type + "]"
	case "out":
		out := method.Outputs[0]
		if out.Type == "bytes" {
			return "RawOutStream"
		}
		return "OutStream[" + out.Type + "]"
	default:
		panic("streamGoType: unknown stream direction " + method.StreamDirection)
	}
}

// streamUpgradeCall returns the rrpc runtime Upgrade*/UpgradeRaw*
// function call expression (with type arguments, if any) for a streaming
// method - e.g. "UpgradeStream[ChatMessage, ChatReply]",
// "UpgradeInStream[LogLine]", "UpgradeRawOutStream".
func streamUpgradeCall(method *def.Method) string {
	switch method.StreamDirection {
	case "both":
		in, out := method.Inputs[0], method.Outputs[0]
		if in.Type == "bytes" {
			return "UpgradeRawStream"
		}
		return "UpgradeStream[" + in.Type + ", " + out.Type + "]"
	case "in":
		in := method.Inputs[0]
		if in.Type == "bytes" {
			return "UpgradeRawInStream"
		}
		return "UpgradeInStream[" + in.Type + "]"
	case "out":
		out := method.Outputs[0]
		if out.Type == "bytes" {
			return "UpgradeRawOutStream"
		}
		return "UpgradeOutStream[" + out.Type + "]"
	default:
		panic("streamUpgradeCall: unknown stream direction " + method.StreamDirection)
	}
}

// streamClientGoType returns the generated Go client's counterpart of
// streamGoType - e.g. "ClientStream[ChatMessage, ChatReply]",
// "ClientSendStream[LogLine]", "ClientRawRecvStream". Note the client's
// "in"/"out" naming is from the CLIENT's own perspective (matching
// rrpc.ClientSendStream/ClientRecvStream's doc comments): an
// "@InputStream" method still means "client sends", so it maps to
// ClientSendStream, not ClientInStream.
func streamClientGoType(method *def.Method) string {
	switch method.StreamDirection {
	case "both":
		in, out := method.Inputs[0], method.Outputs[0]
		if in.Type == "bytes" {
			return "ClientRawStream"
		}
		return "ClientStream[" + in.Type + ", " + out.Type + "]"
	case "in":
		in := method.Inputs[0]
		if in.Type == "bytes" {
			return "ClientRawSendStream"
		}
		return "ClientSendStream[" + in.Type + "]"
	case "out":
		out := method.Outputs[0]
		if out.Type == "bytes" {
			return "ClientRawRecvStream"
		}
		return "ClientRecvStream[" + out.Type + "]"
	default:
		panic("streamClientGoType: unknown stream direction " + method.StreamDirection)
	}
}

// streamDialCall returns the rrpc runtime Dial*/DialRaw* function call
// expression (with type arguments, if any) for a streaming method - the
// dialing counterpart of streamUpgradeCall.
func streamDialCall(method *def.Method) string {
	switch method.StreamDirection {
	case "both":
		in, out := method.Inputs[0], method.Outputs[0]
		if in.Type == "bytes" {
			return "DialRawStream"
		}
		return "DialStream[" + in.Type + ", " + out.Type + "]"
	case "in":
		in := method.Inputs[0]
		if in.Type == "bytes" {
			return "DialRawSendStream"
		}
		return "DialSendStream[" + in.Type + "]"
	case "out":
		out := method.Outputs[0]
		if out.Type == "bytes" {
			return "DialRawRecvStream"
		}
		return "DialRecvStream[" + out.Type + "]"
	default:
		panic("streamDialCall: unknown stream direction " + method.StreamDirection)
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

// goFieldType is the Go type of a struct field. An optional scalar or struct
// becomes a pointer, so "unset" (nil) is distinguishable from the zero value.
// Slices and maps are already nil-able, so they stay as they are.
func goFieldType(field def.Field) string {
	if field.Optional && !strings.HasPrefix(field.Type, "[]") && !strings.HasPrefix(field.Type, "map[") {
		return "*" + field.Type
	}
	return field.Type
}

// goJSONTag is the struct tag body for a field; optional fields are omitted
// from the JSON when unset.
func goJSONTag(field def.Field) string {
	if field.Optional {
		return "json:\"" + field.Name + ",omitempty\""
	}
	return "json:\"" + field.Name + "\""
}
