// Package writer writes parsed xlsx sheet data to files in various formats:
// raw JSON, raw msgpack, proto-validated JSON, proto-validated msgpack,
// and protobuf binary.
package writer

import (
	"bytes"
	"cmp"
	"encoding/json"
	"os"
	"path"
	"slices"
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
	data := xlsxcfg.NewOrderedMap(1)
	data.Set(cfg.Sheet.ListFieldName, rows)
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
	data := xlsxcfg.NewOrderedMap(1)
	data.Set(cfg.Sheet.ListFieldName, rows)
	buf, err := msgpackMarshalOrdered(data)
	if err != nil {
		return "", err
	}
	return writeFile(dir, sheet, ext(cfg.Output.RawMsgpack.Extension, ".msgpack"), buf)
}

// WriteProtoJSON writes proto-validated data as JSON.
// When field_order is "source", columnKeys is used to reorder fields in each row
// to match the xlsx column order.
func WriteProtoJSON(cfg *xlsxcfg.ConfigFile, sheet string, msg proto.Message, columnKeys []string) (string, error) {
	dir := cfg.ResolveDir(cfg.Output.JSON.Dir)
	opts := protojson.MarshalOptions{
		Multiline: cfg.Output.JSON.Indent != "",
		Indent:    cfg.Output.JSON.Indent,
	}
	buf, err := opts.Marshal(msg)
	if err != nil {
		return "", err
	}
	if cfg.IsSourceOrder() && len(columnKeys) > 0 {
		buf, err = reorderProtoJSON(buf, cfg.Sheet.ListFieldName, columnKeys)
		if err != nil {
			return "", err
		}
	}
	return writeFile(dir, sheet, ext(cfg.Output.JSON.Extension, ".json"), buf)
}

// WriteProtoMsgpack writes proto-validated data as msgpack via JSON intermediate.
// Uses OrderedMap to preserve field ordering. When field_order is "source",
// columnKeys reorders fields to match xlsx column order.
func WriteProtoMsgpack(cfg *xlsxcfg.ConfigFile, sheet string, msg proto.Message, columnKeys []string) (string, error) {
	dir := cfg.ResolveDir(cfg.Output.Msgpack.Dir)
	jsonBuf, err := protojson.MarshalOptions{}.Marshal(msg)
	if err != nil {
		return "", err
	}
	if cfg.IsSourceOrder() && len(columnKeys) > 0 {
		jsonBuf, err = reorderProtoJSON(jsonBuf, cfg.Sheet.ListFieldName, columnKeys)
		if err != nil {
			return "", err
		}
	}
	var data xlsxcfg.OrderedMap
	if err := json.Unmarshal(jsonBuf, &data); err != nil {
		return "", err
	}
	buf, err := msgpackMarshalOrdered(&data)
	if err != nil {
		return "", err
	}
	return writeFile(dir, sheet, ext(cfg.Output.Msgpack.Extension, ".msgpack"), buf)
}

// reorderProtoJSON reorders the fields within each list element of a proto JSON
// output to match the given columnKeys order. The JSON has the structure:
//
//	{"ListFieldName": [{"Field1": ..., "Field2": ...}, ...]}
//
// Fields not present in columnKeys retain their original position after
// the ordered fields.
func reorderProtoJSON(jsonBuf []byte, listFieldName string, columnKeys []string) ([]byte, error) {
	var top xlsxcfg.OrderedMap
	if err := json.Unmarshal(jsonBuf, &top); err != nil {
		return nil, err
	}
	listVal, ok := top.Get(listFieldName)
	if !ok {
		return jsonBuf, nil // no list field, return as-is
	}
	list, ok := listVal.([]any)
	if !ok {
		return jsonBuf, nil
	}
	// Build a key-order index for fast lookup
	keyOrder := make(map[string]int, len(columnKeys))
	for i, k := range columnKeys {
		if _, exists := keyOrder[k]; !exists {
			keyOrder[k] = i
		}
	}
	for i, item := range list {
		rowMap, ok := item.(*xlsxcfg.OrderedMap)
		if !ok {
			continue
		}
		list[i] = reorderMap(rowMap, keyOrder)
	}
	// Also reorder the top-level keys to place listFieldName first
	topReordered := reorderMap(&top, map[string]int{listFieldName: 0})
	return json.Marshal(topReordered)
}

// reorderMap returns a new OrderedMap with keys reordered: keys present in
// keyOrder come first (in keyOrder index order), then remaining keys in
// their original order.
func reorderMap(om *xlsxcfg.OrderedMap, keyOrder map[string]int) *xlsxcfg.OrderedMap {
	keys := om.Keys()
	// Separate ordered and remaining keys
	ordered := make([]string, 0, len(keys))
	remaining := make([]string, 0, len(keys))
	for _, k := range keys {
		if _, has := keyOrder[k]; has {
			ordered = append(ordered, k)
		} else {
			remaining = append(remaining, k)
		}
	}
	// Sort ordered keys by their keyOrder index
	slices.SortFunc(ordered, func(a, b string) int {
		return cmp.Compare(keyOrder[a], keyOrder[b])
	})
	result := xlsxcfg.NewOrderedMap(len(keys))
	for _, k := range ordered {
		v, _ := om.Get(k)
		result.Set(k, v)
	}
	for _, k := range remaining {
		v, _ := om.Get(k)
		result.Set(k, v)
	}
	return result
}

// WriteProtoBytes writes proto-validated data as protobuf binary.
// Field ordering is always by proto field number (wire format), regardless of
// field_order setting.
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

// msgpackMarshalOrdered serializes a value to msgpack, preserving the key
// ordering of *OrderedMap values. The standard msgpack library doesn't respect
// json.Marshaler, so we manually walk the structure.
func msgpackMarshalOrdered(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	if err := encodeOrdered(enc, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// encodeOrdered recursively encodes a value using the msgpack encoder,
// preserving *OrderedMap key order.
func encodeOrdered(enc *msgpack.Encoder, v any) error {
	if v == nil {
		return enc.EncodeNil()
	}
	switch val := v.(type) {
	case *xlsxcfg.OrderedMap:
		if err := enc.EncodeMapLen(val.Len()); err != nil {
			return err
		}
		for k, v := range val.All() {
			if err := enc.EncodeString(k); err != nil {
				return err
			}
			if err := encodeOrdered(enc, v); err != nil {
				return err
			}
		}
		return nil
	case []any:
		if err := enc.EncodeArrayLen(len(val)); err != nil {
			return err
		}
		for _, item := range val {
			if err := encodeOrdered(enc, item); err != nil {
				return err
			}
		}
		return nil
	case string:
		return enc.EncodeString(val)
	case int:
		return enc.EncodeInt(int64(val))
	case int64:
		return enc.EncodeInt(val)
	case int32:
		return enc.EncodeInt32(val)
	case float64:
		return enc.EncodeFloat64(val)
	case float32:
		return enc.EncodeFloat32(val)
	case bool:
		return enc.EncodeBool(val)
	case uint64:
		return enc.EncodeUint(val)
	case uint32:
		return enc.EncodeUint(uint64(val))
	default:
		return enc.Encode(val)
	}
}
