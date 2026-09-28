package domain

import "testing"

func TestCheckHostPort(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		wantHost string
		wantErr  string
	}{
		{name: "host and port", addr: "127.0.0.1:29536", wantHost: "127.0.0.1"},
		{name: "empty host", addr: ":29536", wantHost: ""},
		{name: "IP v6", addr: "[::1]:29536", wantHost: "::1"},
		{name: "host name", addr: "can.example:29536", wantHost: "can.example"},
		{name: "max port", addr: "127.0.0.1:65535", wantHost: "127.0.0.1"},
		{
			name:    "no port",
			addr:    "127.0.0.1",
			wantErr: `listen: "127.0.0.1" is not a host:port address`,
		},
		{
			name:    "zero port",
			addr:    "127.0.0.1:0",
			wantErr: `listen: "127.0.0.1:0" has an invalid port (want 1..65535)`,
		},
		{
			name:    "port too large",
			addr:    "127.0.0.1:65536",
			wantErr: `listen: "127.0.0.1:65536" has an invalid port (want 1..65535)`,
		},
		{
			name:    "port is not a number",
			addr:    "127.0.0.1:http",
			wantErr: `listen: "127.0.0.1:http" has an invalid port (want 1..65535)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, err := checkHostPort("listen", tt.addr)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("checkHostPort() = nil, want %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("checkHostPort() = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("checkHostPort() = %v, want nil", err)
			}
			if host != tt.wantHost {
				t.Fatalf("host = %q, want %q", host, tt.wantHost)
			}
		})
	}
}

func TestCheckHTTPURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{name: "http", raw: "http://127.0.0.1:29537"},
		{name: "https with a path", raw: "https://can.example/stats"},
		{
			name:    "no scheme",
			raw:     "127.0.0.1:29537",
			wantErr: `stats_server: "127.0.0.1:29537" is not an http:// or https:// URL`,
		},
		{
			name:    "wrong scheme",
			raw:     "ftp://host/stats",
			wantErr: `stats_server: "ftp://host/stats" is not an http:// or https:// URL`,
		},
		{
			name:    "no host",
			raw:     "http:///stats",
			wantErr: `stats_server: "http:///stats" is not an http:// or https:// URL`,
		},
		{
			name:    "broken URL",
			raw:     "http://%41:8080/",
			wantErr: `stats_server: "http://%41:8080/" is not an http:// or https:// URL`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkHTTPURL("stats_server", tt.raw)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("checkHTTPURL() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("checkHTTPURL() = nil, want %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("checkHTTPURL() = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestHostIsLoopback(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		{name: "empty", host: "", want: false},
		{name: "localhost", host: "localhost", want: true},
		{name: "IPv4 loopback", host: "127.0.0.1", want: true},
		{name: "IPv4 loopback range", host: "127.5.5.5", want: true},
		{name: "IPv6 loopback", host: "::1", want: true},
		{name: "public IPv4", host: "192.168.1.10", want: false},
		{name: "public IPv6", host: "2001:db8::1", want: false},
		{name: "host name", host: "can.example", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostIsLoopback(tt.host); got != tt.want {
				t.Fatalf("hostIsLoopback(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}
