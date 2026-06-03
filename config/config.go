// Package config loads the ssh-mgr application config from
// ~/.config/ssh-mgr/config.toml.
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is the application configuration.
type Config struct {
	Theme string `toml:"theme"`
}

// Default returns the configuration used when no config file is present.
func Default() Config {
	return Config{Theme: "dark"}
}

// Path returns the location of the config file.
func Path() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "ssh-mgr", "config.toml")
}

// Load reads the config file. If the file is missing the defaults are returned
// with no error. If it exists but is malformed, the error is returned and the
// caller is expected to abort before starting the TUI.
func Load() (Config, error) {
	cfg := Default()
	path := Path()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
