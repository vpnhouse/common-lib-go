package xttlmap

import (
	"sync"
	"time"
)

type item[V any] struct {
	value    V
	deadline time.Time
}

func (s *item[V]) expired(now time.Time) bool {
	if s.deadline.IsZero() {
		return false
	}

	return now.After(s.deadline)
}

type node[K comparable, V any] struct {
	key  K
	item item[V]
	prev *node[K, V]
	next *node[K, V]
}

type TTLMap[K comparable, V any] struct {
	lock        sync.RWMutex
	items       map[K]*node[K, V]
	head        *node[K, V]
	tail        *node[K, V]
	lastCleanup time.Time
	cleaning    bool
	maxSize     int
}

func New[K comparable, V any](maxSize int) *TTLMap[K, V] {
	store := &TTLMap[K, V]{
		items:       make(map[K]*node[K, V]),
		lastCleanup: time.Now(),
		maxSize:     maxSize,
	}
	return store
}

func (s *TTLMap[K, V]) Resize(maxSize int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if maxSize < s.maxSize {
		for len(s.items) > maxSize {
			s.removeOldest()
		}
	}

	s.maxSize = maxSize
}

func (s *TTLMap[K, V]) Set(key K, value V, deadline time.Time) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if node, exists := s.items[key]; exists {
		node.item.value = value
		node.item.deadline = deadline
		s.moveToFront(node)
		return
	}

	for len(s.items) >= s.maxSize {
		s.removeOldest()
	}

	newNode := &node[K, V]{
		key: key,
		item: item[V]{
			value:    value,
			deadline: deadline,
		},
	}

	s.addToFront(newNode)
	s.items[key] = newNode

	if time.Since(s.lastCleanup) > time.Second && !s.cleaning {
		s.cleaning = true
		go s.cleanupExpired()
		s.lastCleanup = time.Now()
	}
}

func (s *TTLMap[K, V]) Get(key K) (V, bool) {
	s.lock.RLock()
	node, exists := s.items[key]
	s.lock.RUnlock()

	if !exists {
		var zero V
		return zero, false
	}

	now := time.Now()
	if node.item.expired(now) {
		s.lock.Lock()
		defer s.lock.Unlock()

		// Double checking after lock
		if node, exists := s.items[key]; exists && node.item.expired(now) {
			delete(s.items, node.key)
			s.removeNode(node)
		}

		var zero V
		return zero, false
	}

	s.lock.Lock()
	s.moveToFront(node)
	s.lock.Unlock()

	return node.item.value, true
}

func (s *TTLMap[K, V]) Delete(key K) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if node, exists := s.items[key]; exists {
		delete(s.items, node.key)
		s.removeNode(node)
	}

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

	now := time.Now()

	expiredKeys := make([]K, 0)
	s.lock.RLock()
	for key, node := range s.items {
		if node.item.expired(now) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	s.lock.RUnlock()

	if len(expiredKeys) > 0 {
		s.lock.Lock()
		defer s.lock.Unlock()

		for _, key := range expiredKeys {
			if node, exists := s.items[key]; exists && node.item.expired(now) {
				delete(s.items, node.key)
				s.removeNode(node)
			}
		}
	}
}

func (s *TTLMap[K, V]) addToFront(node *node[K, V]) {
	node.next = s.head
	node.prev = nil
	if s.head != nil {
		s.head.prev = node
	}
	s.head = node
	if s.tail == nil {
		s.tail = node
	}
}

func (s *TTLMap[K, V]) removeNode(node *node[K, V]) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		s.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		s.tail = node.prev
	}
}

func (s *TTLMap[K, V]) moveToFront(node *node[K, V]) {
	if node == s.head {
		return
	}
	s.removeNode(node)
	s.addToFront(node)
}

func (s *TTLMap[K, V]) removeOldest() {
	if s.tail != nil {
		delete(s.items, s.tail.key)
		s.removeNode(s.tail)
	}
}
