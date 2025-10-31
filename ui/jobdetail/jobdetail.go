package jobdetail

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

type Detail struct {
	JobID      uuid.UUID
	Command    string
	Status     string
	ReturnCode *int32
	Stdout     string
	Stderr     string
}

type Model struct {
	view viewport.Model
	data Detail
}

func New() Model {
	vp := viewport.New(0, 0)
	return Model{
		view: vp,
	}
}

func (m *Model) SetDetail(detail Detail) {
	m.data = detail
	m.refresh()
}

func (m *Model) refresh() {
	rc := "pending"
	if m.data.ReturnCode != nil {
		rc = fmt.Sprintf("%d", *m.data.ReturnCode)
	}

	content := fmt.Sprintf("Command: %s\nStatus: %s\nReturn Code: %s\n\nSTDOUT:\n%s\n\nSTDERR:\n%s", m.data.Command, m.data.Status, rc, m.data.Stdout, m.data.Stderr)
	m.view.SetContent(content)
	m.view.GotoBottom()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.view.Width = msg.Width
		m.view.Height = msg.Height
	}

	var cmd tea.Cmd
	m.view, cmd = m.view.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	return m.view.View()
}
