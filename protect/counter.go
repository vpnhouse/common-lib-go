package protect

import (
	"net/netip"
	"sync"
	"syscall"

	"go.uber.org/zap"
)

type ProtectCounter struct {
	lock      sync.Mutex
	protector Protector
	protected map[netip.Addr]int
}

func WithProtectCounter(protector Protector) *ProtectCounter {
	return &ProtectCounter{
		protector: protector,
		protected: make(map[netip.Addr]int),
	}
}

func (pc *ProtectCounter) Lazy() bool {
	return pc.protector.Lazy()
}

func (pc *ProtectCounter) SocketProtector() func(network, address string, conn syscall.RawConn) error {
	return pc.protector.SocketProtector()
}

func (pc *ProtectCounter) ProtectAddresses(addrs []netip.Addr) error {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	unprotected := make([]netip.Addr, 0)
	for _, addr := range addrs {
		_, isProtected := pc.protected[addr]
		if isProtected {
			continue
		} else {
			unprotected = append(unprotected, addr)
		}
	}

	err := pc.protector.ProtectAddresses(unprotected)
	if err != nil {
		return err
	}

	for _, addr := range addrs {
		pc.protected[addr] += 1
	}

	return nil
}

func (pc *ProtectCounter) UnprotectAddresses(addrs []netip.Addr) error {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	unprotect := make([]netip.Addr, 0)
	for _, addr := range addrs {
		count := pc.protected[addr]
		if count == 1 {
			unprotect = append(unprotect, addr)
		} else {
			continue
		}
	}

	err := pc.protector.UnprotectAddresses(unprotect)

	for _, addr := range addrs {
		pc.protected[addr] -= 1
		if pc.protected[addr] == 0 {
			delete(pc.protected, addr)
		}
	}

	return err
}

func (pc *ProtectCounter) UnprotectAll() {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	protected := make([]netip.Addr, 0)
	for addr := range pc.protected {
		protected = append(protected, addr)
	}

	err := pc.protector.UnprotectAddresses(protected)
	if err != nil {
		zap.L().Error("Failed to unprotect addresses", zap.Error(err), zap.Any("aderesses", protected))
	}
	pc.protected = make(map[netip.Addr]int)
}
