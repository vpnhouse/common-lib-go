package stats

import (
	"github.com/vpnhouse/common-lib-go/xcache"
)

const maxBytes = 32 << 20 // 32 Mb

type Service struct {
	c *xcache.Cache
}

func New() (*Service, error) {
	var s Service
	var err error
	s.c, err = xcache.New(maxBytes, s.onEvict)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Service) onEvict(evicted *xcache.Items) {
}
