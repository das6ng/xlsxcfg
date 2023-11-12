package xlsxcfg

import "context"

type sheetParser struct {
	name   string
	maxRow int
	cols   [][]string
	param  *Parameter
}

func newSheetParser(p *Parameter) *sheetParser {
	return &sheetParser{param: p}
}

func (sl *sheetParser) SetName(name string) {
	sl.name = name
}

func (sl *sheetParser) Feed(cells []string) {
	sl.cols = append(sl.cols, cells)
	if sl.maxRow < len(cells) {
		sl.maxRow = len(cells)
	}
}

// Parse one xls sheet
func (sl *sheetParser) Parse(ctx context.Context) ([]any, error) {
	colCount := len(sl.cols)
	rp := &rowParser{
		param: sl.param,
		isStr: func(fieldPath string) bool {
			return sl.param.IsStrField(sl.name, fieldPath)
		},
	}
	res := make([]any, 0, sl.maxRow)
	dataRows := make([][]string, 0, sl.maxRow)
	for i := 0; i < sl.maxRow; i++ {
		row := make([]string, 0, colCount)
		for _, col := range sl.cols {
			if len(col) > i {
				row = append(row, col[i])
			} else {
				row = append(row, "")
			}
		}
		if sl.param.IsMeta(i, row) {
			rp.Meta(ctx, row)
		} else if sl.param.IsComment(i, row) {
			// comment row, skip
		} else if sl.param.IsData(i, row) {
			dataRows = append(dataRows, row)
		}
	}
	for _, row := range dataRows {
		rowData, err := rp.Parse(ctx, row)
		if err != nil {
			return nil, err
		}
		res = append(res, rowData)
	}
	return res, nil
}
