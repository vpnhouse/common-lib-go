package geoip

import (
	"net"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type GeoInfo struct {
	Country string
}

type GeoResolver struct {
	*Instance
	CDNSecrets map[string]string
}

func (s *GeoResolver) ClientInfoFromRequest(r *http.Request) GeoInfo {
	if s == nil || s.Instance == nil {
		return GeoInfo{}
	}

	ip := GetRemoteIP(r, WithIPParser(CDNSecretIPParser(s.CDNSecrets), GetRemoteAddr))
	if ip == "" {
		return GeoInfo{}
	}

	addr := net.ParseIP(ip)
	if addr == nil {
		return GeoInfo{}
	}

	country, err := s.Instance.GetCountry(addr)
	if err != nil {
		zap.L().Error("failed to get country by ip", zap.String("ip", ip), zap.Error(err))
	}

	return GeoInfo{
		Country: strings.ToLower(country),
	}
}
