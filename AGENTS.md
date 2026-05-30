# AGENTS.md

## Project

`xlsxcfg` is a Go CLI that converts Excel (.xlsx) sheets into Protocol Buffer-defined config data (JSON + protobuf binary). It parses `.proto` files at runtime via `protoreflect` — no protoc-gen step is needed for user proto schemas.

**Module**: `github.com/das6ng/xlsxcfg` (Go 1.23)

## Do-Not-Touch

- **`.pb.go` files** are `protoc`-generated; regenerate, never hand-edit:
  - `constant/constant.pb.go` → `make constant`
  - `tests/example.pb.go`, `tests/deps.pb.go` → `make test_pb`
  - Both require `protoc` with the Go plugin installed.

## Build & Run

```sh
make xlscfg                                 # build CLI to build/xlsxcfg
go build -o build/xlsxcfg ./bin/xlsxcfg     # equivalent
```

## Tests

```sh
go test ./...            # all tests (currently only tests/ and constant/)
go test ./tests/         # integration tests only
```

- The only test file is `tests/constant_test.go` (tests the `constant` package). The root `xlsxcfg` package has no `_test.go` files.
- Test fixtures: `.xlsx` files + `xlsxcfg.yaml` in `tests/`. Output artifacts (`*.bytes`, `*.json`) are gitignored.
- If you modify `.proto` files under `tests/`, run `make test_pb` before running tests.

## Architecture

- **Root package** (`xlsxcfg`): core library — proto loading (`proto_src.go`), xlsx parsing, row/sheet/token parsing
- **`bin/xlsxcfg/`**: CLI entrypoint (cobra). Default config template embedded via `//go:embed xlsxcfg.yaml`
- **`constant/`**: standalone sub-package for loading constant/reference lookup tables from xlsx. Own proto schema + test.
- **`tests/`**: integration test with proto fixtures and xlsx files

### Data Flow

1. Parse `.proto` at runtime → build `TypeProvider` (`proto_src.go`)
2. Read `.xlsx` → iterate sheets **column-wise** → feed to `sheetParser`
3. `sheetParser` identifies comment/meta/data rows via config, delegates to `rowParser`
4. `rowParser` uses `tokenReader` to parse cell headers (dot-separated paths like `Phone.Region`, `#N` for list indices)
5. Maps → JSON → dynamic proto messages → `.json` and/or `.bytes` output

### Naming Convention

Sheet name maps to proto messages via configurable suffixes:
- `{SheetName}` + `type_suffix` (default `"Sheet"`) → wrapper message with row list
- `{SheetName}` + `row_type_suffix` (default `"SheetRow"`) → per-row message

### Proto Style

Proto fields use **PascalCase** (`Region`, `No`, `Ext`) — not standard protobuf snake_case. This is intentional; the parser matches on these exact names.

## Release

Tag-driven: push a `v*` tag → CI cross-compiles (linux/amd64, windows/amd64, darwin/amd64, darwin/arm64) → GitHub release.
