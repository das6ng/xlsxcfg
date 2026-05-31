---
name: xlsxcfg
version: 1.0.0
description: |
  Convert Excel (.xlsx) config sheets to structured data using the xlsxcfg CLI.
  Use when user mentions: convert xlsx to config, xlsx to json/msgpack, protobuf config from excel,
  game config export, config data pipeline, xlsxcfg, xlsx sheet parsing,
  proto message from spreadsheet, or any task involving converting Excel sheets into
  JSON/msgpack/protobuf binary output.
metadata:
  requires:
    bins: ["xlsxcfg"]
  cliHelp: "xlsxcfg --help"
---

# xlsxcfg — Excel Config Sheet Converter

Converts `.xlsx` sheets to structured config data (JSON, msgpack, protobuf binary) using an optional
Protocol Buffer schema for validation and type enforcement.

## Quick Start

```bash
# Simplest: no proto, raw JSON output (default config)
xlsxcfg myconfig.xlsx

# With explicit config file
xlsxcfg -c config.yaml data.xlsx

# Generate example config template
xlsxcfg --example-config
```

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-c, --config` | `xlsxcfg.yaml` | Path to YAML config file |
| `--example-config` | — | Write example `xlsxcfg.yaml` to CWD, then exit |

**Flag overrides**: Any unknown `--key=value` args are deep-merged into the config.
Dot-separated paths target nested fields. Values are auto-typed (bool, int, list).

```bash
# Disable raw JSON, change data start row, set output dir
xlsxcfg -c config.yaml data.xlsx \
  --output.raw_json.enabled=false \
  --sheet.data_row_start=4 \
  --output.dir=./out

# Override proto files list (YAML array syntax)
xlsxcfg data.xlsx --proto.files=[hero.proto,item.proto]
```

## Config YAML

Full annotated template. All sections are optional except at least one output format must be enabled.

```yaml
# --- Protocol Buffer schemas ---
proto:
  enabled: true                    # false = no-proto mode (simpler, raw output only)
  files: ["example.proto"]         # .proto files to parse (required if enabled)
  import_path: ["."]               # import search paths (like protoc -I)

# --- Sheet layout ---
sheet:
  comment_rows: [1]                # 1-based row indices to skip
  meta_row: 2                      # 1-based row containing field headers
  data_row_start: 3                # 1-based; first data row (must be > meta_row)
  type_suffix: "Sheet"             # sheet "Hero" → looks up proto message "HeroSheet"
  list_field_name: "List"          # field name in wrapper message for the row list
  row_type_suffix: "SheetRow"     # sheet "Hero" → row message "HeroSheetRow"
  transpose_mark: ""               # prefix/suffix marking transposed sheets (e.g. "~")

# --- Constant lookup tables ---
constant:
  enabled: false
  skip_rows: 1                     # leading rows to skip in constant xlsx files
  comment: "#"                     # prefix marking comment rows
  ref_quote:
    l: "["                         # left delimiter for reference keys
    r: "]"                         # right delimiter
  files: []                        # xlsx files with key-value constant sheets

# --- Output ---
output:
  dir: "."                         # default output directory
  field_order: "schema"            # "schema" (proto field# order) or "source" (xlsx column order)
  raw_json:
    enabled: true                  # no proto validation
    dir: ""                        # inherits output.dir when empty
    extension: "json"
    indent: "  "                   # empty = compact JSON
  raw_msgpack:
    enabled: false                 # no proto validation
    dir: ""
    extension: "msgpack"
  json:
    enabled: false                 # proto-validated JSON
    dir: ""
    extension: "json"
    indent: "  "
  msgpack:
    enabled: false                 # proto-validated msgpack
    dir: ""
    extension: "msgpack"
  pb_bytes:
    enabled: false                 # proto binary wire format
    dir: ""
    extension: "bytes"
```

## Xlsx Sheet Layout

Every sheet follows this row structure:

```
Row 1  (comment_row):  # ignored — comments, notes, or empty
Row 2  (meta_row):      ID | Name | Phone.Region | Phone.No | Phone.Ext
Row 3+ (data_row_start):1 | Alice | 86          | 13800138 | "0001"
Row 4:                  2 | Bob   | 1           | 20955501 | "0002"
```

- **Row numbers are 1-based** in the config. Row 1 = comment, Row 2 = headers, Row 3 = first data row.
- The header row defines the field structure. See header syntax below.

## Header Syntax (meta_row)

Headers use dot-separated paths that map to proto message fields:

```
Simple field:    ID
Nested message:  Phone.Region    → maps to message.Phone.Region
Repeated field:  Tags#1          → 1-based index into repeated field
Nested+repeated: PP#1.Region     → PP is repeated PhoneNumber, index 1
Nested+repeated: PP#1.No
```

Rules:
- **Dots** (`.`) navigate into nested message fields
- **`#N`** (1-based) indexes into `repeated` message fields — each `#N` block maps one element of the repeated list
- For simple `repeated string` fields, use `Tags#1`, `Tags#2`, etc.
- Field names must match proto definitions exactly (PascalCase)

## Proto Schema Conventions

### Naming

Sheet name maps to proto messages via configurable suffixes:
- Sheet **"Hero"** + `type_suffix` → wrapper message **`HeroSheet`**
- Sheet **"Hero"** + `row_type_suffix` → row message **`HeroSheetRow`**

### Field Style

Proto fields use **PascalCase** (not standard snake_case). This is intentional.

```protobuf
syntax = "proto3";

// Shared nested message (in separate .proto if reused)
message PhoneNumber {
    int64  Region = 1;
    int64  No     = 2;
    string Ext    = 3;
}

// Row message — one per config row
message HeroSheetRow {
    int32               ID      = 1;
    string              Name    = 2;
    PhoneNumber         Phone   = 3;       // nested message
    repeated string     Tags    = 4;       // simple repeated
    repeated PhoneNumber PP      = 5;       // repeated message
}

// Wrapper message — contains the row list
message HeroSheet {
    repeated HeroSheetRow List = 1;  // field name matches list_field_name
}
```

### No-Proto Mode

When `proto.enabled: false` (the default), xlsxcfg outputs raw JSON/msgpack
using header paths as JSON keys. No `.proto` files needed.

## Transposed Sheets

Sheets whose names start or end with `transpose_mark` are parsed column-per-record
instead of row-per-record. The mark is stripped before proto type lookup.

Config: `sheet.transpose_mark: "~"` → sheet name `"Hero~"` or `"~Hero"` triggers transpose mode.

## Constant/Value Replacement

Load key-value pairs from xlsx files. Cell values wrapped in `ref_quote` delimiters
(e.g. `[Key]`) are replaced at parse time with the corresponding constant value.

```yaml
constant:
  enabled: true
  ref_quote:
    l: "["
    r: "]"
  files: ["constants.xlsx"]
```

Constant xlsx layout (two columns per sheet):
```
Row 1 (skipped): Header | Header
Row 2: HeroHP   | 100
Row 3: HeroAtk  | 25
```

Then in config data cells, use `[HeroHP]` and it becomes `100`.

## Output Format Guide

| Format | Proto Required | Use Case |
|--------|---------------|----------|
| `raw_json` | No | Quick export, no validation needed |
| `raw_msgpack` | No | Compact binary, no validation |
| `json` | Yes | Type-safe JSON for code consumption |
| `msgpack` | Yes | Compact type-safe binary |
| `pb_bytes` | Yes | Direct protobuf binary for network/storage |

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `sheet.data_row_start must be greater than sheet.meta_row` | Data rows start before or at header row | Ensure `data_row_start > meta_row` |
| `at least one output format must be enabled` | All formats disabled in config | Enable at least one output format |
| `proto.files must not be empty when proto is enabled` | Proto enabled but no files listed | Set `proto.files` or disable proto |
| `sheet.type_suffix must not be empty` | Proto enabled but suffix blank | Set `sheet.type_suffix` and `row_type_suffix` |
| `config file not found` | `-c` flag set but file missing | Fix path or remove `-c` to use defaults |
| Message not found in proto | Sheet name + suffix doesn't match any proto message | Check sheet name matches `{Name}{type_suffix}` |
| `no xlsx files specified` | No input files given | Pass xlsx file paths as positional args |

## Tips

- Start with `--example-config` to get a working template, then customize
- Use `--output.raw_json.enabled=true` with `proto.enabled: false` for quick iteration
- Override specific fields via CLI flags instead of editing YAML for one-off runs
- Use `field_order: "source"` to preserve xlsx column order in output
- `xlsxcfg` is expected to be on PATH — no build step needed
