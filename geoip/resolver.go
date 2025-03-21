package geoip

import (
	"net"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type Info struct {
	Country string
}

type Resolver struct {
	Geo        *Instance
	CDNSecrets map[string]string
}

func (s *Resolver) GetInfo(r *http.Request) Info {
	if s == nil || s.Geo == nil {
		return Info{}
	}

	ip := GetRemoteIP(r, WithIPParser(CDNSecretIPParser(s.CDNSecrets), GetRemoteAddr))
	if ip == "" {
		return Info{}
	}

	addr := net.ParseIP(ip)
	if addr == nil {
		return Info{}
	}

	country, err := s.Geo.GetCountry(addr)
	if err != nil {
		zap.L().Error("failed to get country by ip", zap.String("ip", ip), zap.Error(err))
	}

	return Info{
		Country: strings.ToLower(country),
	}
}
