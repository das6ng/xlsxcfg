package xlsxcfg

import (
	"fmt"
	"log"
	"os"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"
)

// OutputFormat configures a single output format (e.g. protobuf binary, msgpack).
type OutputFormat struct {
	// Enabled controls whether this output format is generated.
	Enabled bool `yaml:"enabled"`
	// Dir is the output directory for this format. When empty, inherits from Output.Dir.
	Dir string `yaml:"dir"`
	// Extension is the output file extension including the dot (e.g. ".json", ".bytes").
	// When empty, uses the format's default extension.
	Extension string `yaml:"extension"`
}

// JSONOutputFormat extends OutputFormat with JSON-specific settings.
type JSONOutputFormat struct {
	OutputFormat `yaml:",inline"`
	// Indent controls the indentation of JSON output (e.g. "  " for 2-space indent).
	// Empty string produces compact JSON.
	Indent string `yaml:"indent"`
}

// ConfigFile represents the YAML configuration file (typically xlsxcfg.yaml)
// that controls how xlsx sheets are mapped to proto messages and how output
// is written. It is divided into three sections: proto, sheet, and output.
type ConfigFile struct {
	// Proto configures which .proto files to load and where to resolve imports.
	Proto struct {
		// Enabled controls whether proto loading is enabled.
		Enabled bool `yaml:"enabled"`
		// Files lists the .proto file names to compile (e.g. ["hero.proto"]).
		Files []string `yaml:"files"`
		// ImportPaths lists directories to search for proto imports (like protoc's -I flag).
		ImportPath []string `yaml:"import_path"`
	} `yaml:"proto"`

	// Sheet configures how rows in each xlsx sheet are classified and how
	// sheet names map to proto message names.
	Sheet struct {
		// CommentRows is a list of 1-based row indices that contain comments
		// (skipped during parsing). Row indices are 1-based to match Excel's display.
		CommentRows []int `yaml:"comment_rows"`
		// MetaRow is the 1-based row index of the metadata row, which typically
		// contains field type hints or other annotations above the data.
		MetaRow int `yaml:"meta_row"`
		// DataRowStart is the 1-based row index where actual data rows begin.
		// Must be greater than MetaRow.
		DataRowStart int `yaml:"data_row_start"`
		// TypeSuffix is appended to the sheet name to form the wrapper message name.
		// For example, with suffix "Sheet", a sheet named "Hero" maps to "HeroSheet".
		TypeSuffix string `yaml:"type_suffix"`
		// ListFieldName is the field name used in the wrapper message for the
		// repeated row list (e.g. "rows" in message HeroSheet { repeated HeroSheetRow rows = 1; }).
		ListFieldName string `yaml:"list_field_name"`
		// RowTypeSuffix is appended to the sheet name to form the per-row message name.
		// For example, with suffix "SheetRow", a sheet named "Hero" maps to "HeroSheetRow".
		RowTypeSuffix string `yaml:"row_type_suffix"`
		// TransposeMark is an optional prefix or suffix that marks a sheet for transposed parsing.
		// When set (e.g., "~"), any xlsx sheet name starting or ending with this mark is parsed
		// column-wise instead of row-wise. The mark is stripped before proto type resolution.
		TransposeMark string `yaml:"transpose_mark"`
	} `yaml:"sheet"`

	// Output configures where and how converted data is written.
	Output struct {
		// Dir is the default directory where output files are written.
		// Individual formats can override this with their own Dir field.
		Dir string `yaml:"dir"`
		// FieldOrder controls the ordering of fields in proto-validated output.
		// "schema" (default) orders by proto field number.
		// "source" orders by xlsx column order.
		// Only applies to proto JSON and proto msgpack output; raw formats always
		// use source order. Proto binary is always field-number ordered.
		FieldOrder string `yaml:"field_order"`
		// RawJSON outputs raw sheet data as JSON without proto schema validation.
		RawJSON JSONOutputFormat `yaml:"raw_json"`
		// RawMsgpack outputs raw sheet data as msgpack without proto schema validation.
		RawMsgpack OutputFormat `yaml:"raw_msgpack"`
		// JSON outputs proto-validated data serialized as JSON.
		JSON JSONOutputFormat `yaml:"json"`
		// Msgpack outputs proto-validated data serialized as msgpack.
		Msgpack OutputFormat `yaml:"msgpack"`
		// PbBytes outputs proto-validated data serialized as protobuf binary.
		PbBytes OutputFormat `yaml:"pb_bytes"`
	} `yaml:"output"`
}

// ConfigFromFile reads and parses a YAML configuration file at the given path.
func ConfigFromFile(f string) (*ConfigFile, error) {
	c := &ConfigFile{}
	if bs, err := os.ReadFile(f); err != nil {
		return nil, err
	} else if err = yaml.Unmarshal(bs, c); err != nil {
		return nil, err
	}
	return c, nil
}

// DefaultConfig returns a ConfigFile with sensible defaults: proto disabled,
// only raw_json enabled. These defaults allow the tool to work without a
// config file for simple use cases.
func DefaultConfig() *ConfigFile {
	c := &ConfigFile{}
	c.Proto.Enabled = false
	c.Sheet.CommentRows = []int{1}
	c.Sheet.MetaRow = 2
	c.Sheet.DataRowStart = 3
	c.Sheet.TypeSuffix = "Sheet"
	c.Sheet.ListFieldName = "List"
	c.Sheet.RowTypeSuffix = "SheetRow"
	c.Output.Dir = "."
	c.Output.RawJSON.Enabled = true
	c.Output.RawJSON.Dir = "."
	c.Output.RawJSON.Indent = "  "
	return c
}

// ResolveDir returns the effective output directory for a format.
// Priority: format's own dir > output.dir > "."
func (c *ConfigFile) ResolveDir(formatDir string) string {
	if formatDir != "" {
		return formatDir
	}
	if c.Output.Dir != "" {
		return c.Output.Dir
	}
	return "."
}

// Config combines the parsed YAML configuration with a TypeProvider for runtime
// proto field type lookups. It is the main configuration object used throughout
// the parsing pipeline.
type Config struct {
	*ConfigFile
	// tp provides proto message descriptors for field type resolution during parsing.
	tp TypeProvider
}

// NewConfig creates a runtime Config by combining a parsed ConfigFile with a
// TypeProvider loaded from the proto files specified in the config.
func NewConfig(cfg *ConfigFile, tp TypeProvider) *Config {
	return &Config{
		ConfigFile: cfg,
		tp:         tp,
	}
}

// IsStrField checks whether the leaf field at the given dot-separated fieldPath
// within the named proto message is a string type. This is used during row
// parsing to decide whether a cell value should be treated as a string literal
// or parsed as a numeric value.
//
// typeName is the short proto message name (e.g. "HeroSheetRow").
// fieldPath is a dot-separated path like "Phone.Region".
// When no TypeProvider is set (proto disabled), all fields are treated as strings.
func (p *Config) IsStrField(typeName, fieldPath string) bool {
	if p.tp == nil {
		return true
	}
	md := p.tp.MessageByName(typeName)
	if md == nil {
		log.Fatalf("message %q cannot be found in proto messages", typeName)
	}
	return IsStrField(md, strings.Split(fieldPath, ".")...)
}

// GetFieldDescriptor returns the proto FieldDescriptor for the leaf field at the
// given dot-separated fieldPath within the named proto message.
// Returns nil if no TypeProvider is set, the message is not found, or the path
// does not resolve to a field.
func (p *Config) GetFieldDescriptor(typeName, fieldPath string) protoreflect.FieldDescriptor {
	if p.tp == nil {
		return nil
	}
	md := p.tp.MessageByName(typeName)
	if md == nil {
		return nil
	}
	return GetFieldDescriptor(md, strings.Split(fieldPath, ".")...)
}

// FieldOrder returns the configured field ordering for output.
// Returns "schema" (proto field number order) by default.
func (c *ConfigFile) FieldOrder() string {
	if c.Output.FieldOrder == "" {
		return "schema"
	}
	return c.Output.FieldOrder
}

// IsSourceOrder reports whether output should use xlsx source column order.
func (c *ConfigFile) IsSourceOrder() bool {
	return c.FieldOrder() == "source"
}

// IsTransposed reports whether the given xlsx sheet name should be parsed in
// transposed mode (column-per-record instead of row-per-record).
// Returns true when TransposeMark is non-empty and sheetName starts or ends with it.
func (c *ConfigFile) IsTransposed(sheetName string) bool {
	mark := c.Sheet.TransposeMark
	return mark != "" && (strings.HasPrefix(sheetName, mark) || strings.HasSuffix(sheetName, mark))
}

// StripTransposeMark removes the leading or trailing TransposeMark from the sheet name.
// If the mark is empty or the name doesn't match it, returns the name unchanged.
func (c *ConfigFile) StripTransposeMark(sheetName string) string {
	if !c.IsTransposed(sheetName) {
		return sheetName
	}
	mark := c.Sheet.TransposeMark
	if strings.HasPrefix(sheetName, mark) {
		return sheetName[len(mark):]
	}
	return sheetName[:len(sheetName)-len(mark)]
}

// IsComment reports whether the row at 0-based index i is a comment row.
// The config stores comment row indices as 1-based (matching Excel display),
// so we compare against i+1.
func (p *Config) IsComment(i int, row []string) bool {
	for _, l := range p.ConfigFile.Sheet.CommentRows {
		if l == i+1 {
			return true
		}
	}
	return false
}

// IsMeta reports whether the row at 0-based index i is the metadata row.
// The config stores MetaRow as a 1-based index, so we compare against i+1.
func (p *Config) IsMeta(i int, row []string) bool {
	return i+1 == p.ConfigFile.Sheet.MetaRow
}

// IsData reports whether the row at 0-based index i is a data row.
// A row is a data row if its 1-based index (i+1) is at or past DataRowStart.
func (p *Config) IsData(i int, row []string) bool {
	return i+1 >= p.Sheet.DataRowStart
}

// Validate checks that the configuration is well-formed:
//   - data_row_start must be greater than meta_row (when both are positive)
//   - comment_rows must not overlap with meta_row
//   - type_suffix and row_type_suffix must be set when proto is enabled
//   - proto.files must not be empty when proto is enabled
//   - at least one output format must be enabled
//
// Returns an error describing the first validation failure, or nil if valid.
func (c *ConfigFile) Validate() error {
	if c.Sheet.MetaRow > 0 && c.Sheet.DataRowStart > 0 && c.Sheet.DataRowStart <= c.Sheet.MetaRow {
		return fmt.Errorf("sheet.data_row_start (%d) must be greater than sheet.meta_row (%d)", c.Sheet.DataRowStart, c.Sheet.MetaRow)
	}
	for _, cr := range c.Sheet.CommentRows {
		if cr == c.Sheet.MetaRow {
			return fmt.Errorf("sheet.meta_row (%d) must not overlap with sheet.comment_rows", c.Sheet.MetaRow)
		}
	}
	// Proto-required validation
	if c.Proto.Enabled {
		if c.Sheet.TypeSuffix == "" {
			return fmt.Errorf("sheet.type_suffix must not be empty when proto is enabled")
		}
		if c.Sheet.RowTypeSuffix == "" {
			return fmt.Errorf("sheet.row_type_suffix must not be empty when proto is enabled")
		}
		if len(c.Proto.Files) == 0 {
			return fmt.Errorf("proto.files must not be empty when proto is enabled")
		}
	}
	// At least one output format must be enabled
	anyEnabled := c.Output.RawJSON.Enabled || c.Output.RawMsgpack.Enabled ||
		c.Output.JSON.Enabled || c.Output.Msgpack.Enabled || c.Output.PbBytes.Enabled
	if !anyEnabled {
		return fmt.Errorf("at least one output format must be enabled")
	}
	return nil
}
