package domain

import "time"

// ConfigServer is the validated daemon configuration: the result of merging
// the defaults, the JSON file, the CANTCP_* environment variables and the
// command line flags.
type ConfigServer struct {
	Listen     string
	AllowPlain bool
	CAN        ConfigCAN
	TLS        ConfigTLS
	Limits     ConfigLimits
	Stats      ConfigStats
	Log        ConfigLog
}

// DefaultConfigServer returns the built-in defaults: a plain daemon on the
// loopback interface, the can0 interface, statistics on the loopback.
func DefaultConfigServer() ConfigServer {
	return ConfigServer{
		Listen: "127.0.0.1:29536",
		CAN:    ConfigCAN{Interface: "can0"},
		TLS:    ConfigTLS{ClientAuth: ClientAuthNone},
		Limits: ConfigLimits{
			MaxConnections: 16,
			ClientQueue:    1024,
			ReadTimeout:    5 * time.Second,
			WriteTimeout:   5 * time.Second,
		},
		Stats: ConfigStats{Enable: true, Listen: "127.0.0.1:29537"},
		Log:   ConfigLog{Level: LogLevelInfo, Format: LogFormatText},
	}
}

// Validate checks every setting. Plain mode on a non-loopback address is
// rejected unless AllowPlain is set: exposing unencrypted CAN traffic to the
// network must be an explicit decision.
func (c ConfigServer) Validate() Error {
	host, err := checkHostPort("listen", c.Listen)
	if err != nil {
		return err
	}
	if err := c.TLS.Validate(); err != nil {
		return err
	}
	if err := c.CAN.Validate(); err != nil {
		return err
	}
	if err := c.Limits.Validate(); err != nil {
		return err
	}
	if err := c.Stats.Validate(); err != nil {
		return err
	}
	if err := c.Log.Validate(); err != nil {
		return err
	}
	if !c.TLS.Enabled() && !c.AllowPlain && !hostIsLoopback(host) {
		return ErrInvalid("listen: plain mode on %q is not loopback; set allow_plain or configure TLS", c.Listen)
	}
	return nil
}
