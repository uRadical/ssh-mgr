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

// Package model is the root Bubble Tea model. It owns the application state
// machine and dispatches to the ui components for rendering and sub-updates.
package model

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"uradical.io/go/sshmgr/ssh"
	"uradical.io/go/sshmgr/theme"
	"uradical.io/go/sshmgr/ui"
)

type state int

const (
	stateList state = iota
	stateDetail
	stateEdit
	stateAdd
	stateFilePicker
	stateConfirm
	stateConnectModal
	stateKnownHost
	stateError
)

// Model is the root application model.
type Model struct {
	hosts  []ssh.Host
	cursor int
	state  state
	styles theme.Styles

	width  int
	height int

	edit       ui.EditForm
	picker     ui.FilePicker
	modal      ui.ConnectModal
	knownModal ui.KnownHostModal

	pickerReturn state // state to return to from the file picker
	parseErr     error // malformed-config error -> stateError
	status       string

	// Pending state while an unknown host key awaits user approval.
	pendingAdd    string // known_hosts lines to write on confirm
	pendingAlias  string
	pendingAction ssh.HostKeyAction
}

// New constructs the root model. If parseErr is non-nil (malformed
// ~/.ssh/config), the model starts in the full-screen error state.
func New(hosts []ssh.Host, parseErr error, styles theme.Styles) Model {
	m := Model{
		hosts:  hosts,
		styles: styles,
		state:  stateList,
	}
	if parseErr != nil {
		m.parseErr = parseErr
		m.state = stateError
	}
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) selected() *ssh.Host {
	if m.cursor < 0 || m.cursor >= len(m.hosts) {
		return nil
	}
	return &m.hosts[m.cursor]
}

func (m *Model) persist() {
	if err := ssh.WriteConfig(m.hosts); err != nil {
		m.status = "write failed: " + err.Error()
		return
	}
	m.status = ""
}

// Update is the central message dispatcher.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.edit.SetWidth(msg.Width)
		m.picker = m.picker.SetHeight(m.pickerHeight())
		return m, nil

	case ssh.ConnectResultMsg:
		m.applyConnectResult(msg)
		return m, nil

	case ssh.HostKeyMsg:
		return m.handleHostKeyMsg(msg)

	case ui.SessionEndedMsg:
		if m.state == stateConnectModal {
			m.state = stateList
		}
		return m, nil

	case tea.KeyMsg:
		// ctrl+c always quits, except on the error screen where any key exits.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleKey(msg)
	}

	// Forward other messages (e.g. cursor blink, filepicker IO) to the active
	// sub-component.
	return m.forward(msg)
}

func (m Model) forward(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.state {
	case stateEdit, stateAdd:
		m.edit, cmd = m.edit.Update(msg)
	case stateFilePicker:
		var path string
		var picked bool
		m.picker, cmd, path, picked = m.picker.Update(msg)
		if picked {
			m.edit.SetIdentity(path)
			m.state = m.pickerReturn
		}
	}
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateError:
		return m, tea.Quit
	case stateList, stateDetail:
		return m.handleListKey(msg)
	case stateEdit, stateAdd:
		return m.handleEditKey(msg)
	case stateFilePicker:
		return m.handlePickerKey(msg)
	case stateConfirm:
		return m.handleConfirmKey(msg)
	case stateConnectModal:
		return m.handleModalKey(msg)
	case stateKnownHost:
		return m.handleKnownHostKey(msg)
	}
	return m, nil
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "j", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "k", "down":
		if m.cursor < len(m.hosts)-1 {
			m.cursor++
		}
	case "e", "enter":
		if h := m.selected(); h != nil {
			m.edit = ui.NewEditForm(*h, false, m.styles)
			m.edit.SetWidth(m.width)
			m.state = stateEdit
			return m, m.edit.Init()
		}
	case "a":
		m.edit = ui.NewEditForm(ssh.Host{Port: 22}, true, m.styles)
		m.edit.SetWidth(m.width)
		m.state = stateAdd
		return m, m.edit.Init()
	case "c":
		if h := m.selected(); h != nil {
			clone := *h
			clone.EnvVars = append([]ssh.EnvVar(nil), h.EnvVars...)
			clone.Alias = m.uniqueAlias(h.Alias + "_copy")
			clone.ConnectStatus = ssh.ConnectUnknown
			clone.ConnectErr = ""
			clone.ConnectMs = 0
			m.edit = ui.NewEditForm(clone, true, m.styles)
			m.edit.SetWidth(m.width)
			m.state = stateAdd
			return m, m.edit.Init()
		}
	case "t":
		if h := m.selected(); h != nil {
			if h.Disabled {
				m.status = "cannot test a disabled host"
				return m, nil
			}
			return m.beginHostKeyFlow(*h, ssh.HostKeyTest)
		}
	case "s":
		if h := m.selected(); h != nil {
			if h.Disabled {
				m.status = "cannot connect to a disabled host"
				return m, nil
			}
			return m.beginHostKeyFlow(*h, ssh.HostKeyConnect)
		}
	case "d":
		if h := m.selected(); h != nil {
			h.Disabled = !h.Disabled
			h.ConnectStatus = ssh.ConnectUnknown
			m.persist()
		}
	case "x":
		if m.selected() != nil {
			m.state = stateConfirm
		}
	}
	return m, nil
}

func (m Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateList
		return m, nil
	case "enter":
		if err := m.edit.Validate(); err != "" {
			m.edit.SetError(err)
			return m, nil
		}
		host := m.edit.Host()
		if m.state == stateAdd {
			m.hosts = append(m.hosts, host)
			m.cursor = len(m.hosts) - 1
		} else if m.cursor >= 0 && m.cursor < len(m.hosts) {
			m.hosts[m.cursor] = host
		}
		m.persist()
		m.state = stateList
		return m, nil
	case " ", "f":
		// Open the identity-file picker only when that field is focused.
		if m.edit.FocusedField() == ui.IdentityFieldIndex {
			m.pickerReturn = m.state
			m.picker = ui.NewFilePicker(m.styles, m.pickerHeight())
			m.state = stateFilePicker
			return m, m.picker.Init()
		}
	}
	var cmd tea.Cmd
	m.edit, cmd = m.edit.Update(msg)
	return m, cmd
}

func (m Model) handlePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.state = m.pickerReturn
		return m, nil
	}
	var cmd tea.Cmd
	var path string
	var picked bool
	m.picker, cmd, path, picked = m.picker.Update(msg)
	if picked {
		m.edit.SetIdentity(path)
		m.state = m.pickerReturn
	}
	return m, cmd
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if m.cursor >= 0 && m.cursor < len(m.hosts) {
			m.hosts = append(m.hosts[:m.cursor], m.hosts[m.cursor+1:]...)
			if m.cursor >= len(m.hosts) && m.cursor > 0 {
				m.cursor--
			}
			m.persist()
		}
		m.state = stateList
	case "n", "esc":
		m.state = stateList
	}
	return m, nil
}

func (m Model) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateList
		return m, nil
	case "f":
		m.modal.Fullscreen = !m.modal.Fullscreen
		if m.modal.Fullscreen {
			return m, tea.EnterAltScreen
		}
		return m, tea.ExitAltScreen
	}
	return m, nil
}

// beginHostKeyFlow starts the user's intent (test or connect). Directly
// reachable hosts are pre-checked against known_hosts first so an unknown key
// can be approved in the TUI; ProxyJump hosts (unreachable by ssh-keyscan) and
// hosts with no HostName fall straight through to ssh's own handling.
func (m Model) beginHostKeyFlow(h ssh.Host, action ssh.HostKeyAction) (tea.Model, tea.Cmd) {
	if h.Hostname == "" || h.ProxyJump != "" {
		return m.startAction(h, action)
	}
	m.status = "checking host key…"
	return m, ssh.CheckHostKey(h, action)
}

// startAction performs the intent assuming the host key is already trusted.
func (m Model) startAction(h ssh.Host, action ssh.HostKeyAction) (tea.Model, tea.Cmd) {
	if action == ssh.HostKeyConnect {
		m.modal = ui.NewConnectModal(h, m.styles)
		m.state = stateConnectModal
		return m, ui.ConnectCmd(h)
	}
	m.setStatus(h.Alias, ssh.ConnectTesting, "")
	m.status = ""
	return m, ssh.Probe(h)
}

// handleHostKeyMsg resumes the pending action once the known-host check returns:
// proceed if known, surface the modal if a new key needs approval, or handle a
// scan failure per action.
func (m Model) handleHostKeyMsg(msg ssh.HostKeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""
	h := m.hostByAlias(msg.Alias)
	if h == nil {
		return m, nil
	}
	switch {
	case msg.Known:
		return m.startAction(*h, msg.Action)
	case msg.Err != "":
		// Couldn't fetch the key. Let an interactive connect fall back to ssh's
		// own verification; report a test as failed.
		if msg.Action == ssh.HostKeyConnect {
			return m.startAction(*h, msg.Action)
		}
		m.setStatus(msg.Alias, ssh.ConnectFailed, "host key check failed: "+msg.Err)
		return m, nil
	default:
		m.pendingAdd = msg.Lines
		m.pendingAlias = msg.Alias
		m.pendingAction = msg.Action
		m.knownModal = ui.NewKnownHostModal(*h, msg.Fingerprints, m.styles)
		m.state = stateKnownHost
		return m, nil
	}
}

// handleKnownHostKey handles approval (y) or rejection (n/esc) of an unknown
// host key. On approval the key is persisted and the original action resumes.
func (m Model) handleKnownHostKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		lines, alias, action := m.pendingAdd, m.pendingAlias, m.pendingAction
		m.clearPending()
		m.state = stateList
		if err := ssh.AddKnownHost(lines); err != nil {
			m.setStatus(alias, ssh.ConnectFailed, "could not write known_hosts: "+err.Error())
			return m, nil
		}
		if h := m.hostByAlias(alias); h != nil {
			return m.startAction(*h, action)
		}
		return m, nil
	case "n", "esc":
		m.clearPending()
		m.state = stateList
		m.status = ""
	}
	return m, nil
}

func (m *Model) clearPending() {
	m.pendingAdd = ""
	m.pendingAlias = ""
}

func (m *Model) hostByAlias(alias string) *ssh.Host {
	for i := range m.hosts {
		if m.hosts[i].Alias == alias {
			return &m.hosts[i]
		}
	}
	return nil
}

// setStatus updates the transient connect state of the host with the given
// alias, if it still exists.
func (m *Model) setStatus(alias string, status ssh.ConnectStatus, errMsg string) {
	if h := m.hostByAlias(alias); h != nil {
		h.ConnectStatus = status
		h.ConnectErr = errMsg
	}
}

func (m *Model) applyConnectResult(msg ssh.ConnectResultMsg) {
	for i := range m.hosts {
		if m.hosts[i].Alias != msg.HostAlias {
			continue
		}
		if msg.OK {
			m.hosts[i].ConnectStatus = ssh.ConnectOK
			m.hosts[i].ConnectErr = ""
		} else {
			m.hosts[i].ConnectStatus = ssh.ConnectFailed
			m.hosts[i].ConnectErr = msg.Err
		}
		m.hosts[i].ConnectMs = msg.ElapsedMs
		return
	}
}

// uniqueAlias returns base, or base with a numeric suffix, such that it does
// not collide with an existing host alias.
func (m Model) uniqueAlias(base string) string {
	exists := func(a string) bool {
		for i := range m.hosts {
			if m.hosts[i].Alias == a {
				return true
			}
		}
		return false
	}
	if !exists(base) {
		return base
	}
	for n := 2; ; n++ {
		cand := fmt.Sprintf("%s%d", base, n)
		if !exists(cand) {
			return cand
		}
	}
}

func (m Model) pickerHeight() int {
	h := m.height - 8
	if h < 5 {
		h = 5
	}
	return h
}

// --- View ---

func (m Model) View() string {
	switch m.state {
	case stateError:
		return m.errorView()
	case stateEdit, stateAdd:
		// Float the form over the list so the main window shows through.
		return m.overlay(m.styles.Modal.Render(m.edit.View()))
	case stateFilePicker:
		return m.overlay(m.styles.Modal.Render(m.picker.View()))
	case stateConnectModal:
		return m.center(m.modal.View())
	case stateKnownHost:
		return m.center(m.knownModal.View())
	case stateConfirm:
		return m.overlay(m.confirmBox())
	default:
		return m.listView()
	}
}

func (m Model) errorView() string {
	s := m.styles
	body := s.ErrorTitle.Render("Could not parse ~/.ssh/config") + "\n\n" +
		s.Error.Render(m.parseErr.Error()) + "\n\n" +
		s.Help.Render("The file was left untouched. Press any key to exit.")
	return m.center(body)
}

func (m Model) listView() string {
	s := m.styles

	header := s.Title.Render(fmt.Sprintf("ssh-mgr · %d hosts", len(m.hosts)))

	listW := 36
	if m.width > 0 && listW > m.width/2 {
		listW = m.width / 2
	}
	detailW := m.width - listW - 6
	if detailW < 20 {
		detailW = 20
	}

	list := s.Panel.Width(listW).Render(
		RenderHeading(s, "Hosts") + "\n" +
			ui.RenderHostList(m.hosts, m.cursor, s, listW, 0),
	)

	var detail string
	if h := m.selected(); h != nil {
		detail = ui.RenderDetail(*h, s)
	} else {
		detail = s.Label.Render("no host selected")
	}
	detailPanel := s.Panel.Width(detailW).Render(detail)

	body := lipgloss.JoinHorizontal(lipgloss.Top, list, " ", detailPanel)

	// The toggle label reflects what 'd' will do to the selected host.
	toggleLabel := "disable"
	if h := m.selected(); h != nil && h.Disabled {
		toggleLabel = "enable"
	}

	help := s.Help.Render(
		keyHelp(s, "j/k", "move") + "  " +
			keyHelp(s, "e", "edit") + "  " +
			keyHelp(s, "a", "add") + "  " +
			keyHelp(s, "c", "clone") + "  " +
			keyHelp(s, "t", "test") + "  " +
			keyHelp(s, "s", "connect") + "  " +
			keyHelp(s, "d", toggleLabel) + "  " +
			keyHelp(s, "x", "delete") + "  " +
			keyHelp(s, "q", "quit"),
	)

	out := header + "\n" + body + "\n" + help
	if m.status != "" {
		out += "\n" + s.StatusFailed.Render(m.status)
	}
	return out
}

func (m Model) confirmBox() string {
	s := m.styles
	alias := ""
	if h := m.selected(); h != nil {
		alias = h.Alias
	}
	body := s.ModalTitle.Render("Delete host") + "\n\n" +
		"Delete " + s.Value.Render(alias) + "?\n\n" +
		s.Help.Render(keyHelp(s, "y", "yes")+"   "+keyHelp(s, "n", "no"))
	return s.Modal.Render(body)
}

// center places content in the middle of the terminal, blanking the rest.
func (m Model) center(content string) string {
	if m.width <= 0 || m.height <= 0 {
		return content
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// overlay floats box centred over the host-list view so the main window
// remains visible behind it.
func (m Model) overlay(box string) string {
	if m.width <= 0 || m.height <= 0 {
		return box
	}
	bg := m.listView()
	x := (m.width - lipgloss.Width(box)) / 2
	y := (m.height - lipgloss.Height(box)) / 2
	return ui.PlaceOverlay(x, y, box, bg)
}

func keyHelp(s theme.Styles, key, label string) string {
	return s.HelpKey.Render(key) + " " + label
}

// RenderHeading renders a small section heading.
func RenderHeading(s theme.Styles, text string) string {
	return s.Label.Render(strings.ToUpper(text))
}
