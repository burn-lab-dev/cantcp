package domain

import (
	"errors"
	"testing"
)

func TestConfigTLSClient_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ConfigTLSClient
		wantErr string
	}{
		{
			name: "plain",
			cfg:  ConfigTLSClient{},
		},
		{
			name: "TLS with CA",
			cfg:  ConfigTLSClient{Enable: true, CAFile: "ca.pem"},
		},
		{
			name: "mTLS",
			cfg:  ConfigTLSClient{Enable: true, CertFile: "c.pem", KeyFile: "k.pem", ServerName: "can.example"},
		},
		{
			name: "insecure TLS",
			cfg:  ConfigTLSClient{Enable: true, Insecure: true},
		},
		{
			name:    "key without certificate",
			cfg:     ConfigTLSClient{Enable: true, KeyFile: "k.pem"},
			wantErr: "tls.cert_file: must be set together with tls.key_file",
		},
		{
			name:    "certificate without key",
			cfg:     ConfigTLSClient{Enable: true, CertFile: "c.pem"},
			wantErr: "tls.key_file: must be set together with tls.cert_file",
		},
		{
			name:    "CA without enable",
			cfg:     ConfigTLSClient{CAFile: "ca.pem"},
			wantErr: "tls: client TLS settings require tls.enable",
		},
		{
			name:    "server name without enable",
			cfg:     ConfigTLSClient{ServerName: "can.example"},
			wantErr: "tls: client TLS settings require tls.enable",
		},
		{
			name:    "insecure without enable",
			cfg:     ConfigTLSClient{Insecure: true},
			wantErr: "tls: client TLS settings require tls.enable",
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
