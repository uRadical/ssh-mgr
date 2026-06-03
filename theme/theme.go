// Package theme loads colour themes from TOML and constructs lipgloss styles.
package theme

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

//go:embed themes/dark.toml
var defaultThemeData []byte

// Theme is the set of named colours a theme defines. Every colour is a hex
// string (e.g. "#58a6ff").
type Theme struct {
	Primary   string `toml:"primary"`
	Secondary string `toml:"secondary"`
	Success   string `toml:"success"`
	Warning   string `toml:"warning"`
	Error     string `toml:"error"`
	Muted     string `toml:"muted"`
	Subtle    string `toml:"subtle"`
	Border    string `toml:"border"`
	BgPanel   string `toml:"bg_panel"`
}

// Default returns the bundled dark theme.
func Default() Theme {
	var t Theme
	// The embedded asset is known-good; ignore the error.
	_ = toml.Unmarshal(defaultThemeData, &t)
	return t
}

// Load reads the theme named name from ~/.config/ssh-mgr/themes/<name>.toml.
// If the file is missing or malformed it silently falls back to the bundled
// default — loading a theme never fails.
func Load(name string) Theme {
	home, err := os.UserHomeDir()
	if err != nil {
		return Default()
	}
	path := filepath.Join(home, ".config", "ssh-mgr", "themes", name+".toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return Default()
	}
	var t Theme
	if err := toml.Unmarshal(data, &t); err != nil {
		return Default()
	}
	return t
}
