package geoip

import (
	"net"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type CountryResolver struct {
	*Instance
	CDNSecrets map[string]string
}

func (s *CountryResolver) ClientCountry(r *http.Request) string {
	if s == nil || s.Instance == nil {
		return ""
	}

	ip := GetRemoteIP(r, WithIPParser(CDNSecretIPParser(s.CDNSecrets), GetRemoteAddr))
	if ip == "" {
		return ""
	}

	addr := net.ParseIP(ip)
	if addr == nil {
		return ""
	}

	country, err := s.Instance.GetCountry(addr)
	if err != nil {
		zap.L().Error("failed to get country by ip", zap.String("ip", ip), zap.Error(err))
	}

	return strings.ToLower(country)
}
