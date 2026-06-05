package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"uradical.io/go/sshmgr/ssh"
	"uradical.io/go/sshmgr/theme"
)

// KnownHostModal asks the user to approve an unknown host's keys before they
// are written to ~/.ssh/known_hosts. It mirrors the prompt ssh itself shows,
// surfacing the fingerprints for out-of-band verification.
type KnownHostModal struct {
	alias        string
	host         string
	fingerprints []string
	styles       theme.Styles
}

// NewKnownHostModal builds the modal for h with the given display fingerprints.
func NewKnownHostModal(h ssh.Host, fingerprints []string, s theme.Styles) KnownHostModal {
	host := h.Hostname
	if h.Port != 0 && h.Port != 22 {
		host = fmt.Sprintf("%s:%d", h.Hostname, h.Port)
	}
	return KnownHostModal{alias: h.Alias, host: host, fingerprints: fingerprints, styles: s}
}

func (m KnownHostModal) View() string {
	s := m.styles
	var b strings.Builder

	b.WriteString(s.ModalTitle.Render("Unknown host key"))
	b.WriteString("\n\n")
	// Keep newlines outside the styled renders: a trailing "\n" inside a styled
	// string pads the blank line and shifts the following text rightward.
	b.WriteString(s.Label.Render("The authenticity of ") + s.Value.Render(m.alias) +
		s.Label.Render(" ("+m.host+") can't be established."))
	b.WriteString("\n\n")

	b.WriteString(s.Label.Render("Key fingerprints:"))
	b.WriteByte('\n')
	if len(m.fingerprints) == 0 {
		b.WriteString("  " + s.Label.Render("(unavailable)"))
		b.WriteByte('\n')
	}
	for _, fp := range m.fingerprints {
		b.WriteString("  " + s.Value.Render(fp))
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(s.Label.Render("Add to ~/.ssh/known_hosts?"))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.JoinHorizontal(
		lipgloss.Top,
		s.Button.Render(s.ButtonKey.Render("y")+" add & continue"),
		"  ",
		s.Button.Render(s.ButtonKey.Render("n")+" cancel"),
	))

	return s.Modal.Render(b.String())
}
