package tlsconfig

import (
	"crypto/tls"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
	"github.com/burn-lab-dev/cantcp/internal/testcert"
)

// serverFiles generates a CA and a server certificate for 127.0.0.1.
func serverFiles(t *testing.T) testcert.Files {
	t.Helper()
	files, err := testcert.ServerFiles(t.TempDir())
	if err != nil {
		t.Fatalf("testcert.ServerFiles() = %v", err)
	}
	return files
}

// clientFiles generates a CA and a client certificate issued by it.
func clientFiles(t *testing.T) testcert.Files {
	t.Helper()
	files, err := testcert.ServerFiles(t.TempDir())
	if err != nil {
		t.Fatalf("testcert.ServerFiles() = %v", err)
	}
	client, err := testcert.ClientFiles(t.TempDir(), files.CA, files.CAKey)
	if err != nil {
		t.Fatalf("testcert.ClientFiles() = %v", err)
	}
	client.CAFile = files.CAFile
	client.CAPEM = files.CAPEM
	return client
}

func TestNewServer_Config(t *testing.T) {
	files := serverFiles(t)
	tests := []struct {
		name       string
		cfg        domain.ConfigTLS
		wantAuth   tls.ClientAuthType
		wantCAFile bool
	}{
		{
			name:     "no client certificates",
			cfg:      domain.ConfigTLS{CertFile: files.CertFile, KeyFile: files.KeyFile, ClientAuth: domain.ClientAuthNone},
			wantAuth: tls.NoClientCert,
		},
		{
			name:       "verify if given",
			cfg:        domain.ConfigTLS{CertFile: files.CertFile, KeyFile: files.KeyFile, CAFile: files.CAFile, ClientAuth: domain.ClientAuthVerifyIfGiven},
			wantAuth:   tls.VerifyClientCertIfGiven,
			wantCAFile: true,
		},
		{
			name:       "require and verify",
			cfg:        domain.ConfigTLS{CertFile: files.CertFile, KeyFile: files.KeyFile, CAFile: files.CAFile, ClientAuth: domain.ClientAuthRequireAndVerify},
			wantAuth:   tls.RequireAndVerifyClientCert,
			wantCAFile: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewServer(tt.cfg)
			if err != nil {
				t.Fatalf("NewServer() = %v, want nil", err)
			}
			conf := r.Config()
			if conf == nil {
				t.Fatal("Config() = nil")
			}
			if conf.MinVersion != tls.VersionTLS13 || conf.MaxVersion != tls.VersionTLS13 {
				t.Fatalf("versions = %x..%x, want TLS 1.3 only", conf.MinVersion, conf.MaxVersion)
			}
			if conf.ClientAuth != tt.wantAuth {
				t.Fatalf("ClientAuth = %v, want %v", conf.ClientAuth, tt.wantAuth)
			}
			if tt.wantCAFile && conf.ClientCAs == nil {
				t.Fatal("ClientCAs = nil, want a pool")
			}
			if !tt.wantCAFile && conf.ClientCAs != nil {
				t.Fatal("ClientCAs must be nil without a CA file")
			}
			if len(conf.Certificates) != 1 {
				t.Fatalf("Certificates = %d, want 1", len(conf.Certificates))
			}
		})
	}
}

func TestNewServer_Errors(t *testing.T) {
	tests := []struct {
		name string
		cfg  domain.ConfigTLS
	}{
		{
			name: "missing certificate",
			cfg:  domain.ConfigTLS{CertFile: "missing.pem", KeyFile: "missing-key.pem", ClientAuth: domain.ClientAuthNone},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewServer(tt.cfg); err == nil {
				t.Fatal("NewServer() = nil, want an error")
			}
		})
	}
}

func TestReloader_ReplacesTheConfiguration(t *testing.T) {
	files := serverFiles(t)
	r, err := NewServer(domain.ConfigTLS{CertFile: files.CertFile, KeyFile: files.KeyFile, ClientAuth: domain.ClientAuthNone})
	if err != nil {
		t.Fatalf("NewServer() = %v, want nil", err)
	}
	first := r.Config()

	// Issue a new certificate over the same files and reload.
	rotated, err := testcert.ServerFiles(t.TempDir())
	if err != nil {
		t.Fatalf("testcert.ServerFiles() = %v", err)
	}
	if err := os.WriteFile(files.CertFile, mustRead(t, rotated.CertFile), 0o600); err != nil {
		t.Fatalf("WriteFile() = %v", err)
	}
	if err := os.WriteFile(files.KeyFile, mustRead(t, rotated.KeyFile), 0o600); err != nil {
		t.Fatalf("WriteFile() = %v", err)
	}
	if err := r.Reload(domain.ConfigTLS{CertFile: files.CertFile, KeyFile: files.KeyFile, ClientAuth: domain.ClientAuthNone}); err != nil {
		t.Fatalf("Reload() = %v, want nil", err)
	}
	second := r.Config()
	if second == first {
		t.Fatal("Reload() did not replace the configuration")
	}

	// A broken configuration must not replace the current one.
	if err := r.Reload(domain.ConfigTLS{CertFile: "missing.pem", KeyFile: "missing.pem", ClientAuth: domain.ClientAuthNone}); err == nil {
		t.Fatal("Reload() = nil, want an error")
	}
	if got := r.Config(); got != second {
		t.Fatal("a failed Reload() must keep the current configuration")
	}
}

// mustRead reads a file or fails the test.
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) = %v", path, err)
	}
	return data
}

// handshake runs a TLS handshake over an in-memory pipe and returns both
// errors.
func handshake(t *testing.T, serverConf, clientConf *tls.Config) (clientErr, serverErr error) {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	// The deadline bounds the negative cases: a rejected client keeps the
	// server waiting for its next record, the deadline surfaces the failure.
	deadline := time.Now().Add(2 * time.Second)
	_ = serverConn.SetDeadline(deadline)
	_ = clientConn.SetDeadline(deadline)
	done := make(chan error, 1)
	go func() {
		sc := tls.Server(serverConn, serverConf)
		done <- sc.Handshake()
	}()
	cc := tls.Client(clientConn, clientConf)
	clientErr = cc.Handshake()
	serverErr = <-done
	return clientErr, serverErr
}

func TestHandshake_TLS13Only(t *testing.T) {
	files := serverFiles(t)
	r, err := NewServer(domain.ConfigTLS{CertFile: files.CertFile, KeyFile: files.KeyFile, ClientAuth: domain.ClientAuthNone})
	if err != nil {
		t.Fatalf("NewServer() = %v, want nil", err)
	}
	pool, err := loadCAPool(files.CAFile)
	if err != nil {
		t.Fatalf("loadCAPool() = %v", err)
	}
	clientConf := &tls.Config{RootCAs: pool, ServerName: "localhost", MinVersion: tls.VersionTLS13}
	clientErr, serverErr := handshake(t, r.Config(), clientConf)
	if clientErr != nil || serverErr != nil {
		t.Fatalf("TLS 1.3 handshake failed: client=%v server=%v", clientErr, serverErr)
	}

	legacy := &tls.Config{RootCAs: pool, ServerName: "localhost", MaxVersion: tls.VersionTLS12}
	clientErr, serverErr = handshake(t, r.Config(), legacy)
	if clientErr == nil && serverErr == nil {
		t.Fatal("TLS 1.2 handshake succeeded, want a protocol version error")
	}
}

func TestHandshake_MTLS(t *testing.T) {
	files := serverFiles(t)
	client, err := testcert.ClientFiles(t.TempDir(), files.CA, files.CAKey)
	if err != nil {
		t.Fatalf("testcert.ClientFiles() = %v", err)
	}
	pool, err := loadCAPool(files.CAFile)
	if err != nil {
		t.Fatalf("loadCAPool() = %v", err)
	}

	requireServer, err := NewServer(domain.ConfigTLS{
		CertFile: files.CertFile, KeyFile: files.KeyFile, CAFile: files.CAFile,
		ClientAuth: domain.ClientAuthRequireAndVerify,
	})
	if err != nil {
		t.Fatalf("NewServer() = %v", err)
	}
	// In TLS 1.3 the client finishes its side of the handshake before the
	// server verifies the certificate, so the failure is observed on the
	// server side: both sides must succeed to call it a handshake.
	withoutCert := &tls.Config{RootCAs: pool, ServerName: "localhost", MinVersion: tls.VersionTLS13}
	if clientErr, serverErr := handshake(t, requireServer.Config(), withoutCert); clientErr == nil && serverErr == nil {
		t.Fatal("mTLS handshake without a client certificate succeeded, want a failure")
	}

	clientPair, err := tls.LoadX509KeyPair(client.CertFile, client.KeyFile)
	if err != nil {
		t.Fatalf("LoadX509KeyPair() = %v", err)
	}
	withCert := &tls.Config{
		RootCAs: pool, ServerName: "localhost", MinVersion: tls.VersionTLS13,
		Certificates: []tls.Certificate{clientPair},
	}
	clientErr, serverErr := handshake(t, requireServer.Config(), withCert)
	if clientErr != nil || serverErr != nil {
		t.Fatalf("mTLS handshake failed: client=%v server=%v", clientErr, serverErr)
	}

	ifGiven, err := NewServer(domain.ConfigTLS{
		CertFile: files.CertFile, KeyFile: files.KeyFile, CAFile: files.CAFile,
		ClientAuth: domain.ClientAuthVerifyIfGiven,
	})
	if err != nil {
		t.Fatalf("NewServer() = %v", err)
	}
	if clientErr, serverErr := handshake(t, ifGiven.Config(), withoutCert); clientErr != nil || serverErr != nil {
		t.Fatalf("verify_if_given rejected a client without a certificate: client=%v server=%v", clientErr, serverErr)
	}
}

func TestNewClient(t *testing.T) {
	files := clientFiles(t)
	tests := []struct {
		name         string
		cfg          domain.ConfigTLSClient
		wantRootCAs  bool
		wantCert     bool
		wantName     string
		wantInsecure bool
	}{
		{
			name: "system roots",
			cfg:  domain.ConfigTLSClient{Enable: true},
		},
		{
			name:        "custom CA",
			cfg:         domain.ConfigTLSClient{Enable: true, CAFile: files.CAFile},
			wantRootCAs: true,
		},
		{
			name: "mTLS",
			cfg: domain.ConfigTLSClient{
				Enable: true, CAFile: files.CAFile,
				CertFile: files.CertFile, KeyFile: files.KeyFile, ServerName: "can.example",
			},
			wantRootCAs: true,
			wantCert:    true,
			wantName:    "can.example",
		},
		{
			name:         "insecure",
			cfg:          domain.ConfigTLSClient{Enable: true, Insecure: true},
			wantInsecure: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf, err := NewClient(tt.cfg)
			if err != nil {
				t.Fatalf("NewClient() = %v, want nil", err)
			}
			if conf.MinVersion != tls.VersionTLS13 || conf.MaxVersion != tls.VersionTLS13 {
				t.Fatalf("versions = %x..%x, want TLS 1.3 only", conf.MinVersion, conf.MaxVersion)
			}
			if (conf.RootCAs != nil) != tt.wantRootCAs {
				t.Fatalf("RootCAs set = %v, want %v", conf.RootCAs != nil, tt.wantRootCAs)
			}
			if (len(conf.Certificates) > 0) != tt.wantCert {
				t.Fatalf("Certificates set = %v, want %v", len(conf.Certificates) > 0, tt.wantCert)
			}
			if conf.ServerName != tt.wantName {
				t.Fatalf("ServerName = %q, want %q", conf.ServerName, tt.wantName)
			}
			if conf.InsecureSkipVerify != tt.wantInsecure {
				t.Fatalf("InsecureSkipVerify = %v, want %v", conf.InsecureSkipVerify, tt.wantInsecure)
			}
		})
	}
}

func TestNewClient_Errors(t *testing.T) {
	dir := t.TempDir()
	emptyCA := filepath.Join(dir, "empty.pem")
	if err := os.WriteFile(emptyCA, []byte("not a certificate"), 0o600); err != nil {
		t.Fatalf("WriteFile() = %v", err)
	}
	tests := []struct {
		name string
		cfg  domain.ConfigTLSClient
	}{
		{
			name: "missing CA file",
			cfg:  domain.ConfigTLSClient{Enable: true, CAFile: filepath.Join(dir, "missing.pem")},
		},
		{
			name: "empty CA file",
			cfg:  domain.ConfigTLSClient{Enable: true, CAFile: emptyCA},
		},
		{
			name: "broken client certificate",
			cfg: domain.ConfigTLSClient{
				Enable: true, CertFile: emptyCA, KeyFile: filepath.Join(dir, "missing.pem"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewClient(tt.cfg); err == nil {
				t.Fatal("NewClient() = nil, want an error")
			}
		})
	}
}
