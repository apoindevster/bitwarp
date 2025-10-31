package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/apoindevster/bitwarp/proto"
	commoncommands "github.com/apoindevster/bitwarp/ui/common/commands"
	connlist "github.com/apoindevster/bitwarp/ui/connlist"
	newconn "github.com/apoindevster/bitwarp/ui/newconn"
	runall "github.com/apoindevster/bitwarp/ui/runall"
	connshell "github.com/apoindevster/bitwarp/ui/shell"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type runAllStartMsg struct {
	ID      uuid.UUID
	Command string
}

type runAllOutputMsg struct {
	ID      uuid.UUID
	Output  string
	IsError bool
}

type runAllResultMsg struct {
	ID         uuid.UUID
	ReturnCode int32
}

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
	RunAll
)

// The model that contains the current state as well as all of the sub-models for the pages intended to be shown.
type Model struct {
	currMod State
	conns   connlist.Model
	newCon  newconn.Model
	runAll  runall.Model
	shell   connshell.Model
	active  uuid.UUID
}

// New function to return the ELM architecture model.
func New(notif chan tea.Msg) Model {
	NotificationChan = notif

	connl := connlist.New(NotificationChan)
	nc := newconn.New(NotificationChan)
	ra := runall.New(NotificationChan)
	sh := connshell.New(NotificationChan)

	return Model{
		currMod: Conns,
		conns:   connl,
		newCon:  nc,
		runAll:  ra,
		shell:   sh,
		active:  uuid.Nil,
	}

}

// Go to the previous page in the BitWarp application
func (m *Model) decrementPage() {
	switch m.currMod {
	case Shell:
		m.active = uuid.Nil
	default:
		m.currMod = Conns
		return
	}
	m.currMod = Conns
}

// Call the ELM Architecture update function for all the sub-models in this model
func (m *Model) updateAllModels(msg tea.Msg) tea.Cmd {
	var concmd, newcmd, runCmd, shcmd tea.Cmd
	m.conns, concmd = m.conns.Update(msg)
	m.newCon, newcmd = m.newCon.Update(msg)
	m.runAll, runCmd = m.runAll.Update(msg)
	m.shell, shcmd = m.shell.Update(msg)

	return tea.Batch(concmd, newcmd, runCmd, shcmd)

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
		if msg.Id > len(clients)-1 {
			return m, waitForResponse(NotificationChan)
		}
		removedID := clients[msg.Id].conid
		clients[msg.Id].con.Close()
		clients = append(clients[:msg.Id], clients[msg.Id+1:]...)
		newconns, concmd := m.conns.Update(msg)
		m.conns = newconns
		if m.active == removedID {
			m.active = uuid.Nil
			m.currMod = Conns
			m.shell.SetCon(nil, nil)
		}
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
		m.active = clients[msg.Id].conid
		return m, waitForResponse(NotificationChan)
	case connlist.RunAllConnReq:
		m.currMod = RunAll
		m.runAll.Reset()
		return m, waitForResponse(NotificationChan)
	case newconn.NewConnParams:
		m.currMod = Conns
		con, err := CreateNewConnection(msg)
		if err != nil {
			return m, waitForResponse(NotificationChan)
		}

		// We can go ahead and create the command client
		client := proto.NewCommandClient(con)

		newCon := Connection{conid: uuid.New(), con: con, comcon: &client, history: []string{}}
		clients = append(clients, newCon)
		newconns, concmd := m.conns.Update(connlist.NewConnReq{Item: connlist.Item{T: msg.Desc, Desc: msg.Ip + ":" + strconv.Itoa(msg.Port)}})
		m.conns = newconns
		return m, tea.Batch(
			concmd,
			waitForResponse(NotificationChan),
		)
	case runall.RunAllCommandMsg:
		if strings.TrimSpace(msg.Command) == "" {
			go func() {
				NotificationChan <- runall.ErrorMsg{Message: "Command cannot be empty"}
			}()
			return m, waitForResponse(NotificationChan)
		}
		if len(clients) == 0 {
			go func() {
				NotificationChan <- runall.ErrorMsg{Message: "No active connections available"}
			}()
			return m, waitForResponse(NotificationChan)
		}
		m.currMod = Conns
		go dispatchRunAll(msg.Command)
		return m, waitForResponse(NotificationChan)
	case runall.ErrorMsg:
		m.runAll, _ = m.runAll.Update(msg)
		return m, waitForResponse(NotificationChan)
	case runall.SuccessMsg:
		m.runAll, _ = m.runAll.Update(msg)
		return m, waitForResponse(NotificationChan)
	case runAllStartMsg:
		appendHistory(msg.ID, fmt.Sprintf("%s\n", msg.Command))
		if m.currMod == Shell && m.active == msg.ID {
			m.shell.Refresh()
		}
		return m, waitForResponse(NotificationChan)
	case runAllOutputMsg:
		if msg.IsError {
			appendHistory(msg.ID, msg.Output)
		} else {
			appendHistory(msg.ID, msg.Output)
		}
		if m.currMod == Shell && m.active == msg.ID {
			m.shell.Refresh()
		}
		return m, waitForResponse(NotificationChan)
	case runAllResultMsg:
		appendHistory(msg.ID, fmt.Sprintf("\nCommand finished with exit code %d\n", msg.ReturnCode))
		if m.currMod == Shell && m.active == msg.ID {
			m.shell.Refresh()
		}
		return m, waitForResponse(NotificationChan)
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
	case RunAll:
		m.runAll, cmd = m.runAll.Update(msg)
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
	case RunAll:
		return m.runAll.View()
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

func appendHistory(id uuid.UUID, entry string) {
	for idx := range clients {
		if clients[idx].conid == id {
			clients[idx].history = append(clients[idx].history, entry)
			return
		}
	}
}

func dispatchRunAll(rawCommand string) {
	cmd, args := splitCommandInput(rawCommand)
	for _, conn := range clients {
		conn := conn
		go runCommandForConnection(conn, rawCommand, cmd, args)
	}
}

func runCommandForConnection(conn Connection, rawCommand, command, args string) {
	if conn.comcon == nil {
		NotificationChan <- runAllOutputMsg{ID: conn.conid, Output: "Connection not ready\n", IsError: true}
		return
	}

	if command == "" {
		NotificationChan <- runAllOutputMsg{ID: conn.conid, Output: "Invalid command\n", IsError: true}
		return
	}

	NotificationChan <- runAllStartMsg{ID: conn.conid, Command: rawCommand}

	_, err := commoncommands.ExecuteCommand(command, args, conn.comcon, commoncommands.Callbacks{
		Stdout: func(data []byte) {
			NotificationChan <- runAllOutputMsg{ID: conn.conid, Output: string(data)}
		},
		Stderr: func(data []byte) {
			NotificationChan <- runAllOutputMsg{ID: conn.conid, Output: string(data), IsError: true}
		},
		Complete: func(code int32) {
			NotificationChan <- runAllResultMsg{ID: conn.conid, ReturnCode: code}
		},
	})
	if err != nil {
		NotificationChan <- runAllOutputMsg{ID: conn.conid, Output: err.Error() + "\n", IsError: true}
		NotificationChan <- runAllResultMsg{ID: conn.conid, ReturnCode: -1}
	}
}

func splitCommandInput(input string) (string, string) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", ""
	}
	command, args, found := strings.Cut(trimmed, " ")
	if !found {
		return trimmed, ""
	}
	return command, strings.TrimSpace(args)
}
