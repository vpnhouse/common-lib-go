package tunnel

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/vpnhouse/tunnel/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	federationAuthHeader = "X-VPNHOUSE-FEDERATION-KEY"
)

type Client struct {
	client     proto.AdminServiceClient
	authSecret string
}

func NewClient(tunnelHostPort string, authSecret string) (*Client, error) {
	tunnelHost, tunnelPort, err := net.SplitHostPort(tunnelHostPort)
	if err != nil || tunnelPort == "" {
		tunnelHost = tunnelHostPort
		tunnelPort = "8089" // Default port
	}

	tunnelAddr := net.JoinHostPort(tunnelHost, tunnelPort)

	conn, err := net.Dial("tcp", tunnelAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %s", tunnelHost, err)
	}
	defer conn.Close()

	config := &tls.Config{ServerName: tunnelHost, InsecureSkipVerify: false}
	tlsConn := tls.Client(conn, config)
	if err := tlsConn.Handshake(); err != nil {
		return nil, fmt.Errorf("failed TLS handshake: %s", err)
	}
	defer tlsConn.Close()

	tlsCert := tls.Certificate{}
	for _, cert := range tlsConn.ConnectionState().PeerCertificates {
		tlsCert.Certificate = append(tlsCert.Certificate, cert.Raw)
	}

	tlsConfig := &tls.Config{
		ServerName:   tunnelHost,
		Certificates: []tls.Certificate{tlsCert},
	}

	creds := credentials.NewTLS(tlsConfig)

	zap.L().Info(
		"handshake tls succeed",
		zap.String("tunnel", tunnelHost),
		zap.Int("certificates", len(tlsConn.ConnectionState().PeerCertificates)),
	)

	cc, err := grpc.Dial(
		net.JoinHostPort(tunnelHost, tunnelPort),
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		zap.L().Fatal("failed to init grps client", zap.Error(err))
		return nil, fmt.Errorf("failed to init grps client: %w", err)
	}

	client := proto.NewAdminServiceClient(cc)
	return &Client{
		client:     client,
		authSecret: authSecret,
	}, nil
}
