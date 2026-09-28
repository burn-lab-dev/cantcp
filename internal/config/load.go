package config

import (
	"errors"
	"flag"
	"io/fs"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
)

// LoadServer merges the configuration sources of the daemon: the built-in
// defaults, the JSON file (the path from --config or the default path), the
// CANTCP_* environment variables and the flags explicitly set on flags. An
// explicitly specified file must exist; the default path may be absent.
// The result is validated: the caller receives either a complete valid
// configuration or an error.
func LoadServer(flags *flag.FlagSet, f *ServerFlags, getenv getenvFunc, read readFileFunc) (domain.ConfigServer, error) {
	cfg := domain.DefaultConfigServer()
	path := f.Config
	explicit := path != ""
	if !explicit {
		path = DefaultServerConfigPath
	}
	file, err := loadFileServer(path, read)
	switch {
	case err == nil:
		if err := applyFileServer(file, &cfg); err != nil {
			return domain.ConfigServer{}, err
		}
	case errors.Is(err, fs.ErrNotExist) && !explicit:
		// The default file is optional.
	default:
		return domain.ConfigServer{}, err
	}
	if err := applyServerEnv(getenv, &cfg); err != nil {
		return domain.ConfigServer{}, err
	}
	applyServerFlags(flags, f, &cfg)
	if err := cfg.Validate(); err != nil {
		return domain.ConfigServer{}, err
	}
	return cfg, nil
}

// LoadClient merges the configuration sources of the client: the built-in
// defaults, the JSON file (the path from --config or the default path), the
// CANTCP_* environment variables and the flags explicitly set on flags.
func LoadClient(flags *flag.FlagSet, f *ClientFlags, getenv getenvFunc, read readFileFunc) (domain.ConfigClient, error) {
	cfg := domain.DefaultConfigClient()
	path := f.Config
	explicit := path != ""
	if !explicit {
		path = DefaultClientConfigPath
	}
	file, err := loadFileClient(path, read)
	switch {
	case err == nil:
		if err := applyFileClient(file, &cfg); err != nil {
			return domain.ConfigClient{}, err
		}
	case errors.Is(err, fs.ErrNotExist) && !explicit:
		// The default file is optional.
	default:
		return domain.ConfigClient{}, err
	}
	if err := applyClientEnv(getenv, &cfg); err != nil {
		return domain.ConfigClient{}, err
	}
	applyClientFlags(flags, f, &cfg)
	if err := cfg.Validate(); err != nil {
		return domain.ConfigClient{}, err
	}
	return cfg, nil
}
