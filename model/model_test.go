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

package model

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"uradical.io/go/sshmgr/ssh"
	"uradical.io/go/sshmgr/theme"
)

func newTestModel() Model {
	styles := theme.NewStyles(theme.Default())
	hosts := []ssh.Host{
		{Alias: "alpha", Hostname: "a.local", User: "u", Port: 22},
		{Alias: "beta", Hostname: "b.local", Port: 2200, Disabled: true},
	}
	m := New(hosts, nil, styles)
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return mm.(Model)
}

func key(s string) tea.KeyMsg {
	if len(s) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestListViewRenders(t *testing.T) {
	m := newTestModel()
	out := m.View()
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Fatalf("list view missing hosts:\n%s", out)
	}
}

func TestNavigation(t *testing.T) {
	m := newTestModel()
	// 'k' navigates down per the spec.
	mm, _ := m.Update(key("k"))
	if mm.(Model).cursor != 1 {
		t.Fatalf("cursor = %d want 1", mm.(Model).cursor)
	}
	// 'j' navigates up.
	mm, _ = mm.(Model).Update(key("j"))
	if mm.(Model).cursor != 0 {
		t.Fatalf("cursor = %d want 0", mm.(Model).cursor)
	}
}

func TestEnterEditAndViewDoesNotPanic(t *testing.T) {
	m := newTestModel()
	mm, _ := m.Update(key("e"))
	m = mm.(Model)
	if m.state != stateEdit {
		t.Fatalf("state = %v want stateEdit", m.state)
	}
	_ = m.View() // must not panic
}

func TestAddHost(t *testing.T) {
	m := newTestModel()
	mm, _ := m.Update(key("a"))
	m = mm.(Model)
	if m.state != stateAdd {
		t.Fatalf("state = %v want stateAdd", m.state)
	}
	_ = m.View()
}

func TestCloneHost(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // saving persists; keep it off the real config
	m := newTestModel()
	mm, _ := m.Update(key("c"))
	m = mm.(Model)
	if m.state != stateAdd {
		t.Fatalf("state = %v want stateAdd", m.state)
	}
	// Clone opens the add form pre-filled; the host count is unchanged until save.
	if len(m.hosts) != 2 {
		t.Fatalf("hosts = %d want 2 (clone not yet saved)", len(m.hosts))
	}
	host := m.edit.Host()
	if host.Alias != "alpha_copy" {
		t.Fatalf("clone alias = %q want alpha_copy", host.Alias)
	}
	if host.Hostname != "a.local" {
		t.Fatalf("clone hostname = %q want a.local", host.Hostname)
	}
	// Saving appends the clone as a new host.
	mm, _ = m.Update(key("enter"))
	m = mm.(Model)
	if len(m.hosts) != 3 || m.hosts[2].Alias != "alpha_copy" {
		t.Fatalf("after save: %d hosts, last = %+v", len(m.hosts), m.hosts[len(m.hosts)-1])
	}
}

func TestUniqueAliasAvoidsCollision(t *testing.T) {
	m := newTestModel()
	m.hosts = append(m.hosts, ssh.Host{Alias: "alpha_copy", Port: 22})
	if got := m.uniqueAlias("alpha_copy"); got != "alpha_copy2" {
		t.Fatalf("uniqueAlias = %q want alpha_copy2", got)
	}
}

func TestDisabledHostCannotTest(t *testing.T) {
	m := newTestModel()
	mm, _ := m.Update(key("k")) // select beta (disabled)
	mm, cmd := mm.(Model).Update(key("t"))
	if cmd != nil {
		t.Fatal("expected no probe command for disabled host")
	}
	if !strings.Contains(mm.(Model).status, "disabled") {
		t.Fatalf("status = %q", mm.(Model).status)
	}
}

func TestDisabledHostCannotConnect(t *testing.T) {
	m := newTestModel()
	mm, _ := m.Update(key("k")) // select beta (disabled)
	mm, cmd := mm.(Model).Update(key("s"))
	if cmd != nil {
		t.Fatal("expected no connect command for disabled host")
	}
	if !strings.Contains(mm.(Model).status, "disabled") {
		t.Fatalf("status = %q", mm.(Model).status)
	}
}

func TestToggleHelpLabelReflectsSelection(t *testing.T) {
	m := newTestModel() // cursor on alpha (enabled)
	if !strings.Contains(m.View(), "disable") {
		t.Fatalf("help should read 'disable' for an enabled host:\n%s", m.View())
	}
	mm, _ := m.Update(key("k")) // move to beta (disabled)
	if !strings.Contains(mm.(Model).View(), "enable") {
		t.Fatalf("help should read 'enable' for a disabled host:\n%s", mm.(Model).View())
	}
}

func TestErrorView(t *testing.T) {
	styles := theme.NewStyles(theme.Default())
	m := New(nil, errors.New("boom at line 5"), styles)
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mm.(Model)
	if m.state != stateError {
		t.Fatalf("state = %v want stateError", m.state)
	}
	if !strings.Contains(m.View(), "boom at line 5") {
		t.Fatalf("error view missing message:\n%s", m.View())
	}
	// Any key quits.
	_, cmd := m.Update(key("x"))
	if cmd == nil {
		t.Fatal("expected quit command on keypress in error state")
	}
}

func TestConnectResultUpdatesHost(t *testing.T) {
	m := newTestModel()
	mm, _ := m.Update(ssh.ConnectResultMsg{HostAlias: "alpha", OK: true, ElapsedMs: 42})
	if got := mm.(Model).hosts[0].ConnectStatus; got != ssh.ConnectOK {
		t.Fatalf("status = %v want ConnectOK", got)
	}
}
