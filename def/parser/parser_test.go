package parser

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
