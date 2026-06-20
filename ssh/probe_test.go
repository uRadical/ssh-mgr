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
	"errors"
	"strings"
	"testing"
)

func TestProbeError(t *testing.T) {
	exit := errors.New("exit status 255")

	tests := []struct {
		name string
		out  string
		err  error
		want string
	}{
		{
			name: "unknown host key gets actionable hint",
			out:  "No ED25519 host key is known for example.com and you have requested strict checking.\nHost key verification failed.",
			err:  exit,
			want: "unknown host key — connect (s) to verify and trust it",
		},
		{
			name: "other failure passes ssh output through",
			out:  "ssh: connect to host example.com port 22: Connection refused",
			err:  exit,
			want: "ssh: connect to host example.com port 22: Connection refused",
		},
		{
			name: "empty output falls back to err",
			out:  "",
			err:  exit,
			want: "exit status 255",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := probeError(tt.out, tt.err); got != tt.want {
				t.Errorf("probeError(%q, %v) = %q, want %q", tt.out, tt.err, got, tt.want)
			}
		})
	}
}

func TestProbeErrorTrimsWhitespace(t *testing.T) {
	got := probeError("  \n connection timed out \n ", errors.New("x"))
	if strings.Contains(got, "\n") || strings.HasPrefix(got, " ") {
		t.Errorf("probeError did not trim surrounding whitespace: %q", got)
	}
}
