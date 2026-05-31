# AGENTS.md

## Project

`xlsxcfg` is a Go CLI that converts Excel (.xlsx) sheets into Protocol Buffer-defined config data (JSON, msgpack, protobuf binary). It parses `.proto` files at runtime via `protocompile` — no protoc-gen step needed for user schemas.

**Module**: `github.com/das6ng/xlsxcfg` (Go 1.26)

## Do-Not-Touch

- **`.pb.go` files** are `protoc`-generated; regenerate, never hand-edit:
  - `constant/constant.pb.go` → `make constant`
  - `tests/*.pb.go` → `make test_pb`
  - Both require `protoc` with the Go plugin installed.

## Build & Run

```sh
make xlscfg                                 # build CLI to build/xlsxcfg
go build -o build/xlsxcfg ./bin/xlsxcfg     # equivalent
./build/xlsxcfg --example-config            # print example YAML config
./build/xlsxcfg -c config.yaml file.xlsx    # run with config
```

## Tests

```sh
go test ./...                    # all tests across all packages
go test ./tests/                 # CLI integration tests
go test ./tests/flat_fields/     # single subdirectory test
go test .                        # root package tests (no-proto mode, streaming)
```

- 11 test files across 8 packages. Root package has tests (`optional_proto_test.go`, `xlsx_src_test.go`).
- Main integration suite: `tests/cli_test.go` (~1500 lines, 25+ test cases covering all output formats, flag overrides, error cases).
- Per-feature test directories under `tests/`: `flat_fields/`, `nested_structs/`, `repeated_fields/`, `edge_cases/`, `multi_sheet/`, `duplicate_sheet/`, `benchmark/`.
- Test helpers: `tests/testutil/helpers.go` — `LoadFixture()` one-call setup.
- Test xlsx fixtures in subdirectories are **gitignored** — they exist locally but aren't in the repo.
- Output artifacts (`*.bytes`, `*.json`) are gitignored.
- If you modify `.proto` files under `tests/`, run `make test_pb` before running tests.

## Architecture

### Packages

| Package | Purpose |
|---------|---------|
| `xlsxcfg` (root) | Core library — config (`config.go`), proto loading (`proto_src.go`), xlsx streaming (`xlsx_src.go`), row parsing (`xlsx_row_parser.go`), token/header parsing (`xlsx_token_reader.go`) |
| `bin/xlsxcfg/` | CLI entrypoint (cobra). Embeds default config via `//go:embed` |
| `app/` | Pipeline orchestrator — wires config→proto→xlsx→writers |
| `convert/` | `map[string]any` → `dynamicpb.Message` conversion via protoreflect |
| `writer/` | 5 output format writers (raw JSON, raw msgpack, proto JSON, proto msgpack, proto bytes) |
| `flagutil/` | Dynamic `--key=value` flag overrides deep-merged into config YAML |
| `constant/` | Standalone sub-package for loading constant/lookup tables from xlsx. Own proto schema + test |
| `tests/` | Integration tests with proto fixtures and xlsx files |

### Data Flow

1. Parse `.proto` at runtime via `protocompile` → build `TypeProvider` (`proto_src.go`)
2. Read `.xlsx` → iterate sheets **column-wise** → feed to `sheetParser`
3. `sheetParser` identifies comment/meta/data rows via config, delegates to `rowParser`
4. `rowParser` uses `tokenReader` to parse cell headers (dot-separated paths like `Phone.Region`, `#N` for list indices)
5. Maps → JSON → dynamic proto messages via `convert.MapToProto()` → writer outputs in all enabled formats

### Naming Convention

Sheet name maps to proto messages via configurable suffixes:
- `{SheetName}` + `type_suffix` (default `"Sheet"`) → wrapper message with row list
- `{SheetName}` + `row_type_suffix` (default `"SheetRow"`) → per-row message

### Proto Style

Proto fields use **PascalCase** (`Region`, `No`, `Ext`) — not standard protobuf snake_case. This is intentional; the parser matches on these exact names.

## Release

Tag-driven: push a `v*` tag → CI cross-compiles (linux/amd64, windows/amd64, darwin/amd64, darwin/arm64, `CGO_ENABLED=0`) → GitHub release.
