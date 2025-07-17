package xhttp

import (
	"crypto/tls"
	"fmt"

	"github.com/go-chi/chi/v5"
)

type CertMasterOpts struct {
	Email        string
	CacheDir     string
	NonSSLRouter chi.Router
	Domains      []string
	ForeignPath  string
}

type CertMaster struct {
	stopped bool
	issuers []*Issuer
	foreign *CertWatcher
}

func NewCertMaster(opts *CertMasterOpts) (*CertMaster, error) {
	m := &CertMaster{
		issuers: []*Issuer{},
	}
	var err error
	if opts.ForeignPath != "" {
		m.foreign, err = NewCertWatcher(opts.ForeignPath)
		if err != nil {
			return nil, err
		}
	}
	for _, d := range opts.Domains {
		issuer, err := NewIssuer(&IssuerOpts{
			Domain:       d,
			CacheDir:     opts.CacheDir,
			Email:        opts.Email,
			NonSSLRouter: opts.NonSSLRouter,
		})
		if err != nil {
			m.Shutdown()
			return nil, err
		}
		m.issuers = append(m.issuers, issuer)
	}

	return m, nil
}

func (m *CertMaster) GetCertificate(h *tls.ClientHelloInfo) (*tls.Certificate, error) {
	for _, i := range m.issuers {
		if cert := i.GetCertificate(h.ServerName); cert != nil {
			return cert, nil
		}
	}

	if m.foreign != nil {
		return m.foreign.GetCertificate(h.ServerName)
	}

	return nil, fmt.Errorf("unknown domain name")
}

func (m *CertMaster) Shutdown() error {
	m.stopped = true
	for _, i := range m.issuers {
		i.Shutdown()
	}
	return nil
}

func (m *CertMaster) Running() bool {
	return !m.stopped
}
