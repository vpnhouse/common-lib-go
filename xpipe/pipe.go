package xpipe

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/vpnhouse/common-lib-go/xsync"
)

var MaxBufferSize = 16384

type XPipe struct {
	ctx context.Context

	closed bool
	lock   *sync.Mutex
	buffer []byte

	rTrigger *xsync.Cond
	wTrigger *xsync.Cond

	rdeadline time.Time
	wdeadline time.Time
}

func New(ctx context.Context) (*XPipe, error) {
	lock := &sync.Mutex{}
	pipe := &XPipe{
		ctx:      ctx,
		lock:     lock,
		rTrigger: xsync.NewCond(),
		wTrigger: xsync.NewCond(),
	}

	return pipe, nil
}

func (p *XPipe) Available() int {
	p.lock.Lock()
	defer p.lock.Unlock()

	return len(p.buffer)
}

func (p *XPipe) Read(b []byte) (n int, err error) {
	if isDeadlineHappened(&p.rdeadline) {
		return 0, os.ErrDeadlineExceeded
	}

	p.lock.Lock()
	for len(p.buffer) == 0 {
		err = p.wait(p.rTrigger, &p.rdeadline)

		if err != nil {
			if len(p.buffer) == 0 {
				p.lock.Unlock()
				return
			} else {
				break
			}
		}
	}

	defer p.lock.Unlock()

	min := min(len(p.buffer), len(b))
	copy(b, p.buffer[:min])
	p.buffer = p.buffer[min:]
	p.wTrigger.Broadcast()

	return min, nil
}

func (p *XPipe) Write(b []byte) (n int, err error) {
	if isDeadlineHappened(&p.wdeadline) {
		return 0, os.ErrDeadlineExceeded
	}

	if p.closed {
		return 0, os.ErrClosed
	}

	bufferSpaceLeft := func() int {
		return MaxBufferSize - len(p.buffer)
	}

	p.lock.Lock()
	for len(b) > 0 {
		spaceleft := bufferSpaceLeft()
		if spaceleft == 0 {
			err = p.wait(p.wTrigger, &p.wdeadline)
			if err != nil {
				p.lock.Unlock()
				return
			}
			spaceleft = bufferSpaceLeft()
		}

		if spaceleft > 0 {
			min := min(len(b), spaceleft)
			p.buffer = append(p.buffer, b[:min]...)
			b = b[min:]
			n += min
			p.rTrigger.Broadcast()
		}

	}

	p.lock.Unlock()
	return
}

func (p *XPipe) Close() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.close()
}

func (p *XPipe) SetDeadline(t time.Time) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.rdeadline = t
	p.wdeadline = t
	p.rTrigger.Broadcast()
	p.wTrigger.Broadcast()

	return nil
}

func (p *XPipe) SetReadDeadline(t time.Time) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.rdeadline = t
	p.rTrigger.Broadcast()

	return nil
}

func (p *XPipe) SetWriteDeadline(t time.Time) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.wdeadline = t
	p.wTrigger.Broadcast()

	return nil
}

func isDeadlineHappened(deadline *time.Time) bool {
	if deadline.IsZero() {
		return false
	}

	return time.Now().After(*deadline)
}

func (p *XPipe) close() error {
	if p.closed {
		return os.ErrClosed
	}
	p.closed = true
	p.rTrigger.Broadcast()
	p.wTrigger.Broadcast()

	return nil
}

func (p *XPipe) wait(trigger *xsync.Cond, deadline *time.Time) error {
	if isDeadlineHappened(deadline) {
		return os.ErrDeadlineExceeded
	}

	if p.closed {
		return os.ErrClosed
	}

	err := trigger.Wait(p.ctx, p.lock)
	if err != nil {
		p.close()
		return err
	}

	if p.closed {
		return os.ErrClosed
	}

	if isDeadlineHappened(deadline) {
		return os.ErrDeadlineExceeded
	}

	return nil
}
