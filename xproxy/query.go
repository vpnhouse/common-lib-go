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

type transport interface {
	Dial(addr string) (net.Conn, error)
	HttpClient() *http.Client
}

type Instance struct {
	Transport    transport
	MarkHeader   string
	AuthCallback func(authType, authInfo string) error
}

func (i *Instance) doPairedForward(wg *sync.WaitGroup, src, dst io.ReadWriteCloser) {
	defer wg.Done()
	defer dst.Close()

	for {
		buffer := make([]byte, 4096)
		len, err := src.Read(buffer)
		if err != nil {
			return
		}

		// TODO: Handle length
		_, err = dst.Write(buffer[:len])
		if err != nil {
			return
		}
	}
}

func (i *Instance) handleV1Connect(w http.ResponseWriter, r *http.Request) {
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
	go i.doPairedForward(&wg, clientConn, remoteConn)
	go i.doPairedForward(&wg, remoteConn, clientConn)
	wg.Wait()
}

func (i *Instance) handleV2Connect(w http.ResponseWriter, r *http.Request) {
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
	go i.doPairedForward(&wg, clientConn, remoteConn)
	go i.doPairedForward(&wg, remoteConn, clientConn)
	wg.Wait()
}

func (i *Instance) handleProxy(w http.ResponseWriter, r *http.Request) {
	if r.ProtoMajor == 2 {
		http.Error(w, "Bad request", http.StatusHTTPVersionNotSupported)
	}

	proxyReq, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		zap.L().Error("Error creating proxy request", zap.Error(err))
		http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
		return
	}

	r.Header.Del("Proxy-Connection")
	r.Header.Del("Proxy-Authenticate")
	r.Header.Del("Proxy-Authorization")
	if i.MarkHeader != "" {
		r.Header.Add(i.MarkHeader, xrand.RandomString(8))
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
	io.Copy(w, resp.Body)
}

func (i *Instance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if i.AuthCallback != nil {
		authType, authInfo := xhttp.ExtractAuthorizationInfo(r, xhttp.HeaderProxyAuthorization)
		if authInfo == "" {
			w.Header()["Proxy-Authenticate"] = []string{"Basic realm=\"proxy\""}
			http.Error(w, "Proxy authentication required", http.StatusProxyAuthRequired)
			return
		}

		err := i.AuthCallback(authType, authInfo)
		if err != nil {
			zap.L().Error("Authentication failed", zap.Error(err))
			http.Error(w, "Authentication failed", http.StatusForbidden)
			return
		}
	}

	if r.Method == "CONNECT" {
		if r.ProtoMajor == 1 {
			i.handleV1Connect(w, r)
			return
		}

		if r.ProtoMajor == 2 {
			i.handleV2Connect(w, r)
			return
		}

		http.Error(w, "Unsupported protocol version", http.StatusHTTPVersionNotSupported)
		return
	} else {
		i.handleProxy(w, r)
	}
}
