package protect

import (
	"net/netip"
	"syscall"
)

type Protector interface {
	Lazy() bool
	SocketProtector() func(network, address string, conn syscall.RawConn) error
	ProtectAddresses([]netip.Addr) error
	UnprotectAddresses([]netip.Addr) error
}
