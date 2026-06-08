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
