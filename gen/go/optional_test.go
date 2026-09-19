package rrpc_gen_go

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// structFields parses a generated file and returns, for the named struct,
// field name -> "<type expr> <tag>".
func structFields(t *testing.T, file, structName string) map[string]string {
	t.Helper()
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), file, src, 0)
	if err != nil {
		t.Fatalf("generated file doesn't parse: %v", err)
	}
	out := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != structName {
			return true
		}
		st := ts.Type.(*ast.StructType)
		for _, fld := range st.Fields.List {
			tag := ""
			if fld.Tag != nil {
				tag = fld.Tag.Value
			}
			out[fld.Names[0].Name] = string(src[fld.Type.Pos()-1:fld.Type.End()-1]) + " " + tag
		}
		return false
	})
	return out
}

func TestOptionalFieldsInGoStructs(t *testing.T) {
	out := generate(t, header+`
struct Other { v string }

struct A {
    id string
    @optional tag string
    @optional count int
    @optional flag bool
    @optional other Other
    @optional tags []string
    @optional labels map[string]string
}
`)
	got := structFields(t, filepath.Join(out, "module.gen.go"), "A")
	want := map[string]string{
		"Id":     "string `json:\"id\"`",
		"Tag":    "*string `json:\"tag,omitempty\"`",
		"Count":  "*int `json:\"count,omitempty\"`",
		"Flag":   "*bool `json:\"flag,omitempty\"`",
		"Other":  "*Other `json:\"other,omitempty\"`",
		"Tags":   "[]string `json:\"tags,omitempty\"`",
		"Labels": "map[string]string `json:\"labels,omitempty\"`",
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("field %s = %q, want %q", k, got[k], w)
		}
	}
	if len(got) != len(want) {
		t.Errorf("fields = %v", got)
	}
}
