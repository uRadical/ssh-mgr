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

package ssh

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// HostKeyAction is what the caller intends to do once a host's key is trusted.
// It is carried through the asynchronous check so the result handler knows
// whether to resume a test or an interactive connection.
type HostKeyAction int

const (
	HostKeyTest HostKeyAction = iota
	HostKeyConnect
)

// HostKeyMsg reports the outcome of a pre-flight known-host check.
//
//   - Known:        the host is already in known_hosts; proceed directly.
//   - Err != "":    the keys could not be fetched (host down, no ssh-keyscan).
//   - otherwise:    Lines/Fingerprints describe an unknown host awaiting the
//     user's approval before being written to known_hosts.
type HostKeyMsg struct {
	Alias        string
	Action       HostKeyAction
	Known        bool
	Lines        string
	Fingerprints []string
	Err          string
}

// KnownHostsPath returns the path to ~/.ssh/known_hosts.
func KnownHostsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "known_hosts")
}

// knownHostTarget formats host as ssh records it in known_hosts: a bare
// hostname on the default port, or [host]:port otherwise.
func knownHostTarget(host string, port int) string {
	if port == 0 {
		port = 22
	}
	if port == 22 {
		return host
	}
	return "[" + host + "]:" + strconv.Itoa(port)
}

// CheckHostKey returns a command that reports whether h is already trusted and,
// if not, scans its host keys so the user can approve them. It is meant for
// directly reachable hosts; callers should skip it for ProxyJump hosts (which
// ssh-keyscan cannot reach) and hosts with no HostName.
func CheckHostKey(h Host, action HostKeyAction) tea.Cmd {
	return func() tea.Msg {
		if IsHostKnown(h) {
			return HostKeyMsg{Alias: h.Alias, Action: action, Known: true}
		}
		lines, fingerprints, err := ScanHostKey(h)
		if err != nil {
			return HostKeyMsg{Alias: h.Alias, Action: action, Err: err.Error()}
		}
		return HostKeyMsg{Alias: h.Alias, Action: action, Lines: lines, Fingerprints: fingerprints}
	}
}

// IsHostKnown reports whether known_hosts already holds a key for h. ssh-keygen
// exits 0 when it finds a matching line.
func IsHostKnown(h Host) bool {
	if h.Hostname == "" {
		return false
	}
	return exec.Command("ssh-keygen", "-F", knownHostTarget(h.Hostname, h.Port)).Run() == nil
}

// ScanHostKey fetches h's public host keys with ssh-keyscan and returns the raw
// known_hosts lines to persist plus human-readable fingerprints for display.
func ScanHostKey(h Host) (lines string, fingerprints []string, err error) {
	port := h.Port
	if port == 0 {
		port = 22
	}

	var out, stderr bytes.Buffer
	scan := exec.Command("ssh-keyscan", "-T", "5", "-p", strconv.Itoa(port), "-t", "ed25519,ecdsa,rsa", h.Hostname)
	scan.Stdout = &out
	scan.Stderr = &stderr
	if err := scan.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", nil, errors.New(msg)
	}

	lines = keyLines(out.String())
	if lines == "" {
		return "", nil, fmt.Errorf("no host keys returned by %s", h.Hostname)
	}
	return lines, fingerprintsOf(lines), nil
}

// keyLines keeps only the key entries from ssh-keyscan output, dropping the
// informational comment lines it may emit.
func keyLines(out string) string {
	var kept []string
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// fingerprintsOf computes display fingerprints for known_hosts lines by piping
// them through ssh-keygen -l. It returns nil if ssh-keygen is unavailable.
func fingerprintsOf(knownHostsLines string) []string {
	cmd := exec.Command("ssh-keygen", "-l", "-f", "-")
	cmd.Stdin = strings.NewReader(knownHostsLines + "\n")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return parseFingerprints(string(out))
}

// parseFingerprints turns `ssh-keygen -l` output, e.g.
//
//	256 SHA256:abc… host (ED25519)
//
// into display strings like "ED25519  SHA256:abc…".
func parseFingerprints(out string) []string {
	var fps []string
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 4 {
			continue
		}
		hash := fields[1]
		keyType := strings.Trim(fields[len(fields)-1], "()")
		fps = append(fps, keyType+"  "+hash)
	}
	return fps
}

// AddKnownHost appends keyscan lines to ~/.ssh/known_hosts under an exclusive
// lock, creating the file (and ~/.ssh) if needed and ensuring a trailing
// newline so the next entry stays on its own line.
func AddKnownHost(lines string) error {
	path := KnownHostsPath()
	if path == "" {
		return errors.New("cannot locate home directory")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := lockFile(f); err != nil {
		return err
	}
	defer func() { _ = unlockFile(f) }()

	_, err = f.WriteString(strings.TrimRight(lines, "\n") + "\n")
	return err
}
