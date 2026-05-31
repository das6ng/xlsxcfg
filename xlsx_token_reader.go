package xlsxcfg

import (
	"bytes"
	"strconv"
)

// tokenReader parses dot-separated cell header paths:
//
//   - Dot-separated segments represent nested message fields: "Phone.Region"
//   - A "#N" suffix indicates a 1-based list index: "Items#1.Name"
//
// Example walkthrough for input "Items#1.Name":
//   - Next() → ident="Items", listIdx=1
//   - Next() → ident="Name",  listIdx=-1
//   - FullIdent() → "Items#1.Name"
//   - TypePath()  → "Items.Name"  (list indices stripped, for proto field lookup)
type tokenReader struct {
	buf []byte
	idx int

	curr []byte
	// listIdx holds the parsed list index from a "#N" suffix, or -1 if none.
	listIdx int
	err error
}

func newTokenReader(src string) *tokenReader {
	return &tokenReader{buf: []byte(src)}
}

func (r *tokenReader) HasNext() bool {
	return r.idx < len(r.buf)
}

// Next advances to the next dot-separated segment, extracting the identifier
// and optional "#N" list index. Returns false at end of input.
func (r *tokenReader) Next() bool {
	r.curr = nil
	r.listIdx = -1
	if r.idx >= len(r.buf) {
		return false
	}
	leftBuf := r.buf[r.idx:]
	i := bytes.IndexByte(leftBuf, '.')
	if i > 0 {
		r.curr = leftBuf[:i]
		r.idx += i + 1
	} else {
		r.curr = leftBuf[:]
		r.idx = len(r.buf)
	}
	i = bytes.IndexByte(r.curr, '#')
	if i > 0 {
		idxBuf := r.curr[i+1:]
		r.curr = r.curr[:i]
		n, e := strconv.ParseInt(string(idxBuf), 10, 64)
		if e != nil {
			r.err = e
		} else {
			r.listIdx = int(n)
		}
	}
	return true
}

func (r *tokenReader) Ident() string {
	return string(r.curr)
}

func (r *tokenReader) ListIndex() int {
	return r.listIdx
}

// FullPrev returns the path before the current segment (e.g., "Items#1." when
// positioned at "Name"). Returns "" at root level.
func (r *tokenReader) FullPrev() string {
	i := bytes.LastIndex(r.buf[:r.idx], r.curr)
	if i <= 0 {
		return ""
	}
	return string(r.buf[:i-1])
}

// FullIdent returns the full dot-separated path consumed so far, including "#N" markers.
func (r *tokenReader) FullIdent() string {
	return string(bytes.TrimSuffix(r.buf[:r.idx], []byte{'.'}))
}

// TypePath returns the consumed path with "#N" list indices stripped,
// yielding a clean field path for proto field lookup.
func (r *tokenReader) TypePath() string {
	buf := make([]byte, r.idx+1)
	copy(buf, r.buf[:r.idx])
	for i := 0; i < len(buf); i++ {
		if buf[i] != '#' {
			continue
		}
		j := i + 1
		for ; j < len(buf); j++ {
			if buf[j] == '.' {
				break
			}
		}
		if j < len(buf) {
			buf = append(buf[:i], buf[j:]...)
		} else {
			buf = buf[:i]
		}
	}
	return string(bytes.Trim(buf, "\x00"))
}
