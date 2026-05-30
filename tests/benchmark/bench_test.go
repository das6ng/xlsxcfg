package benchmark

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/das6ng/xlsxcfg"
	"github.com/xuri/excelize/v2"
)

// benchRowCount returns the number of rows for benchmark fixtures,
// configurable via XLSXCFG_BENCH_ROWS environment variable (default 10000).
func benchRowCount() int {
	if v := os.Getenv("XLSXCFG_BENCH_ROWS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 10000
}

// generateBenchXlsx creates a temporary xlsx file with the given number of data rows.
// The sheet is named "Bench" to match BenchSheetRow proto message.
// Layout: row 1 = comment (empty), row 2 = meta headers, row 3+ = data.
// Columns: ID, Name, Addr.City, Tags#1, Tags#2
func generateBenchXlsx(t testing.TB, rowCount int) string {
	t.Helper()
	f := excelize.NewFile()
	sheet := "Bench"
	f.SetSheetName("Sheet1", sheet)

	// Row 1: comment (empty)
	// Row 2: meta headers
	headers := []string{"ID", "Name", "Addr.City", "Tags#1", "Tags#2"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheet, cell, h)
	}

	// Data rows (row 3 to row 2+rowCount)
	for i := 0; i < rowCount; i++ {
		row := i + 3
		id := i + 1

		// ID
		cell, _ := excelize.CoordinatesToCellName(1, row)
		f.SetCellValue(sheet, cell, strconv.Itoa(id))

		// Name
		cell, _ = excelize.CoordinatesToCellName(2, row)
		f.SetCellValue(sheet, cell, fmt.Sprintf("Name_%d", id))

		// Addr.City
		cell, _ = excelize.CoordinatesToCellName(3, row)
		f.SetCellValue(sheet, cell, fmt.Sprintf("City_%d", id%100))

		// Tags#1
		cell, _ = excelize.CoordinatesToCellName(4, row)
		f.SetCellValue(sheet, cell, fmt.Sprintf("tag_%d_a", id%10))

		// Tags#2
		cell, _ = excelize.CoordinatesToCellName(5, row)
		f.SetCellValue(sheet, cell, fmt.Sprintf("tag_%d_b", id%10))
	}

	path := fmt.Sprintf("bench_%d_rows.xlsx", rowCount)
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("generate bench xlsx failed: %v", err)
	}
	f.Close()
	return path
}

// BenchmarkParseLargeSheet measures xlsx parsing performance.
// Config loading and proto compilation are done once before the timed loop
// so only the xlsx parsing itself is measured.
func BenchmarkParseLargeSheet(b *testing.B) {
	rowCount := benchRowCount()
	xlsxPath := generateBenchXlsx(b, rowCount)
	defer os.Remove(xlsxPath)

	// Setup: load config and compile proto once (not measured)
	ctx := context.Background()
	cfg, err := xlsxcfg.ConfigFromFile("xlsxcfg.yaml")
	if err != nil {
		b.Fatalf("read config failed: %v", err)
	}
	tp, err := xlsxcfg.LoadProtoFiles(ctx, cfg.Proto.ImportPath, cfg.Proto.Files...)
	if err != nil {
		b.Fatalf("load proto failed: %v", err)
	}
	param := xlsxcfg.NewConfig(cfg, tp)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := xlsxcfg.LoadXlsxFiles(ctx, param, xlsxPath)
		if err != nil {
			b.Fatalf("load xlsx failed: %v", err)
		}
		rows, ok := data["Bench"]
		if !ok {
			b.Fatal("expected 'Bench' sheet in output")
		}
		if len(rows) != rowCount {
			b.Fatalf("expected %d rows, got %d", rowCount, len(rows))
		}
	}
}

// TestBenchFixtureGeneration verifies that the benchmark fixture can be generated
// and parsed correctly with a small number of rows.
func TestBenchFixtureGeneration(t *testing.T) {
	rowCount := 100
	xlsxPath := generateBenchXlsx(t, rowCount)
	defer os.Remove(xlsxPath)

	ctx := context.Background()
	cfg, err := xlsxcfg.ConfigFromFile("xlsxcfg.yaml")
	if err != nil {
		t.Fatalf("read config failed: %v", err)
	}
	tp, err := xlsxcfg.LoadProtoFiles(ctx, cfg.Proto.ImportPath, cfg.Proto.Files...)
	if err != nil {
		t.Fatalf("load proto failed: %v", err)
	}
	data, err := xlsxcfg.LoadXlsxFiles(ctx, xlsxcfg.NewConfig(cfg, tp), xlsxPath)
	if err != nil {
		t.Fatalf("load xlsx failed: %v", err)
	}

	rows, ok := data["Bench"]
	if !ok {
		t.Fatal("expected 'Bench' sheet in output")
	}
	if len(rows) != rowCount {
		t.Fatalf("expected %d rows, got %d", rowCount, len(rows))
	}

	// Verify first row structure
	row1 := rows[0].(map[string]any)
	if row1["ID"] != int64(1) {
		t.Errorf("expected ID=1, got %v", row1["ID"])
	}
	if row1["Name"] != "Name_1" {
		t.Errorf("expected Name=Name_1, got %v", row1["Name"])
	}
}
