package protect

import (
	"net/netip"
	"syscall"
	"testing"
)

type protector struct {
	protects   map[netip.Addr]int
	unprotects map[netip.Addr]int
}

func (p *protector) Lazy() bool {
	return true
}

func (p *protector) SocketProtector() func(network, address string, conn syscall.RawConn) error {
	return func(network, address string, conn syscall.RawConn) error {
		return nil
	}
}

func (p *protector) ProtectAddresses(addrs []netip.Addr) error {
	for _, ip := range addrs {
		p.protects[ip] += 1
	}

	return nil
}

func (p *protector) UnprotectAddresses(addrs []netip.Addr) error {
	for _, ip := range addrs {
		p.unprotects[ip] += 1
	}

	return nil
}

func (p *protector) testProtects(t *testing.T, addr string, count int) {
	value := p.protects[netip.MustParseAddr(addr)]
	if value != count {
		t.Fatalf("Invalid protects count addr %s expected %d got %d", addr, count, value)
	}
}

func (p *protector) testUnprotects(t *testing.T, addr string, count int) {
	value := p.unprotects[netip.MustParseAddr(addr)]
	if value != count {
		t.Fatalf("Invalid unprotects count addr %s expected %d got %d", addr, count, value)
	}
}

func mkSlice(addrs ...string) []netip.Addr {
	s := make([]netip.Addr, len(addrs))
	for idx, addr := range addrs {
		s[idx] = netip.MustParseAddr(addr)
	}

	return s
}

func testOnce(t *testing.T, p *protector, c *ProtectCounter, round int) {
	if c.ProtectAddresses(mkSlice("192.168.33.12", "1.2.3.4")) != nil {
		t.Fatal("Protecting failed")
	}
	p.testProtects(t, "192.168.33.12", round+1)
	p.testProtects(t, "1.2.3.4", round+1)

	if c.ProtectAddresses(mkSlice("1.2.3.4")) != nil {
		t.Fatal("Protecting failed")
	}
	p.testProtects(t, "192.168.33.12", round+1)
	p.testProtects(t, "1.2.3.4", round+1)

	if c.ProtectAddresses(mkSlice("2.3.4.5")) != nil {
		t.Fatal("Protecting failed")
	}
	p.testProtects(t, "192.168.33.12", round+1)
	p.testProtects(t, "1.2.3.4", round+1)
	p.testProtects(t, "2.3.4.5", round+1)

	if c.UnprotectAddresses(mkSlice("192.168.33.12", "1.2.3.4", "2.3.4.5")) != nil {
		t.Fatal("Unprotecting failed")
	}
	p.testProtects(t, "192.168.33.12", round+1)
	p.testProtects(t, "1.2.3.4", round+1)
	p.testProtects(t, "2.3.4.5", round+1)

	p.testUnprotects(t, "192.168.33.12", round+1)
	p.testUnprotects(t, "1.2.3.4", round+0)
	p.testUnprotects(t, "2.3.4.5", round+1)

	c.UnprotectAll()
	p.testUnprotects(t, "192.168.33.12", round+1)
	p.testUnprotects(t, "1.2.3.4", round+1)
	p.testUnprotects(t, "2.3.4.5", round+1)
}

func TestCounter(t *testing.T) {
	p := &protector{
		protects:   make(map[netip.Addr]int),
		unprotects: make(map[netip.Addr]int),
	}

	c := WithProtectCounter(p)

	testOnce(t, p, c, 0)
	testOnce(t, p, c, 1)
}
