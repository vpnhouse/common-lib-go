package xttlmap

import (
	"sync"
	"time"
)

type item[V any] struct {
	value   V
	created time.Time
	ttl     time.Duration
}

type TTLMap[K comparable, V any] struct {
	lock        sync.RWMutex
	items       map[K]item[V]
	stop        chan struct{}
	stopped     bool
	lastCleanup time.Time
	cleaning    bool
}

func New[K comparable, V any]() *TTLMap[K, V] {
	store := &TTLMap[K, V]{
		items:       make(map[K]item[V]),
		stop:        make(chan struct{}),
		lastCleanup: time.Now(),
	}
	return store
}

func (s *TTLMap[K, V]) Set(key K, value V, ttl time.Duration) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.stopped {
		return
	}

	s.items[key] = item[V]{
		value:   value,
		created: time.Now(),
		ttl:     ttl,
	}

	if time.Since(s.lastCleanup) > time.Second && !s.cleaning {
		s.cleaning = true
		go s.cleanupExpired()
		s.lastCleanup = time.Now()
	}
}

func (s *TTLMap[K, V]) Get(key K) (V, bool) {
	s.lock.RLock()
	if s.stopped {
		s.lock.RUnlock()
		var zero V
		return zero, false
	}

	item, exists := s.items[key]
	s.lock.RUnlock()

	if !exists {
		var zero V
		return zero, false
	}

	if time.Since(item.created) > item.ttl {
		s.lock.Lock()
		defer s.lock.Unlock()

		if item, exists := s.items[key]; exists && time.Since(item.created) > item.ttl {
			delete(s.items, key)
		}

		var zero V
		return zero, false
	}

	return item.value, true
}

func (s *TTLMap[K, V]) Delete(key K) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.stopped {
		return
	}

	delete(s.items, key)

	if time.Since(s.lastCleanup) > time.Second && !s.cleaning {
		s.cleaning = true
		go s.cleanupExpired()
		s.lastCleanup = time.Now()
	}
}

func (s *TTLMap[K, V]) cleanupExpired() {
	defer func() {
		s.lock.Lock()
		s.cleaning = false
		s.lock.Unlock()
	}()

	expiredKeys := make([]K, 0)
	s.lock.RLock()
	now := time.Now()
	for key, item := range s.items {
		if now.Sub(item.created) > item.ttl {
			expiredKeys = append(expiredKeys, key)
		}
	}
	s.lock.RUnlock()

	if len(expiredKeys) > 0 {
		s.lock.Lock()
		defer s.lock.Unlock()

		now := time.Now()
		for _, key := range expiredKeys {
			if item, exists := s.items[key]; exists && now.Sub(item.created) > item.ttl {
				delete(s.items, key)
			}
		}
	}
}

func (s *TTLMap[K, V]) Stop() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.stopped = true
	close(s.stop)
}
