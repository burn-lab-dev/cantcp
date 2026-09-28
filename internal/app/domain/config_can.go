package domain

// ConfigCAN selects the CAN interface served by the daemon. One daemon
// instance serves exactly one interface; run one instance per bus.
type ConfigCAN struct {
	Interface   string
	ErrorFrames bool
}

// maxIfaceLen is IFNAMSIZ-1: Linux network interface names are at most 15
// characters long.
const maxIfaceLen = 15

// Validate checks the interface name syntax. Whether the interface exists and
// is a CAN interface is checked by the socketcan adapter at startup.
func (c ConfigCAN) Validate() Error {
	if c.Interface == "" {
		return ErrInvalid("can.interface: must not be empty")
	}
	if len(c.Interface) > maxIfaceLen {
		return ErrInvalid("can.interface: %q is longer than %d characters", c.Interface, maxIfaceLen)
	}
	for _, r := range c.Interface {
		if r <= ' ' || r == '/' || r == ':' {
			return ErrInvalid("can.interface: %q contains an invalid character %q", c.Interface, r)
		}
	}
	return nil
}
