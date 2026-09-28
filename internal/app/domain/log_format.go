package domain

// LogFormat selects the logger encoding.
type LogFormat string

// Supported log formats.
const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

// valid reports whether the format is one of the supported values.
func (f LogFormat) valid() bool {
	switch f {
	case LogFormatText, LogFormatJSON:
		return true
	default:
		return false
	}
}
