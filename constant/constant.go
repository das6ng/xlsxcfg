package constant

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/xuri/excelize/v2"
)

type Config struct {
	Enabled  bool   `yaml:"enabled"`
	SkipRows int    `yaml:"skip_rows"`
	Comment  string `yaml:"comment"`
	RefQuote struct {
		L string `yaml:"l"`
		R string `yaml:"r"`
	} `yaml:"ref_quote"`
	Files []string `yaml:"files"`
}

type Data struct {
	config *Config
	data   map[string]string
}

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

func (d *Data) Export(ctx context.Context) (data *ConstantList) {
	data = &ConstantList{Data: make([]*ConstantEntry, 0, len(d.data))}
	for k, v := range d.data {
		data.Data = append(data.Data, &ConstantEntry{Key: k, Val: v})
	}
	return
}

func (d *Data) ExportJSON(ctx context.Context) (data []byte, err error) {
	obj := d.Export(ctx)
	data, err = json.Marshal(obj)
	return
}

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
			d.data[key] = cells[1]
		}
	}
	return nil
}
