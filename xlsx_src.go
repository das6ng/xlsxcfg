package xlsxcfg

import (
	"context"
	"fmt"
	"iter"

	"github.com/xuri/excelize/v2"
)

// SheetResult holds the sheet name and a row iterator for one sheet.
// The Rows iterator yields parsed data rows (map[string]any) one at a time.
// Rows must be fully consumed before the outer iteration advances to the
// next sheet, because the underlying xlsx reader is shared.
type SheetResult struct {
	// Name is the xlsx sheet name (e.g., "Hero", "Item").
	Name string
	// Rows yields parsed data rows one at a time. Each row is a map[string]any
	// matching the proto message structure. Empty rows are skipped.
	Rows iter.Seq2[map[string]any, error]
}

// IterXlsxFiles returns an iterator that yields one SheetResult per sheet
// across all provided xlsx files. Sheets are streamed one at a time — only
// one sheet's rows are held in memory at any point.
//
// Duplicate sheet names across files produce an error.
//
// Usage:
//
//	for sr, err := range xlsxcfg.IterXlsxFiles(ctx, param, "data.xlsx") {
//	    if err != nil { ... }
//	    fmt.Println("Sheet:", sr.Name)
//	    for row, err := range sr.Rows {
//	        if err != nil { ... }
//	        process(row)
//	    }
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

				var rowIter iter.Seq2[map[string]any, error]
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

// makeRowIter creates a row iterator that streams parsed data rows one at a time.
// It reads rows from excelize, classifies each (meta/comment/data), processes the
// meta row through the rowParser to build closure mappings, then yields each data
// row as it's parsed. Each call creates its own rowParser with isolated state,
// avoiding closure-in-loop variable capture issues.
func makeRowIter(ctx context.Context, param *Config, rows *excelize.Rows, typeName string) iter.Seq2[map[string]any, error] {
	// Per-sheet state — each call to makeRowIter gets its own copy.
	rp := newRowParser(typeName, param)
	rowNum := 0
	hasMeta := false

	return func(yield func(map[string]any, error) bool) {
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
				// skip comment rows
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
// is treated as one logical row. It mirrors makeRowIter but uses excelize.Cols to
// iterate column-wise. The meta row index (meta_row), comment rows, and data_row_start
// apply to column indices in this mode.
func makeColIter(ctx context.Context, param *Config, cols *excelize.Cols, typeName string) iter.Seq2[map[string]any, error] {
	rp := newRowParser(typeName, param)
	colNum := 0
	hasMeta := false

	return func(yield func(map[string]any, error) bool) {
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
				// skip comment columns
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

// LoadXlsxFiles loads and parses multiple Excel files, collecting all sheet data
// into a single map keyed by sheet name. This is a convenience wrapper around
// IterXlsxFiles that eagerly loads all rows into memory.
//
// For streaming (one row at a time), use IterXlsxFiles instead.
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
