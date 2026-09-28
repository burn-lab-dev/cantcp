package domain

// ClientAuth selects how the TLS server treats client certificates.
type ClientAuth string

// Supported client authentication modes.
const (
	// ClientAuthNone accepts clients without a certificate.
	ClientAuthNone ClientAuth = "none"
	// ClientAuthVerifyIfGiven verifies a client certificate when one is
	// presented, but accepts clients without one.
	ClientAuthVerifyIfGiven ClientAuth = "verify_if_given"
	// ClientAuthRequireAndVerify requires a client certificate signed by the
	// configured CA.
	ClientAuthRequireAndVerify ClientAuth = "require_and_verify"
)

// valid reports whether the mode is one of the supported values.
func (a ClientAuth) valid() bool {
	switch a {
	case ClientAuthNone, ClientAuthVerifyIfGiven, ClientAuthRequireAndVerify:
		return true
	default:
		return false
	}
}
