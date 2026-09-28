package domain

// ConfigLog selects the logger output. The format is fixed for the lifetime
// of the process: the SIGHUP reload applies the level only.
type ConfigLog struct {
	Level  LogLevel
	Format LogFormat
}

// Validate checks the level and the format.
func (c ConfigLog) Validate() Error {
	if !c.Level.valid() {
		return ErrInvalid("log.level: unknown value %q (want debug, info, warn, error)", c.Level)
	}
	if !c.Format.valid() {
		return ErrInvalid("log.format: unknown value %q (want text, json)", c.Format)
	}
	return nil
}
