package client

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/miekg/dns"
	"github.com/vpnhouse/common-lib-go/pkg/protect"
	"go.uber.org/zap"
)

const (
	infiniteTtl = uint32(365 * 24 * 3600)
)

var (
	directDefaultTimeout   = 15 * time.Second
	directDefaultProtector = &protect.Dummy{}
)

type DirectResolver struct {
	server    netip.Addr
	timeout   time.Duration
	protector protect.Protector
}

func NewDirectResolver(server netip.Addr, opts *options) *DirectResolver {
	return &DirectResolver{
		server:    server,
		timeout:   opts.directTimeout,
		protector: opts.directProtector,
	}
}

func (r *DirectResolver) WithTimeout(timeout time.Duration) *DirectResolver {
	ret := *r
	ret.timeout = timeout
	return &ret
}

func (r *DirectResolver) WithProtector(protector protect.Protector) *DirectResolver {
	ret := *r
	ret.protector = protector
	return &ret
}

func (r *DirectResolver) Lookup(ctx context.Context, request *Request) (*Response, error) {
	if r.protector.Lazy() {
		return r.once(ctx, request, true)
	}

	response, err := r.once(ctx, request, false)
	if err == nil {
		return response, err
	}

	return r.once(ctx, request, true)
}

// (addresses []netip.Addr, ttl uint32, err error)
func (r *DirectResolver) once(ctx context.Context, request *Request, protected bool) (*Response, error) {
	zap.L().Debug("DNS query started", zap.String("domain", request.Domain), zap.String("server", r.server.String()))
	defer zap.L().Debug("DNS query over", zap.String("domain", request.Domain), zap.String("server", r.server.String()))

	client := dns.Client{
		Timeout: defaultDNSTimeout,
		Dialer: &net.Dialer{
			Deadline: time.Now().Add(r.timeout),
		},
	}

	if protected {
		client.Dialer.Control = r.protector.SocketProtector()
		err := r.protector.ProtectAddresses([]netip.Addr{r.server})
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = r.protector.UnprotectAddresses([]netip.Addr{r.server})
		}()
	}

	msgQuery := dns.Msg{}
	msgQuery.SetQuestion(dns.Fqdn(request.Domain), request.QueryType)

	serverAddr := net.UDPAddrFromAddrPort(netip.AddrPortFrom(r.server, 53))
	msgResponse, _, err := client.ExchangeContext(ctx, &msgQuery, serverAddr.String())
	if err != nil {
		return nil, err
	}

	addresses := make([]netip.Addr, 0)
	ttl := infiniteTtl
	for _, rr := range msgResponse.Answer {
		if rr.Header().Rrtype != request.QueryType {
			continue
		}

		var address netip.Addr
		ok := false
		switch a := rr.(type) {
		case *dns.A:
			address, ok = netip.AddrFromSlice(a.A)
			ttl = min(ttl, rr.Header().Ttl)
		case *dns.AAAA:
			address, ok = netip.AddrFromSlice(a.AAAA)
			ttl = min(ttl, rr.Header().Ttl)
		}

		if !ok {
			zap.L().Error("Skipping invalid address", zap.Any("record", rr))
			continue
		}

		addresses = append(addresses, address)
	}

	if len(addresses) == 0 {
		zap.L().Debug("Empty response", zap.String("server", r.server.String()), zap.String("domain", request.Domain), zap.Uint16("type", request.QueryType))
		return nil, ErrEmptyResponse
	}

	return &Response{
		Exists:             true,
		Addresses:          addresses,
		CreatedAt:          time.Now(),
		TTL:                time.Duration(ttl) * time.Second,
		ProtectionRequired: protected,
	}, nil
}
