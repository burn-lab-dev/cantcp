package domain

// ConfigTLS holds the server-side TLS material. TLS is enabled when the
// certificate and the key are set; without them the daemon runs in plain
// mode. TLS 1.3 is the only accepted version.
type ConfigTLS struct {
	CertFile   string
	KeyFile    string
	CAFile     string
	ClientAuth ClientAuth
}

// Enabled reports whether the TLS mode is selected.
func (c ConfigTLS) Enabled() bool { return c.CertFile != "" || c.KeyFile != "" }

// Validate checks the certificate/key pair and the client authentication
// mode. File readability and key matching are checked by the tlsconfig
// adapter at startup.
func (c ConfigTLS) Validate() Error {
	if c.CertFile == "" && c.KeyFile != "" {
		return ErrInvalid("tls.cert_file: must be set together with tls.key_file")
	}
	if c.KeyFile == "" && c.CertFile != "" {
		return ErrInvalid("tls.key_file: must be set together with tls.cert_file")
	}
	if !c.ClientAuth.valid() {
		return ErrInvalid("tls.client_auth: unknown value %q (want none, verify_if_given, require_and_verify)", c.ClientAuth)
	}
	if c.CAFile != "" && !c.Enabled() {
		return ErrInvalid("tls.ca_file: requires tls.cert_file and tls.key_file")
	}
	if c.ClientAuth != ClientAuthNone && !c.Enabled() {
		return ErrInvalid("tls.client_auth: %q requires tls.cert_file and tls.key_file", c.ClientAuth)
	}
	if c.ClientAuth != ClientAuthNone && c.CAFile == "" {
		return ErrInvalid("tls.ca_file: must be set when tls.client_auth is %q", c.ClientAuth)
	}
	return nil
}
