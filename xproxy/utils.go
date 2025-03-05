package xproxy

import (
	"errors"
	"io"
	"net/http"
	"os"
	"regexp"
	"syscall"
)

var hasPort = regexp.MustCompile(`:\d+$`)

func remoteEndpoint(r *http.Request) string {
	host := r.URL.Host
	if !hasPort.MatchString(host) {
		host += ":80"
	}

	return host
}

func isConnectionClosed(err error) bool {
	if err == nil {
		return false
	}

	if err == io.EOF {
		return true
	}

	var syscallError *os.SyscallError
	if errors.As(err, &syscallError) {
		if syscallError.Err == syscall.EPIPE || syscallError.Err == syscall.ECONNRESET || syscallError.Err == syscall.EPROTOTYPE {
			return true
		}
	}

	return false
}
