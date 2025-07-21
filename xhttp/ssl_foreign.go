package xhttp

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type CertWatcher struct {
	path       string
	certs      atomic.Pointer[map[string]*tls.Certificate]
	shutdown   chan struct{}
	shutdownWg sync.WaitGroup
}

func NewCertWatcher(path string) (*CertWatcher, error) {
	i := &CertWatcher{
		path: path,
	}

	i.shutdownWg.Add(1)
	go i.run()

	return i, nil
}

func (i *CertWatcher) GetCertificate(domain string) (*tls.Certificate, error) {
	certs := i.certs.Load()
	cert, ok := (*certs)[domain]
	if !ok {
		return nil, fmt.Errorf("unknown domain name")
	}

	return cert, nil
}

func (i *CertWatcher) run() {
	defer i.shutdownWg.Done()
	for {
		i.sync()
		select {
		case <-time.After(time.Second * 60):
			continue
		case <-i.shutdown:
			return
		}
	}
}

func (i *CertWatcher) list() ([]string, error) {
	entries, err := os.ReadDir(i.path)
	if err != nil {
		return nil, err
	}

	paths := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			paths = append(paths, e.Name())
		}
	}
	return paths, nil
}

func (i *CertWatcher) sync() {
	//TODO: Sergey Kovalev: Implement mtime lookup
	if i.path == "" {
		return
	}

	paths, err := i.list()
	if err != nil {
		return
	}

	certs := map[string]*tls.Certificate{}
	for _, path := range paths {
		certPath := i.path + "/" + path
		domain, cert, err := i.read(certPath)
		if err != nil {
			zap.L().Error("Can't read cert", zap.Error(err), zap.String("path", certPath))
		}

		certs[domain] = cert
	}

	i.certs.Store(&certs)
}

func (i *CertWatcher) read(path string) (domainName string, cert *tls.Certificate, err error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}

	var leCert leCertInfo
	err = json.Unmarshal(bs, &leCert)
	if err != nil {
		return "", nil, err
	}

	tlsCert, err := parseX509(leCert.Cert, leCert.Key)
	if err != nil {
		return "", nil, err
	}

	return leCert.Domain, tlsCert, nil
}

func (i *CertWatcher) Shutdown() error {
	close(i.shutdown)

	return nil
}
