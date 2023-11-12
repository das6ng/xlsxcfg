package xlsxcfg

import (
	"bytes"
	"strconv"
)

type tokenReader struct {
	buf []byte
	idx int

	curr    []byte
	listIdx int
	err     error
}

func newTokenReader(src string) *tokenReader {
	return &tokenReader{buf: []byte(src)}
}

func (r *tokenReader) HasNext() bool {
	return r.idx < len(r.buf)
}

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

func (r *tokenReader) FullPrev() string {
	i := bytes.LastIndex(r.buf[:r.idx], r.curr)
	if i <= 0 {
		return ""
	}
	return string(r.buf[:i-1])
}

func (r *tokenReader) FullIdent() string {
	return string(bytes.TrimSuffix(r.buf[:r.idx], []byte{'.'}))
}

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
