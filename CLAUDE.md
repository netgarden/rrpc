# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

RRPC is a REST RPC framework consisting of multiple Go modules. It defines RPC schemas in `.rrpc` files, generates type-safe Go and TypeScript code from those definitions, and provides an HTTP/REST server runtime.

## Module Structure

This repository is a **single git repo, multi-module Go workspace** — each subdirectory below is an independent Go module with its own `go.mod`, nested by role rather than given a flat `rrpc-xxx` name (that flat naming is a holdover from when each was its own separate repo). All modules use local `replace` directives to reference each other by relative path.

| Directory | Go module | Purpose |
|---|---|---|
| `./` (repo root) | `github.com/netgarden/rrpc` | Core HTTP server runtime library |
| `def/` | `github.com/netgarden/rrpc/def` | `.rrpc` file lexer, parser, and AST entities |
| `gen/` | `github.com/netgarden/rrpc/gen` | CLI tool (`rrpc-gen`) that orchestrates code generation |
| `gen/go/` | `github.com/netgarden/rrpc/gen/go` | Go code generator (uses `text/template`) |
| `gen/ts/` | `github.com/netgarden/rrpc/gen/ts` | TypeScript code generator (uses `text/template`) |
| `test/` | `github.com/netgarden/rrpc/test` | Example application demonstrating the framework |
| `ts/` | (npm package, no go.mod) | Vendored TypeScript runtime — the TS-side counterpart to the root Go runtime |

**Important**: since the root of this repo IS the `rrpc` module itself (not a wrapper directory), `go build ./...`, `go test ./...`, etc. from the repo root operate on the core runtime. To work on a submodule, `cd` into it first — each has its own `go.mod` and is not part of a `go.work`.

## Commands

Run from within the relevant module directory:

```bash
# Build the generator CLI
cd gen && go build ./cmd/rrpc-gen/

# Run tests (test files exist in def/lexer, root, gen/*)
go test ./...

# Run a single test
go test ./path/to/package -run TestName

# Run the test server
cd test && go run .

# Generate code from a .rrpc file
rrpc-gen --source module.rrpc --target ./output --type go
rrpc-gen --source module.rrpc --target ./output --type ts
```

`RRPC_DEVEL=true` makes `gen/go` and `gen/ts` load their templates from `./go/templates` / `./ts/templates` (relative to `gen/`, the CLI's working directory when invoked per the Makefile pattern below) instead of the embedded FS — useful when editing templates without rebuilding.

Root `Makefile` and `test/Makefile` both invoke the generator via `RRPC_GEN_DIR := $(abspath ./gen)` / `../gen` respectively, then `cd $(RRPC_GEN_DIR) && RRPC_DEVEL=true go run ./cmd/rrpc-gen/ ...` — the relative path depends on where the Makefile lives (root Makefile is a *parent* of `gen/`, `test/Makefile` is a *sibling*), so don't copy one pattern into the other without checking.

## RRPC Definition File Format

```
name = "module_name"
version = "0.0.1"
option go_package = "package_name"

include "other_module.rrpc"

struct MyRequest {
    field1 string
    field2 int
}

struct MyResponse {
    success bool
}

@path("/custom/path")
service MyService {
    @method("POST")
    @path("/{id:string}")
    myMethod(request MyRequest) (MyResponse)
}
```

Supported primitive types: `string`, `int`, `float`, `bool`, `bytes`. Fields can be optional with the `@optional` annotation (Go pointer / TS `?:`, see README) or arrays with `[]` prefix. `#` starts a line comment.

## Architecture: How Code Generation Works

1. **Lexer** (`def/lexer/`) tokenizes `.rrpc` files into typed tokens
2. **Parser** (`def/parser/parser.go`) converts tokens into a `SourceOrigin` AST
3. **Processor** (`def/parser/processor.go`) merges included sources into a unified `Module`, resolves paths and annotations
4. **Validator** (`def/parser/validator.go`) performs semantic checks (valid types, bytes parameter constraints, etc.)
5. **Generator** (`gen/go/generator.go` or `gen/ts/generator.go`) renders Go templates into output files

The core AST types are in `def/entities.go`: `Module`, `Source`, `Struct`, `Service`, `Method`, `Field`. Note `Source.Module` back-references the aggregated `Module` — per-source templates (e.g. `gen/go/templates/source.tpl`) must read options like `go_package` via `.Module.Options`, not `.Options`, since an `include`d source's own `Options` map is empty (options are only set on the main source file).

## Architecture: Runtime Server

`server.go` (repo root) implements `http.Handler`. `RegisterModule` flattens every (service, method) pair from the module's `Services()` map into `route.go`'s route table and keeps it sorted by specificity (see `routeSpecificity.less`); `handle()` just walks that precompiled, presorted list and takes the first regex match. This is deliberate: `Services()` is a `map[string]ServiceHandler`, whose iteration order Go randomizes per process, so routing can never rely on registration/map order — an exact path always wins over a `{name:path}` catch-all that also matches, regardless of which module or map entry either came from.

Path parameters use `{paramName:type}` syntax (e.g., `{id:string}`, `{filepath:path}`) — see `paths.go` for the two supported types (`string`: one segment; `path`: a multi-segment catch-all) and `EscapePathValue` for building URLs with a `path`-typed value client-side. `context.go` wraps each request with parsed path/query params and the raw `http.ResponseWriter`/`*http.Request`.

## Generated Code (per module)

The Go generator produces five files per `.rrpc` module:

| Generated file | Contents |
|---|---|
| `*.gen.go` | Structs matching the `.rrpc` definition |
| `*_handler.gen.go` | HTTP handler implementations |
| `rrpc.module.gen.go` | Module registration and service holder |
| `rrpc.client.gen.go` | Client for calling the service |
| `rrpc.errors.gen.go` | Module-specific error type definitions |

**Do not edit generated files** (files ending in `.gen.go` or `.gen.ts`). Regenerate them using `rrpc-gen` instead.

## Implementing a Service

After generating code, implement the generated service interface in an `impl/` subdirectory. See `test/modules/auth/impl/auth.go` for a working example. Register the implementation by calling the generated module initializer in `server.go`.

## Web Frontend (test/web/)

The test application includes a Svelte/TypeScript frontend. Build it with standard Node.js tooling (`npm install`, `npm run build`) from the `test/web/` directory.