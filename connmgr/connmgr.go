package connmgr

import (
	"fmt"
	"net"
	"sync"
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
	lock     sync.Mutex
	nextId   int
	connById map[int]net.Conn
}

func NewConnMgr() *Instance {
	return &Instance{
		nextId:   connMinimalID,
		connById: make(map[int]net.Conn),
	}
}

func (m *Instance) Lookup(id int) (net.Conn, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	c, ok := m.connById[id]
	if !ok {
		return nil, ErrNotExist
	} else {
		return c, nil
	}
}

func (m *Instance) Register(c net.Conn) (int, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if len(m.connById) >= maxConns {
		return -1, ErrOutOfResources
	}

	incId := func() {
		m.nextId += 1
		if m.nextId >= connMaximalID || m.nextId < connMinimalID {
			m.nextId = connMinimalID
		}
	}

	for {
		id := m.nextId
		_, busy := m.connById[id]
		incId()

		if busy {
			continue
		} else {
			m.connById[id] = c
			return id, nil
		}
	}
}

func (m *Instance) Close(id int) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	c, ok := m.connById[id]
	if ok {
		delete(m.connById, id)
		return c.Close()
	} else {
		return ErrNotExist
	}
}

func (m *Instance) CloseAll() {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, c := range m.connById {
		c.Close()
	}

	m.connById = make(map[int]net.Conn)
}
