package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRxTx(t *testing.T) {
	rx := int64(1000)
	tx := int64(2000)

	d := RxTx(rx, tx, nil)
	assert.Equal(t, len(d), 16)

	d1 := IncRxTx(123, 321, d)
	assert.Equal(t, d, d1)

	rx1, tx1 := ParseRxTx(d1)
	assert.Equal(t, rx1, int64(1123))
	assert.Equal(t, tx1, int64(2321))
}
