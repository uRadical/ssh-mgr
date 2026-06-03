// Package ui holds the TUI components: the host list and detail renderers, the
// edit/add form, the identity-file picker and the connect modal.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"uradical.io/go/sshmgr/ssh"
	"uradical.io/go/sshmgr/theme"
)

// statusBadge returns a coloured reachability dot for a host. The colour alone
// conveys state; the detail panel carries the full text.
func statusBadge(h ssh.Host, s theme.Styles) string {
	switch h.ConnectStatus {
	case ssh.ConnectTesting:
		return s.StatusTesting.Render("●")
	case ssh.ConnectOK:
		return s.StatusOK.Render("●")
	case ssh.ConnectFailed:
		return s.StatusFailed.Render("●")
	default:
		return s.StatusUnknown.Render("○")
	}
}

// RenderHostList renders the list of hosts with the cursor highlighted.
func RenderHostList(hosts []ssh.Host, cursor int, s theme.Styles, width, height int) string {
	if len(hosts) == 0 {
		return s.Item.Render(s.Label.Render("no hosts — press 'a' to add one"))
	}

	var b strings.Builder
	for i, h := range hosts {
		alias := h.Alias

		line := lipgloss.JoinHorizontal(
			lipgloss.Left,
			fmt.Sprintf("%-22s", truncate(alias, 22)),
			" ",
			statusBadge(h, s),
		)

		switch {
		case i == cursor:
			b.WriteString(s.ItemSelected.Render("▌ " + line))
		case h.Disabled:
			b.WriteString(s.ItemDisabled.Render("  " + line))
		default:
			b.WriteString(s.Item.Render("  " + line))
		}
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func truncate(s string, n int) string {
	if lipgloss.Width(s) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string([]rune(s)[:n-1]) + "…"
}
