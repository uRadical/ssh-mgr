package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"uradical.io/go/sshmgr/ssh"
	"uradical.io/go/sshmgr/theme"
)

// Fixed (non-env) field indices into EditForm.inputs. IdentityFieldIndex is
// exported so the root model knows when to open the file picker.
const (
	fieldAlias = iota
	fieldHostname
	fieldUser
	fieldPort
	IdentityFieldIndex
	fieldProxyJump
	numFixedFields
)

var fixedLabels = [numFixedFields]string{
	fieldAlias:         "Alias",
	fieldHostname:      "HostName",
	fieldUser:          "User",
	fieldPort:          "Port",
	IdentityFieldIndex: "IdentityFile",
	fieldProxyJump:     "ProxyJump",
}

type envInput struct {
	name  textinput.Model
	value textinput.Model
}

// EditForm edits a single host. The same form serves both edit and add; Adding
// only changes the title and that there is no pre-existing block to preserve.
type EditForm struct {
	inputs   [numFixedFields]textinput.Model
	envs     []envInput
	focus    int // index into the flat focusable list
	styles   theme.Styles
	original ssh.Host
	Adding   bool
	width    int
	errMsg   string
}

// NewEditForm builds a form populated from h. Pass Adding=true and a zero Host
// for the add flow.
func NewEditForm(h ssh.Host, adding bool, s theme.Styles) EditForm {
	f := EditForm{styles: s, original: h, Adding: adding}

	for i := range f.inputs {
		in := textinput.New()
		in.Prompt = ""
		in.CharLimit = 256
		in.Width = 40
		f.inputs[i] = in
	}
	f.inputs[fieldAlias].SetValue(h.Alias)
	f.inputs[fieldHostname].SetValue(h.Hostname)
	f.inputs[fieldUser].SetValue(h.User)
	port := h.Port
	if port == 0 {
		port = 22
	}
	f.inputs[fieldPort].SetValue(strconv.Itoa(port))
	f.inputs[IdentityFieldIndex].SetValue(h.IdentityFile)
	f.inputs[IdentityFieldIndex].Placeholder = "space to browse"
	f.inputs[fieldProxyJump].SetValue(h.ProxyJump)

	for _, e := range h.EnvVars {
		f.envs = append(f.envs, newEnvInput(e.Name, e.Value))
	}

	f.setFocus(0)
	return f
}

// Init starts the cursor blink for the focused field.
func (f EditForm) Init() tea.Cmd { return textinput.Blink }

func newEnvInput(name, value string) envInput {
	n := textinput.New()
	n.Prompt = ""
	n.CharLimit = 128
	n.Width = 18
	n.Placeholder = "KEY"
	n.SetValue(name)

	v := textinput.New()
	v.Prompt = ""
	v.CharLimit = 256
	v.Width = 28
	v.Placeholder = "value"
	v.SetValue(value)

	return envInput{name: n, value: v}
}

// focusCount is the number of focusable inputs (fixed fields + 2 per env row).
func (f *EditForm) focusCount() int {
	return numFixedFields + 2*len(f.envs)
}

// focusableAt returns a pointer to the input at flat focus index i.
func (f *EditForm) focusableAt(i int) *textinput.Model {
	if i < numFixedFields {
		return &f.inputs[i]
	}
	j := i - numFixedFields
	row := j / 2
	if j%2 == 0 {
		return &f.envs[row].name
	}
	return &f.envs[row].value
}

func (f *EditForm) setFocus(i int) {
	if n := f.focusCount(); n > 0 {
		i = (i%n + n) % n
	} else {
		i = 0
	}
	for j := 0; j < f.focusCount(); j++ {
		f.focusableAt(j).Blur()
	}
	f.focus = i
	if f.focusCount() > 0 {
		f.focusableAt(i).Focus()
	}
}

// FocusedField reports the current flat focus index. The root model compares it
// against IdentityFieldIndex to decide whether space/f opens the file picker.
func (f *EditForm) FocusedField() int { return f.focus }

// SetIdentity writes a path back into the IdentityFile field (used after the
// file picker returns).
func (f *EditForm) SetIdentity(path string) {
	f.inputs[IdentityFieldIndex].SetValue(path)
}

// SetWidth updates the layout width.
func (f *EditForm) SetWidth(w int) { f.width = w }

// Update handles navigation, env add/remove and text entry. Navigation:
// tab/down -> next, shift+tab/up -> previous, ctrl+a adds an env row, ctrl+d
// removes the focused env row.
func (f EditForm) Update(msg tea.Msg) (EditForm, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab", "down":
			f.setFocus(f.focus + 1)
			return f, nil
		case "shift+tab", "up":
			f.setFocus(f.focus - 1)
			return f, nil
		case "ctrl+a":
			f.envs = append(f.envs, newEnvInput("", ""))
			f.setFocus(numFixedFields + 2*(len(f.envs)-1))
			return f, nil
		case "ctrl+d":
			if f.focus >= numFixedFields {
				row := (f.focus - numFixedFields) / 2
				f.envs = append(f.envs[:row], f.envs[row+1:]...)
				f.setFocus(f.focus)
			}
			return f, nil
		}
	}

	var cmd tea.Cmd
	if f.focusCount() > 0 {
		*f.focusableAt(f.focus), cmd = f.focusableAt(f.focus).Update(msg)
	}
	return f, cmd
}

// Validate checks required fields. It returns "" when the form is valid.
func (f *EditForm) Validate() string {
	if strings.TrimSpace(f.inputs[fieldAlias].Value()) == "" {
		return "Alias is required"
	}
	if p := strings.TrimSpace(f.inputs[fieldPort].Value()); p != "" {
		if n, err := strconv.Atoi(p); err != nil || n <= 0 {
			return "Port must be a positive number"
		}
	}
	return ""
}

// Host builds a Host from the current field values, preserving fields that the
// form does not edit (Disabled, transient connect state).
func (f *EditForm) Host() ssh.Host {
	h := f.original
	h.Alias = strings.TrimSpace(f.inputs[fieldAlias].Value())
	h.Hostname = strings.TrimSpace(f.inputs[fieldHostname].Value())
	h.User = strings.TrimSpace(f.inputs[fieldUser].Value())
	h.IdentityFile = strings.TrimSpace(f.inputs[IdentityFieldIndex].Value())
	h.ProxyJump = strings.TrimSpace(f.inputs[fieldProxyJump].Value())

	h.Port = 22
	if n, err := strconv.Atoi(strings.TrimSpace(f.inputs[fieldPort].Value())); err == nil && n > 0 {
		h.Port = n
	}

	h.EnvVars = nil
	for _, e := range f.envs {
		name := strings.TrimSpace(e.name.Value())
		if name == "" {
			continue
		}
		h.EnvVars = append(h.EnvVars, ssh.EnvVar{Name: name, Value: e.value.Value()})
	}
	return h
}

func (f EditForm) View() string {
	s := f.styles
	var b strings.Builder

	title := "Edit host"
	if f.Adding {
		title = "Add host"
	}
	b.WriteString(s.Title.Render(title))
	b.WriteString("\n\n")

	for i := range f.inputs {
		b.WriteString(f.renderField(fixedLabels[i], &f.inputs[i], f.focus == i))
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(s.Label.Render("Environment (SetEnv)"))
	b.WriteByte('\n')
	for i := range f.envs {
		nameIdx := numFixedFields + 2*i
		valIdx := nameIdx + 1
		name := f.fieldBox(&f.envs[i].name, f.focus == nameIdx)
		val := f.fieldBox(&f.envs[i].value, f.focus == valIdx)
		row := lipgloss.JoinHorizontal(lipgloss.Top, name, s.Label.Render(" = "), val)
		b.WriteString("  " + row)
		b.WriteByte('\n')
	}
	b.WriteString(s.Label.Render("  ctrl+a add var · ctrl+d remove var"))
	b.WriteString("\n\n")

	if f.errMsg != "" {
		b.WriteString(s.StatusFailed.Render(f.errMsg))
		b.WriteByte('\n')
	}
	b.WriteString(s.Help.Render("tab/↑↓ move · enter save · esc cancel"))

	return b.String()
}

// SetError sets a validation message to show on the next render.
func (f *EditForm) SetError(msg string) { f.errMsg = msg }

func (f EditForm) renderField(label string, in *textinput.Model, active bool) string {
	s := f.styles
	lbl := s.FieldLabel
	if active {
		lbl = s.FieldActive
	}
	return lbl.Render(padRight(label, 14)) + f.fieldBox(in, active)
}

func (f EditForm) fieldBox(in *textinput.Model, active bool) string {
	box := f.styles.Input
	if active {
		box = f.styles.InputFocus
	}
	return box.Render(in.View())
}

func padRight(s string, n int) string {
	for lipgloss.Width(s) < n {
		s += " "
	}
	return s
}
