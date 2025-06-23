package xhttp

import (
	"net/http"

	"github.com/mileusna/useragent"
)

func TryParsePlatform(r *http.Request) string {
	ua := r.Header.Get("User-Agent")
	if ua == "" {
		return ""
	}
	info := useragent.Parse(ua)
	return info.OS
}
