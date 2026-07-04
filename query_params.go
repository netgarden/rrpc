package rrpc

import (
	"encoding/json"
	"net/url"
	"strconv"
)

// DecodeQueryParams decodes URL query parameters into a struct using JSON tags.
// Numeric and boolean strings are parsed to their native types before unmarshaling.
func DecodeQueryParams(values url.Values, v interface{}) error {
	m := make(map[string]interface{})
	for k, vals := range values {
		if len(vals) == 0 {
			continue
		}
		val := vals[0]
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			m[k] = n
		} else if f, err := strconv.ParseFloat(val, 64); err == nil {
			m[k] = f
		} else if b, err := strconv.ParseBool(val); err == nil {
			m[k] = b
		} else {
			m[k] = val
		}
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
