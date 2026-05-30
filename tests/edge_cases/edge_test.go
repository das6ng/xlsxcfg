package edge_cases

import (
	"testing"

	"github.com/das6ng/xlsxcfg/tests/testutil"
	"github.com/stretchr/testify/assert"
)

func TestEdgeCases_SkipEmptyRows(t *testing.T) {
	data := testutil.LoadFixture(t, "xlsxcfg.yaml", "edge.xlsx")
	rows, ok := data["Edge"]
	assert.True(t, ok, "expected 'Edge' sheet in output")
	// All-empty rows (rows 2 and 4) should be skipped, leaving 4 data rows.
	assert.Equal(t, 4, len(rows), "expected 4 data rows (empty rows skipped)")

	// Row 1: ID=1, Label=row1, Value=10
	row1 := rows[0].(map[string]any)
	assert.Equal(t, int64(1), row1["ID"])
	assert.Equal(t, "row1", row1["Label"])
	assert.Equal(t, int64(10), row1["Value"])

	// Row 3: ID=3, Label=row3, Value=0
	row3 := rows[1].(map[string]any)
	assert.Equal(t, int64(3), row3["ID"])
	assert.Equal(t, "row3", row3["Label"])
	assert.Equal(t, int64(0), row3["Value"])

	// Row 5: ID=5, Label=row5, Value=50
	row5 := rows[2].(map[string]any)
	assert.Equal(t, int64(5), row5["ID"])
	assert.Equal(t, "row5", row5["Label"])
	assert.Equal(t, int64(50), row5["Value"])

	// Row 6: ID=6, Label=row6, Value=60
	row6 := rows[3].(map[string]any)
	assert.Equal(t, int64(6), row6["ID"])
	assert.Equal(t, "row6", row6["Label"])
	assert.Equal(t, int64(60), row6["Value"])
}
