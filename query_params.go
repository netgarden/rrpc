package rrpc

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

// DecodeQueryParams decodes URL query parameters into a struct using JSON tags.
// Numeric and boolean strings are parsed to their native types before unmarshaling.
//
// Keys using bracket notation ("filters[name]=foo") are grouped into a
// nested map under the outer key ("filters": {"name": "foo"}) before
// unmarshaling — the encoding a map-typed field (e.g. a generated
// DatatableRequest.Filters map[string]string) arrives in, since a
// browser's URLSearchParams has no native nested-object representation.
// Only one level of nesting is supported (matching map[string]V fields);
// a key with more than one bracket pair (e.g. "a[b][c]") is not
// specially parsed.
func DecodeQueryParams(values url.Values, v interface{}) error {
	m := make(map[string]interface{})
	for k, vals := range values {
		if len(vals) == 0 {
			continue
		}
		val := coerceQueryValue(vals[0])

		if outer, inner, ok := parseBracketKey(k); ok {
			nested, _ := m[outer].(map[string]interface{})
			if nested == nil {
				nested = make(map[string]interface{})
				m[outer] = nested
			}
			nested[inner] = val
			continue
		}

		m[k] = val
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// coerceQueryValue parses a raw query string value to its native numeric
// or boolean type where possible, falling back to the raw string.
func coerceQueryValue(val string) interface{} {
	if n, err := strconv.ParseInt(val, 10, 64); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return f
	}
	if b, err := strconv.ParseBool(val); err == nil {
		return b
	}
	return val
}

// parseBracketKey reports whether key has the form "outer[inner]" (both
// non-empty), splitting it into its two parts.
func parseBracketKey(key string) (outer, inner string, ok bool) {
	open := strings.IndexByte(key, '[')
	if open <= 0 || key[len(key)-1] != ']' {
		return "", "", false
	}
	inner = key[open+1 : len(key)-1]
	if inner == "" {
		return "", "", false
	}
	return key[:open], inner, true
}
