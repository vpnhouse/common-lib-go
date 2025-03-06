package client

import (
	"context"
	"sync"
)

type lastActiveEntity struct {
	tag      string
	resolver Resolver
}

type LastActiveResolver struct {
	lock     sync.RWMutex
	entities []*lastActiveEntity
}

func NewLastActive(opst *options) *LastActiveResolver {
	return &LastActiveResolver{
		entities: make([]*lastActiveEntity, 0),
	}
}

func (r *LastActiveResolver) With(tag string, resolver Resolver) *LastActiveResolver {
	_ = r.add(tag, resolver, true)
	return r
}

func (r *LastActiveResolver) Add(tag string, resolver Resolver) error {
	return r.add(tag, resolver, false)
}

func (r *LastActiveResolver) add(tag string, resolver Resolver, replace bool) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	for idx, entity := range r.entities {
		if entity.tag == tag {
			if replace {
				r.entities[idx].resolver = resolver
				return nil
			} else {
				return ErrExists
			}
		}
	}

	r.entities = append(r.entities, &lastActiveEntity{
		tag, resolver,
	})
	return nil
}

func (r *LastActiveResolver) Unset(tag string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	for idx, entity := range r.entities {
		if entity.tag == tag {
			r.entities = append(r.entities[:idx], r.entities[idx+1:]...)
			return nil
		}
	}

	return ErrNotExists
}

func (r *LastActiveResolver) Lookup(ctx context.Context, request *Request) (*Response, error) {
	r.lock.RLock()

	for idx, entity := range r.entities {
		result, err := entity.resolver.Lookup(ctx, request)
		if err == nil && result.Successful() {
			r.lock.RUnlock()
			r.lock.Lock()
			moveToTop(r.entities, idx)
			r.lock.Unlock()
			return result, nil
		}

	}

	r.lock.RUnlock()
	return nil, ErrNoResponse
}

func moveToTop[T any](slice []T, index int) {
	if len(slice) <= 2 {
		return
	}

	first := slice[0]
	slice[0] = slice[index]
	slice[index] = first
}
