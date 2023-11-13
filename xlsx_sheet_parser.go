package xlsxcfg

import "context"

type sheetParser struct {
	typeName string
	maxRow   int
	cols     [][]string
	param    *Config
}

func newSheetParser(p *Config) *sheetParser {
	return &sheetParser{param: p}
}

func (p *sheetParser) SetName(name string) {
	p.typeName = name
}

func (p *sheetParser) Feed(cells []string) {
	p.cols = append(p.cols, cells)
	if p.maxRow < len(cells) {
		p.maxRow = len(cells)
	}
}

// Parse one xls sheet
func (p *sheetParser) Parse(ctx context.Context) ([]any, error) {
	colCount := len(p.cols)
	rowParser := newRowParser(p.typeName, p.param)
	result := make([]any, 0, p.maxRow)
	dataRows := make([][]string, 0, p.maxRow)
	for i := 0; i < p.maxRow; i++ {
		row := make([]string, 0, colCount)
		for _, col := range p.cols {
			if len(col) > i {
				row = append(row, col[i])
			} else {
				row = append(row, "")
			}
		}
		if p.param.IsMeta(i, row) {
			rowParser.Meta(ctx, row)
		} else if p.param.IsComment(i, row) {
			// comment row, skip
		} else if p.param.IsData(i, row) {
			dataRows = append(dataRows, row)
		}
	}
	for _, row := range dataRows {
		rowData, err := rowParser.Parse(ctx, row)
		if err != nil {
			return nil, err
		}
		if rowData == nil {
			continue
		}
		result = append(result, rowData)
	}
	return result, nil
}
