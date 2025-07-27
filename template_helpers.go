package rrpc_def

import (
	"reflect"
	"strings"
	"text/template"
)

func GetTemplateFuncMap() template.FuncMap {
	return map[string]interface{}{
		"isLast": func(x int, a interface{}) bool {
			return x == reflect.ValueOf(a).Len()-1
		},
		"firstLetterToLower": func(s string) string {

			if len(s) == 0 {
				return ""
			} else if len(s) == 1 {
				return strings.ToLower(s)
			}

			s = strings.ToLower(string(s[0])) + s[1:]

			return s
		},
		"firstLetterToUpper": func(s string) string {

			if len(s) == 0 {
				return ""
			} else if len(s) == 1 {
				return strings.ToUpper(s)
			}

			s = strings.ToUpper(string(s[0])) + s[1:]

			return s
		},
		"mklist": func(args ...interface{}) []interface{} {
			return args
		},
		"append": func(l []interface{}, items ...interface{}) []interface{} {
			return append(l, items...)
		},
	}
}
