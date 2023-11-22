# xlsxcfg

xlsxcfg load config data from excel sheets.

## Installation

Use one of the following ways:

1. Download from [release](https://github.com/das6ng/xlsxcfg/releases).

    Recommended to add it to your `PATH` env.

2. Run the following command:

    *⚠need [golang](https://go.dev/) sdk installed*

    `go install github.com/dashengyeah/xlsxcfg/bin/xlsxcfg`

## Usage

```
Usage:
  xlsxcfg [flags] [xlsx files...]
Flags:
  -c, --config string    config file (default "xlsxcfg.yaml")
      --example-config   export an example config file here
  -h, --help             help for xlsxcfg
```

Dir structure:

```
./
|- proto
|  |
|  |- example.proto
|
|- example.xlsx
|- example1.xlsx
|- example2.xlsx
|- xlsxcfg.yaml

```

`xlsxcfg.yaml`
```yaml
proto:
  files: ["proto/example.proto"]
  import_path: ["proto"]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  write_json: true
  json_indent: "  "
  write_bytes: true
```

1. full example:

    `xlsxcfg -c xlsxcfg.yaml example.xlsx`

2. if the config file is named `xlsxcfg.yaml` and is in current dir, the config file flag can be ommitted.

    `xlsxcfg example.xlsx`

3. it's allowed to specified more than one excel files in the same time.

    `xlsxcfg -c xlsxcfg.yaml example.xlsx example1.xlsx example2.xlsx`

## Config

Default config file `xlsxcfg.yaml`:

```yaml
# Protocol Buffer source config
proto:
  # proto files need to parse.
  files: ["example.proto"]
  # proto message import paths.
  import_path: ["."]
# xlsx sheet config
sheet:
  # rows that is comment, will ignore
  # in parsing
  comment_rows: [1]
  # row contains metadata
  meta_row: 2
  # rows contain config data from this
  data_row_start: 3
  # sheet name should add this suffix
  # to find the proto message.
  type_suffix: "Sheet"
  # config data rows' list name in the
  # sheet proto message.
  list_field_name: "List"
  # sheet name should add this suffix
  # to find the data row proto message.
  row_type_suffix: "SheetRow"
# output file config
output:
  # output dir
  dir: "."
  # json
  write_json: true
  json_indent: "  "
  # proto bytes
  write_bytes: true
```

## Examples

1. Basic

- Protocol Buffer code:

```protobuf
message PhoneNumber {
    int64  Region = 1;
    int64  No     = 2;
    string Ext    = 3;
}

message MemberSheetRow {
    int32                ID      = 1;
    string               Name    = 2;
    string               Address = 3;
    PhoneNumber          Phone   = 4;
    repeated string      Cities  = 5;
    repeated PhoneNumber PP      = 6;
}
message MemberSheet {
    repeated MemberSheetRow List = 1;
}
```

- Excel sheet:

![image](./doc/example-sheet.png)
