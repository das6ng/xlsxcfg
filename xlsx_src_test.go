package xlsxcfg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIterXlsxFilesStreaming verifies the streaming iterator API (IterXlsxFiles)
// produces correct results when consuming sheets one row at a time.
func TestIterXlsxFilesStreaming(t *testing.T) {
	cfg := &ConfigFile{
		Proto: struct {
			Enabled    bool     `yaml:"enabled"`
			Files      []string `yaml:"files"`
			ImportPath []string `yaml:"import_path"`
		}{
			Enabled:    true,
			Files:      []string{"multi.proto"},
			ImportPath: []string{"tests"},
		},
		Sheet: struct {
			CommentRows   []int  `yaml:"comment_rows"`
			MetaRow       int    `yaml:"meta_row"`
			DataRowStart  int    `yaml:"data_row_start"`
			TypeSuffix    string `yaml:"type_suffix"`
			ListFieldName string `yaml:"list_field_name"`
			RowTypeSuffix string `yaml:"row_type_suffix"`
			TransposeMark string `yaml:"transpose_mark"`
		}{
			CommentRows:   []int{1},
			MetaRow:       2,
			DataRowStart:  3,
			TypeSuffix:    "Sheet",
			ListFieldName: "List",
			RowTypeSuffix: "SheetRow",
		},
		Output: struct {
			Dir        string           `yaml:"dir"`
			RawJSON    JSONOutputFormat `yaml:"raw_json"`
			RawMsgpack OutputFormat     `yaml:"raw_msgpack"`
			JSON       JSONOutputFormat `yaml:"json"`
			Msgpack    OutputFormat     `yaml:"msgpack"`
			PbBytes    OutputFormat     `yaml:"pb_bytes"`
		}{
			Dir: ".",
			JSON: JSONOutputFormat{
				OutputFormat: OutputFormat{Enabled: true, Dir: "."},
			},
		},
	}
	require.NoError(t, cfg.Validate())

	tp, err := LoadProtoFiles(context.Background(), cfg.Proto.ImportPath, cfg.Proto.Files...)
	require.NoError(t, err)
	param := NewConfig(cfg, tp)

	sheetCount := 0
	rowCounts := map[string]int{}

	for sr, err := range IterXlsxFiles(context.Background(), param, "tests/multi_sheet/multi.xlsx") {
		require.NoError(t, err)
		sheetCount++

		for row, err := range sr.Rows {
			require.NoError(t, err)
			rowCounts[sr.Name]++
			assert.NotNil(t, row)
			assert.NotEmpty(t, row)
		}
	}

	assert.Equal(t, 2, sheetCount, "expected 2 sheets")
	assert.Equal(t, 3, rowCounts["Hero"], "expected 3 Hero rows")
	assert.Equal(t, 3, rowCounts["Item"], "expected 3 Item rows")
}

// TestIterXlsxFilesRowValues verifies actual row data values from the streaming API.
func TestIterXlsxFilesRowValues(t *testing.T) {
	cfg := makeTestConfig("flat_fields.proto")
	tp, err := LoadProtoFiles(context.Background(), cfg.Proto.ImportPath, cfg.Proto.Files...)
	require.NoError(t, err)
	param := NewConfig(cfg, tp)

	sheetCount := 0
	var rows []map[string]any

	for sr, err := range IterXlsxFiles(context.Background(), param, "tests/flat_fields/flat_fields.xlsx") {
		require.NoError(t, err)
		sheetCount++

		for row, err := range sr.Rows {
			require.NoError(t, err)
			rows = append(rows, row)
		}
	}

	assert.Equal(t, 1, sheetCount)
	require.Equal(t, 7, len(rows), "expected 7 data rows")

	assert.Equal(t, int64(1), rows[0]["ID"])
	assert.Equal(t, int64(100), rows[0]["Count"])
	assert.Equal(t, "Alice", rows[0]["Name"])
	assert.Equal(t, int64(-50), rows[1]["Count"])
	// Row 4: empty numeric cells are absent (not int64(0))
	_, hasCount := rows[3]["Count"]
	assert.False(t, hasCount, "empty numeric field should not be set")
	assert.Equal(t, "Diana", rows[3]["Name"])
}

// TestIterXlsxFilesDuplicateSheet verifies duplicate detection with the streaming API.
func TestIterXlsxFilesDuplicateSheet(t *testing.T) {
	cfg := makeTestConfig("example.proto")
	tp, err := LoadProtoFiles(context.Background(), cfg.Proto.ImportPath, cfg.Proto.Files...)
	require.NoError(t, err)
	param := NewConfig(cfg, tp)

	found := false
	for _, err := range IterXlsxFiles(context.Background(), param, "tests/duplicate_sheet/dup1.xlsx", "tests/duplicate_sheet/dup2.xlsx") {
		if err != nil {
			assert.Contains(t, err.Error(), "duplicated sheet")
			found = true
			break
		}
	}
	assert.True(t, found, "expected duplicated sheet error")
}

// TestIterXlsxFilesRepeatedFields verifies repeated fields through the streaming API.
func TestIterXlsxFilesRepeatedFields(t *testing.T) {
	cfg := makeTestConfig("repeated.proto")
	tp, err := LoadProtoFiles(context.Background(), cfg.Proto.ImportPath, cfg.Proto.Files...)
	require.NoError(t, err)
	param := NewConfig(cfg, tp)

	for sr, err := range IterXlsxFiles(context.Background(), param, "tests/repeated_fields/repeated.xlsx") {
		require.NoError(t, err)
		assert.Equal(t, "Repeated", sr.Name)

		var rows []map[string]any
		for row, err := range sr.Rows {
			require.NoError(t, err)
			rows = append(rows, row)
		}
		require.Equal(t, 4, len(rows))

		tags := rows[0]["Tags"].([]any)
		assert.Equal(t, 3, len(tags))
		assert.Equal(t, "go", tags[0])
		assert.Equal(t, "rust", tags[1])
		assert.Equal(t, "python", tags[2])
	}
}

// TestIterXlsxFilesNestedStructs verifies nested structs through the streaming API.
func TestIterXlsxFilesNestedStructs(t *testing.T) {
	cfg := makeTestConfig("nested.proto")
	tp, err := LoadProtoFiles(context.Background(), cfg.Proto.ImportPath, cfg.Proto.Files...)
	require.NoError(t, err)
	param := NewConfig(cfg, tp)

	for sr, err := range IterXlsxFiles(context.Background(), param, "tests/nested_structs/nested.xlsx") {
		require.NoError(t, err)
		assert.Equal(t, "Nested", sr.Name)

		var rows []map[string]any
		for row, err := range sr.Rows {
			require.NoError(t, err)
			rows = append(rows, row)
		}
		require.Equal(t, 4, len(rows))

		home := rows[0]["Home"].(map[string]any)
		assert.Equal(t, "123 Main St", home["Street"])
		assert.Equal(t, "NYC", home["City"])
	}
}

// TestIterXlsxFilesEdgeCases verifies empty-row skipping through the streaming API.
func TestIterXlsxFilesEdgeCases(t *testing.T) {
	cfg := makeTestConfig("edge.proto")
	tp, err := LoadProtoFiles(context.Background(), cfg.Proto.ImportPath, cfg.Proto.Files...)
	require.NoError(t, err)
	param := NewConfig(cfg, tp)

	for sr, err := range IterXlsxFiles(context.Background(), param, "tests/edge_cases/edge.xlsx") {
		require.NoError(t, err)
		assert.Equal(t, "Edge", sr.Name)

		var rows []map[string]any
		for row, err := range sr.Rows {
			require.NoError(t, err)
			rows = append(rows, row)
		}
		assert.Equal(t, 4, len(rows))
		assert.Equal(t, int64(1), rows[0]["ID"])
		assert.Equal(t, int64(3), rows[1]["ID"])
		assert.Equal(t, int64(5), rows[2]["ID"])
		assert.Equal(t, int64(6), rows[3]["ID"])
	}
}

// makeTestConfig creates a ConfigFile with standard sheet settings for testing.
func makeTestConfig(protoFile string) *ConfigFile {
	return &ConfigFile{
		Proto: struct {
			Enabled    bool     `yaml:"enabled"`
			Files      []string `yaml:"files"`
			ImportPath []string `yaml:"import_path"`
		}{
			Enabled:    true,
			Files:      []string{protoFile},
			ImportPath: []string{"tests"},
		},
		Sheet: struct {
			CommentRows   []int  `yaml:"comment_rows"`
			MetaRow       int    `yaml:"meta_row"`
			DataRowStart  int    `yaml:"data_row_start"`
			TypeSuffix    string `yaml:"type_suffix"`
			ListFieldName string `yaml:"list_field_name"`
			RowTypeSuffix string `yaml:"row_type_suffix"`
			TransposeMark string `yaml:"transpose_mark"`
		}{
			CommentRows:   []int{1},
			MetaRow:       2,
			DataRowStart:  3,
			TypeSuffix:    "Sheet",
			ListFieldName: "List",
			RowTypeSuffix: "SheetRow",
		},
		Output: struct {
			Dir        string           `yaml:"dir"`
			RawJSON    JSONOutputFormat `yaml:"raw_json"`
			RawMsgpack OutputFormat     `yaml:"raw_msgpack"`
			JSON       JSONOutputFormat `yaml:"json"`
			Msgpack    OutputFormat     `yaml:"msgpack"`
			PbBytes    OutputFormat     `yaml:"pb_bytes"`
		}{
			Dir: ".",
			JSON: JSONOutputFormat{
				OutputFormat: OutputFormat{Enabled: true, Dir: "."},
			},
		},
	}
}
