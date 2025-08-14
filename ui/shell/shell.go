package shell

import (
	"fmt"
	"strings"

	"github.com/apoindevster/bitwarp/proto"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var NotificationChan chan tea.Msg

type Model struct {
	viewPort  viewport.Model
	textInput textinput.Model
	conn      *proto.CommandClient
	err       error
	history   []string
}

func New(client *proto.CommandClient, notif chan tea.Msg) Model {
	vp := viewport.New(0, 0)

	ti := textinput.New()
	ti.Placeholder = "Command"
	ti.Focus()
	ti.Width = 0

	NotificationChan = notif

	return Model{
		viewPort:  vp,
		textInput: ti,
		conn:      client,
		err:       nil,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		waitForResponse(NotificationChan),
	)
}

func waitForResponse(sub chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

// Might want to check out the following for trying to get tea.Cmd
// https://github.com/charmbracelet/bubbletea/tree/main/tutorials/commands
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			command, args, found := strings.Cut(m.textInput.Value(), " ")
			if found {
				go ExecuteCommand(command, args, m.conn)
			} else if m.textInput.Value() != "" {
				go ExecuteCommand(m.textInput.Value(), "", m.conn)
			}
			m.history = append(m.history, m.textInput.Value()+"\n")
			m.viewPort.SetContent(strings.Join(m.history, "\n"))
			m.textInput.Reset()
			return m, nil
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.viewPort.Width = msg.Width
		m.viewPort.Height = msg.Height - lipgloss.Height(m.textInput.View())
		m.textInput.Width = msg.Width
	case RunExecutableUpdate:
		m.history = append(m.history, msg.appstring)
		m.viewPort.SetContent(strings.Join(m.history, ""))
		m.viewPort.GotoBottom()
		return m, waitForResponse(NotificationChan)
	case error:
		fmt.Println("Got error message")
		m.err = msg
		return m, nil
	}
	m.viewPort, cmd = m.viewPort.Update(msg)
	cmds = append(cmds, cmd)
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.viewPort.View() + "\n" + m.textInput.View()
}
