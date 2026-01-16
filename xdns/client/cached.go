package client

import (
	"context"
	"fmt"
	"time"

	"github.com/vpnhouse/common-lib-go/xttlmap"
	"go.uber.org/zap"
)

const (
	cachedDefaultMaxSize  = 1024
	cachedDefaultMinTtl   = 60 * time.Second
	cachedDefaultMaxTtl   = 3600 * time.Second
	cachedDefaultKeepTime = 24 * 3600 * time.Second
)

type cacheKey struct {
	domain    string
	queryType uint16
}

type CachedResolver struct {
	cache          *xttlmap.TTLMap[cacheKey, Response]
	nestedResolver Resolver
	minTtl         time.Duration
	maxTtl         time.Duration
	keepTime       time.Duration
}

func NewCachedResolver(nested Resolver, opts *options) *CachedResolver {
	return &CachedResolver{
		cache:          xttlmap.New[cacheKey, Response](opts.cacheMaxSize),
		nestedResolver: nested,
		minTtl:         opts.cacheMinTtl,
		maxTtl:         opts.cacheMaxTtl,
		keepTime:       opts.cacheKeepTime,
	}
}

func (r *CachedResolver) WithMaxSize(value int) *CachedResolver {
	r.cache.Resize(value)
	return r
}

func (r *CachedResolver) WithMinTtl(value time.Duration) *CachedResolver {
	r.minTtl = value
	return r
}

func (r *CachedResolver) WithMaxTtl(value time.Duration) *CachedResolver {
	r.maxTtl = value
	return r
}

func (r *CachedResolver) WithKeepTime(value time.Duration) *CachedResolver {
	r.keepTime = value
	return r
}

func (r *CachedResolver) Lookup(ctx context.Context, request *Request) (*Response, error) {
	key := cacheKey{request.Domain, request.QueryType}

	cachedResult, found := r.cache.Get(key)
	if found && !cachedResult.Expired() {
		zap.L().Debug("Returning cached DNS value", zap.String("domain", request.Domain), zap.Any("value", cachedResult.Addresses))
		return &cachedResult, nil
	}

	lookupResult, err := r.nestedResolver.Lookup(ctx, request)
	if err != nil {
		return &cachedResult, nil
	}

	r.cacheResult(cacheKey{request.Domain, request.QueryType}, lookupResult)
	return lookupResult, nil
}

func (r *CachedResolver) Preset(domain string, queryType uint16, response *Response) {
	if response == nil {
		r.cache.Delete(cacheKey{domain, queryType})
	} else {
		r.cacheResult(cacheKey{domain, queryType}, response)
	}
}

func (r *CachedResolver) cacheResult(key cacheKey, result *Response) {
	result = result.Clone()

	now := time.Now()
	if !result.Expires.IsZero() {
		ttl := result.Expires.Sub(now)
		if ttl < r.minTtl {
			result.Expires = now.Add(r.minTtl)
		}
		if ttl > r.maxTtl {
			result.Expires = now.Add(r.maxTtl)
		}

		r.cache.Set(key, *result, now.Add(r.keepTime))
	} else {
		r.cache.Set(key, *result, time.Time{})
	}

	zap.L().Debug("Cached", zap.String("key", fmt.Sprintf("%s:%d", key.domain, key.queryType)), zap.Any("value", result.Addresses))
}
