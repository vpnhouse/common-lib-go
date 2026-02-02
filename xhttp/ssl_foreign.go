package xhttp

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	// To be safe in GetCertificate init the empty cert's list first
	certs := make(map[string]*tls.Certificate, 1)
	i.certs.Store(&certs)
	// Perform sync
	i.sync()

	i.shutdownWg.Add(1)
	go i.run()

	return i, nil
}

func (i *CertWatcher) GetCertificate(domain string) (*tls.Certificate, error) {
	certs := i.certs.Load()
	cert, ok := (*certs)[domain]
	if ok {
		return cert, nil
	}

	if len(*certs) == 0 {
		// TODO: add fallback to some default certificate
		return nil, fmt.Errorf("certificates is not loaded yet")
	}

	return nil, fmt.Errorf("unknown domain name")
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

	paths := make([]string, 0, 1)
	for _, e := range entries {
		if e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".json") {
			paths = append(paths, filepath.Join(i.path, e.Name()))
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
		zap.L().Error("cannot detect certs in directory", zap.String("dir", i.path), zap.Error(err))
		return
	}

	certs := make(map[string]*tls.Certificate, 1)
	for _, certPath := range paths {
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
