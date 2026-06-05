package ssh

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestKnownHostTarget(t *testing.T) {
	cases := []struct {
		host string
		port int
		want string
	}{
		{"example.com", 22, "example.com"},
		{"example.com", 0, "example.com"}, // 0 defaults to 22
		{"10.0.0.5", 2222, "[10.0.0.5]:2222"},
	}
	for _, c := range cases {
		if got := knownHostTarget(c.host, c.port); got != c.want {
			t.Errorf("knownHostTarget(%q, %d) = %q want %q", c.host, c.port, got, c.want)
		}
	}
}

func TestParseFingerprints(t *testing.T) {
	out := "256 SHA256:abc123 10.0.0.5 (ED25519)\n" +
		"3072 SHA256:def456 10.0.0.5 (RSA)\n" +
		"garbage\n"
	got := parseFingerprints(out)
	want := []string{"ED25519  SHA256:abc123", "RSA  SHA256:def456"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseFingerprints = %v want %v", got, want)
	}
}

func TestKeyLinesDropsComments(t *testing.T) {
	out := "# 10.0.0.5:22 SSH-2.0-OpenSSH_9.6\n" +
		"10.0.0.5 ssh-ed25519 AAAAC3Nza...\n" +
		"\n" +
		"10.0.0.5 ssh-rsa AAAAB3Nza...\n"
	want := "10.0.0.5 ssh-ed25519 AAAAC3Nza...\n10.0.0.5 ssh-rsa AAAAB3Nza..."
	if got := keyLines(out); got != want {
		t.Fatalf("keyLines = %q want %q", got, want)
	}
}

func TestAddKnownHost(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// A line with no trailing newline must still land on its own line, and the
	// file must be created with 0600 perms inside a fresh ~/.ssh.
	if err := AddKnownHost("10.0.0.5 ssh-ed25519 AAAAFIRST"); err != nil {
		t.Fatalf("AddKnownHost: %v", err)
	}
	if err := AddKnownHost("10.0.0.6 ssh-ed25519 AAAASECOND\n"); err != nil {
		t.Fatalf("AddKnownHost (2): %v", err)
	}

	path := filepath.Join(tmp, ".ssh", "known_hosts")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("perm = %o want 600", perm)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "10.0.0.5 ssh-ed25519 AAAAFIRST\n10.0.0.6 ssh-ed25519 AAAASECOND\n"
	if string(data) != want {
		t.Fatalf("known_hosts =\n%q\nwant\n%q", data, want)
	}
	if strings.Count(string(data), "\n") != 2 {
		t.Fatalf("expected exactly 2 lines, got %q", data)
	}
}
