package connmgr

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

const (
	connMinimalID = 1024
	connMaximalID = 2147483647
	maxConns      = connMaximalID - connMinimalID + 1
)

var (
	ErrNotExist       = fmt.Errorf("not exists")
	ErrOutOfResources = fmt.Errorf("out of resources")
)

type Instance struct {
	pmap   atomic.Pointer[sync.Map]
	nextId int
}

func NewConnMgr() *Instance {
	m := &Instance{
		nextId: connMinimalID,
	}
	m.switchMap()

	return m
}

func (m *Instance) Lookup(id int) (net.Conn, error) {
	connById := m.pmap.Load()

	c, ok := connById.Load(id)
	if !ok {
		return nil, ErrNotExist
	} else {
		return c.(net.Conn), nil
	}
}

func (m *Instance) Register(c net.Conn) (int, error) {
	incId := func() {
		m.nextId += 1
		if m.nextId >= connMaximalID || m.nextId < connMinimalID {
			m.nextId = connMinimalID
		}
	}

	connById := m.pmap.Load()
	for {
		id := m.nextId
		_, busy := connById.LoadOrStore(id, c)
		incId()

		if busy {
			continue
		} else {
			return id, nil
		}
	}
}

func (m *Instance) Close(id int) error {
	connById := m.pmap.Load()

	c, ok := connById.LoadAndDelete(id)
	if ok {
		return c.(net.Conn).Close()
	} else {
		return ErrNotExist
	}
}

func (m *Instance) CloseAll() {
	connById := m.switchMap()

	connById.Range(func(id, c any) bool {
		c.(net.Conn).Close()
		return true
	})
}

func (m *Instance) switchMap() *sync.Map {
	return m.pmap.Swap(&sync.Map{})
}
