package geoip

import (
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/oschwald/maxminddb-golang"
	"github.com/vpnhouse/common-lib-go/xerror"
	"go.uber.org/zap"
)

const reloadTimeout = time.Minute

type Instance struct {
	dbCountry atomic.Pointer[db]
	stop      chan struct{}
	done      chan struct{}
}

func NewGeoip(path string) (*Instance, error) {
	reader, modTime, err := load(path, time.Time{})
	if err != nil {
		return nil, err
	}

	s := &Instance{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	s.dbCountry.Store(newDb(reader))

	go s.run(path, modTime)

	return s, nil
}

func (s *Instance) GetCountry(ip net.IP) (string, error) {
	db := s.dbCountry.Load()
	if db == nil {
		return "", xerror.EInternalError("maxmind database instance was stopped or never started", nil)
	}
	var record struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}

	err := db.Lookup(ip, &record)
	if err != nil {
		return "", xerror.EInternalError("can't lookup country", err)
	}

	return record.Country.ISOCode, nil
}

func (s *Instance) TryGetCountryFromRequest(r *http.Request) string {
	if s == nil {
		return ""
	}
	clientIP := GetRemoteIP(r)
	ip := net.ParseIP(clientIP)
	if ip == nil {
		zap.L().Error("failed to get client ip by request")
		return ""
	}

	country, err := s.GetCountry(ip)
	if err != nil {
		zap.L().Error("failed to get client country by ip", zap.Error(err))
		return ""
	}
	return country
}

func (s *Instance) Shutdown() error {
	db := s.dbCountry.Swap(nil)
	if db == nil {
		return nil
	}

	close(s.stop)
	<-s.done

	err := db.Close()
	if err != nil {
		return xerror.EInternalError("can't close maxmind db", err)
	}
	return nil
}

func (s *Instance) Running() bool {
	return s.dbCountry.Load() != nil
}

func (s *Instance) run(path string, modTime time.Time) {
	ticker := time.NewTicker(reloadTimeout)
	defer ticker.Stop()

	var reader *maxminddb.Reader
	var err error

	for {
		select {
		case <-s.stop:
			close(s.done)
			return
		case <-ticker.C:
			reader, modTime, err = load(path, modTime)

			if err != nil {
				zap.L().Error("failed to load maxmind db",
					zap.String("path", path), zap.Error(err))
				continue
			}
			if reader == nil {
				continue
			}
			db := s.dbCountry.Swap(newDb(reader))
			if db == nil {
				continue
			}
			err = db.Close()
			if err != nil {
				zap.L().Error("failed to close old maxmind db",
					zap.String("path", path), zap.Error(err))
			}
		}
	}
}

func load(path string, prevModTime time.Time) (*maxminddb.Reader, time.Time, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, prevModTime, xerror.EInternalError("can't stat maxmind db", err, zap.String("path", path))
	}

	modTime := fi.ModTime()
	if modTime.Equal(prevModTime) {
		zap.L().Debug("maxmind db remains unchanged", zap.Time("modification_time", prevModTime))
		return nil, prevModTime, nil
	}

	if !prevModTime.IsZero() {
		zap.L().Debug("maxmind db modified, reloading...",
			zap.Time("last_modification_time", prevModTime),
			zap.Time("modification_time", modTime),
			zap.Duration("modified_ago", modTime.Sub(prevModTime)),
		)
	}

	db, err := maxminddb.Open(path)
	if err != nil || db == nil {
		return nil, modTime, xerror.EInternalError("can't open maxmind db", err, zap.String("path", path))
	}

	zap.L().Info("maxmind db is successfully loaded",
		zap.String("path", path), zap.Time("modification_time", modTime))

	return db, modTime, nil
}
