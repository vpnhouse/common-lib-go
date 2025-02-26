package xsync

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	LoadCapacity             = 1024 * 1024
	LoadMaxWaitInterval      = time.Microsecond
	LoadMaxBroadcastInterval = time.Microsecond * time.Duration(100)
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

func TestHighLoad(t *testing.T) {
	fmt.Println("Testing random load")
	var (
		locker   = &sync.Mutex{}
		xcond    = NewCond(locker)
		wg       sync.WaitGroup
		capacity = LoadCapacity
		started  = time.Now()
	)

	//Broadcaster
	go func() {
		for capacity > 0 {
			time.Sleep(time.Duration(rand.Int63n(int64(LoadMaxBroadcastInterval))))
			locker.Lock()
			xcond.Broadcast()
			locker.Unlock()
		}
	}()

	var timeouts atomic.Int32
	for capacity > 0 {
		time.Sleep(time.Duration(rand.Int63n(int64(LoadMaxWaitInterval))))
		capacity -= 1
		wg.Add(1)
		go func() {
			locker.Lock()
			ctx, _ := context.WithTimeout(context.Background(), LoadMaxBroadcastInterval*9/10)
			result := xcond.Wait(ctx)
			if !result {
				timeouts.Add(1)
			}
			locker.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Zero(t, xcond.waiters)
	fmt.Println("Load test done,", timeouts.Load(), "timeouts out of", LoadCapacity, "duration", time.Since(started))

}
