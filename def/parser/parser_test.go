package parser

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	def "github.com/netgarden/rrpc/def"
)

// TestParse_SameFileIncludedFromTwoDifferentSources is the regression
// test for a real deadlock: when the same file is "include"d from two
// different sources in the same module, and the second (already-
// processed) copy happens to be the last item still queued, Parse used
// to hang forever on `range toProcess` — the "already processed" branch
// skipped the check that closes the channel once nothing else is queued.
// Reproduced here with a small module: two struct files (a.rrpc, b.rrpc)
// both including the same shared.rrpc.
func TestParse_SameFileIncludedFromTwoDifferentSources(t *testing.T) {
	dir := t.TempDir()

	mustWrite(t, filepath.Join(dir, "shared.rrpc"), `
struct Shared {
    value string
}
`)
	mustWrite(t, filepath.Join(dir, "a.rrpc"), `
include "shared.rrpc"

struct A {
    shared Shared
}
`)
	mustWrite(t, filepath.Join(dir, "b.rrpc"), `
include "shared.rrpc"

struct B {
    shared Shared
}
`)
	mustWrite(t, filepath.Join(dir, "module.rrpc"), `
name = "test"
version = "0.0.1"
option go_package = "test"

include "a.rrpc"
include "b.rrpc"
`)

	done := make(chan error, 1)
	go func() {
		_, err := New().Parse(filepath.Join(dir, "module.rrpc"))
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Parse deadlocked: a file included from two different sources in the same module never let the process loop terminate")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func parseModule(t *testing.T, body string) error {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "module.rrpc"), "name = \"test\"\nversion = \"0.0.1\"\noption go_package = \"test\"\n\n"+body)
	_, err := New().Parse(filepath.Join(dir, "module.rrpc"))
	return err
}

func TestParse_Comments(t *testing.T) {
	err := parseModule(t, `
# leading comment
struct A {
    # own-line comment inside a struct
    id   string # trailing comment
    name string
}

# comment before a service
@Path("/a")
service S {
    # comment inside a service
    @method("POST")
    get(request A) (A)
}
`)
	if err != nil {
		t.Fatalf("comments should be ignored, got: %v", err)
	}
}

// Unsupported syntax must give an error, not crash the parser.
func TestParse_UnsupportedCharacterIsAnErrorNotAPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("parser panicked: %v", r)
		}
	}()
	if err := parseModule(t, "struct A {\n    tag? string\n}\n"); err == nil {
		t.Fatal("expected an error for '?' in a field name")
	}
}

func parseStruct(t *testing.T, fields string) (map[string]bool, error) {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "module.rrpc"),
		"name = \"test\"\nversion = \"0.0.1\"\noption go_package = \"test\"\n\nstruct A {\n"+fields+"\n}\n")
	mod, err := New().Parse(filepath.Join(dir, "module.rrpc"))
	if err != nil {
		return nil, err
	}
	got := map[string]bool{}
	for _, s := range mod.Structs {
		if s.Name == "A" {
			for _, f := range s.Fields {
				got[f.Name] = f.Optional
			}
		}
	}
	return got, nil
}

func TestParse_OptionalAnnotation(t *testing.T) {
	cases := map[string]string{
		"same line":         "    id string\n    @optional tag string\n    n int",
		"own line":          "    id string\n    @optional\n    tag string\n    n int",
		"blank line + note": "    id string\n\n    # explain\n    @optional\n\n    tag string\n    n int",
		"first field":       "    @optional tag string\n    id string\n    n int",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := parseStruct(t, body)
			if err != nil {
				t.Fatal(err)
			}
			if !got["tag"] {
				t.Errorf("tag should be optional: %v", got)
			}
			if got["id"] || got["n"] {
				t.Errorf("only tag should be optional: %v", got)
			}
		})
	}

	// Last field, directly before the closing brace.
	got, err := parseStruct(t, "    id string\n    @optional tag string")
	if err != nil || !got["tag"] || got["id"] {
		t.Errorf("last field: %v, %v", got, err)
	}
	// Types with brackets and struct-typed fields work too.
	got, err = parseStruct(t, "    @optional tags []string\n    @optional labels map[string]string\n    @optional other A")
	if err != nil || !got["tags"] || !got["labels"] || !got["other"] {
		t.Errorf("complex types: %v, %v", got, err)
	}
}

func TestParse_OptionalAnnotationErrors(t *testing.T) {
	cases := map[string]string{
		"unknown annotation": "    @optonal tag string",
		"with argument":      "    @optional(\"x\") tag string",
		"duplicate":          "    @optional @optional tag string",
		"dangling":           "    id string\n    @optional",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panicked instead of returning an error: %v", r)
				}
			}()
			if _, err := parseStruct(t, body); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// parseService parses one service named "S" (with a shared "Info" struct
// available for multipart methods) and returns its methods by name, for
// inspecting the fields the processor computed (ContentType,
// ResponseContentType, BytesPartContentType).
func parseService(t *testing.T, serviceBody string) (map[string]*def.Method, error) {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "module.rrpc"),
		"name = \"test\"\nversion = \"0.0.1\"\noption go_package = \"test\"\n\n"+
			"struct Info {\n    n int\n}\n\n@Path(\"/s\")\nservice S {\n"+serviceBody+"\n}\n")
	mod, err := New().Parse(filepath.Join(dir, "module.rrpc"))
	if err != nil {
		return nil, err
	}
	got := map[string]*def.Method{}
	for _, s := range mod.Services {
		if s.Name == "S" {
			for _, m := range s.Methods {
				got[m.Name] = m
			}
		}
	}
	return got, nil
}

func TestParse_RequestContentType_SingleBytes(t *testing.T) {
	methods, err := parseService(t, `
    @method("POST")
    @RequestContentType("  Application/JSON  ")
    upload(data bytes)
`)
	if err != nil {
		t.Fatal(err)
	}
	m := methods["upload"]
	if m == nil {
		t.Fatal("method not found")
	}
	if m.ContentType != "application/json" {
		t.Errorf("ContentType = %q, want application/json (trimmed and lowercased)", m.ContentType)
	}
	if m.BytesPartContentType != "" {
		t.Errorf("BytesPartContentType = %q, want empty for a non-multipart method", m.BytesPartContentType)
	}
}

func TestParse_RequestContentType_Multipart(t *testing.T) {
	methods, err := parseService(t, `
    @method("POST")
    @RequestContentType("application/pdf")
    upload(info Info, data bytes)
`)
	if err != nil {
		t.Fatal(err)
	}
	m := methods["upload"]
	if m == nil {
		t.Fatal("method not found")
	}
	if m.ContentType != "multipart/form-data" {
		t.Errorf("ContentType = %q, want multipart/form-data (the envelope, untouched)", m.ContentType)
	}
	if m.BytesPartContentType != "application/pdf" {
		t.Errorf("BytesPartContentType = %q, want application/pdf (the part, overridden)", m.BytesPartContentType)
	}
}

func TestParse_RequestContentType_DefaultsWhenUnannotated(t *testing.T) {
	methods, err := parseService(t, `
    @method("POST")
    single(data bytes)

    @method("POST")
    multipart(info Info, data bytes)
`)
	if err != nil {
		t.Fatal(err)
	}
	if got := methods["single"].ContentType; got != "application/octet-stream" {
		t.Errorf("single.ContentType = %q, want application/octet-stream", got)
	}
	if got := methods["multipart"].BytesPartContentType; got != "application/octet-stream" {
		t.Errorf("multipart.BytesPartContentType = %q, want application/octet-stream", got)
	}
}

func TestParse_ResponseContentType(t *testing.T) {
	methods, err := parseService(t, `
    @method("GET")
    @ResponseContentType("application/pdf")
    download() bytes
`)
	if err != nil {
		t.Fatal(err)
	}
	if got := methods["download"].ResponseContentType; got != "application/pdf" {
		t.Errorf("ResponseContentType = %q, want application/pdf", got)
	}
}

func TestParse_RequestAndResponseContentType_SameMethod(t *testing.T) {
	methods, err := parseService(t, `
    @method("POST")
    @RequestContentType("application/json")
    @ResponseContentType("application/pdf")
    transform(data bytes) bytes
`)
	if err != nil {
		t.Fatal(err)
	}
	m := methods["transform"]
	if m == nil {
		t.Fatal("method not found")
	}
	if m.ContentType != "application/json" {
		t.Errorf("ContentType = %q, want application/json", m.ContentType)
	}
	if m.ResponseContentType != "application/pdf" {
		t.Errorf("ResponseContentType = %q, want application/pdf", m.ResponseContentType)
	}
}

func TestParse_ContentTypeAnnotationErrors(t *testing.T) {
	cases := map[string]string{
		"@RequestContentType, no bytes input": `
    @method("POST")
    @RequestContentType("application/json")
    create(info Info)
`,
		"@RequestContentType, no value": `
    @method("POST")
    @RequestContentType
    upload(data bytes)
`,
		"@ResponseContentType, no bytes output": `
    @method("POST")
    @ResponseContentType("application/json")
    create(info Info) (Info)
`,
		"@ResponseContentType, no value": `
    @method("GET")
    @ResponseContentType
    download() bytes
`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseService(t, body); err == nil {
				t.Error("expected an error")
			}
		})
	}
}
