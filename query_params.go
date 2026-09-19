package rrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// DecodeQueryParams decodes URL query parameters into a struct using JSON tags.
//
// Each value is converted according to the type of the field it lands in, not
// by guessing from its text: a string field keeps "2024", "true" or "007"
// exactly as sent, a bool field accepts anything strconv.ParseBool does, and
// numeric fields parse as numbers. A value that doesn't parse as its field's
// type is passed through unconverted so the JSON decode reports it, which
// surfaces to the caller as an error (a 400 from generated handlers). Keys
// that match no field of v (or a v that isn't a struct pointer) fall back to
// guessing numeric/boolean values from their text.
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
	fields := queryFieldTypes(v)
	m := make(map[string]interface{})
	for k, vals := range values {
		if len(vals) == 0 {
			continue
		}

		if outer, inner, ok := parseBracketKey(k); ok {
			elem := mapElemType(lookupField(fields, outer))
			nested, _ := m[outer].(map[string]interface{})
			if nested == nil {
				nested = make(map[string]interface{})
				m[outer] = nested
			}
			nested[inner] = coerceQueryValue(vals[0], elem)
			continue
		}

		m[k] = coerceQueryValue(vals[0], lookupField(fields, k))
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// EncodeQueryParams is the inverse of DecodeQueryParams: it turns a struct
// (or pointer to one) into URL query parameters using its JSON tags, so a
// value survives the round trip through DecodeQueryParams unchanged.
//
// Fields JSON would omit (nil pointers, omitempty zero values) and null
// values are left out. A map[string]V field (or a struct of scalars, which
// JSON cannot tell apart from a map) becomes one level of bracket notation
// ("filters[name]=foo"). []byte fields are sent as the base64 text JSON uses
// for them. Slices and anything nested deeper than one level have no
// query-string form and return an error instead of being silently mangled -
// the rrpc validator rejects such fields for GET methods for the same reason.
func EncodeQueryParams(v interface{}) (url.Values, error) {
	out := url.Values{}
	if v == nil {
		return out, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber() // keep numbers as written: no 1e+06 for a million
	var m map[string]interface{}
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("query params: value must encode to a JSON object: %w", err)
	}

	for k, val := range m {
		switch x := val.(type) {
		case nil:
			continue
		case map[string]interface{}:
			for sub, subval := range x {
				s, ok, err := queryScalar(subval)
				if err != nil {
					return nil, fmt.Errorf("query params: %s[%s]: %w", k, sub, err)
				}
				if ok {
					out.Set(k+"["+sub+"]", s)
				}
			}
		default:
			s, ok, err := queryScalar(val)
			if err != nil {
				return nil, fmt.Errorf("query params: %s: %w", k, err)
			}
			if ok {
				out.Set(k, s)
			}
		}
	}
	return out, nil
}

// queryScalar renders a decoded JSON scalar as its query-string text.
// ok is false for null, which is omitted.
func queryScalar(v interface{}) (s string, ok bool, err error) {
	switch x := v.(type) {
	case nil:
		return "", false, nil
	case string:
		return x, true, nil
	case bool:
		return strconv.FormatBool(x), true, nil
	case json.Number:
		return x.String(), true, nil
	default:
		return "", false, fmt.Errorf("%T has no query-string form (only scalars and map[string]scalar do)", v)
	}
}

// coerceQueryValue converts a raw query string value for a target of type t
// (nil when unknown). See DecodeQueryParams for the rules.
func coerceQueryValue(val string, t reflect.Type) interface{} {
	if t == nil {
		return guessQueryValue(val)
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return val
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 { // []byte: base64 text, not a number
			return val
		}
	case reflect.Bool:
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			return n
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if n, err := strconv.ParseUint(val, 10, 64); err == nil {
			return n
		}
	case reflect.Float32, reflect.Float64:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	case reflect.Interface:
		return guessQueryValue(val)
	}
	// Doesn't parse as the field's type (or the type has no scalar form):
	// hand the raw text to encoding/json so it reports the mismatch.
	return val
}

// guessQueryValue parses a raw query string value to its native numeric or
// boolean type where possible, falling back to the raw string. Used only when
// the destination type is unknown.
func guessQueryValue(val string) interface{} {
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

// queryFieldTypes maps the JSON key of each field of the struct v points to
// onto its Go type, flattening embedded structs the way encoding/json does.
// It returns nil when v isn't a pointer to a struct.
func queryFieldTypes(v interface{}) map[string]reflect.Type {
	t := reflect.TypeOf(v)
	if t == nil || t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return nil
	}
	fields := map[string]reflect.Type{}
	collectFieldTypes(t.Elem(), fields)
	return fields
}

func collectFieldTypes(t reflect.Type, into map[string]reflect.Type) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag, hasTag := f.Tag.Lookup("json")
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" && !strings.Contains(tag, ",") {
			continue
		}
		if f.Anonymous && name == "" {
			ft := f.Type
			for ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				collectFieldTypes(ft, into)
				continue
			}
		}
		if !f.IsExported() {
			continue
		}
		if !hasTag || name == "" {
			name = f.Name
		}
		if _, exists := into[name]; !exists {
			into[name] = f.Type
		}
	}
}

// lookupField finds a field's type by JSON key: exact match first, then
// case-insensitively, since encoding/json matches keys that way too.
func lookupField(fields map[string]reflect.Type, key string) reflect.Type {
	if t, ok := fields[key]; ok {
		return t
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names) // deterministic when two names differ only by case
	for _, name := range names {
		if strings.EqualFold(name, key) {
			return fields[name]
		}
	}
	return nil
}

// mapElemType returns the value type of a map field, or nil if t isn't a map.
func mapElemType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Map {
		return nil
	}
	return t.Elem()
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
