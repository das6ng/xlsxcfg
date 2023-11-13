package xlsxcfg

import (
	"context"
	"log"

	"github.com/xuri/excelize/v2"
)

func LoadXlsxFiles(ctx context.Context, param *Config, files ...string) (data map[string][]any, err error) {
	data = map[string][]any{}
	var configData map[string][]any
	for _, xlsFile := range files {
		configData, err = loadXlsxFile(ctx, xlsFile, param)
		if err != nil {
			return
		}
		for sht, d := range configData {
			if data[sht] != nil {
				log.Printf("duplicated sheet[%s] in file: %s", sht, xlsFile)
			}
			data[sht] = d
		}
	}
	return
}

func loadXlsxFile(ctx context.Context, filePath string, param *Config) (data map[string][]any, err error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	sheets := f.WorkBook.Sheets.Sheet
	data = make(map[string][]any, len(sheets))
	for _, sh := range sheets {
		sht, err1 := f.Cols(sh.Name)
		if err1 != nil {
			return nil, err1
		}
		sp := newSheetParser(param)
		sp.SetName(sh.Name + param.Sheet.RowTypeSuffix)
		for sht.Next() {
			cells, err := sht.Rows()
			if err != nil {
				return nil, err
			}
			sp.Feed(cells)
		}
		sheetData, err1 := sp.Parse(ctx)
		if err1 != nil {
			return nil, err1
		}
		data[sh.Name] = sheetData
	}
	return
}
