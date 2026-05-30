package xlsxcfg

import (
	"bytes"
	"strconv"
)

// tokenReader is a lightweight scanner that parses dot-separated cell header
// paths used in xlsx column headers. Headers use a path notation to address
// nested proto fields and list indices:
//
//   - Dot-separated segments represent nested message fields: "Phone.Region"
//   - A "#N" suffix on a segment indicates a 1-based list index: "Items#1.Name"
//     (stored internally as 0-based index after parsing)
//
// Example walkthrough for input "Items#1.Name":
//   - Next() → ident="Items", listIdx=1
//   - Next() → ident="Name",  listIdx=-1
//   - FullIdent() → "Items#1.Name"
//   - TypePath()  → "Items.Name"  (list indices stripped, for proto field lookup)
type tokenReader struct {
	// buf is the full source string being scanned.
	buf []byte
	// idx is the current read position in buf, advancing past each dot-separated segment.
	idx int

	// curr is the current segment's identifier (without the #index suffix).
	curr []byte
	// listIdx holds the parsed list index from a "#N" suffix, or -1 if no index was present.
	listIdx int
	// err holds any parsing error from the most recent Next() call.
	err error
}

// newTokenReader creates a tokenReader for the given dot-separated header path.
func newTokenReader(src string) *tokenReader {
	return &tokenReader{buf: []byte(src)}
}

// HasNext reports whether there are remaining bytes to scan.
func (r *tokenReader) HasNext() bool {
	return r.idx < len(r.buf)
}

// Next advances the scanner to the next dot-separated segment. It extracts the
// identifier and optional list index ("#N") from that segment.
//
// Returns false if the end of the input has been reached.
// After a successful call, use Ident() and ListIndex() to retrieve the parsed values.
func (r *tokenReader) Next() bool {
	r.curr = nil
	r.listIdx = -1 // Reset: -1 means "no list index" for this segment.
	if r.idx >= len(r.buf) {
		return false
	}
	leftBuf := r.buf[r.idx:]
	// Find the next dot separator; everything before it is the current segment.
	i := bytes.IndexByte(leftBuf, '.')
	if i > 0 {
		r.curr = leftBuf[:i]
		r.idx += i + 1 // advance past the dot
	} else {
		// No more dots — the remainder is the last segment.
		r.curr = leftBuf[:]
		r.idx = len(r.buf)
	}
	// Check for a "#N" list index suffix within the current segment.
	i = bytes.IndexByte(r.curr, '#')
	if i > 0 {
		idxBuf := r.curr[i+1:]
		r.curr = r.curr[:i] // strip the "#N" suffix from the identifier
		n, e := strconv.ParseInt(string(idxBuf), 10, 64)
		if e != nil {
			r.err = e
		} else {
			r.listIdx = int(n)
		}
	}
	return true
}

// Ident returns the field name of the current segment (without any "#N" suffix).
func (r *tokenReader) Ident() string {
	return string(r.curr)
}

// ListIndex returns the parsed list index for the current segment, or -1 if
// the segment had no "#N" suffix.
func (r *tokenReader) ListIndex() int {
	return r.listIdx
}

// FullPrev returns the path of the parent — everything before the current
// segment's identifier. For example, after scanning "Items#1.Name" and being
// positioned at "Name", FullPrev returns "Items#1.".
//
// Returns an empty string if the current segment is the first (root-level).
func (r *tokenReader) FullPrev() string {
	i := bytes.LastIndex(r.buf[:r.idx], r.curr)
	if i <= 0 {
		return ""
	}
	// Everything up to (but not including) the dot before the current ident.
	return string(r.buf[:i-1])
}

// FullIdent returns the full dot-separated path consumed so far by all Next()
// calls, including any "#N" list index markers. Trailing dots are trimmed.
//
// For input "Phone.Region" after two Next() calls, returns "Phone.Region".
func (r *tokenReader) FullIdent() string {
	return string(bytes.TrimSuffix(r.buf[:r.idx], []byte{'.'}))
}

// TypePath returns the consumed path with all list index segments stripped,
// yielding a clean dot-separated field path suitable for proto field lookup
// (since proto field names don't include list indices).
//
// For input "Items#1.Name", returns "Items.Name".
// For input "Phone.Region", returns "Phone.Region" (unchanged).
func (r *tokenReader) TypePath() string {
	// Copy the consumed portion of buf so we can modify it in-place.
	buf := make([]byte, r.idx+1)
	copy(buf, r.buf[:r.idx])
	// Remove all "#N" segments by finding '#' and cutting through the next '.' (or end).
	for i := 0; i < len(buf); i++ {
		if buf[i] != '#' {
			continue
		}
		// Find the end of the "#N" suffix — either a dot or end of buffer.
		j := i + 1
		for ; j < len(buf); j++ {
			if buf[j] == '.' {
				break
			}
		}
		if j < len(buf) {
			// Cut from '#' up to and including the dot: "Items#1.Name" → "Items.Name"
			buf = append(buf[:i], buf[j:]...)
		} else {
			// "#N" is at the end with no trailing dot: just truncate.
			buf = buf[:i]
		}
	}
	return string(bytes.Trim(buf, "\x00"))
}
