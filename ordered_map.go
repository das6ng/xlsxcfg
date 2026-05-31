package xlsxcfg

import (
	"bytes"
	"encoding/json"
	"iter"
	"reflect"
	"slices"
	"strings"
)

// OrderedMap is a map[string]any that preserves key insertion order.
// It implements json.Marshaler and json.Unmarshaler. The zero value is ready to use.
type OrderedMap struct {
	keys   []string
	values []any
	index  map[string]int // key → position in keys/values
}

// NewOrderedMap creates an OrderedMap pre-allocated for n entries.
func NewOrderedMap(n int) *OrderedMap {
	return &OrderedMap{
		keys:   make([]string, 0, n),
		values: make([]any, 0, n),
		index:  make(map[string]int, n),
	}
}

// Set stores a key-value pair, updating in place if the key exists.
func (m *OrderedMap) Set(key string, val any) {
	if m.index == nil {
		m.index = make(map[string]int)
	}
	if i, ok := m.index[key]; ok {
		m.values[i] = val
		return
	}
	m.index[key] = len(m.keys)
	m.keys = append(m.keys, key)
	m.values = append(m.values, val)
}

// Get returns the value for a key. The second return indicates whether the key was found.
func (m *OrderedMap) Get(key string) (any, bool) {
	if m.index == nil {
		return nil, false
	}
	i, ok := m.index[key]
	if !ok {
		return nil, false
	}
	return m.values[i], true
}

// Keys returns the keys in insertion order.
func (m *OrderedMap) Keys() []string {	return slices.Clone(m.keys)
}

// Len returns the number of entries.
func (m *OrderedMap) Len() int {	return len(m.keys)
}

// All returns an iterator over key-value pairs in insertion order.
func (m *OrderedMap) All() iter.Seq2[string, any] {	return func(yield func(string, any) bool) {
		for i, k := range m.keys {
			if !yield(k, m.values[i]) {
				return
			}
		}
	}
}

// MarshalJSON implements json.Marshaler. Keys are written in insertion order.
func (m *OrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range m.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		vb, err := json.Marshal(m.values[i])
		if err != nil {
			return nil, err
		}
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// UnmarshalJSON implements json.Unmarshaler, preserving key order.
func (m *OrderedMap) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if t != json.Delim('{') {
		return &json.UnmarshalTypeError{Value: "object", Type: reflect.TypeOf((*OrderedMap)(nil))}
	}
	*m = OrderedMap{
		index: make(map[string]int),
	}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := t.(string)
		if !ok {
			return &json.UnmarshalTypeError{Value: "object key", Type: reflect.TypeOf((*OrderedMap)(nil))}
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return err
		}
		val, err := unmarshalRaw(raw)
		if err != nil {
			return err
		}
		m.Set(key, val)
	}
	if _, err := dec.Token(); err != nil {
		return err
	}
	return nil
}

// unmarshalRaw decodes a json.RawMessage into a Go value, producing *OrderedMap for objects.
func unmarshalRaw(raw json.RawMessage) (any, error) {
	trimmed := strings.TrimSpace(string(raw))
	switch {
	case trimmed == "null":
		return nil, nil
	case trimmed == "true":
		return true, nil
	case trimmed == "false":
		return false, nil
	case trimmed[0] == '"':
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, err
		}
		return s, nil
	case trimmed[0] == '{':
		var om OrderedMap
		if err := json.Unmarshal(raw, &om); err != nil {
			return nil, err
		}
		return &om, nil
	case trimmed[0] == '[':
		var arr []any
		dec := json.NewDecoder(bytes.NewReader(raw))
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		for dec.More() {
			var elem json.RawMessage
			if err := dec.Decode(&elem); err != nil {
				return nil, err
			}
			v, err := unmarshalRaw(elem)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		return arr, nil
	default:
		var n float64
		if err := json.Unmarshal(raw, &n); err != nil {
			return nil, err
		}
		return n, nil
	}
}
