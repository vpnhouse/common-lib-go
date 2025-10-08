package timewait

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeWait_BasicFunctionality(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tw := NewTimeWait[int, string](ctx, time.Millisecond, time.Millisecond*100)

	evicted := []bool{false, false, false}
	tw.Push(0, "zero", func(k int, v string) {
		evicted[k] = true
		assert.Equal(t, "zero", v)
	})
	tw.Push(1, "one", func(k int, v string) {
		evicted[k] = true
		assert.Equal(t, "one", v)
	})
	time.Sleep(time.Millisecond * 50)
	tw.Push(2, "two", func(k int, v string) {
		evicted[k] = true
		assert.Equal(t, "two", v)
	})
	time.Sleep(time.Millisecond * 70)

	assert.Equal(t, 1, len(tw.byId))
	assert.Equal(t, 1, tw.byOrder.Len())

	assert.True(t, evicted[0])
	assert.True(t, evicted[1])

	value, found := tw.Pop(2)
	assert.True(t, found)
	assert.Equal(t, "two", value)

	assert.Equal(t, 0, len(tw.byId))
	assert.Equal(t, 0, tw.byOrder.Len())
}
