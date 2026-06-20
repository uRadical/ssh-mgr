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
