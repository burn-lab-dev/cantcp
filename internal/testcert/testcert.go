// Package testcert generates self-signed certificate chains for tests: a CA,
// server certificates and client certificates for mTLS.
package testcert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Files holds the certificate material written to disk by ServerFiles and
// ClientFiles.
type Files struct {
	// CertFile, KeyFile and CAFile are the written paths; an empty path
	// means the file was not requested.
	CertFile string
	KeyFile  string
	CAFile   string
	// CAPEM is the PEM encoding of the CA certificate.
	CAPEM []byte
	// CA and CAKey are the CA material used for issuing.
	CA    *x509.Certificate
	CAKey *ecdsa.PrivateKey
}

// ServerFiles generates a CA and a server certificate for localhost and
// 127.0.0.1 and writes them into dir.
func ServerFiles(dir string) (Files, error) {
	ca, caKey, caPEM, err := NewCA("cantcp test CA")
	if err != nil {
		return Files{}, err
	}
	certPEM, keyPEM, err := Issue(ca, caKey, "cantcp test server",
		[]string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, false)
	if err != nil {
		return Files{}, err
	}
	return store(dir, ca, caKey, caPEM, certPEM, keyPEM)
}

// ClientFiles issues a client certificate from the given CA and writes it
// into dir.
func ClientFiles(dir string, ca *x509.Certificate, caKey *ecdsa.PrivateKey) (Files, error) {
	certPEM, keyPEM, err := Issue(ca, caKey, "cantcp test client", nil, nil, true)
	if err != nil {
		return Files{}, err
	}
	return store(dir, nil, nil, nil, certPEM, keyPEM)
}

// store writes the non-empty PEM blocks into dir and returns the paths.
func store(dir string, ca *x509.Certificate, caKey *ecdsa.PrivateKey, caPEM, certPEM, keyPEM []byte) (Files, error) {
	files := Files{CAPEM: caPEM, CA: ca, CAKey: caKey}
	list := []struct {
		path *string
		name string
		data []byte
	}{
		{&files.CertFile, "cert.pem", certPEM},
		{&files.KeyFile, "key.pem", keyPEM},
		{&files.CAFile, "ca.pem", caPEM},
	}
	for _, item := range list {
		if len(item.data) == 0 {
			continue
		}
		*item.path = filepath.Join(dir, item.name)
		if err := os.WriteFile(*item.path, item.data, 0o600); err != nil {
			return Files{}, fmt.Errorf("testcert: write %s: %w", *item.path, err)
		}
	}
	return files, nil
}

// validity is the certificate lifetime used by the tests.
const validity = time.Hour

// NewCA generates a self-signed ECDSA P-256 CA and returns its certificate,
// its key and the PEM encoding of the certificate.
func NewCA(commonName string) (*x509.Certificate, *ecdsa.PrivateKey, []byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("testcert: generate CA key: %w", err)
	}
	serial, err := serialNumber()
	if err != nil {
		return nil, nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(validity),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("testcert: create CA certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("testcert: parse CA certificate: %w", err)
	}
	return cert, key, pemCert(der), nil
}

// Issue generates a leaf certificate signed by the CA. dnsNames and ips are
// added to the subject alternative names; client selects the client
// authentication usage instead of the server one. It returns the PEM
// certificate and the PEM key.
func Issue(ca *x509.Certificate, caKey *ecdsa.PrivateKey, commonName string, dnsNames []string, ips []net.IP, client bool) ([]byte, []byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("testcert: generate key: %w", err)
	}
	serial, err := serialNumber()
	if err != nil {
		return nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(validity),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		DNSNames:     dnsNames,
		IPAddresses:  ips,
	}
	if client {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("testcert: create certificate: %w", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("testcert: marshal key: %w", err)
	}
	return pemCert(der), pemKey(keyDER), nil
}

// serialNumber returns a random positive certificate serial number.
func serialNumber() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	n, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return nil, fmt.Errorf("testcert: generate serial number: %w", err)
	}
	return n, nil
}

// pemCert encodes a certificate as PEM.
func pemCert(der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// pemKey encodes a key as PEM.
func pemKey(der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
}
