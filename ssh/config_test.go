package ssh

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kevinburke/ssh_config"
)

func parse(t *testing.T, text string) []Host {
	t.Helper()
	cfg, err := ssh_config.DecodeBytes([]byte(text))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return scanHosts([]byte(text), cfg)
}

func TestParseEnabledHost(t *testing.T) {
	hosts := parse(t, `Host web
    HostName web.example.com
    User deploy
    Port 2222
    IdentityFile ~/.ssh/id_ed25519
    SetEnv FOO=bar
    SetEnv BAZ=qux
`)
	want := []Host{{
		Alias:        "web",
		Hostname:     "web.example.com",
		User:         "deploy",
		Port:         2222,
		IdentityFile: "~/.ssh/id_ed25519",
		EnvVars:      []EnvVar{{"FOO", "bar"}, {"BAZ", "qux"}},
	}}
	if !reflect.DeepEqual(hosts, want) {
		t.Fatalf("got %+v want %+v", hosts, want)
	}
}

func TestDefaultPort(t *testing.T) {
	hosts := parse(t, "Host h\n    HostName h.local\n")
	if hosts[0].Port != 22 {
		t.Fatalf("port = %d want 22", hosts[0].Port)
	}
}

func TestParseDisabledHost(t *testing.T) {
	hosts := parse(t, `# [ssh-mgr:disabled]
# Host völund
#     HostName völund.local
#     User pi
#     Port 22
#     IdentityFile ~/.ssh/id_ed25519_pi
#     SetEnv ARCH=arm64
#     SetEnv GOARM=7
`)
	if len(hosts) != 1 {
		t.Fatalf("got %d hosts", len(hosts))
	}
	h := hosts[0]
	if !h.Disabled || h.Alias != "völund" || h.Hostname != "völund.local" || h.User != "pi" {
		t.Fatalf("bad disabled host: %+v", h)
	}
	if len(h.EnvVars) != 2 || h.EnvVars[1] != (EnvVar{"GOARM", "7"}) {
		t.Fatalf("bad env vars: %+v", h.EnvVars)
	}
}

func TestRoundTrip(t *testing.T) {
	in := []Host{
		{Alias: "a", Hostname: "a.local", User: "u", Port: 22, IdentityFile: "~/.ssh/k",
			EnvVars: []EnvVar{{"K", "V"}}},
		{Alias: "b", Hostname: "b.local", Port: 2200, Disabled: true},
	}
	text := serialize(in)

	cfg, err := ssh_config.DecodeBytes([]byte(text))
	if err != nil {
		t.Fatalf("serialized output is malformed: %v\n%s", err, text)
	}
	out := scanHosts([]byte(text), cfg)

	// Transient connect fields are never serialised; compare the rest.
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("roundtrip mismatch:\n got %+v\nwant %+v\ntext:\n%s", out, in, text)
	}
}

func TestSetEnvValueWithEquals(t *testing.T) {
	hosts := parse(t, "Host h\n    HostName h\n    SetEnv URL=https://x/y?a=b\n")
	if got := hosts[0].EnvVars[0]; got.Name != "URL" || got.Value != "https://x/y?a=b" {
		t.Fatalf("bad env split: %+v", got)
	}
}

// A quoted IdentityFile must have its delimiter quotes stripped on read so the
// stored path resolves to a real file rather than a literally-quoted name.
func TestParseQuotedIdentityFile(t *testing.T) {
	enabled := parse(t, "Host h\n    HostName h\n    IdentityFile \"~/.ssh/metro-vpn-ec2.pem\"\n")
	if got := enabled[0].IdentityFile; got != "~/.ssh/metro-vpn-ec2.pem" {
		t.Fatalf("enabled identity = %q want unquoted", got)
	}

	disabled := parse(t, "# [ssh-mgr:disabled]\n# Host h\n#     HostName h\n#     IdentityFile \"~/.ssh/metro-vpn-ec2.pem\"\n")
	if got := disabled[0].IdentityFile; got != "~/.ssh/metro-vpn-ec2.pem" {
		t.Fatalf("disabled identity = %q want unquoted", got)
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	want := filepath.Join(home, ".ssh", "k.pem")
	// Plain, quoted, and single-quoted tilde paths must all resolve to the
	// same absolute file; a non-tilde path is returned unchanged.
	for _, in := range []string{"~/.ssh/k.pem", `"~/.ssh/k.pem"`, "'~/.ssh/k.pem'"} {
		if got := ExpandPath(in); got != want {
			t.Fatalf("ExpandPath(%q) = %q want %q", in, got, want)
		}
	}
	if got := ExpandPath("/abs/k.pem"); got != "/abs/k.pem" {
		t.Fatalf("ExpandPath absolute changed: %q", got)
	}
}

// A path containing spaces must round-trip: quoted on write, unquoted on read.
func TestSpacePathRoundTrip(t *testing.T) {
	in := []Host{{Alias: "h", Hostname: "h", Port: 22, IdentityFile: "~/.ssh/my key.pem"}}
	text := serialize(in)
	if !strings.Contains(text, `IdentityFile "~/.ssh/my key.pem"`) {
		t.Fatalf("space path not quoted on write:\n%s", text)
	}
	cfg, err := ssh_config.DecodeBytes([]byte(text))
	if err != nil {
		t.Fatalf("serialized output malformed: %v\n%s", err, text)
	}
	out := scanHosts([]byte(text), cfg)
	if got := out[0].IdentityFile; got != "~/.ssh/my key.pem" {
		t.Fatalf("space path roundtrip = %q", got)
	}
}
