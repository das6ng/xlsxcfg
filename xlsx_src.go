package xlsxcfg

import (
	"context"
	"fmt"
	"iter"

	"github.com/xuri/excelize/v2"
)

// SheetResult holds a sheet name and a row iterator.
// Rows must be fully consumed before advancing to the next sheet
// because the underlying xlsx reader is shared.
type SheetResult struct {
	Name string                       // xlsx sheet name (e.g., "Hero", "Item")
	Rows iter.Seq2[*OrderedMap, error] // yields parsed data rows; empty rows are skipped
}

// IterXlsxFiles returns an iterator yielding one SheetResult per sheet across all xlsx files.
// Only one sheet's rows are in memory at a time. Duplicate sheet names produce an error.
//
//	for sr, err := range xlsxcfg.IterXlsxFiles(ctx, param, "data.xlsx") {
//	    if err != nil { ... }
//	    for row, err := range sr.Rows { ... }
//	}
func IterXlsxFiles(ctx context.Context, param *Config, files ...string) iter.Seq2[*SheetResult, error] {
	return func(yield func(*SheetResult, error) bool) {
		seen := map[string]bool{}
		for _, xlsFile := range files {
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			default:
			}

			f, err := excelize.OpenFile(xlsFile)
			if err != nil {
				yield(nil, err)
				return
			}

			sheets := f.WorkBook.Sheets.Sheet
			for _, sh := range sheets {
				if seen[sh.Name] {
					f.Close()
					yield(nil, fmt.Errorf("duplicated sheet[%s] in file: %s", sh.Name, xlsFile))
					return
				}
				seen[sh.Name] = true

				cleanName := param.StripTransposeMark(sh.Name)
				typeName := cleanName + param.Sheet.RowTypeSuffix

				var rowIter iter.Seq2[*OrderedMap, error]
				if param.IsTransposed(sh.Name) {
					cols, err := f.Cols(sh.Name)
					if err != nil {
						f.Close()
						yield(nil, err)
						return
					}
					rowIter = makeColIter(ctx, param, cols, typeName)
				} else {
					rows, err := f.Rows(sh.Name)
					if err != nil {
						f.Close()
						yield(nil, err)
						return
					}
					rowIter = makeRowIter(ctx, param, rows, typeName)
				}

				sr := &SheetResult{
					Name: sh.Name,
					Rows: rowIter,
				}
				if !yield(sr, nil) {
					f.Close()
					return
				}
			}
			f.Close()
		}
	}
}

// makeRowIter creates a row iterator that classifies rows (meta/comment/data),
// processes the meta row, then yields parsed data rows. Each call gets its own
// rowParser with isolated state.
func makeRowIter(ctx context.Context, param *Config, rows *excelize.Rows, typeName string) iter.Seq2[*OrderedMap, error] {
	rp := newRowParser(typeName, param)
	rowNum := 0
	hasMeta := false

	return func(yield func(*OrderedMap, error) bool) {
		for rows.Next() {
			cells, err := rows.Columns()
			if err != nil {
				yield(nil, err)
				return
			}
			n := rowNum
			rowNum++

			if param.IsMeta(n, cells) {
				rp.Meta(ctx, cells)
				hasMeta = true
			} else if param.IsComment(n, cells) {
			} else if param.IsData(n, cells) && hasMeta {
				rowData, err := rp.Parse(ctx, cells)
				if err != nil {
					yield(nil, err)
					return
				}
				if rowData != nil {
					if !yield(rowData, nil) {
						return
					}
				}
			}
		}
	}
}

// makeColIter creates a column iterator for transposed sheets, where each column
// is one logical row. Row indices (meta_row, comment_rows, data_row_start) apply
// to column indices in this mode.
func makeColIter(ctx context.Context, param *Config, cols *excelize.Cols, typeName string) iter.Seq2[*OrderedMap, error] {
	rp := newRowParser(typeName, param)
	colNum := 0
	hasMeta := false

	return func(yield func(*OrderedMap, error) bool) {
		for cols.Next() {
			cells, err := cols.Rows()
			if err != nil {
				yield(nil, err)
				return
			}
			n := colNum
			colNum++

			if param.IsMeta(n, cells) {
				rp.Meta(ctx, cells)
				hasMeta = true
			} else if param.IsComment(n, cells) {
			} else if param.IsData(n, cells) && hasMeta {
				rowData, err := rp.Parse(ctx, cells)
				if err != nil {
					yield(nil, err)
					return
				}
				if rowData != nil {
					if !yield(rowData, nil) {
						return
					}
				}
			}
		}
	}
}

// LoadXlsxFiles eagerly loads all sheets from multiple xlsx files into a map keyed
// by sheet name. For streaming, use IterXlsxFiles instead.
func LoadXlsxFiles(ctx context.Context, param *Config, files ...string) (map[string][]any, error) {
	data := map[string][]any{}
	for sr, err := range IterXlsxFiles(ctx, param, files...) {
		if err != nil {
			return nil, err
		}
		rows := make([]any, 0)
		for row, err := range sr.Rows {
			if err != nil {
				return nil, err
			}
			rows = append(rows, row)
		}
		data[sr.Name] = rows
	}
	return data, nil
}
