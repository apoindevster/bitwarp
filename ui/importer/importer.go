package importer

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

var NotificationChan chan tea.Msg

type ImportRequest struct {
	ConnectionID uuid.UUID
	Path         string
}

type StatusMsg struct {
	Message string
	IsError bool
}

type Model struct {
	connID   uuid.UUID
	connDesc string
	input    textinput.Model
	status   StatusMsg
}

func New(notif chan tea.Msg) Model {
	NotificationChan = notif

	ti := textinput.New()
	ti.Placeholder = "Path to command batch JSON"
	ti.Prompt = "> "
	ti.Focus()

	return Model{
		input: ti,
		status: StatusMsg{
			Message: "Enter path to command batch JSON file.",
			IsError: false,
		},
	}
}

func (m *Model) Reset() {
	m.input.Reset()
	m.input.Focus()
	m.status = StatusMsg{
		Message: "Enter path to command batch JSON file.",
		IsError: false,
	}
}

func (m *Model) SetConnection(id uuid.UUID, desc string) {
	m.connID = id
	m.connDesc = desc
}

func (m *Model) SetStatus(msg StatusMsg) {
	m.status = msg
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			path := strings.TrimSpace(m.input.Value())
			if path == "" {
				m.status = StatusMsg{Message: "Path cannot be empty.", IsError: true}
				return m, nil
			}
			if !filepath.IsAbs(path) {
				path = filepath.Clean(path)
			}
			go func() {
				NotificationChan <- ImportRequest{ConnectionID: m.connID, Path: path}
			}()
			return m, nil
		}
	case StatusMsg:
		m.status = msg
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	var statusLine string
	if m.status.IsError {
		statusLine = fmt.Sprintf("[error] %s", m.status.Message)
	} else {
		statusLine = m.status.Message
	}
	header := fmt.Sprintf("Import commands for connection: %s\n", m.connDesc)
	return header + m.input.View() + "\n" + statusLine
}
