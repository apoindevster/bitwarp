package shell

import (
	"strings"

	"github.com/apoindevster/bitwarp/proto"
	commoncommands "github.com/apoindevster/bitwarp/ui/common/commands"
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

type RunExecutableUpdate struct {
	appstring string
}

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
	m.Refresh()
}

func (m *Model) Refresh() {
	if m.history == nil {
		m.viewPort.SetContent("")
		return
	}
	m.viewPort.SetContent(strings.Join(*m.history, ""))
	m.viewPort.GotoBottom()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			input := strings.TrimSpace(m.textInput.Value())
			if input != "" {
				if m.Conn == nil {
					NotificationChan <- RunExecutableUpdate{appstring: "Connection not ready\n"}
				} else {
					cmd, args := splitCommandInput(input)
					go func(command, argLine string, client *proto.CommandClient) {
						_, err := commoncommands.ExecuteCommand(command, argLine, client, commoncommands.Callbacks{
							Stdout: func(data []byte) {
								NotificationChan <- RunExecutableUpdate{appstring: string(data)}
							},
							Stderr: func(data []byte) {
								NotificationChan <- RunExecutableUpdate{appstring: string(data)}
							},
						})
						if err != nil {
							NotificationChan <- RunExecutableUpdate{appstring: err.Error() + "\n"}
						}
					}(cmd, args, m.Conn)
				}
				*m.history = append(*m.history, input+"\n")
			}
			m.Refresh()
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
		m.Refresh()
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

func splitCommandInput(input string) (string, string) {
	command, args, found := strings.Cut(input, " ")
	if !found {
		return input, ""
	}
	return command, strings.TrimSpace(args)
}
