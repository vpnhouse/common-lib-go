package xdns

import (
	"context"
	"errors"
	"net/netip"
)

type Builder struct {
	root Resolver
}

func New() *Builder {
	return &Builder{
		root: NewPriorityResolver(),
	}
}
func (r *Builder) Lookup(ctx context.Context, request *Request) (*Response, error) {
	return r.root.Lookup(ctx, request)
}
func (b *Builder) WithCache(opts *options) *Builder {
	switch b.root.(type) {
	case *CachedResolver:
		return b
	default:
		b.root = NewCachedResolver(b.root, opts)
		return b
	}
}

func (b *Builder) WithDNSServers(priority int, servers []netip.Addr, opts *options) *Builder {
	priorityResolver := b.priorityResolver()
	lastActiveResolver, err := priorityResolver.Get(priority)

	if errors.Is(err, ErrNotExists) {
		lastActiveResolver = NewLastActive(opts)
		priorityResolver.With(priority,
			lastActiveResolver,
			opts,
		)
	}

	for _, server := range servers {
		direct := NewDirectResolver(server, opts)
		lastActiveResolver.(*LastActiveResolver).With(server.String(), direct)
	}

	return b
}

func (b *Builder) WithSystem(priority int, opts *options) *Builder {
	systemResolver := NewSystemResolver(opts)
	priorityResolver := b.priorityResolver()
	priorityResolver.With(priority, systemResolver, opts)
	return b
}

func (b *Builder) Preset(domain string, queryType uint16, response *Response) error {
	cache := b.cachedResolver()
	if cache == nil {
		return ErrNoCache
	}

	cache.Preset(domain, queryType, response)
	return nil
}

func (b *Builder) priorityResolver() *PriorityResolver {
	switch r := b.root.(type) {
	case *CachedResolver:
		return r.nestedResolver.(*PriorityResolver)
	default:
		return r.(*PriorityResolver)
	}
}

func (b *Builder) cachedResolver() *CachedResolver {
	switch r := b.root.(type) {
	case *CachedResolver:
		return r
	default:
		return nil
	}
}
