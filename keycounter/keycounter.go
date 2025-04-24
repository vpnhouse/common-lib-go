package keycounter

import (
	"sync"
)

type KeyCounter[K comparable] struct {
	lock sync.Mutex
	ids  map[K]int
}

func New[K comparable]() *KeyCounter[K] {
	return &KeyCounter[K]{
		ids: map[K]int{},
	}
}

func (s *KeyCounter[K]) Inc(k K) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.ids[k]++
}

func (s *KeyCounter[K]) Dec(k K) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.ids[k]--
	if s.ids[k] == 0 {
		delete(s.ids, k)
	}
}

func (s *KeyCounter[K]) Count() int {
	s.lock.Lock()
	defer s.lock.Unlock()

	return len(s.ids)
}
