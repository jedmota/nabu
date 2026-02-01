package proxy

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/elazarl/goproxy"

	"proxy-tui/pkg/ca"
)

// MITMConfig holds MITM configuration
type MITMConfig struct {
	CA             *ca.CA
	SkipVerify     bool
	ConnectTimeout time.Duration
	IdleTimeout    time.Duration
}

// DefaultMITMConfig returns default MITM configuration
func DefaultMITMConfig(certificate *ca.CA) *MITMConfig {
	return &MITMConfig{
		CA:             certificate,
		SkipVerify:     false,
		ConnectTimeout: 30 * time.Second,
		IdleTimeout:    60 * time.Second,
	}
}

// TLSConfigFactory creates TLS configurations for MITM
type TLSConfigFactory struct {
	ca        *ca.CA
	configs   sync.Map
	transport *http.Transport
}

// NewTLSConfigFactory creates a new TLS config factory
func NewTLSConfigFactory(certificate *ca.CA) *TLSConfigFactory {
	return &TLSConfigFactory{
		ca: certificate,
		transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// GetConfigForHost returns a TLS config for the given host
func (f *TLSConfigFactory) GetConfigForHost(host string) (*tls.Config, error) {
	// Check cache
	if cached, ok := f.configs.Load(host); ok {
		return cached.(*tls.Config), nil
	}

	// Generate certificate for this host
	cert, err := f.ca.GenerateCert(host)
	if err != nil {
		return nil, err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{*cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Cache the config
	f.configs.Store(host, config)

	return config, nil
}

// SetupMITM configures the proxy for MITM on HTTPS connections
func SetupMITM(proxy *goproxy.ProxyHttpServer, config *MITMConfig) {
	factory := NewTLSConfigFactory(config.CA)

	// Configure HTTPS handling - always MITM
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	// Set up TLS config for MITM
	goproxy.MitmConnect.TLSConfig = func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
		// Extract hostname without port
		hostname := host
		if h, _, err := net.SplitHostPort(host); err == nil {
			hostname = h
		}
		return factory.GetConfigForHost(hostname)
	}

	// Configure the CA certificate for goproxy
	keyPath := ca.GetCADir() + "/ca.key"
	keyPEM, err := os.ReadFile(keyPath)
	if err == nil {
		if caCert, err := tls.X509KeyPair(config.CA.CertPEM(), keyPEM); err == nil {
			goproxy.GoproxyCa = caCert
		}
	}
}
