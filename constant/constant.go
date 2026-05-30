// Package constant provides a standalone loader for key-value reference/lookup tables
// from Excel (.xlsx) files. It is designed for "constant" data — simple two-column
// (key, value) sheets that other config sheets can reference at parse time via
// quoted delimiters (e.g., [SomeKey] → looks up "SomeKey" and substitutes its value).
//
// The loaded data can also be exported as a protobuf ConstantList message or raw JSON.
package constant

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Config holds the YAML configuration for the constant loader.
// It controls which xlsx files to read, how many header rows to skip,
// which prefix marks comment lines, and what delimiters wrap reference keys.
type Config struct {
	// Enabled controls whether constant loading is active. If false, Load returns empty data.
	Enabled bool `yaml:"enabled"`
	// SkipRows is the number of leading rows to skip in each sheet (e.g., 1 to skip a header row).
	SkipRows int `yaml:"skip_rows"`
	// Comment is the prefix that marks a key as a comment line. Rows whose first cell
	// starts with this prefix are ignored. For example, "#" skips rows like "# This is a comment".
	Comment string `yaml:"comment"`
	// RefQuote defines the left and right delimiters used to wrap reference keys in cell values.
	// For example, L="[" and R="]" means cell value "[HeroHP]" references the constant key "HeroHP".
	RefQuote struct {
		L string `yaml:"l"` // Left delimiter (e.g., "[")
		R string `yaml:"r"` // Right delimiter (e.g., "]")
	} `yaml:"ref_quote"`
	// Files is the list of .xlsx file paths to load constants from.
	// Each file is expected to have sheets with two columns: key (col 0) and value (col 1).
	Files []string `yaml:"files"`
}

// Data holds the loaded key-value constant pairs and provides lookup via Get.
// It is populated by calling Load with a Config.
type Data struct {
	// config retains the configuration for use during key lookups (e.g., RefQuote delimiters).
	config *Config
	// data maps constant keys to their string values, loaded from xlsx files.
	data map[string]string
}

// Load reads all configured xlsx files and extracts key-value pairs into a Data.
// If the config is nil or not enabled, it returns an empty Data without error.
// Each file is processed in order; later files may overwrite keys from earlier ones.
func Load(ctx context.Context, c *Config) (data *Data, err error) {
	data = &Data{config: c, data: map[string]string{}}
	if c == nil || !c.Enabled {
		return
	}
	for _, f := range c.Files {
		err = data.loadFile(ctx, f)
		if err != nil {
			return
		}
	}
	return
}

// Get looks up a value by key, expecting the key to be wrapped in the configured
// RefQuote delimiters. For example, with delimiters "[" and "]", passing "[类型1]"
// will strip the brackets and look up "类型1".
//
// Returns the value and true if found, or ("", false) if:
//   - the Data is nil or empty
//   - no left delimiter is configured
//   - the key doesn't start with the left delimiter
//   - the stripped key is not found in the loaded data
func (d *Data) Get(key string) (string, bool) {
	if d == nil || len(d.data) == 0 {
		return "", false
	}
	// Without a left delimiter, reference lookup is disabled
	if d.config.RefQuote.L == "" {
		return "", false
	}
	// Only process keys that start with the left delimiter — plain keys are not looked up
	if !strings.HasPrefix(key, d.config.RefQuote.L) {
		return "", false
	}
	// Strip the delimiters to get the raw lookup key
	key = strings.TrimPrefix(key, d.config.RefQuote.L)
	key = strings.TrimSuffix(key, d.config.RefQuote.R)
	v, ok := d.data[key]
	return v, ok
}

// Export converts the loaded key-value data into a protobuf ConstantList message.
// This can be used to serialize the constants as protobuf binary or JSON.
func (d *Data) Export(_ context.Context) (data *ConstantList) {
	data = &ConstantList{Data: make([]*ConstantEntry, 0, len(d.data))}
	for k, v := range d.data {
		data.Data = append(data.Data, &ConstantEntry{Key: k, Val: v})
	}
	return
}

// ExportJSON converts the loaded key-value data to JSON using the protobuf ConstantList
// message structure. Returns the raw JSON bytes.
func (d *Data) ExportJSON(ctx context.Context) (data []byte, err error) {
	obj := d.Export(ctx)
	data, err = json.Marshal(obj)
	return
}

// loadFile reads a single xlsx file row-by-row, extracting key-value pairs.
// For each sheet in the workbook:
//   - Skips the configured number of header rows (SkipRows).
//   - Reads the first two columns as key and value.
//   - Skips rows where the key starts with the configured comment prefix.
//   - Rows with fewer than 2 columns are silently ignored.
func (d *Data) loadFile(_ context.Context, file string) error {
	f, err := excelize.OpenFile(file)
	if err != nil {
		return err
	}
	defer f.Close()

	// Iterate over all sheets in the workbook
	sheets := f.WorkBook.Sheets.Sheet
	for _, sheet := range sheets {
		rows, err := f.Rows(sheet.Name)
		if err != nil {
			return err
		}
		n := 0
		for rows.Next() {
			n++
			// Skip header rows (e.g., n < SkipRows skips rows 1 through SkipRows-1)
			if n < d.config.SkipRows {
				continue
			}
			cells, err := rows.Columns()
			if err != nil {
				return err
			}
			// Need at least 2 columns: key and value
			if len(cells) < 2 {
				continue
			}
			key := cells[0]
			// Skip comment lines — keys starting with the configured prefix (e.g., "#")
			if d.config.Comment != "" && strings.HasPrefix(key, d.config.Comment) {
				continue
			}
			d.data[key] = cells[1]
		}
	}
	return nil
}
