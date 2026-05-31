package xlsxcfg

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/das6ng/xlsxcfg/constant"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"
)

// OutputFormat configures a single output format.
type OutputFormat struct {
	Enabled   bool   `yaml:"enabled"`
	Dir       string `yaml:"dir"`       // inherits from Output.Dir when empty
	Extension string `yaml:"extension"` // e.g. ".json", ".bytes"; uses format default when empty
}

// JSONOutputFormat adds JSON-specific settings to OutputFormat.
type JSONOutputFormat struct {
	OutputFormat `yaml:",inline"`
	Indent string `yaml:"indent"` // e.g. "  "; empty produces compact JSON
}

// ConfigFile represents the YAML configuration that controls how xlsx sheets map to
// proto messages and how output is written.
type ConfigFile struct {
	// Proto configures which .proto files to load and where to resolve imports.
	Proto struct {
		Enabled     bool     `yaml:"enabled"`
		Files       []string `yaml:"files"`        // e.g. ["hero.proto"]
		ImportPath  []string `yaml:"import_path"`  // like protoc's -I flag
	} `yaml:"proto"`

	// Sheet configures row classification and sheet-to-proto name mapping.
	Sheet struct {
		CommentRows   []int  `yaml:"comment_rows"`    // 1-based row indices to skip
		MetaRow       int    `yaml:"meta_row"`         // 1-based row index for metadata/type hints
		DataRowStart  int    `yaml:"data_row_start"`   // 1-based; must be > MetaRow
		TypeSuffix    string `yaml:"type_suffix"`      // e.g. "Sheet" → "Hero" maps to "HeroSheet"
		ListFieldName string `yaml:"list_field_name"`  // e.g. "rows" in `repeated HeroSheetRow rows = 1;`
		RowTypeSuffix string `yaml:"row_type_suffix"`  // e.g. "SheetRow" → "Hero" maps to "HeroSheetRow"
		TransposeMark string `yaml:"transpose_mark"`   // prefix/suffix marking transposed sheets (e.g. "~")
	} `yaml:"sheet"`

	// Constant configures optional key-value constant loading from xlsx files.
	// Cell values matching [Key] are replaced with the corresponding constant value.
	Constant constant.Config `yaml:"constant"`

	// Output configures where and how converted data is written.
	Output struct {
		Dir         string          `yaml:"dir"`          // default output directory
		FieldOrder  string          `yaml:"field_order"`  // "schema" (proto field#) or "source" (xlsx column order)
		RawJSON     JSONOutputFormat  `yaml:"raw_json"`     // raw JSON without proto validation
		RawMsgpack  OutputFormat      `yaml:"raw_msgpack"`  // raw msgpack without proto validation
		JSON        JSONOutputFormat  `yaml:"json"`         // proto-validated JSON
		Msgpack     OutputFormat      `yaml:"msgpack"`      // proto-validated msgpack
		PbBytes     OutputFormat      `yaml:"pb_bytes"`     // proto-validated protobuf binary
	} `yaml:"output"`
}

// ConfigFromFile reads and parses a YAML configuration file.
func ConfigFromFile(f string) (*ConfigFile, error) {
	c := &ConfigFile{}
	if bs, err := os.ReadFile(f); err != nil {
		return nil, err
	} else if err = yaml.Unmarshal(bs, c); err != nil {
		return nil, err
	}
	return c, nil
}

// DefaultConfig returns a ConfigFile with proto disabled and only raw_json enabled.
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

// Config combines ConfigFile with a TypeProvider for runtime proto field lookups.
type Config struct {
	*ConfigFile
	tp        TypeProvider     // proto message descriptors for field type resolution
	ConstData *constant.Data   // nil when constant loading is disabled
}

// NewConfig creates a runtime Config from a ConfigFile and TypeProvider.
func NewConfig(cfg *ConfigFile, tp TypeProvider) *Config {
	return &Config{
		ConfigFile: cfg,
		tp:         tp,
	}
}

// IsStrField checks whether the leaf field at the dot-separated fieldPath within
// the named proto message is a string type. Returns true for all fields when proto is disabled.
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
// dot-separated fieldPath, or nil if not resolvable.
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

// FieldOrder returns the configured field ordering, defaulting to "schema" (proto field number order).
func (c *ConfigFile) FieldOrder() string {
	if c.Output.FieldOrder == "" {
		return "schema"
	}
	return c.Output.FieldOrder
}

// IsSourceOrder reports whether output should use xlsx column order.
func (c *ConfigFile) IsSourceOrder() bool {
	return c.FieldOrder() == "source"
}

// IsTransposed reports whether sheetName should be parsed column-per-record
// instead of row-per-record.
func (c *ConfigFile) IsTransposed(sheetName string) bool {
	mark := c.Sheet.TransposeMark
	return mark != "" && (strings.HasPrefix(sheetName, mark) || strings.HasSuffix(sheetName, mark))
}

// StripTransposeMark removes the leading or trailing TransposeMark from sheetName.
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
// Compares against 1-based config values (i+1).
func (p *Config) IsComment(i int, row []string) bool {
	for _, l := range p.ConfigFile.Sheet.CommentRows {
		if l == i+1 {
			return true
		}
	}
	return false
}

// IsMeta reports whether the row at 0-based index i is the metadata row (compares i+1 against 1-based MetaRow).
func (p *Config) IsMeta(i int, row []string) bool {
	return i+1 == p.ConfigFile.Sheet.MetaRow
}

// IsData reports whether the row at 0-based index i is a data row (i+1 >= DataRowStart).
func (p *Config) IsData(i int, row []string) bool {
	return i+1 >= p.Sheet.DataRowStart
}

// Validate checks that the configuration is well-formed: data_row_start > meta_row,
// no comment/meta overlap, suffixes and proto files set when proto is enabled,
// and at least one output format enabled.
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
