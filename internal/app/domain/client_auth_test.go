package domain

import "testing"

func TestClientAuth_valid(t *testing.T) {
	tests := []struct {
		name string
		mode ClientAuth
		want bool
	}{
		{name: "none", mode: ClientAuthNone, want: true},
		{name: "verify if given", mode: ClientAuthVerifyIfGiven, want: true},
		{name: "require and verify", mode: ClientAuthRequireAndVerify, want: true},
		{name: "unknown", mode: "optional", want: false},
		{name: "empty", mode: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.valid(); got != tt.want {
				t.Fatalf("valid(%q) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}
