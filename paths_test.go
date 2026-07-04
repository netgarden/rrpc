package rrpc_test

import (
	"regexp"
	"testing"

	"github.com/netgarden/rrpc"
)

func TestParsePath_StringParamMatchesSingleSegmentOnly(t *testing.T) {
	path := rrpc.ParsePath("{id:string}")

	re := regexp.MustCompile("^" + path.Regex + "$")

	if !re.MatchString("abc") {
		t.Error("expected a \"string\"-typed param to match a single segment")
	}
	if re.MatchString("abc/def") {
		t.Error("expected a \"string\"-typed param not to match multiple segments")
	}
}

func TestParsePath_PathParamMatchesMultipleSegments(t *testing.T) {
	path := rrpc.ParsePath("{filepath:path}")

	re := regexp.MustCompile("^" + path.Regex + "$")

	if !re.MatchString("abc") {
		t.Error("expected a \"path\"-typed param to match a single segment too")
	}
	if !re.MatchString("assets/js/app-abc123.js") {
		t.Error("expected a \"path\"-typed param to match multiple segments")
	}
}

func TestParsePath_DefaultsToString(t *testing.T) {
	path := rrpc.ParsePath("{id}")

	if path.Params["id"].Type != "string" {
		t.Errorf("expected an untyped param to default to \"string\", got %q", path.Params["id"].Type)
	}
}

func TestParsePath_UnknownTypePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected ParsePath to panic on an unknown param type")
		}
	}()

	rrpc.ParsePath("{id:banana}")
}

func TestEscapePathValue_PreservesSeparators(t *testing.T) {
	got := rrpc.EscapePathValue("assets/js/app.js")
	want := "assets/js/app.js"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEscapePathValue_EscapesEachSegment(t *testing.T) {
	got := rrpc.EscapePathValue("a dir/a file.txt")
	want := "a%20dir/a%20file.txt"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEscapePathValue_DoesNotAllowSegmentToEscapeIntoASlash(t *testing.T) {
	// A literal "/" inside a single logical segment must stay percent-encoded
	// as %2F, not be reinterpreted as a path separator, since Split already
	// drew the segment boundaries before escaping.
	got := rrpc.EscapePathValue("weird%2Fsegment")
	want := "weird%252Fsegment"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
