package geoip

import (
	"net"
	"net/http"
	"strings"
)

type IPParser func(*http.Request) string

var DefaultIPAddressParsers = []IPParser{
	GetForwardedIP, // "x-forwarded-for"
	GetRealIP,      // "x-real-ip"
	GetRemoteAddr,  // r.RemoteAddr
}

type (
	IPParserOption         func(opts *ipAddressParserOptions)
	ipAddressParserOptions struct {
		Parsers []IPParser
	}
)

func WithIPParser(ipParses ...IPParser) IPParserOption {
	return func(opts *ipAddressParserOptions) {
		opts.Parsers = ipParses
	}
}

func CDNSecretIPParser(secrets map[string]string) IPParser {
	return func(r *http.Request) string {
		return GetRemoteIPFromCDN(r, secrets)
	}
}

func GetRemoteIPFromCDN(r *http.Request, secrets map[string]string) string {
	if !hasValidCDNSecret(r, secrets) {
		return ""
	}
	return GetForwardedIP(r)
}

func GetRemoteIP(r *http.Request, opts ...IPParserOption) string {
	var options ipAddressParserOptions
	for _, opt := range opts {
		opt(&options)
	}

	if len(options.Parsers) == 0 {
		options.Parsers = DefaultIPAddressParsers
	}

	for _, parseIP := range options.Parsers {
		ip := parseIP(r)
		if ip != "" {
			return ip
		}
	}

	return ""
}

func GetForwardedIP(r *http.Request) string {
	header := r.Header.Get("X-Forwarded-For")
	addrs := strings.Split(strings.Replace(header, " ", "", -1), ",")
	return addrs[len(addrs)-1]
}

func GetRealIP(r *http.Request) string {
	return r.Header.Get("X-Real-Ip")
}

func GetRemoteAddr(r *http.Request) string {
	addr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	return addr
}

func hasValidCDNSecret(r *http.Request, cdnSecrets map[string]string) bool {
	if len(cdnSecrets) == 0 {
		// CDN secrets is not configured thus CDN secret check is disabled
		// So beleive the request
		return true
	}
	for header, secret := range cdnSecrets {
		if r.Header.Get(header) == secret {
			return true
		}
	}

	// Noone configured header secret matched
	return false
}
