package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRxTx(t *testing.T) {
	rx := uint64(1000)
	tx := uint64(2000)

	d := RxTx(rx, tx, nil)
	assert.Equal(t, len(d), 16)

	d1 := AddRxTx(123, 321, d)
	assert.Equal(t, d, d1)

	rx1, tx1 := ParseRxTx(d1)
	assert.Equal(t, rx1, uint64(1123))
	assert.Equal(t, tx1, uint64(2321))
}
