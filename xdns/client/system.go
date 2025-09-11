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
		switch dnsErr := err.(type) {
		case *net.DNSError:
			if dnsErr.IsTimeout {
				return nil, ErrDNSNoResponse
			}

			if dnsErr.IsNotFound {
				return nil, ErrDNSNotExists
			}
		}

		return nil, err
	}

	if len(addrs) == 0 {
		return nil, ErrDNSEmptyResponse
	}

	expires := time.Now().Add(r.ttl)
	response := &Response{
		Expires: &expires,
		Exists:  len(addrs) > 0,
	}

	for _, addr := range addrs {
		netipAddr, ok := netip.AddrFromSlice(addr.IP)
		if ok {
			response.Addresses = append(response.Addresses, netipAddr)
		}
	}

	return response, nil
}
func (r *SystemResolver) LookupNonCached(ctx context.Context, request *Request) (*Response, error) {
	return r.Lookup(ctx, request)
}
