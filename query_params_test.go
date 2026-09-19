package rrpc

import (
	"net/url"
	"reflect"
	"testing"
)

func TestDecodeQueryParams_PlainScalars(t *testing.T) {
	type Request struct {
		Page   int    `json:"page"`
		SortBy string `json:"sortBy"`
		Active bool   `json:"active"`
	}

	values := url.Values{"page": {"2"}, "sortBy": {"name"}, "active": {"true"}}
	var req Request
	if err := DecodeQueryParams(values, &req); err != nil {
		t.Fatalf("DecodeQueryParams: %v", err)
	}
	if req.Page != 2 || req.SortBy != "name" || !req.Active {
		t.Errorf("got %+v", req)
	}
}

func TestDecodeQueryParams_BracketNotationDecodesIntoMap(t *testing.T) {
	type Request struct {
		Filters map[string]string `json:"filters"`
	}

	values := url.Values{
		"filters[name]":     {"foo"},
		"filters[hostname]": {"bar"},
	}
	var req Request
	if err := DecodeQueryParams(values, &req); err != nil {
		t.Fatalf("DecodeQueryParams: %v", err)
	}
	if req.Filters["name"] != "foo" || req.Filters["hostname"] != "bar" {
		t.Errorf("got %+v", req.Filters)
	}
	if len(req.Filters) != 2 {
		t.Errorf("expected exactly 2 filter entries, got %d: %+v", len(req.Filters), req.Filters)
	}
}

func TestDecodeQueryParams_MixedScalarAndMapFields(t *testing.T) {
	type Request struct {
		Page    int               `json:"page"`
		SortBy  string            `json:"sortBy"`
		Filters map[string]string `json:"filters"`
	}

	values := url.Values{
		"page":          {"1"},
		"sortBy":        {"hostname"},
		"filters[name]": {"web"},
	}
	var req Request
	if err := DecodeQueryParams(values, &req); err != nil {
		t.Fatalf("DecodeQueryParams: %v", err)
	}
	if req.Page != 1 || req.SortBy != "hostname" {
		t.Errorf("scalar fields wrong: %+v", req)
	}
	if req.Filters["name"] != "web" {
		t.Errorf("map field wrong: %+v", req.Filters)
	}
}

func TestDecodeQueryParams_NoBracketKeys_UnchangedBehavior(t *testing.T) {
	type Request struct {
		Name string `json:"name"`
	}

	values := url.Values{"name": {"plain"}}
	var req Request
	if err := DecodeQueryParams(values, &req); err != nil {
		t.Fatalf("DecodeQueryParams: %v", err)
	}
	if req.Name != "plain" {
		t.Errorf("got %+v", req)
	}
}

func TestDecodeQueryParams_EmptyMap(t *testing.T) {
	type Request struct {
		Filters map[string]string `json:"filters"`
	}

	var req Request
	if err := DecodeQueryParams(url.Values{}, &req); err != nil {
		t.Fatalf("DecodeQueryParams: %v", err)
	}
	if len(req.Filters) != 0 {
		t.Errorf("expected no filters, got %+v", req.Filters)
	}
}

func TestParseBracketKey(t *testing.T) {
	cases := []struct {
		key         string
		wantOuter   string
		wantInner   string
		wantMatched bool
	}{
		{"filters[name]", "filters", "name", true},
		{"page", "", "", false},
		{"filters[]", "", "", false},
		{"[name]", "", "", false},
		{"filters[a][b]", "filters", "a][b", true}, // documented single-level limitation
	}
	for _, c := range cases {
		outer, inner, ok := parseBracketKey(c.key)
		if ok != c.wantMatched || outer != c.wantOuter || inner != c.wantInner {
			t.Errorf("parseBracketKey(%q) = (%q, %q, %v), want (%q, %q, %v)",
				c.key, outer, inner, ok, c.wantOuter, c.wantInner, c.wantMatched)
		}
	}
}

// Optional fields are generated as pointers with omitempty: an absent query
// param must stay nil (unset), a present one must be set - including a
// zero value, which is the whole point of using pointers.
func TestDecodeQueryParams_OptionalPointerFields(t *testing.T) {
	type Request struct {
		Q     string  `json:"q"`
		Limit *int    `json:"limit,omitempty"`
		All   *bool   `json:"all,omitempty"`
		Name  *string `json:"name,omitempty"`
	}

	var absent Request
	if err := DecodeQueryParams(url.Values{"q": {"x"}}, &absent); err != nil {
		t.Fatal(err)
	}
	if absent.Limit != nil || absent.All != nil || absent.Name != nil {
		t.Errorf("absent params must stay nil: %+v", absent)
	}

	var zero Request
	if err := DecodeQueryParams(url.Values{"q": {"x"}, "limit": {"0"}, "all": {"false"}}, &zero); err != nil {
		t.Fatal(err)
	}
	if zero.Limit == nil || *zero.Limit != 0 || zero.All == nil || *zero.All {
		t.Errorf("explicit zero values must be set, not nil: %+v", zero)
	}
}

// ---- type-directed decoding ----

// The destination field's type decides how a value is read, never its text:
// these strings all look like other types.
func TestDecodeQueryParams_StringFieldsKeepTheirText(t *testing.T) {
	type Request struct {
		Q    string  `json:"q"`
		Ptr  *string `json:"ptr,omitempty"`
		Data []byte  `json:"data"`
	}
	for _, v := range []string{"2024", "true", "false", "1.5", "007", "1e3", "-0", "null", " 5 ", ""} {
		var r Request
		if err := DecodeQueryParams(url.Values{"q": {v}, "ptr": {v}}, &r); err != nil {
			t.Errorf("%q: %v", v, err)
			continue
		}
		if r.Q != v {
			t.Errorf("q = %q, want %q", r.Q, v)
		}
		if r.Ptr == nil || *r.Ptr != v {
			t.Errorf("ptr = %v, want %q", r.Ptr, v)
		}
	}
	// []byte is base64 text; a numeric-looking value must not be read as a number.
	var r Request
	if err := DecodeQueryParams(url.Values{"data": {"MTIz"}}, &r); err != nil || string(r.Data) != "123" {
		t.Errorf("bytes = %q, %v", r.Data, err)
	}
}

func TestDecodeQueryParams_TypedScalars(t *testing.T) {
	type Request struct {
		I  int     `json:"i"`
		U  uint8   `json:"u"`
		F  float64 `json:"f"`
		B  bool    `json:"b"`
		PI *int    `json:"pi,omitempty"`
	}
	var r Request
	err := DecodeQueryParams(url.Values{"i": {"-42"}, "u": {"200"}, "f": {"2.5"}, "b": {"1"}, "pi": {"0"}}, &r)
	if err != nil {
		t.Fatal(err)
	}
	if r.I != -42 || r.U != 200 || r.F != 2.5 || !r.B || r.PI == nil || *r.PI != 0 {
		t.Errorf("got %+v", r)
	}
}

// A value that isn't valid for its field must be an error, not silently zero.
func TestDecodeQueryParams_RejectsMismatchedValues(t *testing.T) {
	type Request struct {
		I int  `json:"i"`
		U uint `json:"u"`
		B bool `json:"b"`
	}
	for _, q := range []url.Values{
		{"i": {"abc"}}, {"i": {"1.5"}}, {"u": {"-1"}}, {"b": {"maybe"}}, {"i": {""}},
	} {
		var r Request
		if err := DecodeQueryParams(q, &r); err == nil {
			t.Errorf("%v: expected an error, got %+v", q, r)
		}
	}
}

func TestDecodeQueryParams_MapValuesFollowTheElementType(t *testing.T) {
	type Request struct {
		Strs map[string]string `json:"strs"`
		Ints map[string]int    `json:"ints"`
	}
	var r Request
	err := DecodeQueryParams(url.Values{"strs[a]": {"2024"}, "strs[b]": {"true"}, "ints[n]": {"7"}}, &r)
	if err != nil {
		t.Fatal(err)
	}
	if r.Strs["a"] != "2024" || r.Strs["b"] != "true" || r.Ints["n"] != 7 {
		t.Errorf("got %+v", r)
	}
}

func TestDecodeQueryParams_KeyMatchingAndEmbedding(t *testing.T) {
	type Base struct {
		Page int `json:"page"`
	}
	type Request struct {
		Base
		Name string `json:"name"`
		Skip string `json:"-"`
	}
	var r Request
	// "NAME" matches "name" case-insensitively, as encoding/json does.
	if err := DecodeQueryParams(url.Values{"page": {"3"}, "NAME": {"2024"}}, &r); err != nil {
		t.Fatal(err)
	}
	if r.Page != 3 || r.Name != "2024" {
		t.Errorf("got %+v", r)
	}
}

// Without a struct pointer to consult, values fall back to guessing.
func TestDecodeQueryParams_UnknownDestinationFallsBackToGuessing(t *testing.T) {
	m := map[string]interface{}{}
	if err := DecodeQueryParams(url.Values{"n": {"5"}, "b": {"true"}, "s": {"x"}}, &m); err != nil {
		t.Fatal(err)
	}
	if m["n"] != float64(5) || m["b"] != true || m["s"] != "x" {
		t.Errorf("got %#v", m)
	}
}

// ---- encoding ----

func TestEncodeQueryParams(t *testing.T) {
	type Request struct {
		Q       string            `json:"q"`
		N       int               `json:"n"`
		Big     int64             `json:"big"`
		F       float64           `json:"f"`
		On      bool              `json:"on"`
		Opt     *string           `json:"opt,omitempty"`
		Missing *int              `json:"missing,omitempty"`
		Empty   string            `json:"empty,omitempty"`
		Filters map[string]string `json:"filters"`
		Hidden  string            `json:"-"`
	}
	opt := "x"
	got, err := EncodeQueryParams(&Request{
		Q: "a b&c=d", N: -3, Big: 1000000, F: 0.5, On: true, Opt: &opt,
		Filters: map[string]string{"name": "foo"}, Hidden: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"q": {"a b&c=d"}, "n": {"-3"}, "big": {"1000000"}, "f": {"0.5"}, "on": {"true"},
		"opt": {"x"}, "filters[name]": {"foo"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, w := range want {
		if len(got[k]) != 1 || got[k][0] != w[0] {
			t.Errorf("%s = %v, want %v", k, got[k], w)
		}
	}
	// A nil pointer, an omitempty zero and a `json:"-"` field are all left out.
	for _, k := range []string{"missing", "empty", "Hidden"} {
		if _, present := got[k]; present {
			t.Errorf("%q must be omitted", k)
		}
	}
	// An explicit zero behind a pointer is a real value and is kept.
	zero := 0
	got, _ = EncodeQueryParams(&struct {
		Z *int `json:"z,omitempty"`
	}{&zero})
	if got.Get("z") != "0" {
		t.Errorf("explicit zero pointer dropped: %v", got)
	}
}

func TestEncodeQueryParams_NilAndEmpty(t *testing.T) {
	if got, err := EncodeQueryParams(nil); err != nil || len(got) != 0 {
		t.Errorf("nil: %v, %v", got, err)
	}
	var p *struct {
		A string `json:"a,omitempty"`
	}
	if got, err := EncodeQueryParams(p); err != nil || len(got) != 0 {
		t.Errorf("nil pointer: %v, %v", got, err)
	}
}

// Shapes with no query-string form are refused rather than mangled.
func TestEncodeQueryParams_UnsupportedShapes(t *testing.T) {
	for name, v := range map[string]interface{}{
		"slice": &struct {
			A []string `json:"a"`
		}{[]string{"x"}},
		"two levels": &struct {
			A map[string]map[string]string `json:"a"`
		}{map[string]map[string]string{"k": {"v": "x"}}},
		"map of slice": &struct {
			A map[string][]string `json:"a"`
		}{map[string][]string{"k": {"v"}}},
		"not an object": []int{1, 2},
	} {
		if _, err := EncodeQueryParams(v); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

// Whatever EncodeQueryParams writes, DecodeQueryParams must read back unchanged.
func TestQueryParamsRoundTrip(t *testing.T) {
	type Request struct {
		S       string            `json:"s"`
		N       int               `json:"n"`
		U       uint16            `json:"u"`
		F       float64           `json:"f"`
		B       bool              `json:"b"`
		Data    []byte            `json:"data"`
		PS      *string           `json:"ps,omitempty"`
		PI      *int              `json:"pi,omitempty"`
		PB      *bool             `json:"pb,omitempty"`
		Filters map[string]string `json:"filters"`
	}
	zero, empty, f := 0, "", false
	numeric := "2024"
	for name, in := range map[string]Request{
		"typical":   {S: "hello", N: 5, U: 7, F: 1.25, B: true},
		"zero":      {},
		"lookalike": {S: "true", PS: &numeric, N: 1},
		"pointers":  {PS: &empty, PI: &zero, PB: &f},
		"symbols":   {S: "a&b=c?d#e f+g%h/é🐴", Filters: map[string]string{"k": "1", "n a": "x&y"}},
		"bytes":     {Data: []byte{0, 1, 2, 250, 255}},
		"extremes":  {N: -9007199254740993, F: -1e-7},
	} {
		t.Run(name, func(t *testing.T) {
			q, err := EncodeQueryParams(&in)
			if err != nil {
				t.Fatal(err)
			}
			// Through a real URL, so escaping is part of the round trip.
			parsed, err := url.ParseQuery(q.Encode())
			if err != nil {
				t.Fatal(err)
			}
			var out Request
			if err := DecodeQueryParams(parsed, &out); err != nil {
				t.Fatalf("decode %q: %v", q.Encode(), err)
			}
			if !reflect.DeepEqual(in, out) {
				t.Errorf("round trip changed the value\n in: %+v\nout: %+v\nvia: %s", in, out, q.Encode())
			}
		})
	}
}

// A struct of scalars encodes like a map (JSON can't tell them apart) and the
// decoder reads that back into the struct.
func TestQueryParamsRoundTrip_NestedScalarStruct(t *testing.T) {
	type Range struct {
		From int `json:"from"`
		To   int `json:"to"`
	}
	type Request struct {
		Range Range `json:"range"`
	}
	in := Request{Range{From: 2, To: 9}}
	q, err := EncodeQueryParams(&in)
	if err != nil {
		t.Fatal(err)
	}
	if q.Get("range[from]") != "2" || q.Get("range[to]") != "9" {
		t.Fatalf("encoded as %v", q)
	}
	var out Request
	if err := DecodeQueryParams(q, &out); err != nil || out != in {
		t.Errorf("round trip = %+v, %v", out, err)
	}
}
