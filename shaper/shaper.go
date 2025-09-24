package shaper

import (
	"context"
	"sync"
	"time"
)

type TimeBucket struct {
	lock  sync.Mutex
	ctx   context.Context
	depth time.Time
	burst time.Duration
	kiBps int
}

func NewTimeBucket(ctx context.Context, speedKiBps, burstKiB int) *TimeBucket {
	if speedKiBps <= 0 || burstKiB <= 0 {
		panic("speedKiBps and burstBytes must be positive")
	}

	return &TimeBucket{
		ctx:   ctx,
		burst: time.Millisecond * time.Duration((burstKiB<<10)/speedKiBps),
		kiBps: speedKiBps,
	}
}

func (s *TimeBucket) Shape(length int) bool {
	if s == nil {
		return true
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	now := time.Now()
	if now.Sub(s.depth) > s.burst {
		s.depth = now.Add(-s.burst)
	}

	need := time.Microsecond * time.Duration((length<<10)/s.kiBps)
	s.depth = s.depth.Add(need)
	if now.After(s.depth) {
		return true
	}

	await := s.depth.Sub(now)

	select {
	case <-s.ctx.Done():
		return false
	case <-time.After(await):
		return true
	}
}
