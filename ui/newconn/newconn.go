package newconn

import (
	"errors"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var NotificationChan chan tea.Msg

// The following Types are the possible custom tea.Msg types
// objects of these types get propagated back up to NotificationChan
type NewConnParams struct {
	Desc string
	Ip   string
	Port int
}

// End

type Focus int

const (
	Desc Focus = iota
	Ip
	Port
	Max
)

type Model struct {
	focus Focus
	desc  textinput.Model
	ip    textinput.Model
	port  textinput.Model
}

func ValidateParams(ip string, port int) error {
	if ip == "" {
		return errors.New("invalid address")
	}

	if port <= 0 || port > 65535 {
		return errors.New("invalid port number")
	}

	return nil
}

func SendNewConnection(desc string, ip string, port string) {
	p, err := strconv.Atoi(port)
	if err != nil {
		// Failed to parse the port
		// TODO: Error case emit an error instead of empty NewConnParam
		NotificationChan <- NewConnParams{}
		return
	}

	err = ValidateParams(ip, p)
	if err != nil {
		// Invalid ip or port
		// TODO: Error case emit an error instead of empty NewConnParam
		NotificationChan <- NewConnParams{}
		return
	}

	NotificationChan <- NewConnParams{Desc: desc, Ip: ip, Port: p}
}

func New(notif chan tea.Msg) Model {
	NotificationChan = notif

	d := textinput.New()
	i := textinput.New()
	p := textinput.New()

	d.Focus()

	return Model{
		focus: Desc,
		desc:  d,
		ip:    i,
		port:  p,
	}
}

func (m *Model) IncFocus() tea.Cmd {
	switch m.focus {
	case Desc:
		m.desc.Blur()
		m.focus = Ip
		return m.ip.Focus()
	case Ip:
		m.ip.Blur()
		m.focus = Port
		return m.port.Focus()
	case Port:
		m.port.Blur()
		m.focus = Desc
		return m.desc.Focus()
	}
	m.focus = Desc
	return m.desc.Focus()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			go SendNewConnection(m.desc.Value(), m.ip.Value(), m.port.Value())
			// TODO: Implement more elegant way of doing this
			m.desc.Reset()
			m.ip.Reset()
			m.port.Reset()
			m.focus = Desc
			m.desc.Focus()
			m.ip.Blur()
			m.port.Blur()
			return m, nil
		case "tab":
			return m, m.IncFocus()
		}
	case tea.WindowSizeMsg:
		m.desc.Width = msg.Width
		m.ip.Width = msg.Width
		m.port.Width = msg.Width
	}

	var dcmd, icmd, pcmd tea.Cmd
	m.desc, dcmd = m.desc.Update(msg)
	m.ip, icmd = m.ip.Update(msg)
	m.port, pcmd = m.port.Update(msg)

	return m, tea.Batch(
		dcmd,
		icmd,
		pcmd,
	)
}

// TODO: Could update this so that the fields are much nicer looking instead of just using an input box
func (m Model) View() string {
	return "Description " + m.desc.View() + "\nIP " + m.ip.View() + "\nPort " + m.port.View()
}
