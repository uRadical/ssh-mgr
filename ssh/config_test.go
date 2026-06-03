package ssh

import (
	"reflect"
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
