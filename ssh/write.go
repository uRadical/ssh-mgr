package ssh

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// WriteConfig serialises hosts and atomically replaces ~/.ssh/config.
//
// The file is regenerated from the in-memory model. An exclusive flock is held
// on the original for the whole operation; the new content is written to a temp
// file in the same directory and renamed over the original so a crash mid-write
// can never leave a truncated config.
func WriteConfig(hosts []Host) error {
	path := ConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	// Open (creating if needed) the original purely to hold the lock.
	lockf, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = lockf.Close() }()

	if err := lockFile(lockf); err != nil {
		return err
	}
	defer func() { _ = unlockFile(lockf) }()

	tmp, err := os.CreateTemp(dir, ".ssh-mgr-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Clean up the temp file if anything below fails before the rename.
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.WriteString(serialize(hosts)); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	committed = true
	return nil
}

// serialize renders the full ~/.ssh/config text for hosts. Disabled hosts are
// emitted as a commented block preceded by the disabled sentinel.
func serialize(hosts []Host) string {
	var b strings.Builder
	for i, h := range hosts {
		if i > 0 {
			b.WriteByte('\n')
		}
		writeHost(&b, h)
	}
	return b.String()
}

func writeHost(b *strings.Builder, h Host) {
	lines := hostLines(h)
	if h.Disabled {
		b.WriteString(disabledSentinel)
		b.WriteByte('\n')
		for _, l := range lines {
			b.WriteString("# ")
			b.WriteString(l)
			b.WriteByte('\n')
		}
		return
	}
	for _, l := range lines {
		b.WriteString(l)
		b.WriteByte('\n')
	}
}

// hostLines returns the uncommented config lines for a single host block. Port
// is always written explicitly; empty optional fields are omitted.
func hostLines(h Host) []string {
	port := h.Port
	if port == 0 {
		port = 22
	}
	lines := []string{"Host " + h.Alias}
	if h.Hostname != "" {
		lines = append(lines, "    HostName "+h.Hostname)
	}
	if h.User != "" {
		lines = append(lines, "    User "+h.User)
	}
	lines = append(lines, "    Port "+strconv.Itoa(port))
	if h.IdentityFile != "" {
		lines = append(lines, "    IdentityFile "+quoteIfSpaces(h.IdentityFile))
	}
	for _, e := range h.EnvVars {
		if e.Name == "" {
			continue
		}
		lines = append(lines, "    SetEnv "+e.Name+"="+e.Value)
	}
	return lines
}

// quoteIfSpaces wraps v in double quotes when it contains whitespace. ssh
// requires paths with spaces to be quoted in the config; values without spaces
// are written bare so the common case stays unquoted.
func quoteIfSpaces(v string) string {
	if strings.ContainsAny(v, " \t") {
		return "\"" + v + "\""
	}
	return v
}
