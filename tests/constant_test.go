package tests

import (
	"context"
	"testing"

	"github.com/dashengyeah/xlsxcfg/constant"
	"github.com/stretchr/testify/assert"
)

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
