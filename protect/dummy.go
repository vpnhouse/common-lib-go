package protect

import (
	"net/netip"
	"syscall"
)

type Dummy struct{}

func (d *Dummy) SocketProtector() func(network, address string, conn syscall.RawConn) error {
	return func(network, address string, conn syscall.RawConn) error {
		return nil
	}
}

func (d *Dummy) ProtectAddresses([]netip.Addr) error {
	return nil
}

func (d *Dummy) UnprotectAddresses([]netip.Addr) error {
	return nil
}
