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

package ui

import (
	"fmt"
	"strings"

	"uradical.io/go/sshmgr/ssh"
	"uradical.io/go/sshmgr/theme"
)

// RenderDetail renders the full detail of a single host.
func RenderDetail(h ssh.Host, s theme.Styles) string {
	var b strings.Builder

	row := func(label, value string) {
		if value == "" {
			value = s.Label.Render("—")
		} else {
			value = s.Value.Render(value)
		}
		b.WriteString(s.Label.Render(fmt.Sprintf("%-12s", label)))
		b.WriteString(value)
		b.WriteByte('\n')
	}

	// A disabled host is conveyed purely by greying its heading — no label text.
	if h.Disabled {
		b.WriteString(s.Label.Bold(true).Padding(0, 1).Render(h.Alias))
	} else {
		b.WriteString(s.Title.Render(h.Alias))
	}
	b.WriteString("\n\n")

	port := h.Port
	if port == 0 {
		port = 22
	}
	row("HostName", h.Hostname)
	row("User", h.User)
	row("Port", fmt.Sprintf("%d", port))
	row("IdentityFile", h.IdentityFile)
	row("ProxyJump", h.ProxyJump)

	b.WriteByte('\n')
	b.WriteString(s.Label.Render("SetEnv"))
	b.WriteByte('\n')
	if len(h.EnvVars) == 0 {
		b.WriteString("  " + s.Label.Render("—"))
		b.WriteByte('\n')
	}
	for _, e := range h.EnvVars {
		b.WriteString("  " + s.Value.Render(e.Name+"="+e.Value))
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(s.Label.Render(fmt.Sprintf("%-12s", "Connect")))
	b.WriteString(connectDetail(h, s))
	b.WriteByte('\n')

	return strings.TrimRight(b.String(), "\n")
}

func connectDetail(h ssh.Host, s theme.Styles) string {
	switch h.ConnectStatus {
	case ssh.ConnectTesting:
		return s.StatusTesting.Render("testing…")
	case ssh.ConnectOK:
		return s.StatusOK.Render(fmt.Sprintf("ok (%dms)", h.ConnectMs))
	case ssh.ConnectFailed:
		msg := h.ConnectErr
		if msg == "" {
			msg = "failed"
		}
		return s.StatusFailed.Render("failed: " + msg)
	default:
		return s.StatusUnknown.Render("unknown")
	}
}
