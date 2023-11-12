package xlsxcfg

import (
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
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

func ConfigFromFile(f string) (*Config, error) {
	c := &Config{}
	if bs, err := os.ReadFile(f); err != nil {
		return nil, err
	} else if err = yaml.Unmarshal(bs, c); err != nil {
		return nil, err
	}
	return c, nil
}

type Parameter struct {
	*Config
	tp TypeProvidor
}

func NewParameter(cfg *Config, tp TypeProvidor) *Parameter {
	return &Parameter{
		Config: cfg,
		tp:     tp,
	}
}

func (p *Parameter) IsStrField(typeName, fieldPath string) bool {
	md := p.tp.MessageByName(typeName)
	if md == nil {
		log.Println("message " + typeName + " cannot find in proto messages")
		return true
	}
	return IsStrField(md, strings.Split(fieldPath, ".")...)
}

func (p *Parameter) IsComment(i int, row []string) bool {
	for _, l := range p.Config.Sheet.CommentRows {
		if l == i+1 {
			return true
		}
	}
	return false
}

func (p *Parameter) IsMeta(i int, row []string) bool {
	return i+1 == p.Config.Sheet.MetaRow
}

func (p *Parameter) IsData(i int, row []string) bool {
	return i+1 >= p.Sheet.DataRowStart
}
