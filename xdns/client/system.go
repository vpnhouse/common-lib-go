package client

import (
	"context"
	"net"
	"net/netip"
	"time"
)

const (
	systemDefaultTtl = 60 * time.Second
)

type SystemResolver struct {
	ttl time.Duration
}

func NewSystemResolver(opts *options) *SystemResolver {
	return &SystemResolver{
		ttl: opts.systemTtl,
	}
}

func (r *SystemResolver) Lookup(ctx context.Context, request *Request) (*Response, error) {
	resolver := net.DefaultResolver

	addrs, err := resolver.LookupIPAddr(ctx, request.Domain)
	if err != nil {
		return nil, err
	}

	response := &Response{
		TTL:    r.ttl,
		Exists: len(addrs) > 0,
	}

	for _, addr := range addrs {
		netipAddr, ok := netip.AddrFromSlice(addr.IP)
		if ok {
			response.Addresses = append(response.Addresses, netipAddr)
		}
	}

	if response.Exists && len(response.Addresses) == 0 {
		return nil, ErrInvalidResponse
	}

	return response, nil
}
func (r *SystemResolver) LookupCached(request *Request) (*Response, error) {
	return nil, ErrCacheMiss

}
func (r *SystemResolver) LookupNonCached(ctx context.Context, request *Request) (*Response, error) {
	return r.Lookup(ctx, request)
}
