package xhttp

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Measure struct {
	requestDuration *prometheus.HistogramVec
	requestCount    *prometheus.CounterVec
	allowedPaths    map[string]bool
}

type MeasureOptions struct {
	Namespace    string
	Subsystem    string
	ServiceName  string
	Buckets      []float64
	AllowedPaths []string
}

func NewMeasure(config MeasureOptions) *Measure {
	if config.Buckets == nil {
		config.Buckets = prometheus.DefBuckets
	}

	if config.Namespace == "" {
		config.Namespace = "http"
	}

	if config.Subsystem == "" {
		config.Subsystem = "undefined"
	}

	allowedPaths := make(map[string]bool)
	for _, path := range config.AllowedPaths {
		allowedPaths[path] = true
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
		[]string{"method", "path", "status"},
	)

	requestCount := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "request_count",
			Help:      "Count of HTTP requests",
			ConstLabels: prometheus.Labels{
				"service": config.ServiceName,
			},
		},
		[]string{"method", "path", "status"},
	)

	return &Measure{
		requestDuration: requestDuration,
		requestCount:    requestCount,
		allowedPaths:    allowedPaths,
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

			m.requestDuration.WithLabelValues(method, path, status).Observe(duration)
			m.requestCount.WithLabelValues(method, path, status).Inc()
		})
	}
}

func (m *Measure) normalizePath(path string) string {
	if m.allowedPaths[path] {
		return path
	}
	return "other"
}
