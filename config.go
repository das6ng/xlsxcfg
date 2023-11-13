package xlsxcfg

import (
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type ConfigFile struct {
	Proto struct {
		Files      []string `yaml:"files"`
		ImportPath []string `yaml:"import_path"`
	} `yaml:"proto"`
	Sheet struct {
		CommentRows   []int  `yaml:"comment_rows"`
		MetaRow       int    `yaml:"meta_row"`
		DataRowStart  int    `yaml:"data_row_start"`
		TypeSuffix    string `yaml:"type_suffix"`
		ListFieldName string `yaml:"list_field_name"`
		RowTypeSuffix string `yaml:"row_type_suffix"`
	} `yaml:"sheet"`
	Output struct {
		Dir        string `yaml:"dir"`
		WriteJSON  bool   `yaml:"write_json"`
		JSONIndent string `yaml:"json_indent"`
		WriteBytes bool   `yaml:"write_bytes"`
	} `yaml:"output"`
}

func ConfigFromFile(f string) (*ConfigFile, error) {
	c := &ConfigFile{}
	if bs, err := os.ReadFile(f); err != nil {
		return nil, err
	} else if err = yaml.Unmarshal(bs, c); err != nil {
		return nil, err
	}
	return c, nil
}

type Config struct {
	*ConfigFile
	tp TypeProvidor
}

func NewConfig(cfg *ConfigFile, tp TypeProvidor) *Config {
	return &Config{
		ConfigFile: cfg,
		tp:         tp,
	}
}

func (p *Config) IsStrField(typeName, fieldPath string) bool {
	md := p.tp.MessageByName(typeName)
	if md == nil {
		log.Println("message " + typeName + " cannot find in proto messages")
		return true
	}
	return IsStrField(md, strings.Split(fieldPath, ".")...)
}

func (p *Config) IsComment(i int, row []string) bool {
	for _, l := range p.ConfigFile.Sheet.CommentRows {
		if l == i+1 {
			return true
		}
	}
	return false
}

func (p *Config) IsMeta(i int, row []string) bool {
	return i+1 == p.ConfigFile.Sheet.MetaRow
}

func (p *Config) IsData(i int, row []string) bool {
	return i+1 >= p.Sheet.DataRowStart
}
