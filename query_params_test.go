package rrpc

import (
	"net/url"
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
