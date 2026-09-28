package tlsconfig

import (
	"crypto/x509"
	"fmt"
	"os"
)

// loadCAPool reads a PEM file into a certificate pool. The file must contain
// at least one certificate.
func loadCAPool(path string) (*x509.CertPool, error) {
	pem, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("tls: read CA file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("tls: CA file %s: no certificates found", path)
	}
	return pool, nil
}
