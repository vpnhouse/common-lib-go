package protect

import (
	"net/netip"

	"go.uber.org/zap"
)

func TryProtecSlice(p Protector, ips []netip.Addr) error {
	err := p.ProtectAddresses(ips)
	if err != nil {
		zap.L().Error("Failed to protect", zap.Error(err), zap.Any("servers", ips))
		return err
	}

	return nil
}

func TryUnprotecSlice(p Protector, ips []netip.Addr) error {
	err := p.UnprotectAddresses(ips)
	if err != nil {
		zap.L().Error("Failed to unprotect", zap.Error(err), zap.Any("servers", ips))
		return err
	}

	return nil
}

func TryProtect(p Protector, ips ...netip.Addr) error {
	return TryProtecSlice(p, ips)
}

func TryUnprotect(p Protector, ips ...netip.Addr) error {
	return TryUnprotecSlice(p, ips)
}
