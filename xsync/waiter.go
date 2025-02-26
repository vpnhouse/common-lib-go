package xsync

import (
	"context"
	"sync"
)

type Cond struct {
	locker  sync.Locker
	waiters int
	notify  chan struct{}
}

func NewCond(locker sync.Locker) *Cond {
	return &Cond{
		locker: locker,
		notify: make(chan struct{}),
	}
}

// Wait() must be called under the lock
func (w *Cond) Wait(ctx context.Context) bool {
	w.waiters += 1
	w.locker.Unlock()
	defer w.locker.Lock()

	select {
	case _, ok := <-w.notify:
		return ok
	case <-ctx.Done():
		return false
	}
}

// Broadcast() must be called under the lock
func (w *Cond) Broadcast() {
	for w.waiters > 0 {
		w.Signal()
	}
}

// Signal() must be called under the lock
func (w *Cond) Signal() {
	select {
	case w.notify <- struct{}{}:
	default:
	}
	w.waiters -= 1
}

func (w *Cond) Destroy() {
	close(w.notify)
}
