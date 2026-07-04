package rrpc

import "strings"

type Path struct {
	Path   string
	Regex  string
	Params map[string]PathParam
}

type PathParam struct {
	Name string
	Type string
}

func ParsePath(path string) *Path {

	ret := &Path{
		Path:   path,
		Params: make(map[string]PathParam),
	}

	var lastEndIndex = 0
	for {

		startIndex := strings.Index(path[lastEndIndex:], "{")
		if startIndex == -1 {
			ret.Regex += path[lastEndIndex:]
			break
		}

		startIndex += lastEndIndex

		ret.Regex += path[lastEndIndex:startIndex]

		endIndex := strings.Index(path[startIndex:], "}")
		if endIndex == -1 {
			panic("Could not find param end in path " + path)
		}

		param := path[startIndex+1 : startIndex+endIndex]
		paramType := "string"

		paramTypeIndex := strings.Index(param, ":")
		if paramTypeIndex != -1 {
			paramType = param[paramTypeIndex+1:]
			param = param[:paramTypeIndex]
		}

		if _, found := ret.Params[param]; found {
			panic("Param " + param + " already defined")
		}

		var paramRegex string

		switch paramType {
		case "string":
			paramRegex = "[^\\/]+"
			break
		case "path":
			paramRegex = ".*"
			break
		default:
			panic("Unknown param type " + paramType)
		}

		ret.Regex += "(?P<" + param + ">" + paramRegex + ")"
		ret.Params[param] = PathParam{Name: param, Type: paramType}

		lastEndIndex = startIndex + endIndex + 1

	}

	return ret
}
