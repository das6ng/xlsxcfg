package xlsxcfg

import (
	"context"
	"fmt"
	"strconv"
)

type rowParser struct {
	param    *Parameter
	isStr    func(fieldPath string) bool
	errStack []error
	hasMeta  bool

	res   map[string]any
	fn    []func(v string)
	fnGet map[string]func() any
	fnSet map[string]func(v any)
}

// Meta feed metadata row to the parser.
func (t *rowParser) Meta(ctx context.Context, row []string) {
	t.fnGet = make(map[string]func() any, len(row))
	t.fnSet = make(map[string]func(v any), len(row))
	t.fn = make([]func(v string), 0, len(row))
	t.res = make(map[string]any)
	t.fnGet[""] = func() any { return t.res } // top level getter
	for _, meta := range row {
		if meta == "" {
			t.fn = append(t.fn, func(_ string) {})
			continue
		}
		tr := newTokenReader(meta)
		for tr.Next() {
			ident := tr.Ident()
			prevIdent := tr.FullPrev()
			fullIdent := tr.FullIdent()
			typePath := tr.TypePath()
			// log.Printf("head: %s -- ident: %s -- prevIdent: %s -- fullIdent: %s -- typePath: %s\n", hd, ident, prevIdent, fullIdent, typePath)
			if ai := tr.ListIndex(); ai >= 0 {
				t.metaList(ident, prevIdent, fullIdent, typePath, ai-1)
				if !tr.HasNext() {
					// end at list item
					t.fn = append(t.fn, func(v string) {
						// log.Printf("arr set [%s] to [%s]\n", fullIdent, v)
						t.fnSet[fullIdent](t.convertVal(typePath, fullIdent, v))
					})
				}
			} else if tr.HasNext() {
				t.metaStruct(ident, prevIdent, fullIdent, typePath)
			} else {
				t.metaField(ident, prevIdent, fullIdent, typePath)
			}
		}
	}
	t.hasMeta = true
}

// Parse parse one data row, MUST called after feed metadata.
func (t *rowParser) Parse(ctx context.Context, row []string) (map[string]any, error) {
	if !t.hasMeta {
		return nil, fmt.Errorf("row parser no metadata")
	}
	t.res = make(map[string]any)
	for i, cell := range row {
		if i < len(t.fn) {
			t.fn[i](cell)
		}
	}
	if len(t.errStack) > 0 {
		return nil, t.errStack[0]
	}
	return t.res, nil
}

func (t *rowParser) convertVal(typePath, fullIdent, v string) any {
	var res any
	if t.isStr(typePath) {
		res = v
	} else {
		n, e := strconv.ParseInt(v, 10, 64)
		if e != nil {
			t.errStack = append(t.errStack, fmt.Errorf("parse col[%s] as number failed: %v", fullIdent, e))
		} else {
			res = n
		}
	}
	return res
}

func (t *rowParser) metaList(ident, prevIdent, fullIdent, typePath string, ai int) {
	fullIdentList := ident
	if prevIdent != "" {
		fullIdentList = prevIdent + "." + ident
	}
	getList := t.fnGet[fullIdentList]
	if getList == nil {
		getList = func() any {
			arr := t.fnGet[prevIdent]().(map[string]any)[ident]
			if arr == nil || len(arr.([]any)) <= ai {
				if arr != nil {
					oa := arr.([]any)
					na := make([]any, ai+1)
					copy(na, oa)
					arr = na
				} else {
					arr = make([]any, ai+1)
				}
				t.fnGet[prevIdent]().(map[string]any)[ident] = arr
			}
			return arr
		}
		t.fnGet[fullIdentList] = getList
	}
	t.fnGet[fullIdent] = func() any {
		arr := getList()
		return arr.([]any)[ai]
	}
	t.fnSet[fullIdent] = func(v any) {
		arr := getList()
		arr.([]any)[ai] = v
	}
}

func (t *rowParser) metaStruct(ident, prevIdent, fullIdent, typePath string) {
	t.fnGet[fullIdent] = func() any {
		return t.fnGet[prevIdent]().(map[string]any)[ident]
	}
	t.fnSet[fullIdent] = func(v any) {
		prev := t.fnGet[prevIdent]()
		if prev == nil {
			prev = map[string]any{}
			t.fnSet[prevIdent](prev)
		}
		prev.(map[string]any)[ident] = v
	}
}

func (t *rowParser) metaField(ident, prevIdent, fullIdent, typePath string) {
	// end at a field
	t.fn = append(t.fn, func(v string) {
		// log.Printf("set [%s] to [%s]\n", fullIdent, v)
		prev := t.fnGet[prevIdent]()
		if prev == nil {
			prev = map[string]any{}
			t.fnSet[prevIdent](prev)
		}
		prev.(map[string]any)[ident] = t.convertVal(typePath, fullIdent, v)
	})
}
