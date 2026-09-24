package rrpc

import (
	"regexp"
	"strings"
)

// route is a single (service, method) endpoint with its precompiled
// matching pattern and specificity, computed once when its module is
// registered. Requests are matched against routes in a fixed, deterministic
// order — not however ServerModule.Services() (a map) happens to iterate,
// which Go randomizes per process.
type route struct {
	serviceHandler ServiceHandler
	method         MethodHolder
	pattern        *regexp.Regexp
	specificity    routeSpecificity
	// template is the service+method path with its "{name:type}"
	// placeholders left in ("/hooks/{token:string}"), what a
	// ResponseObserver gets as ResponseInfo.Route.
	template string
}

// routeSpecificity ranks routes from most to least specific, so e.g. an
// exact path always wins over a catch-all that also happens to match the
// same request, regardless of registration order.
type routeSpecificity struct {
	// hasCatchAll is true if the route has a "path"-typed (multi-segment
	// wildcard) parameter anywhere in its service or method path.
	hasCatchAll bool
	// paramCount is the total number of path parameters, of any type.
	paramCount int
	// literalLength is the number of non-parameter characters in the
	// combined service+method path template.
	literalLength int
	// fullPath is the final, always-available tie-breaker: a route table
	// built from the same set of routes sorts identically no matter what
	// order they were registered in.
	fullPath string
}

// less reports whether a should be tried before b: no catch-all beats any
// catch-all; among those, fewer params beats more; among those, a longer
// literal footprint beats a shorter one; ties are broken by comparing the
// path templates themselves, so the order never depends on registration or
// map-iteration order.
func (a routeSpecificity) less(b routeSpecificity) bool {
	if a.hasCatchAll != b.hasCatchAll {
		return !a.hasCatchAll
	}
	if a.paramCount != b.paramCount {
		return a.paramCount < b.paramCount
	}
	if a.literalLength != b.literalLength {
		return a.literalLength > b.literalLength
	}
	return a.fullPath < b.fullPath
}

func newRoute(serviceHandler ServiceHandler, method MethodHolder) (route, error) {

	serviceUri := serviceHandler.Path().Regex
	if !strings.HasPrefix(serviceUri, "/") {
		serviceUri = "/" + serviceUri
	}

	methodUri := serviceUri
	if method.Path.Regex != "" {
		if !strings.HasSuffix(methodUri, "/") && !strings.HasPrefix(method.Path.Regex, "/") {
			methodUri += "/"
		}
		methodUri += method.Path.Regex
	}

	pattern, err := regexp.Compile("^" + methodUri + "$")
	if err != nil {
		return route{}, err
	}

	hasCatchAll := false
	paramCount := 0
	for _, param := range serviceHandler.Path().Params {
		paramCount++
		if param.Type == "path" {
			hasCatchAll = true
		}
	}
	for _, param := range method.Path.Params {
		paramCount++
		if param.Type == "path" {
			hasCatchAll = true
		}
	}

	template := serviceHandler.Path().Path
	if !strings.HasPrefix(template, "/") {
		template = "/" + template
	}
	if method.Path.Path != "" {
		if !strings.HasSuffix(template, "/") && !strings.HasPrefix(method.Path.Path, "/") {
			template += "/"
		}
		template += method.Path.Path
	}

	return route{
		serviceHandler: serviceHandler,
		method:         method,
		pattern:        pattern,
		template:       template,
		specificity: routeSpecificity{
			hasCatchAll:   hasCatchAll,
			paramCount:    paramCount,
			literalLength: literalPathLength(serviceHandler.Path().Path) + literalPathLength(method.Path.Path),
			fullPath:      methodUri,
		},
	}, nil
}

// literalPathLength counts the characters in rawPath outside of
// "{name:type}" placeholders.
func literalPathLength(rawPath string) int {
	total := 0
	rest := rawPath
	for {
		start := strings.Index(rest, "{")
		if start == -1 {
			total += len(rest)
			break
		}
		total += start
		end := strings.Index(rest[start:], "}")
		if end == -1 {
			total += len(rest) - start
			break
		}
		rest = rest[start+end+1:]
	}
	return total
}
