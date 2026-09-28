package config

import (
	"fmt"
	"strconv"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// getenvFunc looks up an environment variable; it is injected for tests.
type getenvFunc func(string) (string, bool)

// applyServerEnv merges the CANTCP_* variables into cfg.
func applyServerEnv(getenv getenvFunc, cfg *domain.ConfigServer) error {
	envString(getenv, "CANTCP_LISTEN", &cfg.Listen)
	envString(getenv, "CANTCP_CAN", &cfg.CAN.Interface)
	if err := envBool(getenv, "CANTCP_ERROR_FRAMES", &cfg.CAN.ErrorFrames); err != nil {
		return err
	}
	if err := envBool(getenv, "CANTCP_ALLOW_PLAIN", &cfg.AllowPlain); err != nil {
		return err
	}
	envString(getenv, "CANTCP_TLS_CERT", &cfg.TLS.CertFile)
	envString(getenv, "CANTCP_TLS_KEY", &cfg.TLS.KeyFile)
	envString(getenv, "CANTCP_TLS_CA", &cfg.TLS.CAFile)
	if v, ok := getenv("CANTCP_TLS_CLIENT_AUTH"); ok {
		cfg.TLS.ClientAuth = domain.ClientAuth(v)
	}
	if err := envInt(getenv, "CANTCP_MAX_CONNECTIONS", &cfg.Limits.MaxConnections); err != nil {
		return err
	}
	if err := envInt(getenv, "CANTCP_CLIENT_QUEUE", &cfg.Limits.ClientQueue); err != nil {
		return err
	}
	if err := envDuration(getenv, "CANTCP_READ_TIMEOUT", &cfg.Limits.ReadTimeout); err != nil {
		return err
	}
	if err := envDuration(getenv, "CANTCP_WRITE_TIMEOUT", &cfg.Limits.WriteTimeout); err != nil {
		return err
	}
	if err := envDuration(getenv, "CANTCP_IDLE_TIMEOUT", &cfg.Limits.IdleTimeout); err != nil {
		return err
	}
	if err := envInt(getenv, "CANTCP_MAX_FRAMES_PER_SECOND", &cfg.Limits.MaxFramesPerSecond); err != nil {
		return err
	}
	envString(getenv, "CANTCP_STATS_LISTEN", &cfg.Stats.Listen)
	if err := envBool(getenv, "CANTCP_STATS_ENABLE", &cfg.Stats.Enable); err != nil {
		return err
	}
	if v, ok := getenv("CANTCP_LOG_LEVEL"); ok {
		cfg.Log.Level = domain.LogLevel(v)
	}
	if v, ok := getenv("CANTCP_LOG_FORMAT"); ok {
		cfg.Log.Format = domain.LogFormat(v)
	}
	return nil
}

// applyClientEnv merges the CANTCP_* variables into cfg.
func applyClientEnv(getenv getenvFunc, cfg *domain.ConfigClient) error {
	envString(getenv, "CANTCP_SERVER", &cfg.Server)
	envString(getenv, "CANTCP_STATS_SERVER", &cfg.StatsServer)
	if err := envBool(getenv, "CANTCP_TLS_ENABLE", &cfg.TLS.Enable); err != nil {
		return err
	}
	envString(getenv, "CANTCP_TLS_CA", &cfg.TLS.CAFile)
	envString(getenv, "CANTCP_TLS_CERT", &cfg.TLS.CertFile)
	envString(getenv, "CANTCP_TLS_KEY", &cfg.TLS.KeyFile)
	envString(getenv, "CANTCP_TLS_SERVER_NAME", &cfg.TLS.ServerName)
	if err := envBool(getenv, "CANTCP_TLS_INSECURE", &cfg.TLS.Insecure); err != nil {
		return err
	}
	if v, ok := getenv("CANTCP_LOG_LEVEL"); ok {
		cfg.Log.Level = domain.LogLevel(v)
	}
	if v, ok := getenv("CANTCP_LOG_FORMAT"); ok {
		cfg.Log.Format = domain.LogFormat(v)
	}
	return nil
}

// envString stores the variable value into dst when the variable is set.
func envString(getenv getenvFunc, name string, dst *string) {
	if v, ok := getenv(name); ok {
		*dst = v
	}
}

// envBool parses a boolean variable.
func envBool(getenv getenvFunc, name string, dst *bool) error {
	v, ok := getenv(name)
	if !ok {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fmt.Errorf("%s: invalid boolean %q", name, v)
	}
	*dst = b
	return nil
}

// envInt parses an integer variable.
func envInt(getenv getenvFunc, name string, dst *int) error {
	v, ok := getenv(name)
	if !ok {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("%s: invalid integer %q", name, v)
	}
	*dst = n
	return nil
}

// envDuration parses a Go duration variable.
func envDuration(getenv getenvFunc, name string, dst *time.Duration) error {
	v, ok := getenv(name)
	if !ok {
		return nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fmt.Errorf("%s: invalid duration %q", name, v)
	}
	*dst = d
	return nil
}
