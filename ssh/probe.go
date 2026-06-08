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
//
// StrictHostKeyChecking=yes (rather than accept-new) means a probe never
// silently mutates known_hosts: directly reachable hosts have already had any
// unknown key approved via the in-TUI modal before Probe runs, and an unknown
// key on a host the modal can't reach (ProxyJump, no HostName) surfaces here as
// a clean failure instead.
func Probe(h Host) tea.Cmd {
	return func() tea.Msg {
		// Non-interactive options, then the shared connection args, then a
		// no-op remote command so a successful login exits immediately.
		args := append([]string{
			"-o", "BatchMode=yes",
			"-o", "ConnectTimeout=5",
			"-o", "StrictHostKeyChecking=yes",
		}, ConnectArgs(h)...)
		args = append(args, "exit")

		start := time.Now()
		out, err := exec.Command("ssh", args...).CombinedOutput()
		elapsed := time.Since(start).Milliseconds()

		if err != nil {
			msg := probeError(string(out), err)
			return ConnectResultMsg{HostAlias: h.Alias, OK: false, Err: msg, ElapsedMs: elapsed}
		}
		return ConnectResultMsg{HostAlias: h.Alias, OK: true, ElapsedMs: elapsed}
	}
}

// probeError turns ssh's combined output into a concise failure message. An
// unknown host key is reported with an actionable hint, since strict checking
// means the probe won't add it; the user must connect interactively (which can
// reach ProxyJump hosts that ssh-keyscan can't) to verify and trust the key.
func probeError(out string, err error) string {
	msg := strings.TrimSpace(out)
	if strings.Contains(msg, "Host key verification failed") {
		return "unknown host key — connect (s) to verify and trust it"
	}
	if msg == "" {
		return err.Error()
	}
	return msg
}

// ConnectArgs builds the ssh command-line arguments common to probing and
// interactive connection: identity file, jump host, port and target. Both
// callers reconstruct the connection from the in-memory model rather than
// relying on the alias, so every option that affects reachability — including
// ProxyJump — must be passed explicitly here.
func ConnectArgs(h Host) []string {
	port := h.Port
	if port == 0 {
		port = 22
	}

	var args []string
	if h.IdentityFile != "" {
		args = append(args, "-i", ExpandPath(h.IdentityFile))
	}
	if h.ProxyJump != "" {
		args = append(args, "-J", h.ProxyJump)
	}
	args = append(args, "-p", strconv.Itoa(port))

	target := h.Hostname
	if h.User != "" {
		target = h.User + "@" + h.Hostname
	}
	return append(args, target)
}

// ExpandPath expands a leading ~ to the user's home directory. ssh would do
// this for shell-expanded arguments, but exec.Command bypasses the shell.
// Any surrounding quotes are stripped first so a value that survived as a
// quoted literal still expands and resolves to a real file.
func ExpandPath(p string) string {
	p = unquote(p)
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
		}
	}
	return p
}
