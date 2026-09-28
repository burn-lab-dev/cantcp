package domain

// ConfigStats selects the HTTP statistics listener of the daemon.
type ConfigStats struct {
	Enable bool
	Listen string
}

// Validate checks the statistics address when the listener is enabled.
func (c ConfigStats) Validate() Error {
	if !c.Enable {
		return nil
	}
	if _, err := checkHostPort("stats.listen", c.Listen); err != nil {
		return err
	}
	return nil
}
