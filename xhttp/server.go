// Copyright 2021 The VPN House Authors. All rights reserved.
// Use of this source code is governed by a AGPL-style
// license that can be found in the LICENSE file.

package xhttp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chi_middleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
	middlewarestd "github.com/slok/go-http-metrics/middleware/std"
	openapi "github.com/vpnhouse/api/go/server/common"
	"go.uber.org/zap"
	"golang.org/x/net/idna"

	"github.com/vpnhouse/common-lib-go/xerror"
)

// initialize the measuring middleware only once
var (
	measureMW = middleware.New(middleware.Config{
		Recorder:      metrics.NewRecorder(metrics.Config{}),
		GroupedStatus: true,
	})
	MetricsSourceAllowed = []netip.Prefix{
		netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("172.16.0.0/12"),
	}
)

type Middleware = func(http.Handler) http.Handler

type Option func(w *Server)

func WithMiddleware(mw Middleware) Option {
	return func(w *Server) {
		w.router.Middlewares()
		w.router.Use(mw)
	}
}

func WithMetrics() Option {
	return func(w *Server) {
		// the measurement middleware
		w.router.Use(func(handler http.Handler) http.Handler {
			return middlewarestd.Handler("", measureMW, handler)
		})
		// route to obtain metrics
		w.router.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			s := strings.SplitN(r.RemoteAddr, ":", 2)
			zap.L().Debug("Metrics requested", zap.String("addr", s[0]))
			addr := netip.MustParseAddr(s[0])
			isAllowed := false
			for _, allowed := range MetricsSourceAllowed {
				if allowed.Contains(addr) {
					isAllowed = true
					break
				}
			}

			if !isAllowed {
				notFoundHandler(w, r)
				return
			}

			promhttp.Handler().ServeHTTP(w, r)

		})
	}
}

func WithCORS() Option {
	return func(w *Server) {
		cfg := cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{
				http.MethodHead,
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
			},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
		}
		w.router.Use(cors.Handler(cfg))
	}
}

func WithDisableHTTPv2() Option {
	return func(w *Server) {
		w.disablev2 = true
	}
}

func WithLogger() Option {
	return func(w *Server) {
		w.router.Use(requestLogger)
	}
}

func WithSSL(cfg *tls.Config) Option {
	return func(w *Server) {
		w.tlsConfig = cfg
	}
}

func WithPprof() Option {
	return func(w *Server) {
		w.router.Mount("/debug", chi_middleware.Profiler())
	}
}

type Server struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	tlsConfig *tls.Config
	router    chi.Router
	disablev2 bool
}

// Run starts the http server asynchronously.
func (w *Server) Run(addr string) error {
	srv := &http.Server{
		Handler:     w.router,
		Addr:        addr,
		TLSConfig:   w.tlsConfig,
		ReadTimeout: 10 * time.Second,
	}

	if w.disablev2 {
		srv.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return xerror.EInternalError("failed to start http listener", err, zap.String("addr", addr))
	}

	withTLS := w.tlsConfig != nil
	zap.L().Info("starting HTTP server", zap.String("addr", addr), zap.Bool("with_tls", withTLS))

	w.wg.Add(2)
	go func() {
		defer w.wg.Done()

		defer w.cancel()
		<-w.ctx.Done()
		srv.Shutdown(context.Background())
	}()

	go func() {
		defer w.wg.Done()

		var err error
		if withTLS {
			err = srv.ServeTLS(lis, "", "")
		} else {
			err = srv.Serve(lis)
		}

		if err != nil {
			zap.L().Error("http listener failed", zap.String("addr", addr), zap.Error(err))
		}
	}()

	return nil
}

// Router exposes chi.Router for the external registration of handlers.
// usage:
//
//	h.Router().Get("/apt/path", myHandler)
//	h.Router().Post("/apt/verb", myOtherHandler)
func (w *Server) Router() chi.Router {
	return w.router
}

func New(opts ...Option) *Server {
	r := chi.NewRouter()
	// always respond with JSON by using the custom error handlers
	r.NotFound(notFoundHandler)
	r.MethodNotAllowed(notAllowedHandler)

	ctx, cancel := context.WithCancel(context.Background())
	h := &Server{
		ctx:    ctx,
		cancel: cancel,
		router: r,
	}
	for _, o := range opts {
		o(h)
	}

	return h
}

func NewDefault() *Server {
	return New(
		WithLogger(),
		// WithMetrics must be declared last
		WithMetrics(),
	)
}

func NewDefaultSSL(cfg *tls.Config) *Server {
	return New(
		WithLogger(),
		WithMetrics(),
		WithSSL(cfg),
	)
}

func discoverRequestHost(r *http.Request) (string, error) {
	if r.Host == "" {
		return "", fmt.Errorf("host header is not set")
	}

	segments := strings.Split(r.Host, ":")
	if len(segments) > 2 {
		return "", fmt.Errorf("too many colon-separated segments")
	}

	if len(segments) > 1 {
		_, err := strconv.Atoi(segments[1])
		if err != nil {
			return "", fmt.Errorf("last segment is not integer")
		}
	}

	return idna.ToASCII(segments[0])
}

func NewRedirectToSSL(primaryHost string) *Server {
	r := chi.NewRouter()
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		host, err := discoverRequestHost(r)
		if err != nil {
			if primaryHost != "" {
				zap.L().Info("Can't determine request hostname, using primary", zap.Error(err))
				host = primaryHost
			} else {
				zap.L().Error("Can't determine redirection URL")
				w.Header().Set("Upgrade", "TLS/1.2, HTTP/1.1")
				w.WriteHeader(http.StatusUpgradeRequired)
				return
			}
		}

		url2 := *r.URL
		url2.Scheme = "https"
		url2.Host = host
		w.Header().Set("Location", url2.String())
		w.WriteHeader(http.StatusTemporaryRedirect)
	})

	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		ctx:    ctx,
		cancel: cancel,
		router: r,
	}
}

func (w *Server) Shutdown() error {
	w.cancel()
	w.wg.Wait()
	return nil
}

func (w *Server) Running() bool {
	return w.ctx.Err() == nil
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	txt := "Not found"
	err := openapi.Error{
		Result: "404",
		Error:  &txt,
	}
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(err)
}

func notAllowedHandler(w http.ResponseWriter, r *http.Request) {
	txt := "Method not allowed"
	err := openapi.Error{
		Result: "405",
		Error:  &txt,
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(err)
}
