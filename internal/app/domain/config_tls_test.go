package domain

import (
	"errors"
	"testing"
)

func TestConfigTLS_Enabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  ConfigTLS
		want bool
	}{
		{name: "plain", cfg: ConfigTLS{ClientAuth: ClientAuthNone}, want: false},
		{name: "cert only", cfg: ConfigTLS{CertFile: "c.pem"}, want: true},
		{name: "key only", cfg: ConfigTLS{KeyFile: "k.pem"}, want: true},
		{name: "pair", cfg: ConfigTLS{CertFile: "c.pem", KeyFile: "k.pem"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.Enabled(); got != tt.want {
				t.Fatalf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigTLS_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ConfigTLS
		wantErr string
	}{
		{
			name: "plain",
			cfg:  ConfigTLS{ClientAuth: ClientAuthNone},
		},
		{
			name: "server certificate without client auth",
			cfg:  ConfigTLS{CertFile: "c.pem", KeyFile: "k.pem", ClientAuth: ClientAuthNone},
		},
		{
			name: "mTLS",
			cfg: ConfigTLS{
				CertFile: "c.pem", KeyFile: "k.pem", CAFile: "ca.pem", ClientAuth: ClientAuthRequireAndVerify,
			},
		},
		{
			name:    "key without certificate",
			cfg:     ConfigTLS{KeyFile: "k.pem", ClientAuth: ClientAuthNone},
			wantErr: "tls.cert_file: must be set together with tls.key_file",
		},
		{
			name:    "certificate without key",
			cfg:     ConfigTLS{CertFile: "c.pem", ClientAuth: ClientAuthNone},
			wantErr: "tls.key_file: must be set together with tls.cert_file",
		},
		{
			name:    "unknown client auth",
			cfg:     ConfigTLS{ClientAuth: "optional"},
			wantErr: `tls.client_auth: unknown value "optional" (want none, verify_if_given, require_and_verify)`,
		},
		{
			name:    "CA without TLS",
			cfg:     ConfigTLS{CAFile: "ca.pem", ClientAuth: ClientAuthNone},
			wantErr: "tls.ca_file: requires tls.cert_file and tls.key_file",
		},
		{
			name:    "client auth without TLS",
			cfg:     ConfigTLS{ClientAuth: ClientAuthVerifyIfGiven},
			wantErr: `tls.client_auth: "verify_if_given" requires tls.cert_file and tls.key_file`,
		},
		{
			name: "client auth without CA",
			cfg: ConfigTLS{
				CertFile: "c.pem", KeyFile: "k.pem", ClientAuth: ClientAuthRequireAndVerify,
			},
			wantErr: `tls.ca_file: must be set when tls.client_auth is "require_and_verify"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("Validate() = %q, want %q", err.Error(), tt.wantErr)
			}
			var de domainError
			if !errors.As(err, &de) || de.Type() != ErrorInvalid {
				t.Fatalf("Validate() type = %v, want ErrorInvalid", err.Type())
			}
		})
	}
}
