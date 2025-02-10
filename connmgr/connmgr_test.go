package connmgr

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testMaxConns = 2
)

var mgr = NewConnMgr()

type conn struct {
	id     int
	closed bool
}

// Close implements net.Conn.
func (c *conn) Close() error {
	c.closed = true
	return nil
}

// LocalAddr implements net.Conn.
func (c *conn) LocalAddr() net.Addr {
	panic("unimplemented")
}

// Read implements net.Conn.
func (c *conn) Read(b []byte) (n int, err error) {
	panic("unimplemented")
}

// RemoteAddr implements net.Conn.
func (c *conn) RemoteAddr() net.Addr {
	panic("unimplemented")
}

// SetDeadline implements net.Conn.
func (c *conn) SetDeadline(t time.Time) error {
	panic("unimplemented")
}

// SetReadDeadline implements net.Conn.
func (c *conn) SetReadDeadline(t time.Time) error {
	panic("unimplemented")
}

// SetWriteDeadline implements net.Conn.
func (c *conn) SetWriteDeadline(t time.Time) error {
	panic("unimplemented")
}

// Write implements net.Conn.
func (c *conn) Write(b []byte) (n int, err error) {
	panic("unimplemented")
}

func TestRegister(t *testing.T) {

	conns := make([]*conn, 0)

	for idx := 0; idx < testMaxConns; idx++ {
		c := &conn{}
		id, err := mgr.Register(c)
		assert.Nil(t, err)
		assert.GreaterOrEqual(t, id, connMinimalID)
		c.id = id
		conns = append(conns, c)
	}

	assert.Equal(t, testMaxConns, len(mgr.connById))

	for _, c := range conns {
		mgrc, err := mgr.Lookup(c.id)
		assert.Nil(t, err)
		assert.Equal(t, c.id, mgrc.(*conn).id)
	}

	for _, c := range conns {
		err := mgr.Close(c.id)
		assert.Nil(t, err)

		err = mgr.Close(c.id)
		assert.ErrorIs(t, ErrNotExist, err)
		assert.True(t, c.closed)
	}

	assert.Equal(t, 0, len(mgr.connById))
}
