package xproxy

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/posener/h2conn"
	"github.com/vpnhouse/common-lib-go/xhttp"
	"github.com/vpnhouse/common-lib-go/xrand"
	"go.uber.org/zap"
)

var (
	ErrAuthCallbackNotSet  = errors.New("auth callback is not set")
	ErrCantExtractAuthInfo = errors.New("can't extract authorization info")
)

type (
	Reporter   func(customInfo any, n uint64)
	Authorizer func(r *http.Request) (customInfo any, err error)
	Releaser   func(customInfo any)
	Transport  interface {
		Dial(addr string) (net.Conn, error)
		HttpClient() *http.Client
	}
)

type Instance struct {
	Name            string
	MarkHeaderName  string
	Transport       Transport
	AuthCallback    Authorizer
	ReleaseCallback Releaser
	StatsReportTx   Reporter
	StatsReportRx   Reporter
}

func (i *Instance) doPairedForward(wg *sync.WaitGroup, src, dst io.ReadWriteCloser, customInfo any, rep Reporter) {
	defer wg.Done()
	defer dst.Close()

	for {
		buffer := make([]byte, 4096)
		n, err := src.Read(buffer)
		if err != nil {
			return
		}

		n, err = dst.Write(buffer[:n])
		if err != nil {
			return
		}
		rep(customInfo, uint64(n))
	}
}

func (i *Instance) handleV1Connect(w http.ResponseWriter, r *http.Request, customInfo any) {
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
	go i.doPairedForward(&wg, clientConn, remoteConn, customInfo, i.StatsReportTx)
	go i.doPairedForward(&wg, remoteConn, clientConn, customInfo, i.StatsReportRx)
	wg.Wait()
}

func (i *Instance) handleV2Connect(w http.ResponseWriter, r *http.Request, customInfo any) {
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
	go i.doPairedForward(&wg, clientConn, remoteConn, customInfo, i.StatsReportTx)
	go i.doPairedForward(&wg, remoteConn, clientConn, customInfo, i.StatsReportRx)
	wg.Wait()
}

func (i *Instance) handleProxy(w http.ResponseWriter, r *http.Request, customInfo any) {
	// We can't actually receive remote url scheme from HTTP2 connection.
	// If it's fixed in golang - feel free to remove it. Also check listener to enable HTTP2 back.
	if r.ProtoMajor == 2 {
		http.Error(w, "Bad request", http.StatusHTTPVersionNotSupported)
	}

	// Check if we do not process https as plain text
	if r.URL.Scheme == "https" && r.Method != http.MethodOptions {
		zap.L().Warn("Attempt to proxy https", zap.String("host", r.URL.Host))
		http.Error(w, "Proxying HTTPS as plain text is dumb idea", http.StatusTeapot)
		return
	}

	// Create new request
	proxyReq, err := http.NewRequest(r.Method, r.URL.String(),
		&accounter{
			customInfo,
			i.StatsReportTx,
			r.Body,
		},
	)
	if err != nil {
		zap.L().Error("Error creating proxy request", zap.Error(err))
		http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
		return
	}

	// Remove proxy-related headers
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
			customInfo,
			i.StatsReportTx,
			resp.Body,
		},
	)
}
func (i *Instance) handleAuth(r *http.Request) (customInfo any, err error) {
	if i.AuthCallback == nil {
		return nil, ErrAuthCallbackNotSet
	}

	_, authInfo := xhttp.ExtractAuthorizationInfo(r, xhttp.HeaderProxyAuthorization)
	if authInfo == "" {
		return nil, ErrCantExtractAuthInfo
	}

	customInfo, err = i.AuthCallback(r)
	if err != nil {
		return nil, err
	}

	return customInfo, nil
}

func (i *Instance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	customInfo, err := i.handleAuth(r)
	if err != nil {
		// Preflight requests never send authorization headers
		// To prevent annoying users with login form simply bypass request to the target host
		// to get valid response
		if r.Method != http.MethodOptions {
			zap.L().Info("Proxy authentication failed",
				zap.String("method", r.Method),
				zap.Stringer("url", r.URL),
				zap.Any("headers", r.Header),
				zap.Int("protocol", r.ProtoMajor),
				zap.Error(err),
			)
			name := "proxy"
			if i.Name != "" {
				name = i.Name
			}
			w.Header()["Proxy-Authenticate"] = []string{fmt.Sprintf("Basic realm=\"%s\"", name)}
			http.Error(w, "Proxy authentication required", http.StatusProxyAuthRequired)
			return
		}
		zap.L().Info("Bypass proxy OPTIONS request to target server")
	}

	defer i.ReleaseCallback(customInfo)

	if r.Method == http.MethodConnect {
		if r.ProtoMajor == 1 {
			i.handleV1Connect(w, r, customInfo)
			return
		}

		if r.ProtoMajor == 2 {
			i.handleV2Connect(w, r, customInfo)
			return
		}

		http.Error(w, "Unsupported protocol version", http.StatusHTTPVersionNotSupported)
		return
	} else {
		i.handleProxy(w, r, customInfo)
	}
}
