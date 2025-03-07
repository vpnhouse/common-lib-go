package client

import (
	"context"
	"sync"
	"time"
)

type priorityEntity struct {
	priority int
	resolver Resolver

	failureTimeout time.Duration
	failedAt       time.Time
}

type PriorityResolver struct {
	lock      sync.RWMutex
	resolvers []*priorityEntity
}

func NewPriorityResolver() *PriorityResolver {
	return &PriorityResolver{
		resolvers: make([]*priorityEntity, 0),
	}
}

func (r *PriorityResolver) With(priority int, resolver Resolver, opts *options) *PriorityResolver {
	_ = r.add(priority, resolver, opts, true)
	return r
}

func (r *PriorityResolver) Add(priority int, resolver Resolver, opts *options) error {
	return r.add(priority, resolver, opts, false)
}

func (r *PriorityResolver) Get(priority int) (Resolver, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for _, entity := range r.resolvers {
		if entity.priority == priority {
			return entity.resolver, nil
		}
	}

	return nil, ErrNotExists
}

func (r *PriorityResolver) add(priority int, resolver Resolver, opts *options, replace bool) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	entity := &priorityEntity{
		priority:       priority,
		resolver:       resolver,
		failureTimeout: opts.priorityFailureTimeout,
	}

	for idx, entity := range r.resolvers {
		if entity.priority == priority && replace {
			r.resolvers[idx].resolver = resolver
			return nil
		}

		if entity.priority > priority {
			r.resolvers = append(
				append(
					r.resolvers[:idx],
					entity,
				),
				r.resolvers[idx:]...,
			)
		}
	}

	r.resolvers = append(r.resolvers, entity)
	return nil
}

func (r *PriorityResolver) Unset(priority int) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	for idx, entity := range r.resolvers {
		if entity.priority == priority {
			r.resolvers = append(r.resolvers[:idx], r.resolvers[idx+1:]...)
			return nil
		}
	}

	return ErrNotExists
}
func (r *PriorityResolver) Lookup(ctx context.Context, request *Request) (*Response, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	for _, entity := range r.resolvers {
		if !entity.failedAt.IsZero() {
			if time.Since(entity.failedAt) > entity.failureTimeout {
				entity.failedAt = time.Time{}
			} else {
				continue
			}
		}

		result, err := entity.resolver.Lookup(ctx, request)
		if err != nil {
			entity.failedAt = time.Now()
			continue
		}

		return result, nil
	}

	return nil, ErrNoResponse
}
