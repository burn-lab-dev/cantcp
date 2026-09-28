package domain

// ConfigTLSClient holds the client-side TLS material. Enable switches the
// client to TLS 1.3; CAFile replaces the system roots, CertFile and KeyFile
// add a client certificate (mTLS), ServerName overrides the verified name,
// and Insecure disables verification for debugging only.
type ConfigTLSClient struct {
	Enable     bool
	CAFile     string
	CertFile   string
	KeyFile    string
	ServerName string
	Insecure   bool
}

// Validate checks the TLS client settings.
func (c ConfigTLSClient) Validate() Error {
	if c.CertFile == "" && c.KeyFile != "" {
		return ErrInvalid("tls.cert_file: must be set together with tls.key_file")
	}
	if c.KeyFile == "" && c.CertFile != "" {
		return ErrInvalid("tls.key_file: must be set together with tls.cert_file")
	}
	if !c.Enable && (c.CAFile != "" || c.CertFile != "" || c.ServerName != "" || c.Insecure) {
		return ErrInvalid("tls: client TLS settings require tls.enable")
	}
	return nil
}
