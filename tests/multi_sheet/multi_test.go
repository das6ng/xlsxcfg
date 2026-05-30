package multi_sheet

import (
	"testing"

	"github.com/das6ng/xlsxcfg/tests/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMultiSheet(t *testing.T) {
	data := testutil.LoadFixture(t, "xlsxcfg.yaml", "multi.xlsx")

	// Hero sheet
	heroRows, ok := data["Hero"]
	assert.True(t, ok, "expected 'Hero' sheet in output")
	assert.Equal(t, 3, len(heroRows), "expected 3 Hero rows")

	hero1 := heroRows[0].(map[string]any)
	assert.Equal(t, int64(1), hero1["ID"])
	assert.Equal(t, "Warrior", hero1["Name"])
	assert.Equal(t, int64(10), hero1["Level"])

	hero2 := heroRows[1].(map[string]any)
	assert.Equal(t, int64(2), hero2["ID"])
	assert.Equal(t, "Mage", hero2["Name"])
	assert.Equal(t, int64(5), hero2["Level"])

	hero3 := heroRows[2].(map[string]any)
	assert.Equal(t, int64(3), hero3["ID"])
	assert.Equal(t, "Rogue", hero3["Name"])
	assert.Equal(t, int64(8), hero3["Level"])

	// Item sheet
	itemRows, ok := data["Item"]
	assert.True(t, ok, "expected 'Item' sheet in output")
	assert.Equal(t, 3, len(itemRows), "expected 3 Item rows")

	item1 := itemRows[0].(map[string]any)
	assert.Equal(t, int64(1), item1["ID"])
	assert.Equal(t, "Sword", item1["Name"])
	assert.Equal(t, int64(100), item1["Price"])

	item2 := itemRows[1].(map[string]any)
	assert.Equal(t, int64(2), item2["ID"])
	assert.Equal(t, "Shield", item2["Name"])
	assert.Equal(t, int64(50), item2["Price"])

	item3 := itemRows[2].(map[string]any)
	assert.Equal(t, int64(3), item3["ID"])
	assert.Equal(t, "Potion", item3["Name"])
	assert.Equal(t, int64(10), item3["Price"])
}
