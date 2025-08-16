package main

import (
	"fmt"
	"strconv"

	"github.com/apoindevster/bitwarp/proto"
	connlist "github.com/apoindevster/bitwarp/ui/connlist"
	newconn "github.com/apoindevster/bitwarp/ui/newconn"
	connshell "github.com/apoindevster/bitwarp/ui/shell"
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/grpc"
)

type Connection struct {
	con     *grpc.ClientConn
	comcon  *proto.CommandClient
	history []string
}

var clients []Connection
var Prog *tea.Program
var NotificationChan chan tea.Msg

type State int

const (
	Conns State = iota
	NewCon
	Shell
)

type Model struct {
	currMod State
	conns   connlist.Model
	newCon  newconn.Model
	shell   connshell.Model
}

func New(notif chan tea.Msg) Model {
	NotificationChan = notif

	connl := connlist.New(NotificationChan)
	nc := newconn.New(NotificationChan)
	sh := connshell.New(NotificationChan)

	return Model{
		currMod: Conns,
		conns:   connl,
		newCon:  nc,
		shell:   sh,
	}

}

func (m *Model) decrementPage() {
	switch m.currMod {
	default:
		m.currMod = Conns
	}
}

func (m *Model) updateAllModels(msg tea.Msg) tea.Cmd {
	var concmd, newcmd, shcmd tea.Cmd
	m.conns, concmd = m.conns.Update(msg)
	m.newCon, newcmd = m.newCon.Update(msg)
	m.shell, shcmd = m.shell.Update(msg)

	return tea.Batch(concmd, newcmd, shcmd)

}

func waitForResponse(sub chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

func (m Model) Init() tea.Cmd {
	return waitForResponse(NotificationChan)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// TODO: Need to check on our own custom messages and call return m, waitForResponse(NotifcationChan) if it is a custom one
	switch msg := msg.(type) {
	case connlist.NewConnReq:
		m.currMod = NewCon
		return m, waitForResponse(NotificationChan)
	case connlist.DelConnReq:
		clients[msg.Id].con.Close()
		clients = append(clients[:msg.Id], clients[msg.Id+1:]...)
		newconns, concmd := m.conns.Update(msg)
		m.conns = newconns
		return m, tea.Batch(
			concmd,
			waitForResponse(NotificationChan),
		)
	case connlist.InteractConnReq:
		if msg.Id > len(clients)-1 {
			return m, nil
		}

		// TODO This will also need to validate that the connection is still alive
		m.currMod = Shell
		m.shell.SetCon(clients[msg.Id].comcon, &clients[msg.Id].history)
		return m, waitForResponse(NotificationChan)
	case newconn.NewConnParams:
		m.currMod = Conns
		con, comcon, err := CreateNewConnection(msg)
		if err != nil {
			return m, waitForResponse(NotificationChan)
		}

		newCon := Connection{con: con, comcon: comcon, history: []string{}}
		clients = append(clients, newCon)
		newconns, concmd := m.conns.Update(connlist.NewConnReq{Item: connlist.Item{T: msg.Desc, Desc: msg.Ip + ":" + strconv.Itoa(msg.Port)}})
		m.conns = newconns
		return m, tea.Batch(
			concmd,
			waitForResponse(NotificationChan),
		)
	case connshell.RunExecutableUpdate:
		newshell, shcmd := m.shell.Update(msg)
		m.shell = newshell
		return m, tea.Batch(
			shcmd,
			waitForResponse(NotificationChan),
		)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m.decrementPage()
			return m, nil
		}
	case tea.WindowSizeMsg:
		cmd := m.updateAllModels(msg)
		return m, cmd
	}

	// TODO: Need to check for the escape key. If it is pressed, go back to the previous screen
	var cmd tea.Cmd
	switch m.currMod {
	case Conns:
		m.conns, cmd = m.conns.Update(msg)
	case NewCon:
		m.newCon, cmd = m.newCon.Update(msg)
	case Shell:
		m.shell, cmd = m.shell.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	switch m.currMod {
	case Conns:
		return m.conns.View()
	case NewCon:
		return m.newCon.View()
	case Shell:
		return m.shell.View()
	default:
		return m.conns.View()
	}
}

func main() {
	// TODO: Upgrade the commandclient library to have hooks into the underlying connection.GetState()
	// This will be needed if tracking the connection state is necessary
	notif := make(chan tea.Msg)

	Prog = tea.NewProgram(New(notif), tea.WithAltScreen())
	if _, err := Prog.Run(); err != nil {
		fmt.Printf("Failed to run tui interface with error: %v\n", err)
		return
	}
}
