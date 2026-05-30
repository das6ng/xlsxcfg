package tests

import (
	"context"
	"testing"

	"github.com/das6ng/xlsxcfg/constant"
	"github.com/stretchr/testify/assert"
)

// TestLoad verifies the constant package's end-to-end loading and lookup flow.
//
// Setup:
//   - Loads "constant.xlsx" with SkipRows=1 (skips the header row).
//   - Uses "#" as the comment prefix, so keys starting with "#" are ignored.
//   - Configures reference quote delimiters "[" and "]" so that Get("[key]")
//     strips the brackets and looks up "key".
//
// Assertions:
//   - Loading completes without error.
//   - data.Get("[类型1]") returns value "1" with ok=true, confirming that:
//     1. The header row was skipped (SkipRows=1).
//     2. The key was not treated as a comment.
//     3. RefQuote delimiters were correctly stripped during lookup.
func TestLoad(t *testing.T) {
	c := &constant.Config{
		Enabled:  true,
		SkipRows: 1,
		Comment:  "#",
		Files:    []string{"constant.xlsx"},
	}
	c.RefQuote.L = "["
	c.RefQuote.R = "]"
	data, err := constant.Load(context.Background(), c)
	assert.Nil(t, err)
	v, o := data.Get("[类型1]")
	assert.Equal(t, true, o)
	assert.Equal(t, "1", v)
}
