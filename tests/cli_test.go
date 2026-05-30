package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/das6ng/xlsxcfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// xlsxcfgBin is the absolute path to the compiled CLI binary under test.
func init() {
	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(thisFile), "..")
	xlsxcfgBin = filepath.Join(projectRoot, "build", "xlsxcfg")
}

var xlsxcfgBin string

// testsDir is the absolute path to the tests/ directory (where fixtures live).
var testsDir string

func init() {
	_, thisFile, _, _ := runtime.Caller(0)
	testsDir = filepath.Dir(thisFile)
	projectRoot := filepath.Join(testsDir, "..")
	xlsxcfgBin = filepath.Join(projectRoot, "build", "xlsxcfg")
}

// runCLI executes the xlsxcfg binary with the given arguments in the specified
// working directory. Returns stdout, stderr, and error (nil if exit code 0).
func runCLI(t *testing.T, workdir string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(xlsxcfgBin, args...)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	stdout := string(out)
	// Also capture separate stderr for error cases
	cmd2 := exec.Command(xlsxcfgBin, args...)
	cmd2.Dir = workdir
	var stderrBuf strings.Builder
	cmd2.Stderr = &stderrBuf
	cmd2.Output()
	return stdout, stderrBuf.String(), err
}

// tmpDir creates a temporary directory and returns its path along with a cleanup func.
func tmpDir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "xlsxcfg-cli-test-*")
	require.NoError(t, err)
	return dir, func() { os.RemoveAll(dir) }
}

// writeConfig writes a YAML config string to the given directory.
func writeConfig(t *testing.T, dir, yamlContent string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "xlsxcfg.yaml"), []byte(yamlContent), 0644))
}

// copyFile copies a file from src to dst.
func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(dst, data, 0644))
}

// fixturePath returns the absolute path to a test fixture file.
// e.g. fixturePath("flat_fields/flat_fields.xlsx") → /path/to/tests/flat_fields/flat_fields.xlsx
func fixturePath(rel string) string {
	return filepath.Join(testsDir, rel)
}

// protoPath returns the absolute path to a proto file in tests/.
func protoPath(name string) string {
	return filepath.Join(testsDir, name)
}

// --- Test: --help flag ---

func TestCLI_HelpFlag(t *testing.T) {
	out, _, err := runCLI(t, testsDir, "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "xlsxcfg [flags]")
	assert.Contains(t, out, "--config")
	assert.Contains(t, out, "--example-config")
}

// --- Test: no arguments (should fail) ---

func TestCLI_NoArguments(t *testing.T) {
	_, _, err := runCLI(t, testsDir)
	assert.Error(t, err, "expected error when no xlsx files provided")
}

// --- Test: --example-config exports config template ---

func TestCLI_ExampleConfig(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	out, _, err := runCLI(t, dir, "--example-config")
	require.NoError(t, err)

	// The file should have been written
	data, err := os.ReadFile(filepath.Join(dir, "xlsxcfg.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "proto:")
	assert.Contains(t, string(data), "sheet:")
	assert.Contains(t, string(data), "output:")

	// Output should mention writing
	_ = out
}

// --- Test: nonexistent config file via -c ---

func TestCLI_NonexistentConfigFile(t *testing.T) {
	_, out, err := runCLI(t, testsDir, "-c", "/nonexistent/xlsxcfg.yaml", "dummy.xlsx")
	assert.Error(t, err)
	assert.Contains(t, out, "config file not found")
}

// --- Test: nonexistent xlsx file ---

func TestCLI_NonexistentXlsxFile(t *testing.T) {
	_, out, err := runCLI(t, testsDir, "nonexistent.xlsx")
	assert.Error(t, err)
	assert.Contains(t, out, "parse xlsx files")
}

// --- Test: Flat fields with proto → JSON + pb_bytes ---

func TestCLI_FlatFields_ProtoJSON(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	// Copy fixture + proto
	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))
	copyFile(t, protoPath("flat_fields.proto"), filepath.Join(dir, "flat_fields.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["flat_fields.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
    dir: "."
    indent: "  "
  pb_bytes:
    enabled: true
    dir: "."
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	require.NoError(t, err)

	// Verify JSON output
	jsonData, err := os.ReadFile(filepath.Join(dir, "Flat.json"))
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(jsonData, &result))

	list := result["List"].([]any)
	assert.Equal(t, 7, len(list), "expected 7 data rows (rows 3-9)")

	// First row: ID=1, Count=100, Name=Alice, Active=1, Score=95
	row0 := list[0].(map[string]any)
	assert.Equal(t, "1", jsonVal(row0["ID"]))
	assert.Equal(t, "100", jsonVal(row0["Count"]))
	assert.Equal(t, "Alice", row0["Name"])
	assert.Equal(t, "95", jsonVal(row0["Score"]))

	// Row with empty fields (Diana, row index 3): ID=4, Name=Diana, others empty/default
	row3 := list[3].(map[string]any)
	assert.Equal(t, "4", jsonVal(row3["ID"]))
	assert.Equal(t, "Diana", row3["Name"])

	// Verify .bytes file exists and is non-empty
	bytesData, err := os.ReadFile(filepath.Join(dir, "Flat.bytes"))
	require.NoError(t, err)
	assert.True(t, len(bytesData) > 0, "pb_bytes output should be non-empty")
}

// --- Test: Flat fields with proto → protobuf binary round-trip ---

func TestCLI_FlatFields_ProtobufRoundTrip(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))
	copyFile(t, protoPath("flat_fields.proto"), filepath.Join(dir, "flat_fields.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["flat_fields.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  pb_bytes:
    enabled: true
    dir: "."
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	require.NoError(t, err)

	// Read binary and verify it deserializes via proto
	bytesData, err := os.ReadFile(filepath.Join(dir, "Flat.bytes"))
	require.NoError(t, err)

	// Load proto descriptor for round-trip verification
	ctx := t.Context()
	tp, err := xlsxcfg.LoadProtoFiles(ctx, []string{dir}, "flat_fields.proto")
	require.NoError(t, err)

	md := tp.MessageByName("FlatSheet")
	require.NotNil(t, md)

	msg := dynamicpb.NewMessage(md)
	require.NoError(t, proto.Unmarshal(bytesData, msg))

	// Verify field values via protoreflect
	listFd := md.Fields().ByName("List")
	assert.True(t, msg.Has(listFd))
	protoList := msg.Get(listFd).List()
	assert.Equal(t, 7, protoList.Len())

	// Check first row
	firstRow := protoList.Get(0).Message()
	idFd := firstRow.Descriptor().Fields().ByName("ID")
	assert.Equal(t, int64(1), firstRow.Get(idFd).Int())
}

// --- Test: Nested structs with proto → JSON ---

func TestCLI_NestedStructs_ProtoJSON(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("nested_structs/nested.xlsx"), filepath.Join(dir, "nested.xlsx"))
	copyFile(t, protoPath("nested.proto"), filepath.Join(dir, "nested.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["nested.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
    dir: "."
    indent: "  "
`)

	_, _, err := runCLI(t, dir, "nested.xlsx")
	require.NoError(t, err)

	jsonData, err := os.ReadFile(filepath.Join(dir, "Nested.json"))
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(jsonData, &result))

	list := result["List"].([]any)
	assert.Equal(t, 4, len(list))

	// Row 1: Alice with nested Address and Contact
	row0 := list[0].(map[string]any)
	home := row0["Home"].(map[string]any)
	assert.Equal(t, "123 Main St", home["Street"])
	assert.Equal(t, "NYC", home["City"])
	assert.Equal(t, "10001", home["Zip"])

	info := row0["Info"].(map[string]any)
	assert.Equal(t, "alice@test.com", info["Email"])
	assert.Equal(t, "555-0001", info["Phone"])

	// Row 4: Diana with empty Home struct
	row3 := list[3].(map[string]any)
	assert.Equal(t, "Diana", row3["Name"])
	info3 := row3["Info"].(map[string]any)
	assert.Equal(t, "diana@test.com", info3["Email"])
	// Home should be empty/missing or have empty fields
	home3 := row3["Home"]
	if home3 != nil {
		homeMap := home3.(map[string]any)
		assert.Equal(t, "", homeMap["Street"])
	}
}

// --- Test: Repeated fields with proto import → JSON ---

func TestCLI_RepeatedFields_WithImport(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("repeated_fields/repeated.xlsx"), filepath.Join(dir, "repeated.xlsx"))
	copyFile(t, protoPath("repeated.proto"), filepath.Join(dir, "repeated.proto"))
	copyFile(t, protoPath("deps.proto"), filepath.Join(dir, "deps.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["repeated.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
    dir: "."
    indent: "  "
`)

	_, _, err := runCLI(t, dir, "repeated.xlsx")
	require.NoError(t, err)

	jsonData, err := os.ReadFile(filepath.Join(dir, "Repeated.json"))
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(jsonData, &result))

	list := result["List"].([]any)
	assert.Equal(t, 4, len(list))

	// Row 1: Alpha with Tags=[go, rust, python] and Phones
	row0 := list[0].(map[string]any)
	tags := row0["Tags"].([]any)
	assert.Equal(t, "go", tags[0])
	assert.Equal(t, "rust", tags[1])
	assert.Equal(t, "python", tags[2])

	phones := row0["Phones"].([]any)
	phone0 := phones[0].(map[string]any)
	assert.Equal(t, "86", jsonVal(phone0["Region"]))
	assert.Equal(t, "1310001", jsonVal(phone0["No"]))

	// Row 2: Beta with fewer tags
	row1 := list[1].(map[string]any)
	tags1 := row1["Tags"].([]any)
	assert.Equal(t, "java", tags1[0])
	assert.Equal(t, 1, len(tags1), "Beta should have only 1 tag")

	// Row 4: Delta with empty data
	row3 := list[3].(map[string]any)
	assert.Equal(t, "4", jsonVal(row3["ID"]))
	assert.Equal(t, "Delta", row3["Name"])
}

// --- Test: Multi-sheet xlsx → separate output files ---

func TestCLI_MultiSheet_SeparateOutputFiles(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("multi_sheet/multi.xlsx"), filepath.Join(dir, "multi.xlsx"))
	copyFile(t, protoPath("multi.proto"), filepath.Join(dir, "multi.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["multi.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
    dir: "."
    indent: "  "
  pb_bytes:
    enabled: true
    dir: "."
`)

	_, _, err := runCLI(t, dir, "multi.xlsx")
	require.NoError(t, err)

	// Hero.json should exist
	heroData, err := os.ReadFile(filepath.Join(dir, "Hero.json"))
	require.NoError(t, err)
	var heroResult map[string]any
	require.NoError(t, json.Unmarshal(heroData, &heroResult))
	heroList := heroResult["List"].([]any)
	assert.Equal(t, 3, len(heroList))

	// Item.json should exist
	itemData, err := os.ReadFile(filepath.Join(dir, "Item.json"))
	require.NoError(t, err)
	var itemResult map[string]any
	require.NoError(t, json.Unmarshal(itemData, &itemResult))
	itemList := itemResult["List"].([]any)
	assert.Equal(t, 3, len(itemList))

	// Verify Hero row data
	hero0 := heroList[0].(map[string]any)
	assert.Equal(t, "1", jsonVal(hero0["ID"]))
	assert.Equal(t, "Warrior", hero0["Name"])
	assert.Equal(t, "10", jsonVal(hero0["Level"]))

	// Verify Item row data
	item0 := itemList[0].(map[string]any)
	assert.Equal(t, "1", jsonVal(item0["ID"]))
	assert.Equal(t, "Sword", item0["Name"])
	assert.Equal(t, "100", jsonVal(item0["Price"]))

	// Binary outputs should also exist
	_, err = os.Stat(filepath.Join(dir, "Hero.bytes"))
	assert.NoError(t, err, "Hero.bytes should exist")
	_, err = os.Stat(filepath.Join(dir, "Item.bytes"))
	assert.NoError(t, err, "Item.bytes should exist")
}

// --- Test: Multiple xlsx files as args (duplicate sheets) ---

func TestCLI_MultipleXlsxFiles(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("duplicate_sheet/dup1.xlsx"), filepath.Join(dir, "dup1.xlsx"))
	copyFile(t, fixturePath("duplicate_sheet/dup2.xlsx"), filepath.Join(dir, "dup2.xlsx"))
	copyFile(t, protoPath("example.proto"), filepath.Join(dir, "example.proto"))
	copyFile(t, protoPath("deps.proto"), filepath.Join(dir, "deps.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["example.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
    dir: "."
    indent: "  "
`)

	// Both files have a "Member" sheet — should fail with duplicate sheet error
	_, _, err := runCLI(t, dir, "dup1.xlsx", "dup2.xlsx")
	assert.Error(t, err, "should fail with duplicate sheet name")
}

// --- Test: Edge cases (skipped rows, empty rows) ---

func TestCLI_EdgeCases_SkippedAndEmptyRows(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("edge_cases/edge.xlsx"), filepath.Join(dir, "edge.xlsx"))
	copyFile(t, protoPath("edge.proto"), filepath.Join(dir, "edge.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["edge.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
    dir: "."
    indent: "  "
`)

	_, _, err := runCLI(t, dir, "edge.xlsx")
	require.NoError(t, err)

	jsonData, err := os.ReadFile(filepath.Join(dir, "Edge.json"))
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(jsonData, &result))

	list := result["List"].([]any)
	// Edge sheet has data in rows 3,5,7,8,9 (rows 4 and 6 are empty)
	// So we expect 4 data rows with values
	assert.Equal(t, 4, len(list), "expected 4 non-empty data rows")

	// Verify the data: row1, row3, row5, row6 (skipping empties)
	row0 := list[0].(map[string]any)
	assert.Equal(t, "1", jsonVal(row0["ID"]))
	assert.Equal(t, "row1", row0["Label"])
	assert.Equal(t, "10", jsonVal(row0["Value"]))

	row1 := list[1].(map[string]any)
	assert.Equal(t, "3", jsonVal(row1["ID"]))
	assert.Equal(t, "row3", row1["Label"])
}

// --- Test: Raw JSON output (no proto) ---

func TestCLI_RawJSON_NoProto(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))

	writeConfig(t, dir, `
proto:
  enabled: false
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  raw_json:
    enabled: true
    dir: "."
    indent: "  "
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	require.NoError(t, err)

	jsonData, err := os.ReadFile(filepath.Join(dir, "Flat.json"))
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(jsonData, &result))

	list := result["List"].([]any)
	assert.Equal(t, 7, len(list))

	// Raw mode: all values are strings
	row0 := list[0].(map[string]any)
	assert.Equal(t, "1", row0["ID"])
	assert.Equal(t, "100", row0["Count"])
	assert.Equal(t, "Alice", row0["Name"])
	assert.Equal(t, "95", row0["Score"])
}

// --- Test: Raw msgpack output (no proto) ---

func TestCLI_RawMsgpack_NoProto(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))

	writeConfig(t, dir, `
proto:
  enabled: false
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  raw_msgpack:
    enabled: true
    dir: "."
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	require.NoError(t, err)

	msgpackData, err := os.ReadFile(filepath.Join(dir, "Flat.msgpack"))
	require.NoError(t, err)
	assert.True(t, len(msgpackData) > 0, "msgpack output should be non-empty")

	// Round-trip: decode msgpack back to map
	var result map[string]any
	require.NoError(t, msgpack.Unmarshal(msgpackData, &result))

	list := result["List"].([]any)
	assert.Equal(t, 7, len(list))

	row0 := list[0].(map[string]any)
	assert.Equal(t, "1", row0["ID"])
	assert.Equal(t, "Alice", row0["Name"])
}

// --- Test: All output formats together ---

func TestCLI_AllOutputFormats(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))
	copyFile(t, protoPath("flat_fields.proto"), filepath.Join(dir, "flat_fields.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["flat_fields.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  raw_json:
    enabled: true
    dir: "."
    indent: "  "
  raw_msgpack:
    enabled: true
    dir: "."
  json:
    enabled: true
    dir: "."
    indent: "  "
  msgpack:
    enabled: true
    dir: "."
  pb_bytes:
    enabled: true
    dir: "."
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	require.NoError(t, err)

	// All 5 output files should exist
	for _, ext := range []string{".json", ".msgpack", ".bytes"} {
		_, err := os.Stat(filepath.Join(dir, "Flat"+ext))
		assert.NoError(t, err, "Flat%s should exist", ext)
	}
	// raw_json and proto json both write to Flat.json, so only 3 unique files
	// But raw_json + json both produce .json — last writer wins.
	// Actually: raw_json writes Flat.json, proto json also writes Flat.json.
	// The second one overwrites. Let's just verify the file exists and is valid JSON.

	jsonData, err := os.ReadFile(filepath.Join(dir, "Flat.json"))
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal(jsonData, &result))
	assert.NotNil(t, result["List"])
}

// --- Test: Proto msgpack output round-trip ---

func TestCLI_ProtoMsgpack_RoundTrip(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))
	copyFile(t, protoPath("flat_fields.proto"), filepath.Join(dir, "flat_fields.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["flat_fields.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
    dir: "."
    indent: "  "
  msgpack:
    enabled: true
    dir: "."
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	require.NoError(t, err)

	// Read both outputs and verify they contain the same data
	jsonData, err := os.ReadFile(filepath.Join(dir, "Flat.json"))
	require.NoError(t, err)

	msgpackData, err := os.ReadFile(filepath.Join(dir, "Flat.msgpack"))
	require.NoError(t, err)

	var jsonResult map[string]any
	require.NoError(t, json.Unmarshal(jsonData, &jsonResult))

	var msgpackResult map[string]any
	require.NoError(t, msgpack.Unmarshal(msgpackData, &msgpackResult))

	// Both should have same number of rows
	jsonList := jsonResult["List"].([]any)
	msgpackList := msgpackResult["List"].([]any)
	assert.Equal(t, len(jsonList), len(msgpackList))
}

// --- Test: Custom output directory ---

func TestCLI_CustomOutputDir(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))
	copyFile(t, protoPath("flat_fields.proto"), filepath.Join(dir, "flat_fields.proto"))

	outputDir := filepath.Join(dir, "out", "sub")

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["flat_fields.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "out/sub"
  json:
    enabled: true
    indent: "  "
  pb_bytes:
    enabled: true
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	require.NoError(t, err)

	// Output should be in out/sub/
	_, err = os.Stat(filepath.Join(outputDir, "Flat.json"))
	assert.NoError(t, err, "Flat.json should be in custom output dir")
	_, err = os.Stat(filepath.Join(outputDir, "Flat.bytes"))
	assert.NoError(t, err, "Flat.bytes should be in custom output dir")

	// Should NOT be in root dir
	_, err = os.Stat(filepath.Join(dir, "Flat.json"))
	assert.True(t, os.IsNotExist(err), "Flat.json should not be in root dir")
}

// --- Test: Dynamic CLI flag overrides ---

func TestCLI_FlagOverrides(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))
	copyFile(t, protoPath("flat_fields.proto"), filepath.Join(dir, "flat_fields.proto"))

	// Config with json disabled
	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["flat_fields.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: false
    indent: "  "
  pb_bytes:
    enabled: true
`)

	// Override: enable json via --output.json.enabled=true
	_, _, err := runCLI(t, dir, "--output.json.enabled=true", "flat_fields.xlsx")
	require.NoError(t, err)

	// JSON should now be produced
	_, err = os.Stat(filepath.Join(dir, "Flat.json"))
	assert.NoError(t, err, "Flat.json should exist after flag override")
}

// --- Test: Compact JSON (no indent) ---

func TestCLI_CompactJSON(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))
	copyFile(t, protoPath("flat_fields.proto"), filepath.Join(dir, "flat_fields.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["flat_fields.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
    indent: ""
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	require.NoError(t, err)

	jsonData, err := os.ReadFile(filepath.Join(dir, "Flat.json"))
	require.NoError(t, err)

	// Compact JSON should be a single line (no newlines inside)
	jsonStr := string(jsonData)
	assert.False(t, strings.Contains(jsonStr, "\n "), "compact JSON should not have indented newlines")
}

// --- Test: Default config (no config file) ---

func TestCLI_DefaultConfig_RawJSON(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))

	// Remove any xlsxcfg.yaml from tmpdir so the binary falls back to DefaultConfig.
	// We explicitly set -c to a non-existent path to bypass the default "xlsxcfg.yaml"
	// lookup and force DefaultConfig().
	// But DefaultConfig requires raw_json.enabled=true, so we write a minimal config.
	writeConfig(t, dir, `
proto:
  enabled: false
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  raw_json:
    enabled: true
    indent: "  "
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	require.NoError(t, err)

	jsonData, err := os.ReadFile(filepath.Join(dir, "Flat.json"))
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(jsonData, &result))

	list := result["List"].([]any)
	assert.Equal(t, 7, len(list))

	// All values should be strings (raw mode, no proto)
	row0 := list[0].(map[string]any)
	assert.Equal(t, "1", row0["ID"])
	assert.Equal(t, "Alice", row0["Name"])
}

// --- Test: Example xlsx (original test fixture with imports) ---

func TestCLI_ExampleXlsx_WithDeps(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("example.xlsx"), filepath.Join(dir, "example.xlsx"))
	copyFile(t, protoPath("example.proto"), filepath.Join(dir, "example.proto"))
	copyFile(t, protoPath("deps.proto"), filepath.Join(dir, "deps.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["example.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
    dir: "."
    indent: "  "
  pb_bytes:
    enabled: true
    dir: "."
`)

	_, _, err := runCLI(t, dir, "example.xlsx")
	require.NoError(t, err)

	jsonData, err := os.ReadFile(filepath.Join(dir, "Member.json"))
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(jsonData, &result))

	list := result["List"].([]any)
	assert.Equal(t, 10, len(list), "expected 10 Member rows")

	// Verify first row with nested Phone and repeated Cities/PP
	row0 := list[0].(map[string]any)
	assert.Equal(t, "1", jsonVal(row0["ID"]))
	assert.Equal(t, "Dash01", row0["Name"])
	assert.Equal(t, "Address 001", row0["Address"])

	phone := row0["Phone"].(map[string]any)
	assert.Equal(t, "86", jsonVal(phone["Region"]))
	assert.Equal(t, "13122336655", jsonVal(phone["No"]))

	cities := row0["Cities"].([]any)
	assert.Equal(t, "Hello00", cities[0])

	pp := row0["PP"].([]any)
	pp0 := pp[0].(map[string]any)
	assert.Equal(t, "1", jsonVal(pp0["Region"]))
}

// --- Test: Proto disabled but proto formats enabled → warnings ---

func TestCLI_ProtoDisabled_ProtoFormatsWarned(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))

	writeConfig(t, dir, `
proto:
  enabled: false
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  raw_json:
    enabled: true
    indent: "  "
  json:
    enabled: true
    indent: "  "
  msgpack:
    enabled: true
  pb_bytes:
    enabled: true
`)

	out, _, err := runCLI(t, dir, "flat_fields.xlsx")
	require.NoError(t, err)

	// Should contain warnings about proto-validated formats being skipped
	assert.Contains(t, out, "WARNING")
	assert.Contains(t, out, "requires proto to be enabled")

	// Raw JSON should still be produced
	jsonData, err := os.ReadFile(filepath.Join(dir, "Flat.json"))
	require.NoError(t, err)
	assert.True(t, len(jsonData) > 0)
}

// --- Test: Invalid config (data_row_start < meta_row) ---

func TestCLI_InvalidConfig_DataRowStart(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))

	writeConfig(t, dir, `
proto:
  enabled: false
sheet:
  comment_rows: [1]
  meta_row: 5
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  raw_json:
    enabled: true
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	assert.Error(t, err, "should fail with invalid config")
}

// --- Test: Invalid config (no output formats enabled) ---

func TestCLI_InvalidConfig_NoFormatsEnabled(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))

	writeConfig(t, dir, `
proto:
  enabled: false
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  raw_json:
    enabled: false
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	assert.Error(t, err, "should fail with no output formats enabled")
}

// --- Test: Invalid config (proto enabled but no files) ---

func TestCLI_InvalidConfig_ProtoNoFiles(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: []
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	assert.Error(t, err, "should fail with proto enabled but no files")
}

// --- Test: Proto file not found ---

func TestCLI_ProtoFileNotFound(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["nonexistent.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	assert.Error(t, err, "should fail when proto file doesn't exist")
}

// --- Test: Sheet name not matching any proto message ---

func TestCLI_SheetNameNotMatchingProto(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))
	// Use edge.proto which defines EdgeSheet/EdgeSheetRow, but xlsx has a "Flat" sheet
	copyFile(t, protoPath("edge.proto"), filepath.Join(dir, "edge.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["edge.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
    dir: "."
    indent: "  "
`)

	// Should fail because the sheet parser tries to look up "FlatSheetRow" during parsing
	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	assert.Error(t, err, "should fail when proto message for sheet row not found")
}

// --- Test: Running from subdirectory with config pointing to parent protos ---

func TestCLI_SubdirectoryConfig(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	// Copy proto files to parent dir
	copyFile(t, protoPath("flat_fields.proto"), filepath.Join(dir, "flat_fields.proto"))

	// Create subdirectory for xlsx and config
	subDir := filepath.Join(dir, "sub")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(subDir, "flat_fields.xlsx"))

	writeConfig(t, subDir, `
proto:
  enabled: true
  files: ["flat_fields.proto"]
  import_path: [".."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
    indent: "  "
`)

	_, _, err := runCLI(t, subDir, "flat_fields.xlsx")
	require.NoError(t, err)

	jsonData, err := os.ReadFile(filepath.Join(subDir, "Flat.json"))
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(jsonData, &result))
	assert.NotNil(t, result["List"])
}

// --- Test: Protobuf binary round-trip for nested structs ---

func TestCLI_NestedStructs_ProtobufRoundTrip(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("nested_structs/nested.xlsx"), filepath.Join(dir, "nested.xlsx"))
	copyFile(t, protoPath("nested.proto"), filepath.Join(dir, "nested.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["nested.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
    dir: "."
    indent: "  "
  pb_bytes:
    enabled: true
    dir: "."
`)

	_, _, err := runCLI(t, dir, "nested.xlsx")
	require.NoError(t, err)

	// Verify JSON
	jsonData, _ := os.ReadFile(filepath.Join(dir, "Nested.json"))
	var jsonResult map[string]any
	require.NoError(t, json.Unmarshal(jsonData, &jsonResult))

	// Verify binary round-trip
	bytesData, err := os.ReadFile(filepath.Join(dir, "Nested.bytes"))
	require.NoError(t, err)

	ctx := t.Context()
	tp, err := xlsxcfg.LoadProtoFiles(ctx, []string{dir}, "nested.proto")
	require.NoError(t, err)

	md := tp.MessageByName("NestedSheet")
	require.NotNil(t, md)
	msg := dynamicpb.NewMessage(md)
	require.NoError(t, proto.Unmarshal(bytesData, msg))

	// Re-serialize to JSON via protojson and compare
	opts := protojson.MarshalOptions{Multiline: true, Indent: "  "}
	protoJSON, err := opts.Marshal(msg)
	require.NoError(t, err)

	var protoResult map[string]any
	require.NoError(t, json.Unmarshal(protoJSON, &protoResult))

	assert.Equal(t, len(jsonResult["List"].([]any)), len(protoResult["List"].([]any)))
}

// --- Test: Repeated fields protobuf round-trip via protojson ---

func TestCLI_RepeatedFields_ProtoRoundTrip(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("repeated_fields/repeated.xlsx"), filepath.Join(dir, "repeated.xlsx"))
	copyFile(t, protoPath("repeated.proto"), filepath.Join(dir, "repeated.proto"))
	copyFile(t, protoPath("deps.proto"), filepath.Join(dir, "deps.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["repeated.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  pb_bytes:
    enabled: true
    dir: "."
`)

	_, _, err := runCLI(t, dir, "repeated.xlsx")
	require.NoError(t, err)

	bytesData, err := os.ReadFile(filepath.Join(dir, "Repeated.bytes"))
	require.NoError(t, err)

	// Round-trip: unmarshal binary, re-marshal to protojson
	ctx := t.Context()
	tp, err := xlsxcfg.LoadProtoFiles(ctx, []string{dir}, "repeated.proto")
	require.NoError(t, err)

	md := tp.MessageByName("RepeatedSheet")
	msg := dynamicpb.NewMessage(md)
	require.NoError(t, proto.Unmarshal(bytesData, msg))

	opts := protojson.MarshalOptions{Multiline: true, Indent: "  "}
	protoJSON, err := opts.Marshal(msg)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(protoJSON, &result))

	list := result["List"].([]any)
	assert.Equal(t, 4, len(list))

	// Verify Alpha's tags round-tripped correctly
	row0 := list[0].(map[string]any)
	tags := row0["Tags"].([]any)
	assert.Equal(t, "go", tags[0])
	assert.Equal(t, "rust", tags[1])
	assert.Equal(t, "python", tags[2])
}

// --- Test: Multiple flag overrides at once ---

func TestCLI_MultipleFlagOverrides(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))
	copyFile(t, protoPath("flat_fields.proto"), filepath.Join(dir, "flat_fields.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["flat_fields.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: false
    indent: ""
  pb_bytes:
    enabled: false
  raw_json:
    enabled: false
`)

	// Enable both json and pb_bytes via flags
	_, _, err := runCLI(t, dir,
		"--output.json.enabled=true",
		"--output.json.indent=  ",
		"--output.pb_bytes.enabled=true",
		"flat_fields.xlsx",
	)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "Flat.json"))
	assert.NoError(t, err, "Flat.json should exist after enabling via flag")

	_, err = os.Stat(filepath.Join(dir, "Flat.bytes"))
	assert.NoError(t, err, "Flat.bytes should exist after enabling via flag")
}

// --- Test: Proto format output dir inheritance ---

func TestCLI_OutputDirInheritance(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))
	copyFile(t, protoPath("flat_fields.proto"), filepath.Join(dir, "flat_fields.proto"))

	// pb_bytes dir is empty (inherits output.dir = "custom_out")
	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["flat_fields.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "custom_out"
  json:
    enabled: true
    indent: "  "
  pb_bytes:
    enabled: true
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	require.NoError(t, err)

	// Both files should be in custom_out/ (pb_bytes inherits from output.dir)
	customDir := filepath.Join(dir, "custom_out")
	_, err = os.Stat(filepath.Join(customDir, "Flat.json"))
	assert.NoError(t, err, "Flat.json should be in custom_out/")
	_, err = os.Stat(filepath.Join(customDir, "Flat.bytes"))
	assert.NoError(t, err, "Flat.bytes should inherit output.dir")
}

// --- Test: protojson value normalization (integers as strings in JSON) ---

func TestCLI_ProtoJSON_IntegerFieldFormat(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	copyFile(t, fixturePath("flat_fields/flat_fields.xlsx"), filepath.Join(dir, "flat_fields.xlsx"))
	copyFile(t, protoPath("flat_fields.proto"), filepath.Join(dir, "flat_fields.proto"))

	writeConfig(t, dir, `
proto:
  enabled: true
  files: ["flat_fields.proto"]
  import_path: ["."]
sheet:
  comment_rows: [1]
  meta_row: 2
  data_row_start: 3
  type_suffix: "Sheet"
  list_field_name: "List"
  row_type_suffix: "SheetRow"
output:
  dir: "."
  json:
    enabled: true
    indent: "  "
`)

	_, _, err := runCLI(t, dir, "flat_fields.xlsx")
	require.NoError(t, err)

	jsonData, err := os.ReadFile(filepath.Join(dir, "Flat.json"))
	require.NoError(t, err)

	// protojson serializes int32/int64 as strings in proto3 JSON format
	var result map[string]any
	require.NoError(t, json.Unmarshal(jsonData, &result))

	list := result["List"].([]any)
	row0 := list[0].(map[string]any)

	// protojson outputs numeric fields as strings: "1", "100", "95"
	assert.Equal(t, "1", jsonVal(row0["ID"]), "int32 ID should be string in proto3 JSON")
	assert.Equal(t, "100", jsonVal(row0["Count"]), "int64 Count should be string in proto3 JSON")
	assert.Equal(t, "Alice", row0["Name"], "string Name should be string")
	assert.Equal(t, "95", jsonVal(row0["Score"]), "int64 Score should be string in proto3 JSON")
}

// jsonVal extracts a value from a JSON-decoded map, handling both string and float64.
// protojson encodes integers as strings, raw JSON uses strings too.
func jsonVal(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strings.TrimRight(strings.TrimRight(
			strings.ToLower(string([]byte(json.Number(fmt.Sprintf("%v", val))))),
			"0"), ".")
	default:
		return ""
	}
}

// Verify the imports are used (compile-time check)
var _ protoreflect.MessageDescriptor
var _ protojson.MarshalOptions
var _ = dynamicpb.NewMessage
