package connmgr

import "sync"

type idType struct {
	lock     sync.Mutex
	current  int
	min, max int
}

func newId(min, max int) *idType {
	return &idType{
		current: min,
		min:     min,
		max:     max,
	}
}

func (i *idType) shift() (current int) {
	i.lock.Lock()
	defer i.lock.Unlock()

	current = i.current

	i.current++
	if i.current > i.max {
		i.current = i.min
	}

	return
}
