// Copyright 2021 The Uranium Authors. All rights reserved.
// Use of this source code is governed by a AGPL-style
// license that can be found in the LICENSE file.

package xhttp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"time"

	openapi "github.com/Codename-Uranium/api/go/server/common"
	"github.com/Codename-Uranium/tunnel/pkg/xerror"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
	middlewarestd "github.com/slok/go-http-metrics/middleware/std"
	"go.uber.org/zap"
)

// initialize the measuring middleware only once
var measureMW = middleware.New(middleware.Config{
	Recorder:      metrics.NewRecorder(metrics.Config{}),
	GroupedStatus: true,
})

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
		w.router.Handle("/metrics", promhttp.Handler())
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

type Server struct {
	srv       *http.Server
	tlsConfig *tls.Config
	router    chi.Router
}

// Run starts the http server asynchronously.
func (w *Server) Run(addr string) error {
	w.srv = &http.Server{
		Handler:   w.router,
		Addr:      addr,
		TLSConfig: w.tlsConfig,
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return xerror.EInternalError("failed to start http listener", err, zap.String("addr", addr))
	}

	withTLS := w.tlsConfig != nil
	zap.L().Info("starting HTTP server", zap.String("addr", addr), zap.Bool("with_tls", withTLS))
	go func() {
		var err error
		if withTLS {
			err = w.srv.ServeTLS(lis, "", "")
		} else {
			err = w.srv.Serve(lis)
		}

		if err != nil {
			zap.L().Error("http listener failed", zap.String("addr", addr), zap.Error(err))
		}
	}()

	return nil
}

// Router exposes chi.Router for the external registration of handlers.
// usage:
// 		h.Router().Get("/apt/path", myHandler)
// 		h.Router().Post("/apt/verb", myOtherHandler)
func (w *Server) Router() chi.Router {
	return w.router
}

func New(opts ...Option) *Server {
	r := chi.NewRouter()
	// always respond with JSON by using the custom error handlers
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		txt := "Not found"
		err := openapi.Error{
			Result: "404",
			Error:  &txt,
		}
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(err)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		txt := "Method not allowed"
		err := openapi.Error{
			Result: "405",
			Error:  &txt,
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(err)
	})

	h := &Server{router: r}
	for _, o := range opts {
		o(h)
	}

	return h
}

func NewDefault() *Server {
	return New(
		WithLogger(),
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

func NewRedirectToSSL(to string) *Server {
	r := chi.NewRouter()
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		url2 := *r.URL
		if len(to) > 0 {
			// override host if present,
			// if we have ssl enabled with self-signed
			// certificate and no domain - redirect to the
			// required IP but with https schema.
			url2.Host = to
		}
		url2.Scheme = "https"
		w.Header().Set("Location", url2.String())
		w.WriteHeader(http.StatusTemporaryRedirect)
	})

	return &Server{
		router: r,
	}
}

func (w *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := w.srv.Shutdown(ctx)
	w.srv = nil

	return err
}

func (w *Server) Running() bool {
	return w.srv != nil
}