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
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"uradical.io/go/sshmgr/theme"
)

// FilePicker selects an IdentityFile, rooted at ~/.ssh with hidden files shown
// (ssh keys are dotfiles).
type FilePicker struct {
	fp     filepicker.Model
	styles theme.Styles
}

// NewFilePicker constructs a picker rooted at ~/.ssh.
func NewFilePicker(s theme.Styles, height int) FilePicker {
	fp := filepicker.New()
	fp.ShowHidden = true
	fp.DirAllowed = false
	fp.FileAllowed = true
	fp.AutoHeight = false
	if height > 0 {
		fp.Height = height
	}
	if home, err := os.UserHomeDir(); err == nil {
		fp.CurrentDirectory = filepath.Join(home, ".ssh")
	}
	return FilePicker{fp: fp, styles: s}
}

func (f FilePicker) Init() tea.Cmd { return f.fp.Init() }

// Update advances the picker. The returned string is the selected path (only
// when the boolean is true).
func (f FilePicker) Update(msg tea.Msg) (FilePicker, tea.Cmd, string, bool) {
	var cmd tea.Cmd
	f.fp, cmd = f.fp.Update(msg)
	if did, path := f.fp.DidSelectFile(msg); did {
		return f, cmd, path, true
	}
	return f, cmd, "", false
}

func (f FilePicker) SetHeight(h int) FilePicker {
	if h > 0 {
		f.fp.Height = h
	}
	return f
}

func (f FilePicker) View() string {
	s := f.styles
	return s.Title.Render("Select identity file") + "\n" +
		s.Label.Render(f.fp.CurrentDirectory) + "\n\n" +
		f.fp.View() + "\n\n" +
		s.Help.Render("enter select · esc cancel")
}
