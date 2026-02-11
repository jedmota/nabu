package ca

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func setupTestDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	caDirOverride = dir
	t.Cleanup(func() { caDirOverride = "" })
}

// --- Generate ---

func TestGenerate(t *testing.T) {
	setupTestDir(t)

	ca, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Files should exist
	dir := getCADir()
	if _, err := os.Stat(filepath.Join(dir, caCertFile)); err != nil {
		t.Error("ca.crt should exist")
	}
	if _, err := os.Stat(filepath.Join(dir, caKeyFile)); err != nil {
		t.Error("ca.key should exist")
	}

	// CertPEM should be non-empty
	if len(ca.CertPEM()) == 0 {
		t.Error("CertPEM should be non-empty")
	}

	// Fingerprint should be valid hex
	fp := ca.Fingerprint()
	if fp == "" {
		t.Error("Fingerprint should be non-empty")
	}
	// Remove any spaces and verify it's hex
	clean := strings.ReplaceAll(fp, " ", "")
	if _, err := hex.DecodeString(clean); err != nil {
		t.Errorf("Fingerprint %q is not valid hex: %v", fp, err)
	}

	// CertPath should be correct
	expected := filepath.Join(dir, caCertFile)
	if ca.CertPath() != expected {
		t.Errorf("CertPath = %q, want %q", ca.CertPath(), expected)
	}
}

// --- Load ---

func TestLoad_GeneratesIfMissing(t *testing.T) {
	setupTestDir(t)

	ca, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(ca.CertPEM()) == 0 {
		t.Error("should have generated a CA")
	}
}

func TestLoad_ExistingCA(t *testing.T) {
	setupTestDir(t)

	// Generate first
	ca1, err := Generate()
	if err != nil {
		t.Fatal(err)
	}

	// Load should return the same CA
	ca2, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if ca1.Fingerprint() != ca2.Fingerprint() {
		t.Error("loaded CA should have same fingerprint as generated")
	}
}

// --- GenerateCert ---

func TestGenerateCert(t *testing.T) {
	setupTestDir(t)

	ca, err := Generate()
	if err != nil {
		t.Fatal(err)
	}

	cert, err := ca.GenerateCert("example.com")
	if err != nil {
		t.Fatalf("GenerateCert: %v", err)
	}
	if cert == nil {
		t.Fatal("cert should not be nil")
	}

	// Second call should return cached
	cert2, err := ca.GenerateCert("example.com")
	if err != nil {
		t.Fatal(err)
	}
	if cert != cert2 {
		t.Error("second call should return cached certificate")
	}
}

func TestGenerateCert_IPAddress(t *testing.T) {
	setupTestDir(t)

	ca, err := Generate()
	if err != nil {
		t.Fatal(err)
	}

	cert, err := ca.GenerateCert("127.0.0.1")
	if err != nil {
		t.Fatalf("GenerateCert with IP: %v", err)
	}
	if cert == nil {
		t.Fatal("cert should not be nil for IP address")
	}
}

// --- Concurrent GenerateCert ---

func TestGenerateCert_Concurrent(t *testing.T) {
	setupTestDir(t)

	ca, err := Generate()
	if err != nil {
		t.Fatal(err)
	}

	hosts := []string{"a.com", "b.com", "c.com", "d.com", "e.com"}
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		for _, h := range hosts {
			wg.Add(1)
			go func(host string) {
				defer wg.Done()
				_, err := ca.GenerateCert(host)
				if err != nil {
					t.Errorf("GenerateCert(%q): %v", host, err)
				}
			}(h)
		}
	}
	wg.Wait()
}
