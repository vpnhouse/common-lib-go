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

type Resolver interface {
	Lookup(ctx context.Context, request *Request) (*Response, error)
}
