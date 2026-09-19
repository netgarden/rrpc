package rrpc_gen_ts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	rrpcparser "github.com/netgarden/rrpc/def/parser"
)

func TestOptionalFieldsInTSClasses(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "module.rrpc")
	if err := os.WriteFile(src, []byte(`name = "m"
version = "0.0.1"
option go_package = "m"

struct A {
    id string
    @optional tag string
    @optional tags []string
    @optional labels map[string]int
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	mod, err := rrpcparser.New().Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := NewGenerator().Generate(mod, out); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(out)
	var ts string
	for _, e := range entries {
		b, _ := os.ReadFile(filepath.Join(out, e.Name()))
		ts += string(b)
	}
	for _, want := range []string{
		"id!: string;",
		"tag?: string;",
		"tags?: string[];",
		"labels?: Record<string, number>;",
	} {
		if !strings.Contains(ts, want) {
			t.Errorf("generated TS missing %q\n%s", want, ts)
		}
	}
}
