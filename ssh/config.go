// Package ssh reads and writes ~/.ssh/config and probes host reachability.
//
// It owns the core Host data model so that both the UI and the root model can
// depend on it without an import cycle.
package ssh

import (
	"bufio"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kevinburke/ssh_config"
)

// ConnectStatus is the in-memory reachability state of a host. It is never
// persisted to ~/.ssh/config and resets to ConnectUnknown on restart.
type ConnectStatus int

const (
	ConnectUnknown ConnectStatus = iota
	ConnectTesting
	ConnectOK
	ConnectFailed
)

// EnvVar is a single SetEnv KEY=VALUE pair.
type EnvVar struct {
	Name  string
	Value string
}

// Host is one Host block from ~/.ssh/config.
type Host struct {
	Alias        string
	Hostname     string
	User         string
	Port         int
	IdentityFile string
	ProxyJump    string
	EnvVars      []EnvVar
	Disabled     bool

	ConnectStatus ConnectStatus
	ConnectErr    string
	ConnectMs     int64
}

// disabledSentinel marks the Host line of a commented-out (disabled) block so
// it can be recognised on read.
const disabledSentinel = "# [ssh-mgr:disabled]"

// ConfigPath returns the path to ~/.ssh/config.
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "config")
}

// LoadConfig parses ~/.ssh/config and returns its hosts in file order.
//
// Enabled hosts are read via ssh_config (per the spec); disabled hosts —
// which are commented out and therefore invisible to ssh_config — are parsed
// from the raw text using the disabled sentinel. A missing config file is not
// an error (it yields no hosts); a malformed one is.
func LoadConfig() ([]Host, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	// ssh_config validates the file (malformed -> error) and provides field
	// lookups for enabled hosts.
	cfg, err := ssh_config.DecodeBytes(data)
	if err != nil {
		return nil, err
	}

	return scanHosts(data, cfg), nil
}

// scanHosts walks the raw config text in order, building a Host for every
// enabled or disabled block it finds.
func scanHosts(data []byte, cfg *ssh_config.Config) []Host {
	var hosts []Host
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var (
		inDisabled bool
		raw        []string
	)
	flushDisabled := func() {
		if inDisabled {
			if h, ok := parseDisabledBlock(raw); ok {
				hosts = append(hosts, h)
			}
			inDisabled = false
			raw = nil
		}
	}

	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == disabledSentinel {
			flushDisabled()
			inDisabled = true
			continue
		}

		if inDisabled {
			if strings.HasPrefix(trimmed, "#") {
				raw = append(raw, uncomment(trimmed))
				continue
			}
			flushDisabled()
			// fall through to handle this line normally
		}

		key, val := splitDirective(trimmed)
		if strings.EqualFold(key, "Host") {
			alias := firstField(val)
			if alias == "" || strings.ContainsAny(alias, "*?!") {
				continue // skip wildcard / negated patterns
			}
			hosts = append(hosts, enabledHost(cfg, alias))
		}
	}
	flushDisabled()
	return hosts
}

// enabledHost builds a Host from ssh_config lookups for the given alias.
func enabledHost(cfg *ssh_config.Config, alias string) Host {
	get := func(key string) string {
		v, _ := cfg.Get(alias, key)
		return unquote(strings.TrimSpace(v))
	}
	h := Host{
		Alias:        alias,
		Hostname:     get("HostName"),
		User:         get("User"),
		IdentityFile: get("IdentityFile"),
		ProxyJump:    get("ProxyJump"),
		Port:         22,
	}
	if p := get("Port"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			h.Port = n
		}
	}
	if envs, err := cfg.GetAll(alias, "SetEnv"); err == nil {
		for _, e := range envs {
			if name, value, ok := splitEnv(e); ok {
				h.EnvVars = append(h.EnvVars, EnvVar{Name: name, Value: value})
			}
		}
	}
	return h
}

// parseDisabledBlock builds a Host from the uncommented directive lines of a
// disabled block. ok is false if the block contained no Host directive.
func parseDisabledBlock(lines []string) (Host, bool) {
	h := Host{Disabled: true, Port: 22}
	found := false
	for _, l := range lines {
		key, val := splitDirective(l)
		switch strings.ToLower(key) {
		case "host":
			h.Alias = firstField(val)
			found = true
		case "hostname":
			h.Hostname = unquote(val)
		case "user":
			h.User = unquote(val)
		case "port":
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				h.Port = n
			}
		case "identityfile":
			h.IdentityFile = unquote(val)
		case "proxyjump":
			h.ProxyJump = unquote(val)
		case "setenv":
			if name, value, ok := splitEnv(val); ok {
				h.EnvVars = append(h.EnvVars, EnvVar{Name: name, Value: value})
			}
		}
	}
	return h, found && h.Alias != ""
}

// unquote removes a single pair of matching surrounding quotes from an ssh
// config value. ssh treats the quotes as delimiters rather than part of the
// value — IdentityFile "~/.ssh/key" refers to the file ~/.ssh/key — but the
// underlying parser preserves them, so they are stripped on read. Leaving them
// in would defeat tilde expansion and make ssh look for a literally-quoted
// filename.
func unquote(s string) string {
	if len(s) >= 2 {
		q := s[0]
		if (q == '"' || q == '\'') && s[len(s)-1] == q {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// uncomment strips a leading "#" (and one optional following space) from a
// commented config line.
func uncomment(s string) string {
	s = strings.TrimPrefix(s, "#")
	return strings.TrimPrefix(s, " ")
}

// splitDirective splits "Key value" or "Key=value" into key and trimmed value.
func splitDirective(s string) (key, val string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	idx := strings.IndexAny(s, " \t=")
	if idx < 0 {
		return s, ""
	}
	key = s[:idx]
	val = strings.TrimLeft(s[idx:], " \t=")
	return key, strings.TrimSpace(val)
}

// splitEnv splits "KEY=VALUE" on the first "=".
func splitEnv(s string) (name, value string, ok bool) {
	s = strings.TrimSpace(s)
	idx := strings.Index(s, "=")
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(s[:idx]), s[idx+1:], true
}

// firstField returns the first whitespace-delimited token of s.
func firstField(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
