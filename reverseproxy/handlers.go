package reverseproxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"go.uber.org/zap"
)

type Config struct {
	URL      string   `json:"url" yaml:"url"`
	Patterns []string `json:"patterns" yaml:"patterns"`
}

type Handler struct {
	Patterns []string
	Func     http.HandlerFunc
}

func MakeHandler(config *Config) (*Handler, error) {
	targetURL, err := url.Parse(config.URL)
	if err != nil {
		zap.L().Error("SKipping invalid target URL", zap.Error(err), zap.String("target", config.URL))
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Host = targetURL.Host
	}
	return &Handler{
		Patterns: config.Patterns,
		Func: func(w http.ResponseWriter, req *http.Request) {
			proxy.ServeHTTP(w, req)
		},
	}, nil
}

func MakeHandlers(configs []*Config) ([]*Handler, error) {
	result := make([]*Handler, 0, len(configs))

	for _, config := range configs {
		handler, err := MakeHandler(config)
		if err != nil {
			return nil, err
		}

		result = append(result, handler)
	}

	return result, nil
}
