package xproxy

import (
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/posener/h2conn"
	"github.com/vpnhouse/common-lib-go/xhttp"
	"github.com/vpnhouse/common-lib-go/xrand"
	"go.uber.org/zap"
)

type Transport interface {
	Dial(addr string) (net.Conn, error)
	HttpClient() *http.Client
}

type Reporter func(description any, n uint64)
type Authorizer func(authType, authInfo string) (description any, err error)

type Instance struct {
	MarkHeaderName string
	Transport      Transport
	AuthCallback   Authorizer
	StatsReportTx  Reporter
	StatsReportRx  Reporter
}

type accounter struct {
	description any
	reporter    Reporter
	parent      io.ReadCloser
}

func (i *accounter) Read(p []byte) (n int, err error) {
	if i.reporter != nil {
		i.reporter(i.description, uint64(n))
	}

	return i.parent.Read(p)
}

func (i *accounter) Close() error {
	return i.parent.Close()
}

func (i *Instance) doPairedForward(wg *sync.WaitGroup, src, dst io.ReadWriteCloser, description any, rep Reporter) {
	defer wg.Done()
	defer dst.Close()

	for {
		buffer := make([]byte, 4096)
		n, err := src.Read(buffer)
		if err != nil {
			return
		}

		// TODO: Handle length
		n, err = dst.Write(buffer[:n])
		if err != nil {
			return
		}
		rep(description, uint64(n))
	}
}

func (i *Instance) handleV1Connect(w http.ResponseWriter, r *http.Request, description any) {
	remoteConn, err := i.Transport.Dial(remoteEndpoint(r))
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijack not supported", http.StatusServiceUnavailable)
		zap.L().Error("Hijacking is not supported")
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		if r.Body != nil {
			defer r.Body.Close()
		}
		zap.L().Error("Hijack error", zap.Error(err))
		return
	}

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		clientConn.Close()
		remoteConn.Close()
		if !isConnectionClosed(err) {
			zap.L().Error("Can't write 200 OK response", zap.Error(err))
		}
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go i.doPairedForward(&wg, clientConn, remoteConn, description, i.StatsReportTx)
	go i.doPairedForward(&wg, remoteConn, clientConn, description, i.StatsReportRx)
	wg.Wait()
}

func (i *Instance) handleV2Connect(w http.ResponseWriter, r *http.Request, description any) {
	remoteConn, err := i.Transport.Dial(remoteEndpoint(r))
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	clientConn, err := h2conn.Accept(w, r)
	if err != nil {
		zap.L().Error("h2conn error", zap.Error(err))
		remoteConn.Close()
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go i.doPairedForward(&wg, clientConn, remoteConn, description, i.StatsReportTx)
	go i.doPairedForward(&wg, remoteConn, clientConn, description, i.StatsReportRx)
	wg.Wait()
}

func (i *Instance) handleProxy(w http.ResponseWriter, r *http.Request, description any) {
	if r.ProtoMajor == 2 {
		http.Error(w, "Bad request", http.StatusHTTPVersionNotSupported)
	}

	if r.URL.Scheme == "https" {
		zap.L().Warn("Attempt to proxy https", zap.String("host", r.URL.Host))
		http.Error(w, "Proxying HTTPS as plain text is dumb idea", http.StatusTeapot)
		return
	}

	proxyReq, err := http.NewRequest(r.Method, r.URL.String(),
		&accounter{
			description,
			i.StatsReportTx,
			r.Body,
		},
	)
	if err != nil {
		zap.L().Error("Error creating proxy request", zap.Error(err))
		http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
		return
	}

	r.Header.Del("Proxy-Connection")
	r.Header.Del("Proxy-Authenticate")
	r.Header.Del("Proxy-Authorization")
	if i.MarkHeaderName != "" {
		r.Header.Add(i.MarkHeaderName, xrand.RandomString(8))
	}

	// Copy the headers from the original request to the proxy request
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	// Send the proxy request using the custom transport
	resp, err := i.Transport.HttpClient().Do(proxyReq)
	if err != nil {
		http.Error(w, "Error sending proxy request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy the headers from the proxy response to the original response
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Set the status code of the original response to the status code of the proxy response
	w.WriteHeader(resp.StatusCode)

	// Copy the body of the proxy response to the original response
	io.Copy(w,
		&accounter{
			description,
			i.StatsReportTx,
			resp.Body,
		},
	)
}

func (i *Instance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		description any
		err         error
	)

	if i.AuthCallback != nil {
		authType, authInfo := xhttp.ExtractAuthorizationInfo(r, xhttp.HeaderProxyAuthorization)
		if authInfo == "" {
			w.Header()["Proxy-Authenticate"] = []string{"Basic realm=\"proxy\""}
			http.Error(w, "Proxy authentication required", http.StatusProxyAuthRequired)
			return
		}

		description, err = i.AuthCallback(authType, authInfo)
		if err != nil {
			zap.L().Error("Authentication failed", zap.Error(err))
			http.Error(w, "Authentication failed", http.StatusForbidden)
			return
		}
	}

	if r.Method == "CONNECT" {
		if r.ProtoMajor == 1 {
			i.handleV1Connect(w, r, description)
			return
		}

		if r.ProtoMajor == 2 {
			i.handleV2Connect(w, r, description)
			return
		}

		http.Error(w, "Unsupported protocol version", http.StatusHTTPVersionNotSupported)
		return
	} else {
		i.handleProxy(w, r, description)
	}
}
