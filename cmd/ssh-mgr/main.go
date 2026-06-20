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

// Command sshmgr is a terminal UI for managing ~/.ssh/config host entries.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"uradical.io/go/sshmgr/config"
	"uradical.io/go/sshmgr/model"
	"uradical.io/go/sshmgr/ssh"
	"uradical.io/go/sshmgr/theme"
)

func main() {
	// A malformed application config is fatal before the TUI starts.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh-mgr: invalid config %s: %v\n", config.Path(), err)
		os.Exit(1)
	}

	styles := theme.NewStyles(theme.Load(cfg.Theme))

	// A malformed ~/.ssh/config is surfaced inside the TUI (full-screen error)
	// rather than aborting; the file is never touched.
	hosts, parseErr := ssh.LoadConfig()

	m := model.New(hosts, parseErr, styles)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "ssh-mgr:", err)
		os.Exit(1)
	}
}
