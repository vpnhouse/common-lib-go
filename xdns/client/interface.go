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
	ErrNoResponse             = errors.New("no vaild response")
	ErrExists                 = errors.New("already exists")
	ErrNotExists              = errors.New("resource does not exist")
	ErrEmptyResponse          = errors.New("empty response")
	ErrInvalidResponse        = errors.New("invalid response")
	ErrProtectionNotSupported = errors.New("protection is not supported")
	ErrCacheMiss              = errors.New("not found in cachce")
	ErrNoCache                = errors.New("no cache")
)

type Request struct {
	Domain    string
	QueryType uint16
}

type Response struct {
	Exists             bool
	Addresses          []netip.Addr
	CreatedAt          time.Time
	TTL                time.Duration
	ProtectionRequired bool
}

type Resolver interface {
	Lookup(ctx context.Context, request *Request) (*Response, error)
}

func (r *Response) Successful() bool {
	return r != nil && r.Exists && len(r.Addresses) > 0
}

func (r *Response) Expired() bool {
	return time.Since(r.CreatedAt) > r.TTL
}
