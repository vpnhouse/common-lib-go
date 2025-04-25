package keycounter

import (
	"sync"
)

type KeyCounter[K comparable] struct {
	lock sync.Mutex
	keys map[K]int
}

func New[K comparable]() *KeyCounter[K] {
	return &KeyCounter[K]{
		keys: map[K]int{},
	}
}

func (s *KeyCounter[K]) Inc(k K) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.keys[k]++
}

func (s *KeyCounter[K]) Dec(k K) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.keys[k]--
	if s.keys[k] == 0 {
		delete(s.keys, k)
	}
}

func (s *KeyCounter[K]) Count() int {
	s.lock.Lock()
	defer s.lock.Unlock()

	return len(s.keys)
}
