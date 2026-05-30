package flat_fields

import (
	"testing"

	"github.com/das6ng/xlsxcfg/tests/testutil"
	"github.com/stretchr/testify/assert"
)

func TestFlatFields(t *testing.T) {
	data := testutil.LoadFixture(t, "xlsxcfg.yaml", "flat_fields.xlsx")
	rows, ok := data["Flat"]
	assert.True(t, ok, "expected 'Flat' sheet in output")
	assert.Equal(t, 7, len(rows), "expected 7 data rows")

	// Row 1: all fields populated
	row1 := rows[0].(map[string]any)
	assert.Equal(t, int64(1), row1["ID"])
	assert.Equal(t, int64(100), row1["Count"])
	assert.Equal(t, "Alice", row1["Name"])
	assert.Equal(t, int64(1), row1["Active"])
	assert.Equal(t, int64(95), row1["Score"])

	// Row 2: negative values
	row2 := rows[1].(map[string]any)
	assert.Equal(t, int64(2), row2["ID"])
	assert.Equal(t, int64(-50), row2["Count"])
	assert.Equal(t, "Bob", row2["Name"])
	assert.Equal(t, int64(0), row2["Active"])
	assert.Equal(t, int64(80), row2["Score"])

	// Row 3: zero values
	row3 := rows[2].(map[string]any)
	assert.Equal(t, int64(3), row3["ID"])
	assert.Equal(t, int64(0), row3["Count"])
	assert.Equal(t, "Charlie", row3["Name"])
	assert.Equal(t, int64(1), row3["Active"])
	assert.Equal(t, int64(0), row3["Score"])

	// Row 4: empty numeric cells are absent (not set to int64(0))
	row4 := rows[3].(map[string]any)
	assert.Equal(t, int64(4), row4["ID"])
	_, hasCount := row4["Count"]
	assert.False(t, hasCount, "empty numeric field should not be set")
	assert.Equal(t, "Diana", row4["Name"])
	_, hasActive := row4["Active"]
	assert.False(t, hasActive, "empty numeric field should not be set")
	_, hasScore := row4["Score"]
	assert.False(t, hasScore, "empty numeric field should not be set")

	// Row 5: large value and negative score
	row5 := rows[4].(map[string]any)
	assert.Equal(t, int64(5), row5["ID"])
	assert.Equal(t, int64(999999), row5["Count"])
	assert.Equal(t, "Eve", row5["Name"])
	assert.Equal(t, int64(1), row5["Active"])
	assert.Equal(t, int64(-10), row5["Score"])

	// Row 6: empty string field (Name is not set)
	row6 := rows[5].(map[string]any)
	assert.Equal(t, int64(6), row6["ID"])
	assert.Equal(t, int64(1), row6["Count"])
	_, hasName := row6["Name"]
	assert.False(t, hasName, "empty string field should not be set")
	assert.Equal(t, int64(0), row6["Active"])
	assert.Equal(t, int64(100), row6["Score"])

	// Row 7: negative count, empty active
	row7 := rows[6].(map[string]any)
	assert.Equal(t, int64(7), row7["ID"])
	assert.Equal(t, int64(-1), row7["Count"])
	assert.Equal(t, "Grace", row7["Name"])
	_, hasActive7 := row7["Active"]
	assert.False(t, hasActive7, "empty numeric field should not be set")
	assert.Equal(t, int64(50), row7["Score"])
}
