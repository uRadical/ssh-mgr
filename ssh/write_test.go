package ssh

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWriteThenLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	// On some platforms UserHomeDir consults other vars; HOME is honoured on
	// the unix targets this tool runs on.

	hosts := []Host{
		{Alias: "alpha", Hostname: "a.local", User: "alan", Port: 22,
			IdentityFile: "~/.ssh/id_ed25519", EnvVars: []EnvVar{{"FOO", "bar"}}},
		{Alias: "voelund", Hostname: "voelund.local", User: "pi", Port: 2222,
			Disabled: true, EnvVars: []EnvVar{{"ARCH", "arm64"}}},
	}

	if err := WriteConfig(hosts); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	// The file must exist with 0600 perms in ~/.ssh.
	info, err := os.Stat(filepath.Join(tmp, ".ssh", "config"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("perm = %o want 600", perm)
	}

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !reflect.DeepEqual(got, hosts) {
		t.Fatalf("roundtrip mismatch:\n got %+v\nwant %+v", got, hosts)
	}

	// A second write (e.g. after an edit) must replace cleanly.
	hosts[0].User = "root"
	if err := WriteConfig(hosts); err != nil {
		t.Fatalf("second WriteConfig: %v", err)
	}
	got, _ = LoadConfig()
	if got[0].User != "root" {
		t.Fatalf("user = %q want root", got[0].User)
	}
}
