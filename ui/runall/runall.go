package runall

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var NotificationChan chan tea.Msg

// RunAllCommandMsg is emitted when the user confirms a command to run.
type RunAllCommandMsg struct {
	Command string
}

// ErrorMsg allows the parent model to surface validation feedback.
type ErrorMsg struct {
	Message string
}

// SuccessMsg allows the parent model to communicate completion status.
type SuccessMsg struct {
	Message string
}

type Model struct {
	input  textinput.Model
	status string
}

func New(notif chan tea.Msg) Model {
	NotificationChan = notif

	in := textinput.New()
	in.Placeholder = "Enter command to run on all connections"
	in.Focus()

	return Model{
		input: in,
	}
}

func (m *Model) Reset() {
	m.input.Reset()
	m.input.Focus()
	m.status = ""
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			command := strings.TrimSpace(m.input.Value())
			if command == "" {
				m.status = "Command cannot be empty"
				return m, nil
			}
			go func() {
				NotificationChan <- RunAllCommandMsg{Command: command}
			}()
			m.status = ""
			m.input.Reset()
			m.input.Focus()
			return m, nil
		case "tab":
			// single field form, ignore tab to keep focus
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.input.Width = msg.Width
	case ErrorMsg:
		m.status = msg.Message
		return m, nil
	case SuccessMsg:
		m.status = msg.Message
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	if m.status != "" {
		return m.input.View() + "\n" + m.status
	}
	return m.input.View()
}
