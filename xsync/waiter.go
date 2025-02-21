package xsync

import (
	"context"
	"sync"
)

type Cond struct {
	locker sync.Locker
	notify chan struct{}
}

func NewCond(locker sync.Locker) *Cond {
	return &Cond{
		locker: locker,
		notify: make(chan struct{}),
	}
}

func (w *Cond) Wait(ctx context.Context) bool {
	w.locker.Unlock()
	defer w.locker.Lock()

	select {
	case _, ok := <-w.notify:
		return ok
	case <-ctx.Done():
		return false
	}
}

func (w *Cond) Broadcast() {
	for {
		if !w.Signal() {
			return
		}
	}
}

func (w *Cond) Signal() bool {
	select {
	case w.notify <- struct{}{}:
		return true
	default:
		return false
	}
}

func (w *Cond) Destroy() {
	w.locker.Lock()
	defer w.locker.Unlock()

	close(w.notify)
}
