package domain

// ConfigClient is the validated client configuration.
type ConfigClient struct {
	Server      string
	StatsServer string
	TLS         ConfigTLSClient
	Log         ConfigLog
}

// DefaultConfigClient returns the built-in defaults: a plain connection to
// the loopback daemon and its statistics endpoint.
func DefaultConfigClient() ConfigClient {
	return ConfigClient{
		Server:      "127.0.0.1:29536",
		StatsServer: "http://127.0.0.1:29537",
		Log:         ConfigLog{Level: LogLevelInfo, Format: LogFormatText},
	}
}

// Validate checks every client setting.
func (c ConfigClient) Validate() Error {
	if _, err := checkHostPort("server", c.Server); err != nil {
		return err
	}
	if err := checkHTTPURL("stats_server", c.StatsServer); err != nil {
		return err
	}
	if err := c.TLS.Validate(); err != nil {
		return err
	}
	return c.Log.Validate()
}
