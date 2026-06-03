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
