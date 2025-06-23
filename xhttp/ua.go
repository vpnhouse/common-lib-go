package xhttp

import (
	"net/http"
	"strings"

	"github.com/mileusna/useragent"
)

type PlatformParser func(*http.Request) string

var DefaultPlatformParsers = []PlatformParser{
	GetByUserAgent,         // "User-Agent"
	GetByXClientTypeHeader, // "X-Client-Type"
}

func TryParsePlatform(r *http.Request, platformParsers ...PlatformParser) string {
	if len(platformParsers) == 0 {
		platformParsers = DefaultPlatformParsers
	}
	for _, parser := range platformParsers {
		platform := parser(r)
		if platform != "" {
			return platform
		}
	}

	return ""
}

func GetByUserAgent(r *http.Request) string {
	ua := r.Header.Get("User-Agent")
	if ua == "" {
		return ""
	}
	info := useragent.Parse(ua)
	return strings.ToLower(info.OS)
}

func GetByXClientTypeHeader(r *http.Request) string {
	return r.Header.Get("X-Client-Type")
}
