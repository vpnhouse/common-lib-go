package xsync

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func round(t *testing.T, ctx context.Context, triggerInterval time.Duration, expected bool) {
	var triggered bool
	var counter int
	var lock sync.Mutex
	xcond := NewCond(&lock)

	go func() {
		for {
			if triggered {
				break
			}
		}

		lock.Lock()
		defer lock.Unlock()
		xcond.Broadcast()
	}()

	var wg sync.WaitGroup
	for idx := 0; idx < 10; idx++ {
		lock.Lock()
		wg.Add(1)
		go func() {
			counter += 1
			result := xcond.Wait(ctx)
			assert.Equal(t, expected, result)
			lock.Unlock()
			func() { counter -= 1 }()
			wg.Done()
		}()
	}

	lock.Lock()
	assert.Equal(t, 10, counter)
	lock.Unlock()

	time.Sleep(triggerInterval)
	triggered = true
	wg.Wait()
	assert.Equal(t, 00, counter)

}

func TestGeneric(t *testing.T) {
	fmt.Println("Test without timeout")
	round(t, context.Background(), time.Millisecond, true)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	fmt.Println("Test with timeout")
	round(t, ctx, time.Second, false)
}
