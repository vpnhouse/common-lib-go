package geoip

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIP(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/some/path", http.NoBody)
	assert.NoError(t, err)

	r.Header.Add("X-Forwarded-For", "192.0.2.4,192.0.2.3,192.0.2.2")
	r.Header.Add("X-Real-Ip", "137.24.18.67")
	r.RemoteAddr = "10.0.0.2:6578"

	ip := GetRemoteAddr(r)
	assert.Equal(t, "10.0.0.2", ip)

	ip = GetRealIP(r)
	assert.Equal(t, "137.24.18.67", ip)

	ip = GetForwardedIP(r)
	assert.Equal(t, "192.0.2.2", ip)

	ip = GetRemoteIP(r)
	assert.Equal(t, "192.0.2.2", ip)

	cdnSecrets := map[string]string{
		"X-CDN-Secret": "secret",
	}

	ip = GetRemoteIP(r, WithIPParser(CDNSecretIPParser(cdnSecrets)))
	assert.Equal(t, "", ip)

	ip = GetRemoteIP(r, WithIPParser(CDNSecretIPParser(cdnSecrets), GetRemoteAddr))
	assert.Equal(t, "10.0.0.2", ip)

	r.Header.Add("X-CDN-Secret", "secret")

	ip = GetRemoteIP(r, WithIPParser(CDNSecretIPParser(cdnSecrets), GetRemoteAddr))
	assert.Equal(t, "192.0.2.2", ip)
}
