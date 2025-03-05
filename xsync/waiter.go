package xsync

import (
	"context"
	"sync"
)

type Cond struct {
	notify  chan struct{}
	waiters int
}

func NewCond() *Cond {
	return &Cond{
		notify: make(chan struct{}, 1),
	}
}

// Wait() must be called under the lock
func (w *Cond) Wait(ctx context.Context, lock *sync.Mutex) error {
	w.waiters++
	lock.Unlock()

	select {
	case <-w.notify:
		lock.Lock()
		w.waiters--
		return nil
	case <-ctx.Done():
		lock.Lock()
		w.waiters--
		return ctx.Err()
	}
}

// Signal() must be called under the lock
func (w *Cond) Signal() {
	if w.waiters > 0 {
		select {
		case w.notify <- struct{}{}:
		default:
		}
	}
}

// Broadcast() must be called under the lock
func (w *Cond) Broadcast() {
	for range w.waiters {
		w.Signal()
	}
}

func (w *Cond) Destroy() {
	close(w.notify)
}
