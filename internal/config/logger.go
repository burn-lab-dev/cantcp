package config

import (
	"io"
	"log/slog"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// NewLogger builds a logger writing to w and the level variable that the
// SIGHUP reload updates without recreating the handler. The format is fixed
// for the lifetime of the process.
func NewLogger(cfg domain.ConfigLog, w io.Writer) (*slog.Logger, *slog.LevelVar) {
	level := new(slog.LevelVar)
	level.Set(SlogLevel(cfg.Level))
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if cfg.Format == domain.LogFormatJSON {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}
	return slog.New(handler), level
}

// SlogLevel maps a domain level to its slog counterpart. Unknown levels map
// to info, the same fallback the validation uses.
func SlogLevel(l domain.LogLevel) slog.Level {
	switch l {
	case domain.LogLevelDebug:
		return slog.LevelDebug
	case domain.LogLevelWarn:
		return slog.LevelWarn
	case domain.LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
