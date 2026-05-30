// Package app orchestrates the xlsxcfg pipeline: load proto, iterate xlsx sheets,
// collect rows, and write outputs in all configured formats.
package app

import (
	"context"
	"fmt"
	"iter"
	"log"

	"github.com/das6ng/xlsxcfg"
	"github.com/das6ng/xlsxcfg/convert"
	"github.com/das6ng/xlsxcfg/writer"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Run executes the full xlsxcfg pipeline:
//  1. Optionally load proto files.
// 2. Warn about proto-validated formats when proto is disabled.
// 3. Ensure output directories exist.
// 4. Stream sheets from xlsx files and write enabled output formats.
//
// It logs progress and warnings; returns an error for fatal conditions.
func Run(ctx context.Context, cfg *xlsxcfg.ConfigFile, files []string) error {
	var tp xlsxcfg.TypeProvider
	if cfg.Proto.Enabled {
		var err error
		tp, err = xlsxcfg.LoadProtoFiles(ctx, cfg.Proto.ImportPath, cfg.Proto.Files...)
		if err != nil {
			return fmt.Errorf("load proto files: %w", err)
		}
	}

	warnProtoFormats(cfg)
	if err := writer.EnsureOutputDirs(cfg); err != nil {
		return fmt.Errorf("create output dirs: %w", err)
	}

	param := xlsxcfg.NewConfig(cfg, tp)
	for sr, err := range xlsxcfg.IterXlsxFiles(ctx, param, files...) {
		if err != nil {
			return fmt.Errorf("parse xlsx files: %w", err)
		}
		rows := CollectRows(sr.Name, sr.Rows)
		if err := writeSheet(cfg, tp, sr.Name, rows); err != nil {
			return err
		}
	}
	return nil
}

// CollectRows drains the row iterator into a slice.
func CollectRows(name string, rows iter.Seq2[map[string]any, error]) []any {
	var out []any
	for row, err := range rows {
		if err != nil {
			log.Fatalf("sheet[%s] parse row failed: %v\n", name, err)
		}
		out = append(out, row)
	}
	return out
}

func warnProtoFormats(cfg *xlsxcfg.ConfigFile) {
	if cfg.Proto.Enabled {
		return
	}
	for _, f := range []struct {
		name string
		on   bool
	}{
		{"output.json", cfg.Output.JSON.Enabled},
		{"output.msgpack", cfg.Output.Msgpack.Enabled},
		{"output.pb_bytes", cfg.Output.PbBytes.Enabled},
	} {
		if f.on {
			log.Printf("WARNING: %s requires proto to be enabled, skipping\n", f.name)
		}
	}
}

func writeSheet(cfg *xlsxcfg.ConfigFile, tp xlsxcfg.TypeProvider, sheet string, rows []any) error {
	if cfg.Output.RawJSON.Enabled {
		if _, err := writer.WriteRawJSON(cfg, sheet, rows); err != nil {
			return fmt.Errorf("sheet[%s] write raw json: %w", sheet, err)
		}
	}
	if cfg.Output.RawMsgpack.Enabled {
		if _, err := writer.WriteRawMsgpack(cfg, sheet, rows); err != nil {
			return fmt.Errorf("sheet[%s] write raw msgpack: %w", sheet, err)
		}
	}

	if !cfg.Proto.Enabled || tp == nil {
		return nil
	}

	msg, err := BuildProtoMessage(cfg, tp, sheet, rows)
	if err != nil {
		return err
	}
	if msg == nil {
		return nil
	}

	if cfg.Output.JSON.Enabled {
		if _, err := writer.WriteProtoJSON(cfg, sheet, msg); err != nil {
			return fmt.Errorf("sheet[%s] write proto json: %w", sheet, err)
		}
	}
	if cfg.Output.Msgpack.Enabled {
		if _, err := writer.WriteProtoMsgpack(cfg, sheet, msg); err != nil {
			return fmt.Errorf("sheet[%s] write proto msgpack: %w", sheet, err)
		}
	}
	if cfg.Output.PbBytes.Enabled {
		if _, err := writer.WriteProtoBytes(cfg, sheet, msg); err != nil {
			return fmt.Errorf("sheet[%s] write proto bytes: %w", sheet, err)
		}
	}
	return nil
}

// BuildProtoMessage creates a dynamic proto message populated with sheet rows.
// Returns nil (no error) when the proto message type is not found.
func BuildProtoMessage(cfg *xlsxcfg.ConfigFile, tp xlsxcfg.TypeProvider, sheet string, rows []any) (proto.Message, error) {
	sheetTypeName := sheet + cfg.Sheet.TypeSuffix
	md := tp.MessageByName(sheetTypeName)
	if md == nil {
		log.Printf("sheet[%s] cannot find proto: %s\n", sheet, sheetTypeName)
		return nil, nil
	}
	msg := dynamicpb.NewMessage(md)
	data := map[string]any{cfg.Sheet.ListFieldName: rows}
	if err := convert.MapToProto(data, msg); err != nil {
		return nil, fmt.Errorf("sheet[%s] populate proto message: %w", sheet, err)
	}
	return msg, nil
}
