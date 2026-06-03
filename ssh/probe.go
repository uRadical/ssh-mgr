package ssh

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ConnectResultMsg is delivered when a background probe finishes.
type ConnectResultMsg struct {
	HostAlias string
	OK        bool
	Err       string
	ElapsedMs int64
}

// Probe returns a tea.Cmd that tests reachability of h in the background by
// running a non-interactive ssh that connects and immediately exits. It never
// blocks the UI; completion is reported via ConnectResultMsg.
func Probe(h Host) tea.Cmd {
	return func() tea.Msg {
		port := h.Port
		if port == 0 {
			port = 22
		}

		args := []string{
			"-o", "BatchMode=yes",
			"-o", "ConnectTimeout=5",
			"-o", "StrictHostKeyChecking=accept-new",
		}
		if h.IdentityFile != "" {
			args = append(args, "-i", ExpandPath(h.IdentityFile))
		}
		args = append(args, "-p", strconv.Itoa(port))

		target := h.Hostname
		if h.User != "" {
			target = h.User + "@" + h.Hostname
		}
		args = append(args, target, "exit")

		start := time.Now()
		out, err := exec.Command("ssh", args...).CombinedOutput()
		elapsed := time.Since(start).Milliseconds()

		if err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = err.Error()
			}
			return ConnectResultMsg{HostAlias: h.Alias, OK: false, Err: msg, ElapsedMs: elapsed}
		}
		return ConnectResultMsg{HostAlias: h.Alias, OK: true, ElapsedMs: elapsed}
	}
}

// ExpandPath expands a leading ~ to the user's home directory. ssh would do
// this for shell-expanded arguments, but exec.Command bypasses the shell.
func ExpandPath(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
		}
	}
	return p
}
