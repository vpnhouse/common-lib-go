package client

import (
	"context"
	"sync"

	"github.com/vpnhouse/common-lib-go/list"
)

type lastActiveEntity struct {
	tag      string
	resolver Resolver
}

type LastActiveResolver struct {
	lock     sync.RWMutex
	entities *list.List[*lastActiveEntity]
}

func NewLastActive(opst *options) *LastActiveResolver {
	return &LastActiveResolver{
		entities: list.New[*lastActiveEntity](),
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

	for e := r.entities.Front(); e != nil; e = e.Next() {
		entity := e.Value
		if entity.tag == tag {
			if replace {
				entity.resolver = resolver
				return nil
			} else {
				return ErrExists
			}
		}
	}

	r.entities.PushBack(&lastActiveEntity{
		tag, resolver,
	})
	return nil
}

func (r *LastActiveResolver) Unset(tag string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	for e := r.entities.Front(); e != nil; e = e.Next() {
		entity := e.Value
		if entity.tag == tag {
			r.entities.Remove(e)
			return nil
		}
	}

	return ErrNotExists
}

func (r *LastActiveResolver) Lookup(ctx context.Context, request *Request) (*Response, error) {
	r.lock.RLock()

	for e := r.entities.Front(); e != nil; e = e.Next() {
		entity := e.Value

		result, err := entity.resolver.Lookup(ctx, request)
		if err == nil && result.Successful() {
			if e == r.entities.Front() {
				r.lock.RUnlock()
				return result, nil
			}

			r.lock.RUnlock()
			r.lock.Lock()
			r.entities.MoveToFront(e)
			r.lock.Unlock()
			return result, nil
		}
	}

	r.lock.RUnlock()
	return nil, ErrNoResponse
}
