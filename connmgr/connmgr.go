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
	pmap atomic.Pointer[sync.Map]
	id   *idType
}

func NewConnMgr() *Instance {
	m := &Instance{
		id: newId(connMinimalID, connMaximalID),
	}
	m.pmap.Store(&sync.Map{})

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
	connById := m.pmap.Load()
	for {
		id := m.id.shift()
		_, busy := connById.LoadOrStore(id, c)

		if busy {
			continue
		} else {
			return id, nil
		}
	}
}

func (m *Instance) Unregister(id int) error {
	connById := m.pmap.Load()

	_, ok := connById.LoadAndDelete(id)
	if !ok {
		return ErrNotExist
	}

	return nil
}

func (m *Instance) CloseAll() {
	connById := m.pmap.Swap(&sync.Map{})

	connById.Range(func(id, c any) bool {
		c.(net.Conn).Close()
		return true
	})
}
