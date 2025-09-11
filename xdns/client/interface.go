package client

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/miekg/dns"
)

const (
	QueryIp4          = dns.TypeA
	QueryIp6          = dns.TypeAAAA
	defaultDNSTimeout = time.Second * 2
)

var DefaultDNSServers = []netip.Addr{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("8.8.8.8")}

var (
	ErrExists                 = errors.New("already exists")
	ErrNotExists              = errors.New("resource does not exist")
	ErrProtectionNotSupported = errors.New("protection is not supported")
	ErrNoCache                = errors.New("no cache")

	ErrDNSNotExists     = errors.New("DNS entry not exists")
	ErrDNSNoResponse    = errors.New("no vaild DNS response")
	ErrDNSEmptyResponse = errors.New("empty DNS response")
)

type Request struct {
	Domain    string
	QueryType uint16
	NoLazy    bool
}

type Response struct {
	Exists             bool
	Addresses          []netip.Addr
	CreatedAt          time.Time
	TTL                *time.Duration
	ProtectionRequired bool
}

type Resolver interface {
	Lookup(ctx context.Context, request *Request) (*Response, error)
}

func (r *Response) Successful() bool {
	return r != nil && r.Exists && len(r.Addresses) > 0
}

func (r *Response) Expired() bool {
	if r.TTL == nil {
		return false
	}
	return time.Since(r.CreatedAt) > *r.TTL
}

func (r *Response) AddressesAsStrings() []string {
	result := make([]string, len(r.Addresses))
	for idx, addr := range r.Addresses {
		result[idx] = addr.String()
	}

	return result
}
