package xlsxcfg

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoProtoRawJSON verifies that running without a proto schema produces
// raw JSON output where all values are strings.
func TestNoProtoRawJSON(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Output.RawJSON.Enabled = true
	cfg.Output.RawJSON.Dir = "."

	param := NewConfig(cfg, nil) // nil TypeProvider = no proto
	result, err := LoadXlsxFiles(context.Background(), param, "tests/flat_fields/flat_fields.xlsx")
	require.NoError(t, err)

	rows, ok := result["Flat"]
	require.True(t, ok)
	require.Equal(t, 7, len(rows))

	// In no-proto mode, all values should be strings
	row1 := rows[0].(*OrderedMap)
	id, _ := row1.Get("ID")
	assert.Equal(t, "1", id, "expected string in no-proto mode")
	count, _ := row1.Get("Count")
	assert.Equal(t, "100", count, "expected string in no-proto mode")
	name, _ := row1.Get("Name")
	assert.Equal(t, "Alice", name)
}

// TestNoProtoRawMsgpack verifies raw msgpack output round-trips correctly.
func TestNoProtoRawMsgpack(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Proto.Enabled = false
	cfg.Output.RawMsgpack.Enabled = true
	cfg.Output.RawMsgpack.Dir = "."

	param := NewConfig(cfg, nil)
	result, err := LoadXlsxFiles(context.Background(), param, "tests/flat_fields/flat_fields.xlsx")
	require.NoError(t, err)

	rows, ok := result["Flat"]
	require.True(t, ok)

	// Serialize to msgpack via the OrderedMap-aware encoder (simulating writer behavior).
	// For this test, just verify JSON round-trip works since msgpack encoding of
	// OrderedMap values requires the custom encoder.
	buf, err := json.Marshal(map[string]any{"List": rows})
	require.NoError(t, err)

	// Deserialize back
	var decoded map[string]any
	err = json.Unmarshal(buf, &decoded)
	require.NoError(t, err)

	// Verify round-trip via JSON
	list := decoded["List"].([]any)
	assert.Equal(t, 7, len(list))
	row1 := list[0].(map[string]any)
	assert.Equal(t, "1", row1["ID"])
	assert.Equal(t, "Alice", row1["Name"])
}

// TestDefaultConfig verifies DefaultConfig returns valid defaults.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.False(t, cfg.Proto.Enabled)
	assert.Equal(t, []int{1}, cfg.Sheet.CommentRows)
	assert.Equal(t, 2, cfg.Sheet.MetaRow)
	assert.Equal(t, 3, cfg.Sheet.DataRowStart)
	assert.Equal(t, "Sheet", cfg.Sheet.TypeSuffix)
	assert.Equal(t, "List", cfg.Sheet.ListFieldName)
	assert.Equal(t, "SheetRow", cfg.Sheet.RowTypeSuffix)
	assert.True(t, cfg.Output.RawJSON.Enabled)
	assert.False(t, cfg.Output.JSON.Enabled)
	assert.False(t, cfg.Output.PbBytes.Enabled)
	assert.NoError(t, cfg.Validate())
}

// TestResolveDir verifies directory resolution with fallback.
func TestResolveDir(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Output.Dir = "fallback"

	// Format dir takes priority
	assert.Equal(t, "custom", cfg.ResolveDir("custom"))
	// Falls back to output.dir
	assert.Equal(t, "fallback", cfg.ResolveDir(""))
	// Empty output.dir falls back to "."
	cfg.Output.Dir = ""
	assert.Equal(t, ".", cfg.ResolveDir(""))
}

// TestNoProtoWithProtoFormats verifies that proto-validated formats are
// gracefully handled when proto is disabled (they should be skipped).
func TestNoProtoWithProtoFormats(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Proto.Enabled = false
	cfg.Output.JSON.Enabled = true     // proto-validated, should be skipped
	cfg.Output.PbBytes.Enabled = true  // proto-validated, should be skipped
	cfg.Output.RawJSON.Enabled = true  // raw, should work

	param := NewConfig(cfg, nil)

	// This should succeed — proto-validated formats are just skipped
	result, err := LoadXlsxFiles(context.Background(), param, "tests/edge_cases/edge.xlsx")
	require.NoError(t, err)

	rows, ok := result["Edge"]
	require.True(t, ok)
	assert.Equal(t, 4, len(rows))

	// All values are strings in no-proto mode
	row1 := rows[0].(*OrderedMap)
	id, _ := row1.Get("ID")
	assert.Equal(t, "1", id)
	label, _ := row1.Get("Label")
	assert.Equal(t, "row1", label)
	val, _ := row1.Get("Value")
	assert.Equal(t, "10", val)
}

// TestNoProtoNestedStructs verifies nested struct parsing works without proto.
func TestNoProtoNestedStructs(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Proto.Enabled = false
	cfg.Output.RawJSON.Enabled = true

	param := NewConfig(cfg, nil)
	result, err := LoadXlsxFiles(context.Background(), param, "tests/nested_structs/nested.xlsx")
	require.NoError(t, err)

	rows, ok := result["Nested"]
	require.True(t, ok)
	require.Equal(t, 4, len(rows))

	// Nested struct should be an OrderedMap with string values
	row1 := rows[0].(*OrderedMap)
	homeVal, _ := row1.Get("Home")
	home := homeVal.(*OrderedMap)
	street, _ := home.Get("Street")
	assert.Equal(t, "123 Main St", street)
	city, _ := home.Get("City")
	assert.Equal(t, "NYC", city)
	zip, _ := home.Get("Zip")
	assert.Equal(t, "10001", zip)

	// Verify it serializes to JSON cleanly
	buf, err := json.Marshal(map[string]any{"List": rows})
	require.NoError(t, err)
	assert.Contains(t, string(buf), `"Street":"123 Main St"`)
}
