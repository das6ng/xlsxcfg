// Package constant loads key-value lookup tables from Excel files.
// Other config sheets reference these values at parse time via quoted delimiters (e.g. [SomeKey]).
package constant

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Config holds the YAML configuration for the constant loader.
type Config struct {
	Enabled  bool `yaml:"enabled"`
	// SkipRows is the number of leading rows to skip (e.g. 1 to skip a header row).
	SkipRows int `yaml:"skip_rows"`
	// Comment is the prefix that marks a key as a comment line (e.g. "#").
	Comment string `yaml:"comment"`
	// RefQuote defines delimiters wrapping reference keys in cell values (e.g. L="[" R="]").
	RefQuote struct {
		L string `yaml:"l"`
		R string `yaml:"r"`
	} `yaml:"ref_quote"`
	// Files is the list of .xlsx paths. Each sheet expects two columns: key, value.
	Files []string `yaml:"files"`
}

// Data holds loaded key-value constant pairs and provides lookup via Get.
type Data struct {
	config *Config
	data   map[string]string
}

// Load reads all configured xlsx files and returns key-value pairs. Later files overwrite earlier keys.
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

// Get strips RefQuote delimiters from key and looks up the bare key.
func (d *Data) Get(key string) (string, bool) {
	if d == nil || len(d.data) == 0 {
		return "", false
	}
	if d.config.RefQuote.L == "" {
		return "", false
	}
	if !strings.HasPrefix(key, d.config.RefQuote.L) {
		return "", false
	}
	key = strings.TrimPrefix(key, d.config.RefQuote.L)
	key = strings.TrimSuffix(key, d.config.RefQuote.R)
	v, ok := d.data[key]
	return v, ok
}

// Export converts the loaded data into a protobuf ConstantList message.
func (d *Data) Export(_ context.Context) (data *ConstantList) {
	data = &ConstantList{Data: make([]*ConstantEntry, 0, len(d.data))}
	for k, v := range d.data {
		data.Data = append(data.Data, &ConstantEntry{Key: k, Val: v})
	}
	return
}

// ExportJSON returns the loaded data as JSON using the ConstantList message structure.
func (d *Data) ExportJSON(ctx context.Context) (data []byte, err error) {
	obj := d.Export(ctx)
	data, err = json.Marshal(obj)
	return
}

// loadFile reads a single xlsx file, extracting key-value pairs from each sheet.
// Keys wrapped in RefQuote delimiters are stripped to bare form for lookup.
func (d *Data) loadFile(_ context.Context, file string) error {
	f, err := excelize.OpenFile(file)
	if err != nil {
		return err
	}
	defer f.Close()

	sheets := f.WorkBook.Sheets.Sheet
	for _, sheet := range sheets {
		rows, err := f.Rows(sheet.Name)
		if err != nil {
			return err
		}
		n := 0
		for rows.Next() {
			n++
			if n < d.config.SkipRows {
				continue
			}
			cells, err := rows.Columns()
			if err != nil {
				return err
			}
			if len(cells) < 2 {
				continue
			}
			key := cells[0]
			if d.config.Comment != "" && strings.HasPrefix(key, d.config.Comment) {
				continue
			}
			// Strip ref_quote delimiters so xlsx can show keys like "[HeroHP]" for clarity
			if d.config.RefQuote.L != "" {
				key = strings.TrimPrefix(key, d.config.RefQuote.L)
				key = strings.TrimSuffix(key, d.config.RefQuote.R)
			}
			d.data[key] = cells[1]
		}
	}
	return nil
}
