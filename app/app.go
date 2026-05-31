// Package app orchestrates the xlsxcfg pipeline.
package app

import (
	"context"
	"fmt"
	"iter"
	"log"

	"github.com/das6ng/xlsxcfg"
	"github.com/das6ng/xlsxcfg/constant"
	"github.com/das6ng/xlsxcfg/convert"
	"github.com/das6ng/xlsxcfg/writer"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Run executes the full pipeline: load proto/constants, stream xlsx sheets,
// and write all enabled output formats.
func Run(ctx context.Context, cfg *xlsxcfg.ConfigFile, files []string) error {
	var tp xlsxcfg.TypeProvider
	if cfg.Proto.Enabled {
		var err error
		tp, err = xlsxcfg.LoadProtoFiles(ctx, cfg.Proto.ImportPath, cfg.Proto.Files...)
		if err != nil {
			return fmt.Errorf("load proto files: %w", err)
		}
	}

	// Load constant key-value data for value replacement.
	constData, err := constant.Load(ctx, &cfg.Constant)
	if err != nil {
		return fmt.Errorf("load constants: %w", err)
	}

	warnProtoFormats(cfg)
	if err := writer.EnsureOutputDirs(cfg); err != nil {
		return fmt.Errorf("create output dirs: %w", err)
	}

	param := xlsxcfg.NewConfig(cfg, tp)
	param.ConstData = constData
	for sr, err := range xlsxcfg.IterXlsxFiles(ctx, param, files...) {
		if err != nil {
			return fmt.Errorf("parse xlsx files: %w", err)
		}
		rows, err := CollectRows(sr.Name, sr.Rows)
		if err != nil {
			return fmt.Errorf("sheet[%s]: %w", sr.Name, err)
		}
		if err := writeSheet(cfg, tp, sr.Name, rows); err != nil {
			return err
		}
	}
	return nil
}

// CollectRows drains the row iterator into a slice.
func CollectRows(name string, rows iter.Seq2[*xlsxcfg.OrderedMap, error]) ([]any, error) {
	var out []any
	for row, err := range rows {
		if err != nil {
			return nil, fmt.Errorf("parse row failed: %w", err)
		}
		out = append(out, row)
	}
	return out, nil
}

// warnProtoFormats logs warnings when proto-validated formats are enabled but proto is disabled.
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

// writeSheet writes a single sheet's data in all enabled output formats.
func writeSheet(cfg *xlsxcfg.ConfigFile, tp xlsxcfg.TypeProvider, sheet string, rows []any) error {
	cleanSheet := cfg.StripTransposeMark(sheet)

	// Extract xlsx column key order from the first row for source-order proto output.
	var columnKeys []string
	if cfg.IsSourceOrder() && len(rows) > 0 {
		if om, ok := rows[0].(*xlsxcfg.OrderedMap); ok {
			columnKeys = om.Keys()
		}
	}

	if cfg.Output.RawJSON.Enabled {
		if _, err := writer.WriteRawJSON(cfg, cleanSheet, rows); err != nil {
			return fmt.Errorf("sheet[%s] write raw json: %w", sheet, err)
		}
	}
	if cfg.Output.RawMsgpack.Enabled {
		if _, err := writer.WriteRawMsgpack(cfg, cleanSheet, rows); err != nil {
			return fmt.Errorf("sheet[%s] write raw msgpack: %w", sheet, err)
		}
	}

	if !cfg.Proto.Enabled || tp == nil {
		return nil
	}

	msg, err := BuildProtoMessage(cfg, tp, cleanSheet, rows)
	if err != nil {
		return err
	}
	if msg == nil {
		return nil
	}

	if cfg.Output.JSON.Enabled {
		if _, err := writer.WriteProtoJSON(cfg, cleanSheet, msg, columnKeys); err != nil {
			return fmt.Errorf("sheet[%s] write proto json: %w", sheet, err)
		}
	}
	if cfg.Output.Msgpack.Enabled {
		if _, err := writer.WriteProtoMsgpack(cfg, cleanSheet, msg, columnKeys); err != nil {
			return fmt.Errorf("sheet[%s] write proto msgpack: %w", sheet, err)
		}
	}
	if cfg.Output.PbBytes.Enabled {
		if _, err := writer.WriteProtoBytes(cfg, cleanSheet, msg); err != nil {
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
	data := xlsxcfg.NewOrderedMap(1)
	data.Set(cfg.Sheet.ListFieldName, rows)
	if err := convert.MapToProto(data, msg); err != nil {
		return nil, fmt.Errorf("sheet[%s] populate proto message: %w", sheet, err)
	}
	return msg, nil
}
