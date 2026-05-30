package duplicate_sheet

import (
	"context"
	"testing"

	"github.com/das6ng/xlsxcfg"
	"github.com/stretchr/testify/assert"
)

func TestDuplicateSheetAcrossFiles(t *testing.T) {
	ctx := context.Background()
	cfg, err := xlsxcfg.ConfigFromFile("xlsxcfg.yaml")
	assert.NoError(t, err)
	tp, err := xlsxcfg.LoadProtoFiles(ctx, cfg.Proto.ImportPath, cfg.Proto.Files...)
	assert.NoError(t, err)
	_, err = xlsxcfg.LoadXlsxFiles(ctx, xlsxcfg.NewConfig(cfg, tp), "dup1.xlsx", "dup2.xlsx")
	assert.Error(t, err, "expected error when loading two files with the same sheet name")
	assert.Contains(t, err.Error(), "duplicated sheet")
}
