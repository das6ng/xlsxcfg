package repeated_fields

import (
	"testing"

	"github.com/das6ng/xlsxcfg/tests/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRepeatedFields(t *testing.T) {
	data := testutil.LoadFixture(t, "xlsxcfg.yaml", "repeated.xlsx")
	rows, ok := data["Repeated"]
	assert.True(t, ok, "expected 'Repeated' sheet in output")
	assert.Equal(t, 4, len(rows), "expected 4 data rows")

	// Row 1: Alpha — 3 tags, 2 phones (both with region, first with number)
	row1 := rows[0].(map[string]any)
	assert.Equal(t, int64(1), row1["ID"])
	assert.Equal(t, "Alpha", row1["Name"])

	tags1 := row1["Tags"].([]any)
	assert.Equal(t, 3, len(tags1))
	assert.Equal(t, "go", tags1[0])
	assert.Equal(t, "rust", tags1[1])
	assert.Equal(t, "python", tags1[2])

	phones1 := row1["Phones"].([]any)
	assert.Equal(t, 2, len(phones1))
	phone1_1 := phones1[0].(map[string]any)
	assert.Equal(t, int64(86), phone1_1["Region"])
	assert.Equal(t, int64(1310001), phone1_1["No"])
	phone1_2 := phones1[1].(map[string]any)
	assert.Equal(t, int64(1), phone1_2["Region"])

	// Row 2: Beta — 1 tag, 2 phones
	row2 := rows[1].(map[string]any)
	assert.Equal(t, int64(2), row2["ID"])
	assert.Equal(t, "Beta", row2["Name"])

	tags2 := row2["Tags"].([]any)
	assert.Equal(t, 1, len(tags2))
	assert.Equal(t, "java", tags2[0])

	phones2 := row2["Phones"].([]any)
	assert.Equal(t, 2, len(phones2))
	phone2_1 := phones2[0].(map[string]any)
	assert.Equal(t, int64(1), phone2_1["Region"])
	assert.Equal(t, int64(2020001), phone2_1["No"])
	phone2_2 := phones2[1].(map[string]any)
	assert.Equal(t, int64(44), phone2_2["Region"])

	// Row 3: Gamma — 2 tags, 1 phone (second phone region is empty)
	row3 := rows[2].(map[string]any)
	assert.Equal(t, int64(3), row3["ID"])
	assert.Equal(t, "Gamma", row3["Name"])

	tags3 := row3["Tags"].([]any)
	assert.Equal(t, 2, len(tags3))
	assert.Equal(t, "c++", tags3[0])
	assert.Equal(t, "c", tags3[1])

	phones3 := row3["Phones"].([]any)
	assert.Equal(t, 1, len(phones3))
	phone3_1 := phones3[0].(map[string]any)
	assert.Equal(t, int64(86), phone3_1["Region"])
	assert.Equal(t, int64(1310003), phone3_1["No"])

	// Row 4: Delta — empty tags, no phones (all cells empty)
	row4 := rows[3].(map[string]any)
	assert.Equal(t, int64(4), row4["ID"])
	assert.Equal(t, "Delta", row4["Name"])

	_, hasTags := row4["Tags"]
	assert.False(t, hasTags, "empty repeated string field should not be set")
	_, hasPhones := row4["Phones"]
	assert.False(t, hasPhones, "empty repeated message field should not be set")
}
