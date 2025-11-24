package xhttp

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type compiledPattern struct {
	original   string
	parts      []string
	isParam    []bool
	partsCount int
}

type Measure struct {
	requestDuration *prometheus.HistogramVec
	requestCount    *prometheus.CounterVec
	requestResult   *prometheus.CounterVec
	allowedPaths    map[string]bool
	allowedPatterns []compiledPattern
}

type MeasureOptions struct {
	Namespace   string
	Subsystem   string
	ServiceName string
	Buckets     []float64

	// Exact paths or patterns like /some/path/{with_param}/here
	AllowedPaths []string
}

func NewMeasure(config MeasureOptions) *Measure {
	if config.Buckets == nil {
		config.Buckets = prometheus.DefBuckets
	}

	allowedPaths := make(map[string]bool)
	var compiledPatterns []compiledPattern

	for _, path := range config.AllowedPaths {
		if strings.Contains(path, "{") {
			compiled := compilePattern(path)
			compiledPatterns = append(compiledPatterns, compiled)
		} else {
			allowedPaths[path] = true
		}
	}

	requestDuration := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds",
			Buckets:   config.Buckets,
			ConstLabels: prometheus.Labels{
				"service": config.ServiceName,
			},
		},
		[]string{"method", "path", "status", "result"},
	)

	requestCount := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "requests_total",
			Help:      "Total number of HTTP requests",
			ConstLabels: prometheus.Labels{
				"service": config.ServiceName,
			},
		},
		[]string{"method", "path", "status", "result"},
	)

	requestResult := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "request_results_total",
			Help:      "Total number of HTTP requests by result",
			ConstLabels: prometheus.Labels{
				"service": config.ServiceName,
			},
		},
		[]string{"method", "path", "result"},
	)

	return &Measure{
		requestDuration: requestDuration,
		requestCount:    requestCount,
		requestResult:   requestResult,
		allowedPatterns: compiledPatterns,
		allowedPaths:    allowedPaths,
	}
}

func compilePattern(pattern string) compiledPattern {
	parts := strings.Split(pattern, "/")
	isParam := make([]bool, len(parts))

	for i, part := range parts {
		isParam[i] = strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}")
	}

	return compiledPattern{
		original:   pattern,
		parts:      parts,
		isParam:    isParam,
		partsCount: len(parts),
	}
}

func (m *Measure) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			duration := time.Since(start).Seconds()

			path := m.normalizePath(r.URL.Path)
			method := r.Method
			status := strconv.Itoa(ww.Status())
			result := m.getResult(ww.Status())

			m.requestDuration.WithLabelValues(method, path, status, result).Observe(duration)
			m.requestCount.WithLabelValues(method, path, status, result).Inc()
			m.requestResult.WithLabelValues(method, path, result).Inc()
		})
	}
}

func (m *Measure) normalizePath(requestPath string) string {
	if m.allowedPaths[requestPath] {
		return requestPath
	}

	for _, compiled := range m.allowedPatterns {
		if match := m.matchCompiledPattern(compiled, requestPath); match != "" {
			return match
		}
	}

	return "other"
}

func (m *Measure) matchCompiledPattern(compiled compiledPattern, requestPath string) string {
	requestParts := strings.Split(requestPath, "/")

	if len(requestParts) != compiled.partsCount {
		return ""
	}

	for i := 0; i < compiled.partsCount; i++ {
		if !compiled.isParam[i] && compiled.parts[i] != requestParts[i] {
			return ""
		}
	}

	return compiled.original
}

func (m *Measure) getResult(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "success"
	case status >= 300 && status < 400:
		return "redirect"
	case status >= 400 && status < 500:
		return "client_error"
	case status >= 500:
		return "server_error"
	default:
		return "unknown"
	}
}

func (m *Measure) Handler() http.Handler {
	return promhttp.Handler()
}
