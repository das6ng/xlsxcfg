// Package xlsxcfg converts Excel (.xlsx) sheets into Protocol Buffer-defined
// config data. Sheets are read row-wise (or column-wise for transposed sheets),
// then mapped to nested structures via closure-based cell-to-field dispatch.
package xlsxcfg

import (
	"context"
	"fmt"
	"strconv"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// rowParser transforms Excel data rows into nested *OrderedMap structures mirroring
// proto message layouts. During Meta(), it processes the header row to build
// closure-based cell-to-field mappings that avoid reflection. During Parse(), it
// invokes those closures with actual cell values.
type rowParser struct {
	param *Config

	// isStr reports whether the field at the given dot-separated typePath is a
	// string type. Captures the proto type name so callers don't need to pass it.
	isStr func(fieldPath string) bool

	getFieldDesc func(fieldPath string) protoreflect.FieldDescriptor

	// errStack accumulates parse errors during a Parse() call. Only the first error is returned.
	errStack []error

	// hasMeta indicates whether Meta() has been called.
	hasMeta bool

	res *OrderedMap

	// set is an ordered slice of setter closures, one per column. Built during Meta().
	set []func(v string)

	// fnGet maps a full ident path to a getter closure that lazily creates intermediate
	// maps/slices as needed.
	fnGet map[string]func() any

	fnSet map[string]func(v any)
}

// newRowParser creates a rowParser for the given proto message type name.
func newRowParser(typeName string, param *Config) *rowParser {
	return &rowParser{
		param: param,
		isStr: func(fieldPath string) bool {
			return param.IsStrField(typeName, fieldPath)
		},
		getFieldDesc: func(fieldPath string) protoreflect.FieldDescriptor {
			return param.GetFieldDescriptor(typeName, fieldPath)
		},
	}
}

// Meta processes the header row to build column-to-field mappings. Header cells
// use dot-separated paths (e.g., "Phone.Region") for nested structs and "#N"
// tokens for 1-based list indices. Must be called before Parse().
func (p *rowParser) Meta(ctx context.Context, row []string) {
	p.fnGet = make(map[string]func() any, len(row))
	p.fnSet = make(map[string]func(v any), len(row))
	p.set = make([]func(v string), 0, len(row))
	p.res = NewOrderedMap(len(row))
	// Root getter: empty string key returns the top-level result map.
	p.fnGet[""] = func() any { return p.res }
	for _, meta := range row {
		if meta == "" {
			p.set = append(p.set, func(_ string) {})
			continue
		}
		tr := newTokenReader(meta)
		for tr.Next() {
			ident := tr.Ident()
			prevIdent := tr.FullPrev()
			fullIdent := tr.FullIdent()
			typePath := tr.TypePath()
			// Dispatch: list index → metaList, intermediate struct → metaStruct, leaf → metaField.
			if ai := tr.ListIndex(); ai >= 0 {
				p.metaList(ident, prevIdent, fullIdent, typePath, ai-1, !tr.HasNext())
			} else if tr.HasNext() {
				p.metaStruct(ident, prevIdent, fullIdent, typePath)
			} else {
				p.metaField(ident, prevIdent, fullIdent, typePath)
			}
		}
	}
	p.hasMeta = true
}

// Parse processes one data row using the closures built by Meta().
// Returns nil for rows where all cells are empty.
func (p *rowParser) Parse(ctx context.Context, row []string) (*OrderedMap, error) {
	if !p.hasMeta {
		return nil, fmt.Errorf("row parser no metadata")
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	p.res = NewOrderedMap(len(p.set))
	p.errStack = nil

	// Trim trailing empty cells.
	ni := len(row) - 1
	for ; ni >= 0; ni-- {
		if row[ni] != "" {
			break
		}
	}
	if ni < 0 {
		return nil, nil
	}
	row = row[:ni+1]

	for i, cell := range row {
		if i < len(p.set) {
			p.set[i](cell)
		}
	}
	if len(p.errStack) > 0 {
		return nil, p.errStack[0]
	}
	return p.res, nil
}

// resolveEnumValue resolves a cell value for a proto enum field.
// Tries integer parse first (backward compatible), then enum value name lookup.
func resolveEnumValue(fd protoreflect.FieldDescriptor, cellValue string) (int64, error) {
	if n, err := strconv.ParseInt(cellValue, 10, 64); err == nil {
		return n, nil
	}
	ed := fd.Enum()
	if ed == nil {
		return 0, fmt.Errorf("field %q is not an enum", fd.Name())
	}
	evd := ed.Values().ByName(protoreflect.Name(cellValue))
	if evd == nil {
		return 0, fmt.Errorf("enum value %q not found in %s", cellValue, ed.Name())
	}
	return int64(evd.Number()), nil
}

// convertVal converts a cell's string value to the appropriate Go type based on
// the proto field type. Empty values in non-string fields default to "0". Constant
// references (e.g., [Key]) are resolved before type conversion.
func (p *rowParser) convertVal(typePath, fullIdent, v string) any {
	if p.param.ConstData != nil {
		if resolved, ok := p.param.ConstData.Get(v); ok {
			v = resolved
		}
	}
	var res any
	if p.isStr(typePath) {
		res = v
	} else if fd := p.getFieldDesc(typePath); fd != nil && fd.Kind() == protoreflect.EnumKind {
		if v == "" {
			v = "0"
		}
		n, e := resolveEnumValue(fd, v)
		if e != nil {
			p.errStack = append(p.errStack, fmt.Errorf("parse col[%s]: %v", fullIdent, e))
		} else {
			res = n
		}
	} else {
		if v == "" {
			v = "0"
		}
		n, e := strconv.ParseInt(v, 10, 64)
		if e != nil {
			p.errStack = append(p.errStack, fmt.Errorf("parse col[%s] as number failed: %v", fullIdent, e))
		} else {
			res = n
		}
	}
	return res
}

// metaList registers getter/setter closures for a list (repeated field) segment.
// Handles slice navigation and auto-expansion: the slice is grown to fit the
// requested index, preserving existing elements.
func (p *rowParser) metaList(ident, prevIdent, fullIdent, typePath string, idx int, isEnd bool) {
	p.fnGet[fullIdent] = func() any {
		parent := p.fnGet[prevIdent]().(*OrderedMap)
		list, _ := parent.Get(ident)
		if list == nil || len(list.([]any)) <= idx {
			if list != nil {
				oldList := list.([]any)
				newList := make([]any, idx+1)
				copy(newList, oldList)
				list = newList
			} else {
				list = make([]any, idx+1)
			}
			parent.Set(ident, list)
		}
		return list.([]any)[idx]
	}

	p.fnSet[fullIdent] = func(v any) {
		parent := p.fnGet[prevIdent]().(*OrderedMap)
		list, _ := parent.Get(ident)
		if list == nil || len(list.([]any)) <= idx {
			if list != nil {
				oldList := list.([]any)
				newList := make([]any, idx+1)
				copy(newList, oldList)
				list = newList
			} else {
				list = make([]any, idx+1)
			}
			parent.Set(ident, list)
		}
		list.([]any)[idx] = v
	}

	if !isEnd {
		return
	}

	p.set = append(p.set, func(v string) {
		if v == "" {
			return
		}
		p.fnSet[fullIdent](p.convertVal(typePath, fullIdent, v))
	})
}

// metaStruct registers getter/setter closures for an intermediate struct node
// (e.g., "Phone" in "Phone.Region"), enabling downstream handlers to navigate into it.
func (p *rowParser) metaStruct(ident, prevIdent, fullIdent, typePath string) {
	p.fnGet[fullIdent] = func() any {
		v, _ := p.fnGet[prevIdent]().(*OrderedMap).Get(ident)
		return v
	}
	p.fnSet[fullIdent] = func(v any) {
		prev := p.fnGet[prevIdent]()
		if prev == nil {
			prev = NewOrderedMap(4)
			p.fnSet[prevIdent](prev)
		}
		prev.(*OrderedMap).Set(ident, v)
	}
}

// metaField handles a terminal leaf field, appending a setter closure that lazily
// creates the parent struct, converts the cell value, and writes it.
func (p *rowParser) metaField(ident, prevIdent, fullIdent, typePath string) {
	p.set = append(p.set, func(v string) {
		if v == "" {
			return
		}
		prev := p.fnGet[prevIdent]()
		if prev == nil {
			prev = NewOrderedMap(4)
			p.fnSet[prevIdent](prev)
		}
		prev.(*OrderedMap).Set(ident, p.convertVal(typePath, fullIdent, v))
	})
}
