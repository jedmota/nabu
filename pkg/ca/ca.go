package ca

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"time"
)

const (
	defaultCADir  = ".nabu"
	caCertFile    = "ca.crt"
	caKeyFile     = "ca.key"
	certCacheSize = 1000
)

// caDirOverride, when non-empty, is returned by getCADir instead of
// the default ~/. Tests in this package set it directly.
var caDirOverride string

// CA manages certificate generation and signing
type CA struct {
	cert      *x509.Certificate
	key       *rsa.PrivateKey
	certPEM   []byte
	keyPEM    []byte
	certCache sync.Map // map[string]*tls.Certificate
	mu        sync.RWMutex
}

// Load loads an existing CA or generates a new one
func Load() (*CA, error) {
	caDir := getCADir()
	certPath := filepath.Join(caDir, caCertFile)
	keyPath := filepath.Join(caDir, caKeyFile)

	// Try to load existing CA
	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {
			return loadFromFiles(certPath, keyPath)
		}
	}

	// Generate new CA
	return Generate()
}

// Generate creates a new CA certificate and key
func Generate() (*CA, error) {
	caDir := getCADir()
	if err := os.MkdirAll(caDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create CA directory: %w", err)
	}

	// Generate private key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Include hostname in the CA name for identification
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	username := ""
	if u, err := user.Current(); err == nil {
		username = u.Username
	}
	cnParts := "Nabu CA"
	if username != "" && hostname != "" {
		cnParts = fmt.Sprintf("Nabu CA (%s@%s)", username, hostname)
	} else if hostname != "" {
		cnParts = fmt.Sprintf("Nabu CA (%s)", hostname)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"Nabu"},
			OrganizationalUnit: []string{"Development"},
			CommonName:         cnParts,
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0), // Valid for 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	// Self-sign the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	// Save to files
	certPath := filepath.Join(caDir, caCertFile)
	keyPath := filepath.Join(caDir, caKeyFile)

	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return nil, fmt.Errorf("failed to save certificate: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, fmt.Errorf("failed to save private key: %w", err)
	}

	return &CA{
		cert:    cert,
		key:     key,
		certPEM: certPEM,
		keyPEM:  keyPEM,
	}, nil
}

// loadFromFiles loads CA from existing files
func loadFromFiles(certPath, keyPath string) (*CA, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode private key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &CA{
		cert:    cert,
		key:     key,
		certPEM: certPEM,
		keyPEM:  keyPEM,
	}, nil
}

// GenerateCert creates a certificate for a specific host
func (ca *CA) GenerateCert(host string) (*tls.Certificate, error) {
	// Check cache first
	if cached, ok := ca.certCache.Load(host); ok {
		return cached.(*tls.Certificate), nil
	}

	ca.mu.Lock()
	defer ca.mu.Unlock()

	// Double-check after acquiring lock
	if cached, ok := ca.certCache.Load(host); ok {
		return cached.(*tls.Certificate), nil
	}

	// Generate new certificate
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Nabu"},
			CommonName:   host,
		},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().AddDate(1, 0, 0), // Valid for 1 year
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	// Add host as SAN
	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	// Generate key for this certificate
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Sign with CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS certificate: %w", err)
	}

	// Cache the certificate
	ca.certCache.Store(host, &tlsCert)

	return &tlsCert, nil
}

// CertPEM returns the CA certificate in PEM format
func (ca *CA) CertPEM() []byte {
	return ca.certPEM
}

// CertPath returns the path to the CA certificate file
func (ca *CA) CertPath() string {
	return filepath.Join(getCADir(), caCertFile)
}

// Fingerprint returns the SHA256 fingerprint of the CA certificate
func (ca *CA) Fingerprint() string {
	hash := sha256.Sum256(ca.cert.Raw)
	return fmt.Sprintf("%X", hash)
}

// getCADir returns the directory for storing CA files
func getCADir() string {
	if caDirOverride != "" {
		return caDirOverride
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultCADir
	}
	return filepath.Join(home, defaultCADir)
}

// GetCADir returns the CA directory (exported)
func GetCADir() string {
	return getCADir()
}
