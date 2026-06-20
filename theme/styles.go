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

package theme

import "github.com/charmbracelet/lipgloss"

// Styles holds every lipgloss style the UI uses. Components never reference a
// colour directly — they read styles from here, so a theme change is purely a
// matter of rebuilding this struct with NewStyles.
type Styles struct {
	App   lipgloss.Style
	Title lipgloss.Style

	Panel        lipgloss.Style
	PanelFocused lipgloss.Style

	Item         lipgloss.Style
	ItemSelected lipgloss.Style
	ItemDisabled lipgloss.Style

	Label lipgloss.Style
	Value lipgloss.Style

	Help    lipgloss.Style
	HelpKey lipgloss.Style
	Status  lipgloss.Style

	StatusOK      lipgloss.Style
	StatusFailed  lipgloss.Style
	StatusTesting lipgloss.Style
	StatusUnknown lipgloss.Style

	Input       lipgloss.Style
	InputFocus  lipgloss.Style
	FieldLabel  lipgloss.Style
	FieldActive lipgloss.Style

	Modal      lipgloss.Style
	ModalTitle lipgloss.Style
	Button     lipgloss.Style
	ButtonKey  lipgloss.Style

	Error      lipgloss.Style
	ErrorTitle lipgloss.Style
}

// NewStyles constructs all styles from a theme.
func NewStyles(t Theme) Styles {
	c := func(s string) lipgloss.Color { return lipgloss.Color(s) }

	primary := c(t.Primary)
	secondary := c(t.Secondary)
	success := c(t.Success)
	warning := c(t.Warning)
	errc := c(t.Error)
	muted := c(t.Muted)
	subtle := c(t.Subtle)
	border := c(t.Border)
	bgPanel := c(t.BgPanel)

	var s Styles

	s.App = lipgloss.NewStyle()

	s.Title = lipgloss.NewStyle().
		Foreground(primary).
		Bold(true).
		Padding(0, 1)

	s.Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)

	s.PanelFocused = s.Panel.BorderForeground(primary)

	s.Item = lipgloss.NewStyle().Padding(0, 1)

	s.ItemSelected = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(primary).
		Background(bgPanel).
		Bold(true)

	s.ItemDisabled = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(muted)

	s.Label = lipgloss.NewStyle().Foreground(muted)
	s.Value = lipgloss.NewStyle().Foreground(secondary)

	s.Help = lipgloss.NewStyle().Foreground(subtle).Padding(0, 1)
	s.HelpKey = lipgloss.NewStyle().Foreground(muted).Bold(true)
	s.Status = lipgloss.NewStyle().Foreground(warning).Padding(0, 1)

	s.StatusOK = lipgloss.NewStyle().Foreground(success).Bold(true)
	s.StatusFailed = lipgloss.NewStyle().Foreground(errc).Bold(true)
	s.StatusTesting = lipgloss.NewStyle().Foreground(warning).Bold(true)
	s.StatusUnknown = lipgloss.NewStyle().Foreground(muted)

	s.Input = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(border)
	s.InputFocus = s.Input.BorderForeground(primary)
	s.FieldLabel = lipgloss.NewStyle().Foreground(muted)
	s.FieldActive = lipgloss.NewStyle().Foreground(primary).Bold(true)

	s.Modal = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Background(bgPanel).
		Padding(1, 2)
	s.ModalTitle = lipgloss.NewStyle().Foreground(primary).Bold(true)
	s.Button = lipgloss.NewStyle().
		Foreground(secondary).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border)
	s.ButtonKey = lipgloss.NewStyle().Foreground(primary).Bold(true)

	s.Error = lipgloss.NewStyle().
		Foreground(errc).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(errc).
		Padding(1, 2)
	s.ErrorTitle = lipgloss.NewStyle().Foreground(errc).Bold(true)

	return s
}
