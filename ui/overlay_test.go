package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestPlaceOverlayPlain(t *testing.T) {
	bg := strings.Join([]string{
		"aaaaaaaaaa",
		"bbbbbbbbbb",
		"cccccccccc",
		"dddddddddd",
	}, "\n")
	fg := "XX\nXX"

	got := PlaceOverlay(4, 1, fg, bg)
	lines := strings.Split(got, "\n")

	// Row 0 untouched; rows 1-2 have XX at column 4; row 3 untouched.
	want := []string{
		"aaaaaaaaaa",
		"bbbbXXbbbb",
		"ccccXXcccc",
		"dddddddddd",
	}
	for i := range want {
		if got := ansi.Strip(lines[i]); got != want[i] {
			t.Fatalf("row %d = %q want %q", i, got, want[i])
		}
	}
}

func TestPlaceOverlayPreservesBackgroundColour(t *testing.T) {
	// A fully-styled background line must keep its colour on both sides of the
	// overlay.
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	bg := red.Render("0123456789")
	fg := "##"

	got := PlaceOverlay(4, 0, fg, bg)
	if ansi.Strip(got) != "0123##6789" {
		t.Fatalf("visible = %q", ansi.Strip(got))
	}
	// The right segment ("6789") must still carry an SGR colour code.
	rightIdx := strings.Index(got, "6789")
	if rightIdx < 0 || !strings.Contains(got[:rightIdx], "\x1b[") {
		t.Fatalf("right segment lost its styling: %q", got)
	}
}

func TestPlaceOverlayPadsShortBackground(t *testing.T) {
	got := PlaceOverlay(5, 1, "box", "short")
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected background padded to 2 rows, got %d", len(lines))
	}
	if !strings.HasSuffix(ansi.Strip(lines[1]), "box") {
		t.Fatalf("row 1 = %q", ansi.Strip(lines[1]))
	}
}
