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

package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

const escByte = '\x1b'

// PlaceOverlay composites the foreground block fg onto the background block bg
// with fg's top-left corner at cell coordinates (x, y). Background rows that fg
// does not cover are kept verbatim; on covered rows the background to the left
// and right of fg is preserved, ANSI styling intact.
//
// Bubble Tea has no native pop-up layer — View returns a single string — so a
// floating panel is produced by compositing two rendered strings like this.
func PlaceOverlay(x, y int, fg, bg string) string {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	fgLines := strings.Split(fg, "\n")
	bgLines := strings.Split(bg, "\n")

	for len(bgLines) < y+len(fgLines) {
		bgLines = append(bgLines, "")
	}

	for i, fgLine := range fgLines {
		row := y + i
		if row >= len(bgLines) {
			break
		}
		fgw := ansi.StringWidth(fgLine)
		left, right := cutSides(bgLines[row], x, x+fgw)
		bgLines[row] = left + fgLine + right
	}
	return strings.Join(bgLines, "\n")
}

// cutSides splits an ANSI-styled line into the portion before column from and
// the portion from column to onward. The left portion is padded with spaces to
// exactly width from so the foreground always begins at the requested column;
// the right portion is prefixed with whatever SGR style was active at the cut
// so its colours survive.
func cutSides(s string, from, to int) (left, right string) {
	var (
		lb, rb       strings.Builder
		active       strings.Builder
		col          int
		leftWidth    int
		startedRight bool
	)

	runes := []rune(s)
	i := 0
	for i < len(runes) {
		r := runes[i]

		if r == escByte {
			seq, ni := readEscape(runes, i)
			i = ni
			if col < to {
				switch {
				case isReset(seq):
					active.Reset()
				case isSGR(seq):
					active.WriteString(seq)
				}
			}
			if col < from {
				lb.WriteString(seq)
			} else if startedRight {
				rb.WriteString(seq)
			}
			continue
		}

		w := runewidth.RuneWidth(r)
		switch {
		case col < from:
			lb.WriteRune(r)
			leftWidth += w
		case col >= to:
			if !startedRight {
				rb.WriteString(active.String())
				startedRight = true
			}
			rb.WriteRune(r)
		}
		col += w
		i++
	}

	leftStr := lb.String()
	if leftStr != "" {
		leftStr += "\x1b[0m"
	}
	if pad := from - leftWidth; pad > 0 {
		leftStr += strings.Repeat(" ", pad)
	}
	return leftStr, rb.String()
}

// readEscape reads one escape sequence beginning at runes[i] (a 0x1b). For a
// CSI sequence (ESC '[') it consumes through the final byte; otherwise it
// returns the ESC plus the following rune. It returns the sequence and the
// index just past it.
func readEscape(runes []rune, i int) (string, int) {
	start := i
	i++ // past ESC
	if i < len(runes) && runes[i] == '[' {
		i++
		for i < len(runes) {
			r := runes[i]
			i++
			if r >= 0x40 && r <= 0x7e { // final byte
				break
			}
		}
		return string(runes[start:i]), i
	}
	if i < len(runes) {
		i++
	}
	return string(runes[start:i]), i
}

func isSGR(seq string) bool   { return strings.HasSuffix(seq, "m") }
func isReset(seq string) bool { return seq == "\x1b[0m" || seq == "\x1b[m" }
