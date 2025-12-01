package timewait

import (
	"context"
	"time"

	"github.com/vpnhouse/common-lib-go/list"
)

type OnEvict[K comparable, V any] func(k K, v V)

type timeWaitRecord[K comparable, V any] struct {
	key      K
	value    V
	deadline time.Time
	evict    OnEvict[K, V]

	element *list.Element[*timeWaitRecord[K, V]]
}

type TimeWait[K comparable, V any] struct {
	ctx     context.Context
	timeout time.Duration
	byId    map[K]*timeWaitRecord[K, V]
	byOrder *list.List[*timeWaitRecord[K, V]]
}

func NewTimeWait[K comparable, V any](ctx context.Context, cleanupInterval, timeout time.Duration) *TimeWait[K, V] {
	timeWait := &TimeWait[K, V]{
		ctx:     ctx,
		timeout: timeout,
		byId:    map[K]*timeWaitRecord[K, V]{},
		byOrder: list.New[*timeWaitRecord[K, V]](),
	}

	go timeWait.run(cleanupInterval)
	return timeWait
}

func (s *TimeWait[K, V]) Push(k K, v V, onEvict OnEvict[K, V]) {
	record := &timeWaitRecord[K, V]{
		key:      k,
		value:    v,
		deadline: time.Now().Add(s.timeout),
		evict:    onEvict,
	}

	record.element = s.byOrder.PushBack(record)
	s.byId[k] = record
}

func (s *TimeWait[K, V]) Pop(k K) (V, bool) {
	record, found := s.byId[k]
	if !found {
		var zero V
		return zero, false
	}

	s.byOrder.Remove(record.element)
	delete(s.byId, record.key)
	return record.value, true
}

func (s *TimeWait[K, V]) cleanup() {
	now := time.Now()
	record := s.byOrder.Front()
	for {
		if record == nil {
			return
		}

		next := record.Next()

		if now.After(record.Value.deadline) {
			record.Value.evict(record.Value.key, record.Value.value)
			s.byOrder.Remove(record)
			delete(s.byId, record.Value.key)
		} else {
			return
		}

		record = next
	}
}

func (s *TimeWait[K, V]) run(interval time.Duration) {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(interval):
			s.cleanup()
		}
	}
}
