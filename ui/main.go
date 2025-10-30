package main

import (
	"fmt"
	"strconv"

	"github.com/apoindevster/bitwarp/proto"
	connlist "github.com/apoindevster/bitwarp/ui/connlist"
	newconn "github.com/apoindevster/bitwarp/ui/newconn"
	connshell "github.com/apoindevster/bitwarp/ui/shell"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// Global but keeps track of all the connection in the client list.
// TODO: Find a better way to track con and comcon simultaneously
type Connection struct {
	conid   uuid.UUID
	con     *grpc.ClientConn
	comcon  *proto.CommandClient
	history []string
}

var clients []Connection
var Prog *tea.Program
var NotificationChan chan tea.Msg

type State int

// The "enum" to track state for the internal state machine
const (
	Conns State = iota
	NewCon
	Shell
)

// The model that contains the current state as well as all of the sub-models for the pages intended to be shown.
type Model struct {
	currMod State
	conns   connlist.Model
	newCon  newconn.Model
	shell   connshell.Model
}

// New function to return the ELM architecture model.
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

// Go to the previous page in the BitWarp application
func (m *Model) decrementPage() {
	switch m.currMod {
	default:
		m.currMod = Conns
	}
}

// Call the ELM Architecture update function for all the sub-models in this model
func (m *Model) updateAllModels(msg tea.Msg) tea.Cmd {
	var concmd, newcmd, shcmd tea.Cmd
	m.conns, concmd = m.conns.Update(msg)
	m.newCon, newcmd = m.newCon.Update(msg)
	m.shell, shcmd = m.shell.Update(msg)

	return tea.Batch(concmd, newcmd, shcmd)

}

// This function serves as a way to pass custom tea.Msg types between the models. This also makes it much more event driven.
func waitForResponse(sub chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

// This implementation of the BubbleTea interface is currently used as an interface to the state machine that is all the various pages in the BitWarp client.
// See the various modules that also implement the BubbleTea interface for more information on the business logic for the individual pages.
func (m Model) Init() tea.Cmd {
	return waitForResponse(NotificationChan)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// The following messages are messages that are custom BitWarp tea.Msg messages. They come from the returned function from waitForResponse and allow
	// for an event driven architecture.
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

		if clients[msg.Id].con.GetState() == connectivity.Shutdown {
			// Connection has been or is shutting down.
			return m, waitForResponse(NotificationChan)
		}

		m.currMod = Shell
		m.shell.SetCon(clients[msg.Id].comcon, &clients[msg.Id].history)
		return m, waitForResponse(NotificationChan)
	case newconn.NewConnParams:
		m.currMod = Conns
		con, err := CreateNewConnection(msg)
		if err != nil {
			return m, waitForResponse(NotificationChan)
		}

		// We can go ahead and create the command client
		client := proto.NewCommandClient(con)

		newCon := Connection{con: con, comcon: &client, history: []string{}}
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
	// The following are built-in tea messages from BubbleTea.
	case tea.KeyMsg:
		switch msg.Type {
		// Allow it to go back to the previous page/state.
		case tea.KeyEscape:
			m.decrementPage()
			return m, nil
		}
	case tea.WindowSizeMsg:
		// Send this window size message to all of the possible pages you can be in so that when you switch between BitWarp windows, the sizes will be correct.
		cmd := m.updateAllModels(msg)
		return m, cmd
	}

	// Any other keys or message types should be funneled down to the currently active page.
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
	// Conditional UI based upon the current view in the state machine.
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
	notif := make(chan tea.Msg)

	Prog = tea.NewProgram(New(notif), tea.WithAltScreen())
	if _, err := Prog.Run(); err != nil {
		fmt.Printf("Failed to run tui interface with error: %v\n", err)
		return
	}
}
