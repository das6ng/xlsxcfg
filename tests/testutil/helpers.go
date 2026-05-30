package testutil

import (
	"context"
	"testing"

	"github.com/das6ng/xlsxcfg"
)

// LoadFixture loads an xlsx file through the full xlsxcfg pipeline using the given config file.
// configPath is the path to the xlsxcfg.yaml file. xlsxFiles are paths to xlsx files to load.
func LoadFixture(t *testing.T, configPath string, xlsxFiles ...string) map[string][]any {
	t.Helper()
	ctx := context.Background()

	cfg, err := xlsxcfg.ConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("read config %s failed: %v", configPath, err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("invalid config %s: %v", configPath, err)
	}

	typeProvider, err := xlsxcfg.LoadProtoFiles(ctx, cfg.Proto.ImportPath, cfg.Proto.Files...)
	if err != nil {
		t.Fatalf("load proto files failed: %v", err)
	}

	data, err := xlsxcfg.LoadXlsxFiles(ctx, xlsxcfg.NewConfig(cfg, typeProvider), xlsxFiles...)
	if err != nil {
		t.Fatalf("load xlsx files failed: %v", err)
	}
	return data
}
