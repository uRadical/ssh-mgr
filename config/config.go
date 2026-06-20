// ssh-mgr — a terminal UI for managing ~/.ssh/config host entries.
// Copyright (C) 2026 uradical
//
// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for
// more details.
//
// You should have received a copy of the GNU General Public License along
// with this program. If not, see <https://www.gnu.org/licenses/>.

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
