package shell

import (
	"strings"

	"github.com/apoindevster/bitwarp/proto"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	viewPort  viewport.Model
	textInput textinput.Model
	Conn      *proto.CommandClient
	err       error
	history   *[]string
}

var NotificationChan chan tea.Msg

func New(notif chan tea.Msg) Model {
	vp := viewport.New(0, 0)

	ti := textinput.New()
	ti.Placeholder = "Command"
	ti.Focus()
	ti.Width = 0

	NotificationChan = notif

	return Model{
		viewPort:  vp,
		textInput: ti,
		Conn:      nil,
		err:       nil,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) SetCon(conn *proto.CommandClient, history *[]string) {
	m.Conn = conn
	m.history = history
	m.viewPort.SetContent(strings.Join(*m.history, "\n"))
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			command, args, found := strings.Cut(m.textInput.Value(), " ")
			if found {
				// TODO: Might want to check to make sure that m.conn is not nil
				go ExecuteCommand(command, args, m.Conn)
			} else if m.textInput.Value() != "" {
				// TODO: Might want to check to make sure that m.conn is not nil
				go ExecuteCommand(m.textInput.Value(), "", m.Conn)
			}
			*m.history = append(*m.history, m.textInput.Value()+"\n")
			m.viewPort.SetContent(strings.Join(*m.history, "\n"))
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
		// The goroutine that executes the commands passes this message type back to the app so we can display it here.
		*m.history = append(*m.history, msg.appstring)
		m.viewPort.SetContent(strings.Join(*m.history, ""))
		m.viewPort.GotoBottom()
		return m, nil
	case error:
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
