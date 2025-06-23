package geoip

import (
	"net"
	"sync"

	"github.com/oschwald/maxminddb-golang"

	"github.com/vpnhouse/common-lib-go/xerror"
)

type db struct {
	lock   sync.RWMutex
	reader *maxminddb.Reader
}

func newDb(reader *maxminddb.Reader) *db {
	return &db{reader: reader}
}

func (s *db) Lookup(ip net.IP, result interface{}) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.reader == nil {
		return xerror.EInternalError("maxmind database instance is closed", nil)
	}
	return s.reader.Lookup(ip, result)
}

func (s *db) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.reader == nil {
		return nil
	}

	err := s.reader.Close()
	s.reader = nil
	return err
}
