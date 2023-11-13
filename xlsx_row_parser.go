package xlsxcfg

import (
	"context"
	"fmt"
	"strconv"
)

type rowParser struct {
	param    *Config
	isStr    func(fieldPath string) bool
	errStack []error
	hasMeta  bool

	res   map[string]any
	set   []func(v string)
	fnGet map[string]func() any
	fnSet map[string]func(v any)
}

func newRowParser(typeName string, param *Config) *rowParser {
	return &rowParser{
		param: param,
		isStr: func(fieldPath string) bool {
			return param.IsStrField(typeName, fieldPath)
		},
	}
}

// Meta feed metadata row to the parser.
func (p *rowParser) Meta(ctx context.Context, row []string) {
	p.fnGet = make(map[string]func() any, len(row))
	p.fnSet = make(map[string]func(v any), len(row))
	p.set = make([]func(v string), 0, len(row))
	p.res = make(map[string]any)
	p.fnGet[""] = func() any { return p.res } // top level getter
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
			// log.Printf("head: %s -- ident: %s -- prevIdent: %s -- fullIdent: %s -- typePath: %s\n", hd, ident, prevIdent, fullIdent, typePath)
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

// Parse parse one data row, MUST called after feed metadata.
func (p *rowParser) Parse(ctx context.Context, row []string) (map[string]any, error) {
	if !p.hasMeta {
		return nil, fmt.Errorf("row parser no metadata")
	}
	p.res = make(map[string]any)
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

func (p *rowParser) convertVal(typePath, fullIdent, v string) any {
	var res any
	if p.isStr(typePath) {
		res = v
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

func (p *rowParser) metaList(ident, prevIdent, fullIdent, typePath string, idx int, isEnd bool) {
	p.fnGet[fullIdent] = func() any {
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
		return list.([]any)[idx]
	}
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
	if !isEnd {
		return
	}
	// end at list item
	p.set = append(p.set, func(v string) {
		// log.Printf("arr set [%s] to [%s]\n", fullIdent, v)
		if v == "" {
			return
		}
		p.fnSet[fullIdent](p.convertVal(typePath, fullIdent, v))
	})
}

func (p *rowParser) metaStruct(ident, prevIdent, fullIdent, typePath string) {
	p.fnGet[fullIdent] = func() any {
		return p.fnGet[prevIdent]().(map[string]any)[ident]
	}
	p.fnSet[fullIdent] = func(v any) {
		prev := p.fnGet[prevIdent]()
		if prev == nil {
			prev = map[string]any{}
			p.fnSet[prevIdent](prev)
		}
		prev.(map[string]any)[ident] = v
	}
}

func (p *rowParser) metaField(ident, prevIdent, fullIdent, typePath string) {
	// end at a field
	p.set = append(p.set, func(v string) {
		// log.Printf("set [%s] to [%s]\n", fullIdent, v)
		if v == "" {
			return
		}
		prev := p.fnGet[prevIdent]()
		if prev == nil {
			prev = map[string]any{}
			p.fnSet[prevIdent](prev)
		}
		prev.(map[string]any)[ident] = p.convertVal(typePath, fullIdent, v)
	})
}
