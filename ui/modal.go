package ui

import (
	"os/exec"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"uradical.io/go/sshmgr/ssh"
	"uradical.io/go/sshmgr/theme"
)

// SessionEndedMsg is sent when an interactive ssh session started via
// ConnectCmd exits and the TUI resumes.
type SessionEndedMsg struct{}

// ConnectModal is the minimal popup shown while an interactive ssh session is
// active. The session itself runs via tea.ExecProcess, which suspends the TUI.
type ConnectModal struct {
	host       ssh.Host
	styles     theme.Styles
	Fullscreen bool
}

// NewConnectModal builds a modal for h.
func NewConnectModal(h ssh.Host, s theme.Styles) ConnectModal {
	return ConnectModal{host: h, styles: s}
}

// ConnectCmd opens an interactive ssh session to h. tea.ExecProcess hands the
// terminal to ssh and, on exit, delivers SessionEndedMsg.
func ConnectCmd(h ssh.Host) tea.Cmd {
	port := h.Port
	if port == 0 {
		port = 22
	}
	args := []string{}
	if h.IdentityFile != "" {
		args = append(args, "-i", ssh.ExpandPath(h.IdentityFile))
	}
	args = append(args, "-p", strconv.Itoa(port))
	target := h.Hostname
	if h.User != "" {
		target = h.User + "@" + h.Hostname
	}
	args = append(args, target)

	c := exec.Command("ssh", args...)
	return tea.ExecProcess(c, func(error) tea.Msg {
		return SessionEndedMsg{}
	})
}

func (m ConnectModal) View() string {
	s := m.styles
	target := m.host.Hostname
	if m.host.User != "" {
		target = m.host.User + "@" + m.host.Hostname
	}

	title := s.ModalTitle.Render(m.host.Alias) + s.Label.Render("  "+target)

	fsLabel := "fullscreen"
	if m.Fullscreen {
		fsLabel = "windowed"
	}
	buttons := lipgloss.JoinHorizontal(
		lipgloss.Top,
		s.Button.Render(s.ButtonKey.Render("f")+" "+fsLabel),
		"  ",
		s.Button.Render(s.ButtonKey.Render("esc")+" exit"),
	)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		s.Label.Render("Connecting…"),
		"",
		buttons,
	)
	return s.Modal.Render(body)
}
