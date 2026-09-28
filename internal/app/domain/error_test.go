package domain

import (
	"errors"
	"testing"
)

// Compile-time check: domainError implements Error.
var _ Error = domainError{}

func TestError_TypeAndText(t *testing.T) {
	tests := []struct {
		name     string
		err      Error
		wantType TypeError
		want     string
	}{
		{
			name:     "invalid formats the message",
			err:      ErrInvalid("field: bad value %q", "x"),
			wantType: ErrorInvalid,
			want:     `field: bad value "x"`,
		},
		{
			name:     "invalid without arguments",
			err:      ErrInvalid("plain message"),
			wantType: ErrorInvalid,
			want:     "plain message",
		},
		{
			name:     "app carries its own type",
			err:      ErrApp("boom %d", 7),
			wantType: ErrorApp,
			want:     "boom 7",
		},
		{
			name:     "early carries its own type",
			err:      ErrEarly("dependency is not ready"),
			wantType: ErrorEarly,
			want:     "dependency is not ready",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Type(); got != tt.wantType {
				t.Fatalf("Type() = %v, want %v", got, tt.wantType)
			}
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
			var de domainError
			if !errors.As(tt.err, &de) {
				t.Fatalf("errors.As(%T, &domainError) = false, want true", tt.err)
			}
			if de.Type() != tt.wantType {
				t.Fatalf("errors.As value Type() = %v, want %v", de.Type(), tt.wantType)
			}
		})
	}
}
