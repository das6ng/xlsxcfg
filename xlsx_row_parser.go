// Package xlsxcfg provides the core library for converting Excel (.xlsx) sheets
// into Protocol Buffer-defined config data (JSON + protobuf binary).
//
// The parsing pipeline reads sheets column-wise (as excelize provides), transposes
// to rows, then uses a closure-based cell-to-field mapping system to build nested
// map structures that mirror the proto schema.
package xlsxcfg

import (
	"context"
	"fmt"
	"strconv"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// rowParser transforms Excel data rows into nested map[string]any structures
// that mirror proto message layouts. During Meta(), it processes the header row
// to build a system of getter/setter closures that know how to navigate and
// populate the nested map for each column. During Parse(), it invokes those
// closures with actual cell values to produce one row's data.
//
// The map representation uses:
//   - map[string]any for proto message structs (keys = PascalCase field names)
//   - []any for repeated/list fields (auto-expanded to fit the highest index)
//
// This design avoids reflection entirely — closures are built once during Meta()
// and reused for every data row, making Parse() fast.
type rowParser struct {
	// param holds the parsed Config (from xlsxcfg.yaml), used to classify rows
	// and determine field types.
	param *Config

	// isStr reports whether the proto field at the given dot-separated typePath
	// is a string type. Non-string fields are parsed as int64. This closure
	// captures the proto type name so callers don't need to pass it each time.
	isStr func(fieldPath string) bool

	// getFieldDesc returns the proto FieldDescriptor for the leaf field at the
	// given dot-separated typePath within the parser's message type. Returns nil
	// if the path does not resolve to a field.
	getFieldDesc func(fieldPath string) protoreflect.FieldDescriptor

	// errStack accumulates parse errors (e.g., non-numeric value in an int field)
	// during a single Parse() call. Only the first error is returned to the caller.
	errStack []error

	// hasMeta indicates whether Meta() has been called. Parse() will fail if
	// the header row has not been processed first.
	hasMeta bool

	// res is the current row's data as a nested map. It is reset at the start
	// of each Parse() call and populated by the setter closures in set[].
	res map[string]any

	// set is an ordered slice of setter closures, one per column. Each closure
	// knows how to navigate the nested map (using fnGet/fnSet) and write the
	// cell value at the correct location. Built during Meta().
	set []func(v string)

	// fnGet maps a full ident path (e.g., "Phone.Region") to a getter closure
	// that returns the current value at that path in res. Closures lazily create
	// intermediate maps/slices as needed to support forward references.
	fnGet map[string]func() any

	// fnSet maps a full ident path to a setter closure that writes a value at
	// that path in res. Used by metaStruct and metaList to wire up intermediate
	// nodes, and indirectly by set[] for terminal leaf values.
	fnSet map[string]func(v any)
}

// newRowParser creates a rowParser for the given proto message type name.
// The isStr closure is bound to typeName so that field-type lookups during
// convertVal are automatic.
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

// Meta processes the header (metadata) row to build the column-to-field mapping.
// Each cell in the header row defines how the corresponding column in data rows
// maps into the nested result map. Header cells use dot-separated paths like
// "Phone.Region" for nested structs and "#N" tokens for list indices (1-based
// in the header, converted to 0-based internally).
//
// For each header cell, tokenReader breaks the path into segments. The parser
// then dispatches to one of three handlers:
//   - metaList:  segment has a list index (e.g., "Tags.#1") — creates closures
//     that navigate into and auto-expand a []any slice.
//   - metaStruct: segment is an intermediate struct node — creates getter/setter
//     closures that lazily create the nested map.
//   - metaField:  segment is a terminal leaf field — appends a setter closure
//     to set[] that writes the converted cell value at parse time.
//
// This method must be called before Parse(). It is typically called once per sheet,
// and the resulting closures are reused for every data row.
func (p *rowParser) Meta(ctx context.Context, row []string) {
	p.fnGet = make(map[string]func() any, len(row))
	p.fnSet = make(map[string]func(v any), len(row))
	p.set = make([]func(v string), 0, len(row))
	p.res = make(map[string]any)
	// The root getter: empty string key returns the top-level result map.
	// All other fnGet/fnSet closures chain back to this via prevIdent.
	p.fnGet[""] = func() any { return p.res }
	for _, meta := range row {
		// Empty header cell — column has no mapping; append a no-op setter
		// so the column indices stay aligned in set[].
		if meta == "" {
			p.set = append(p.set, func(_ string) {})
			continue
		}
		tr := newTokenReader(meta)
		for tr.Next() {
			ident := tr.Ident()           // current segment name (e.g., "Region")
			prevIdent := tr.FullPrev()    // full path of parent (e.g., "Phone")
			fullIdent := tr.FullIdent()   // full path including this segment (e.g., "Phone.Region")
			typePath := tr.TypePath()     // proto type path for field-type resolution
			// Dispatch based on the kind of segment:
			//   - list index (#N) → metaList (handles slice navigation + auto-expansion)
			//   - intermediate struct → metaStruct (creates getter/setter for nested map)
			//   - terminal leaf → metaField (appends value-setting closure to set[])
			if ai := tr.ListIndex(); ai >= 0 {
				// Convert 1-based header index to 0-based slice index.
				// isEnd indicates this is the last token (i.e., the list element
				// itself is a terminal value, not a struct to navigate into).
				p.metaList(ident, prevIdent, fullIdent, typePath, ai-1, !tr.HasNext())
			} else if tr.HasNext() {
				// Intermediate struct node — more tokens follow, so register
				// getter/setter for navigating into this nested map.
				p.metaStruct(ident, prevIdent, fullIdent, typePath)
			} else {
				// Terminal leaf field — no more tokens, append a setter that
				// writes the converted cell value at this path.
				p.metaField(ident, prevIdent, fullIdent, typePath)
			}
		}
	}
	p.hasMeta = true
}

// Parse processes one data row using the closures built by Meta().
// It resets the result map, trims trailing empty cells for efficiency,
// then invokes each setter closure with the corresponding cell value.
//
// Returns nil (not an empty map) for rows where all cells are empty.
// Must be called after Meta(); returns an error otherwise.
func (p *rowParser) Parse(ctx context.Context, row []string) (map[string]any, error) {
	if !p.hasMeta {
		return nil, fmt.Errorf("row parser no metadata")
	}
	// Check for context cancellation before doing work.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	// Fresh result map for this row. The closures from Meta() reference p.res,
	// so they will write into this new map.
	p.res = make(map[string]any)
	p.errStack = nil

	// Trim trailing empty cells — avoids invoking setters for trailing columns
	// that have no data, which is common when sheets have many optional fields.
	ni := len(row) - 1
	for ; ni >= 0; ni-- {
		if row[ni] != "" {
			break
		}
	}
	// Entire row is empty — skip it.
	if ni < 0 {
		return nil, nil
	}
	row = row[:ni+1]

	// Invoke each column's setter closure. If the row has fewer cells than
	// the header, only the available cells are set (the rest keep zero values).
	for i, cell := range row {
		if i < len(p.set) {
			p.set[i](cell)
		}
	}
	// Return the first conversion error, if any occurred (e.g., non-numeric int).
	if len(p.errStack) > 0 {
		return nil, p.errStack[0]
	}
	return p.res, nil
}

// resolveEnumValue resolves a cell value for a proto enum field. It tries
// ParseInt first (backward compatible with raw integer cells), then falls back
// to looking up the value by name in the enum descriptor.
func resolveEnumValue(fd protoreflect.FieldDescriptor, cellValue string) (int64, error) {
	// Try raw integer first — preserves backward compatibility.
	if n, err := strconv.ParseInt(cellValue, 10, 64); err == nil {
		return n, nil
	}
	// Fall back to enum value name lookup.
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
// the proto field type at typePath. String fields keep the value as-is; enum
// fields try name resolution (with integer fallback); all other fields are
// parsed as int64. Empty values in non-string fields default to "0" so that
// missing cells produce zero rather than a parse error.
func (p *rowParser) convertVal(typePath, fullIdent, v string) any {
	var res any
	if p.isStr(typePath) {
		// String proto fields — keep the raw cell value.
		res = v
	} else if fd := p.getFieldDesc(typePath); fd != nil && fd.Kind() == protoreflect.EnumKind {
		// Enum proto fields — try integer first, then name lookup.
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
		// Non-string, non-enum proto fields (int, bool, etc.) — parse as int64.
		// Default empty cells to "0" to avoid parse errors for optional fields.
		if v == "" {
			v = "0"
		}
		n, e := strconv.ParseInt(v, 10, 64)
		if e != nil {
			// Accumulate errors instead of failing immediately, so we can report
			// the first one after all columns have been processed.
			p.errStack = append(p.errStack, fmt.Errorf("parse col[%s] as number failed: %v", fullIdent, e))
		} else {
			res = n
		}
	}
	return res
}

// metaList registers getter/setter closures for a list (repeated field) segment.
// It handles slice navigation and auto-expansion: if the current slice is nil or
// too small for idx, a new slice is allocated (preserving existing elements) so
// that the element at idx can be read or written.
//
// Parameters:
//   - ident:      field name of the list within its parent struct (e.g., "Tags")
//   - prevIdent:  full path of the parent node (used to look up the parent map)
//   - fullIdent:  full path including this list + index (e.g., "Hero.Tags.#1")
//   - typePath:   proto type path for field-type resolution
//   - idx:        0-based slice index (converted from 1-based header index)
//   - isEnd:      true if this is the last token — the list element is a terminal
//     value (not a struct to navigate further into)
func (p *rowParser) metaList(ident, prevIdent, fullIdent, typePath string, idx int, isEnd bool) {
	// Getter: retrieves the element at idx from the list. Auto-expands the slice
	// if it doesn't exist or is too short, copying existing elements to the new
	// larger slice.
	p.fnGet[fullIdent] = func() any {
		list := p.fnGet[prevIdent]().(map[string]any)[ident]
		if list == nil || len(list.([]any)) <= idx {
			if list != nil {
				// Preserve existing elements when expanding.
				oldList := list.([]any)
				newList := make([]any, idx+1)
				copy(newList, oldList)
				list = newList
			} else {
				list = make([]any, idx+1)
			}
			// Write back the expanded slice to the parent map.
			p.fnGet[prevIdent]().(map[string]any)[ident] = list
		}
		return list.([]any)[idx]
	}

	// Setter: writes a value at idx in the list. Same auto-expansion logic as
	// the getter, then assigns the value at the target index.
	p.fnSet[fullIdent] = func(v any) {
		list := p.fnGet[prevIdent]().(map[string]any)[ident]
		if list == nil || len(list.([]any)) <= idx {
			if list != nil {
				oldList := list.([]any)
				newList := make([]any, idx+1)
				copy(newList, oldList)
				list = newList
			} else {
				list = make([]any, idx+1)
			}
			p.fnGet[prevIdent]().(map[string]any)[ident] = list
		}
		list.([]any)[idx] = v
	}

	// If this is not the last token (e.g., "Items.#1.Name"), the list element
	// is a struct — other handlers will register further closures for the
	// sub-fields. Only when isEnd is true do we add a setter to set[] for
	// the terminal value.
	if !isEnd {
		return
	}

	// Terminal list element — append a setter closure that converts and writes
	// the cell value at this list index. Empty cells are skipped so that
	// sparsely-indexed lists don't get zero-filled unnecessarily.
	p.set = append(p.set, func(v string) {
		if v == "" {
			return
		}
		p.fnSet[fullIdent](p.convertVal(typePath, fullIdent, v))
	})
}

// metaStruct registers getter/setter closures for an intermediate struct node
// in the header path (e.g., the "Phone" in "Phone.Region"). These closures
// enable downstream metaField/metaList calls to navigate into the nested map.
//
// The getter simply looks up the ident key in the parent map (may return nil
// if the struct hasn't been created yet). The setter lazily creates the parent
// if needed, then writes the nested map value at ident.
func (p *rowParser) metaStruct(ident, prevIdent, fullIdent, typePath string) {
	// Getter: return the value at ident in the parent map (may be nil).
	p.fnGet[fullIdent] = func() any {
		return p.fnGet[prevIdent]().(map[string]any)[ident]
	}
	// Setter: write a value at ident in the parent map. If the parent itself
	// doesn't exist yet (nil), create it via the parent's setter first.
	p.fnSet[fullIdent] = func(v any) {
		prev := p.fnGet[prevIdent]()
		if prev == nil {
			prev = map[string]any{}
			p.fnSet[prevIdent](prev)
		}
		prev.(map[string]any)[ident] = v
	}
}

// metaField handles a terminal leaf field in the header path. It appends a setter
// closure to set[] that, when invoked during Parse(), will:
//  1. Skip empty cells (the field is left absent in the map).
//  2. Lazily create the parent struct if it doesn't exist yet.
//  3. Convert the cell value (string or int64 based on proto type) and write it.
func (p *rowParser) metaField(ident, prevIdent, fullIdent, typePath string) {
	p.set = append(p.set, func(v string) {
		// Skip empty cells — absent fields remain unset in the map, which
		// translates to proto default values during serialization.
		if v == "" {
			return
		}
		// Ensure the parent struct exists before writing into it.
		prev := p.fnGet[prevIdent]()
		if prev == nil {
			prev = map[string]any{}
			p.fnSet[prevIdent](prev)
		}
		prev.(map[string]any)[ident] = p.convertVal(typePath, fullIdent, v)
	})
}
