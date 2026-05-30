package nested_structs

import (
	"testing"

	"github.com/das6ng/xlsxcfg/tests/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNestedStructs(t *testing.T) {
	data := testutil.LoadFixture(t, "xlsxcfg.yaml", "nested.xlsx")
	rows, ok := data["Nested"]
	assert.True(t, ok, "expected 'Nested' sheet in output")
	assert.Equal(t, 4, len(rows), "expected 4 data rows")

	// Row 1: all nested fields populated
	row1 := rows[0].(map[string]any)
	assert.Equal(t, int64(1), row1["ID"])
	assert.Equal(t, "Alice", row1["Name"])

	home1 := row1["Home"].(map[string]any)
	assert.Equal(t, "123 Main St", home1["Street"])
	assert.Equal(t, "NYC", home1["City"])
	assert.Equal(t, "10001", home1["Zip"])

	info1 := row1["Info"].(map[string]any)
	assert.Equal(t, "alice@test.com", info1["Email"])
	assert.Equal(t, "555-0001", info1["Phone"])

	// Row 2
	row2 := rows[1].(map[string]any)
	assert.Equal(t, int64(2), row2["ID"])
	assert.Equal(t, "Bob", row2["Name"])

	home2 := row2["Home"].(map[string]any)
	assert.Equal(t, "456 Oak Ave", home2["Street"])
	assert.Equal(t, "LA", home2["City"])
	assert.Equal(t, "90001", home2["Zip"])

	info2 := row2["Info"].(map[string]any)
	assert.Equal(t, "bob@test.com", info2["Email"])
	assert.Equal(t, "555-0002", info2["Phone"])

	// Row 3
	row3 := rows[2].(map[string]any)
	assert.Equal(t, int64(3), row3["ID"])
	assert.Equal(t, "Charlie", row3["Name"])

	home3 := row3["Home"].(map[string]any)
	assert.Equal(t, "789 Pine Rd", home3["Street"])
	assert.Equal(t, "SF", home3["City"])
	assert.Equal(t, "94102", home3["Zip"])

	info3 := row3["Info"].(map[string]any)
	assert.Equal(t, "charlie@test.com", info3["Email"])
	assert.Equal(t, "555-0003", info3["Phone"])

	// Row 4: Diana — empty nested string fields; Home and Info.Phone absent
	row4 := rows[3].(map[string]any)
	assert.Equal(t, int64(4), row4["ID"])
	assert.Equal(t, "Diana", row4["Name"])

	// All Home sub-fields are empty, so Home struct is never created
	_, hasHome := row4["Home"]
	assert.False(t, hasHome, "Home should not be set when all sub-fields are empty")

	// Info.Email has a value so Info is created, but Phone is absent
	info4 := row4["Info"].(map[string]any)
	assert.Equal(t, "diana@test.com", info4["Email"])
	_, hasPhone := info4["Phone"]
	assert.False(t, hasPhone, "empty string field Info.Phone should not be set")
}
