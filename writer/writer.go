// Package writer writes parsed xlsx sheet data to files in various formats:
// raw JSON, raw msgpack, proto-validated JSON, proto-validated msgpack,
// and protobuf binary.
package writer

import (
	"encoding/json"
	"os"
	"path"
	"strings"

	"github.com/das6ng/xlsxcfg"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ext returns the file extension from config, adding a leading dot if needed.
// Falls back to the given default when config extension is empty.
func ext(cfgExt, defaultExt string) string {
	if cfgExt == "" {
		return defaultExt
	}
	if !strings.HasPrefix(cfgExt, ".") {
		return "." + cfgExt
	}
	return cfgExt
}

// WriteRawJSON writes sheet data as JSON without proto validation.
// Returns the output file path.
func WriteRawJSON(cfg *xlsxcfg.ConfigFile, sheet string, rows []any) (string, error) {
	dir := cfg.ResolveDir(cfg.Output.RawJSON.Dir)
	data := map[string]any{cfg.Sheet.ListFieldName: rows}
	var buf []byte
	var err error
	if cfg.Output.RawJSON.Indent != "" {
		buf, err = json.MarshalIndent(data, "", cfg.Output.RawJSON.Indent)
	} else {
		buf, err = json.Marshal(data)
	}
	if err != nil {
		return "", err
	}
	return writeFile(dir, sheet, ext(cfg.Output.RawJSON.Extension, ".json"), buf)
}

// WriteRawMsgpack writes sheet data as msgpack without proto validation.
func WriteRawMsgpack(cfg *xlsxcfg.ConfigFile, sheet string, rows []any) (string, error) {
	dir := cfg.ResolveDir(cfg.Output.RawMsgpack.Dir)
	data := map[string]any{cfg.Sheet.ListFieldName: rows}
	buf, err := msgpack.Marshal(data)
	if err != nil {
		return "", err
	}
	return writeFile(dir, sheet, ext(cfg.Output.RawMsgpack.Extension, ".msgpack"), buf)
}

// WriteProtoJSON writes proto-validated data as JSON.
func WriteProtoJSON(cfg *xlsxcfg.ConfigFile, sheet string, msg proto.Message) (string, error) {
	dir := cfg.ResolveDir(cfg.Output.JSON.Dir)
	opts := protojson.MarshalOptions{
		Multiline: cfg.Output.JSON.Indent != "",
		Indent:    cfg.Output.JSON.Indent,
	}
	buf, err := opts.Marshal(msg)
	if err != nil {
		return "", err
	}
	return writeFile(dir, sheet, ext(cfg.Output.JSON.Extension, ".json"), buf)
}

// WriteProtoMsgpack writes proto-validated data as msgpack via JSON intermediate.
func WriteProtoMsgpack(cfg *xlsxcfg.ConfigFile, sheet string, msg proto.Message) (string, error) {
	dir := cfg.ResolveDir(cfg.Output.Msgpack.Dir)
	jsonBuf, err := protojson.MarshalOptions{}.Marshal(msg)
	if err != nil {
		return "", err
	}
	var data map[string]any
	if err := json.Unmarshal(jsonBuf, &data); err != nil {
		return "", err
	}
	buf, err := msgpack.Marshal(data)
	if err != nil {
		return "", err
	}
	return writeFile(dir, sheet, ext(cfg.Output.Msgpack.Extension, ".msgpack"), buf)
}

// WriteProtoBytes writes proto-validated data as protobuf binary.
func WriteProtoBytes(cfg *xlsxcfg.ConfigFile, sheet string, msg proto.Message) (string, error) {
	dir := cfg.ResolveDir(cfg.Output.PbBytes.Dir)
	buf, err := proto.Marshal(msg)
	if err != nil {
		return "", err
	}
	return writeFile(dir, sheet, ext(cfg.Output.PbBytes.Extension, ".bytes"), buf)
}

// EnsureOutputDirs creates all configured output directories.
func EnsureOutputDirs(cfg *xlsxcfg.ConfigFile) error {
	dirs := []string{
		cfg.ResolveDir(cfg.Output.RawJSON.Dir),
		cfg.ResolveDir(cfg.Output.RawMsgpack.Dir),
		cfg.ResolveDir(cfg.Output.JSON.Dir),
		cfg.ResolveDir(cfg.Output.Msgpack.Dir),
		cfg.ResolveDir(cfg.Output.PbBytes.Dir),
	}
	for _, d := range dirs {
		if d != "" {
			if err := os.MkdirAll(d, 0755); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeFile writes data to dir/sheet+ext and returns the output file path.
func writeFile(dir, sheet, ext string, data []byte) (string, error) {
	outFile := path.Join(dir, sheet+ext)
	return outFile, os.WriteFile(outFile, data, 0644)
}
