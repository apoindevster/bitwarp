package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apoindevster/bitwarp/proto"
	commoncommands "github.com/apoindevster/bitwarp/ui/common/commands"
	jobmsgs "github.com/apoindevster/bitwarp/ui/common/jobs"
	connlist "github.com/apoindevster/bitwarp/ui/connlist"
	cmdimporter "github.com/apoindevster/bitwarp/ui/importer"
	jobdetail "github.com/apoindevster/bitwarp/ui/jobdetail"
	joblist "github.com/apoindevster/bitwarp/ui/joblist"
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

type JobStatus string

const (
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
)

type Job struct {
	ID            uuid.UUID
	ConnectionID  uuid.UUID
	Command       string
	Source        jobmsgs.Source
	Stdout        []string
	Stderr        []string
	ReturnCode    *int32
	StartedAt     time.Time
	CompletedAt   *time.Time
	LastUpdatedAt time.Time
	Cancel        context.CancelFunc
}

type commandFuture struct {
	once sync.Once
	code int32
	ch   chan int32
}

func newCommandFuture() *commandFuture {
	return &commandFuture{ch: make(chan int32, 1)}
}

func (f *commandFuture) resolve(code int32) {
	f.ch <- code
}

func (f *commandFuture) Wait() int32 {
	f.once.Do(func() {
		f.code = <-f.ch
	})
	return f.code
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
	Import
	Jobs
	JobDetail
)

// The model that contains the current state as well as all of the sub-models for the pages intended to be shown.
type Model struct {
	currMod          State
	conns            connlist.Model
	newCon           newconn.Model
	runAll           runall.Model
	shell            connshell.Model
	importer         cmdimporter.Model
	jobList          joblist.Model
	jobDet           jobdetail.Model
	active           uuid.UUID
	jobsByConnection map[uuid.UUID][]*Job
	jobIndex         map[uuid.UUID]*Job
	selectedJob      uuid.UUID
}

// New function to return the ELM architecture model.
func New(notif chan tea.Msg) Model {
	NotificationChan = notif

	connl := connlist.New(NotificationChan)
	nc := newconn.New(NotificationChan)
	ra := runall.New(NotificationChan)
	jl := joblist.New(NotificationChan)
	jd := jobdetail.New()
	sh := connshell.New(NotificationChan)
	imp := cmdimporter.New(NotificationChan)

	return Model{
		currMod:          Conns,
		conns:            connl,
		newCon:           nc,
		runAll:           ra,
		shell:            sh,
		importer:         imp,
		jobList:          jl,
		jobDet:           jd,
		active:           uuid.Nil,
		jobsByConnection: make(map[uuid.UUID][]*Job),
		jobIndex:         make(map[uuid.UUID]*Job),
	}

}

// Go to the previous page in the BitWarp application
func (m *Model) decrementPage() {
	switch m.currMod {
	case Shell:
		m.active = uuid.Nil
		m.currMod = Conns
	case RunAll:
		m.currMod = Conns
	case Import:
		m.currMod = Conns
	case Jobs:
		m.currMod = Conns
	case JobDetail:
		m.currMod = Jobs
		m.selectedJob = uuid.Nil
	default:
		m.currMod = Conns
	}
}

// Call the ELM Architecture update function for all the sub-models in this model
func (m *Model) updateAllModels(msg tea.Msg) tea.Cmd {
	var concmd, newcmd, runCmd, importCmd, jobCmd, detCmd, shcmd tea.Cmd
	m.conns, concmd = m.conns.Update(msg)
	m.newCon, newcmd = m.newCon.Update(msg)
	m.runAll, runCmd = m.runAll.Update(msg)
	m.importer, importCmd = m.importer.Update(msg)
	m.jobList, jobCmd = m.jobList.Update(msg)
	m.jobDet, detCmd = m.jobDet.Update(msg)
	m.shell, shcmd = m.shell.Update(msg)

	return tea.Batch(concmd, newcmd, runCmd, importCmd, jobCmd, detCmd, shcmd)

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
		m.cancelJobsForConnection(removedID)
		clients[msg.Id].con.Close()
		clients = append(clients[:msg.Id], clients[msg.Id+1:]...)
		newconns, concmd := m.conns.Update(msg)
		m.conns = newconns
		if m.active == removedID {
			m.active = uuid.Nil
			m.currMod = Conns
			m.shell.SetCon(uuid.Nil, nil, nil)
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
		m.shell.SetCon(clients[msg.Id].conid, clients[msg.Id].comcon, &clients[msg.Id].history)
		m.active = clients[msg.Id].conid
		return m, waitForResponse(NotificationChan)
	case connlist.ShowJobsReq:
		if msg.Id > len(clients)-1 {
			return m, waitForResponse(NotificationChan)
		}
		conn := clients[msg.Id]
		m.active = conn.conid
		m.currMod = Jobs
		m.jobList.SetConnection(conn.conid)
		m.jobList.SetJobs(buildJobListItems(m.jobsByConnection[conn.conid]))
		return m, waitForResponse(NotificationChan)
	case connlist.ImportCommandsReq:
		if msg.Id > len(clients)-1 {
			return m, waitForResponse(NotificationChan)
		}
		conn := clients[msg.Id]
		m.active = conn.conid
		m.currMod = Import
		target := "<disconnected>"
		if conn.con != nil {
			target = conn.con.Target()
		}
		desc := fmt.Sprintf("%s (%s)", conn.conid.String(), target)
		m.importer.SetConnection(conn.conid, desc)
		m.importer.Reset()
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
	case joblist.JobSelected:
		if job, ok := m.jobIndex[msg.JobID]; ok {
			m.selectedJob = job.ID
			m.currMod = JobDetail
			m.jobDet.SetDetail(buildJobDetail(job))
		}
		return m, waitForResponse(NotificationChan)
	case joblist.JobCancelRequest:
		m.cancelJob(msg.JobID)
		return m, waitForResponse(NotificationChan)
	case jobmsgs.StartedMsg:
		m.handleJobStarted(msg)
		return m, waitForResponse(NotificationChan)
	case jobmsgs.OutputMsg:
		m.handleJobOutput(msg)
		return m, waitForResponse(NotificationChan)
	case jobmsgs.CompletedMsg:
		m.handleJobCompleted(msg)
		return m, waitForResponse(NotificationChan)
	case cmdimporter.ImportRequest:
		conn := m.findConnection(msg.ConnectionID)
		if conn == nil {
			go func() {
				NotificationChan <- cmdimporter.StatusMsg{Message: "Connection not found or unavailable", IsError: true}
			}()
			return m, waitForResponse(NotificationChan)
		}
		batch, err := cmdimporter.LoadCommandBatch(msg.Path)
		if err != nil {
			go func() {
				NotificationChan <- cmdimporter.StatusMsg{Message: err.Error(), IsError: true}
			}()
			return m, waitForResponse(NotificationChan)
		}
		m.startCommandBatch(*conn, batch)
		go func(count int) {
			NotificationChan <- cmdimporter.StatusMsg{Message: fmt.Sprintf("Started import with %d commands", count), IsError: false}
		}(len(batch.Commands))
		m.currMod = Jobs
		m.jobList.SetConnection(conn.conid)
		m.jobList.SetJobs(buildJobListItems(m.jobsByConnection[conn.conid]))
		return m, waitForResponse(NotificationChan)
	case cmdimporter.StatusMsg:
		m.importer, _ = m.importer.Update(msg)
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
	case Import:
		m.importer, cmd = m.importer.Update(msg)
	case Jobs:
		m.jobList, cmd = m.jobList.Update(msg)
	case JobDetail:
		m.jobDet, cmd = m.jobDet.Update(msg)
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
	case Import:
		return m.importer.View()
	case Jobs:
		return m.jobList.View()
	case JobDetail:
		return m.jobDet.View()
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

func (m *Model) handleJobStarted(msg jobmsgs.StartedMsg) {
	job := &Job{
		ID:            msg.JobID,
		ConnectionID:  msg.ConnectionID,
		Command:       msg.Command,
		Source:        msg.Source,
		Stdout:        []string{},
		Stderr:        []string{},
		StartedAt:     time.Now(),
		LastUpdatedAt: time.Now(),
		Cancel:        msg.Cancel,
	}

	m.jobsByConnection[msg.ConnectionID] = append(m.jobsByConnection[msg.ConnectionID], job)
	m.jobIndex[msg.JobID] = job

	if !m.connectionExists(msg.ConnectionID) && job.Cancel != nil {
		job.Cancel()
		job.Cancel = nil
	}

	if m.currMod == Jobs && m.active == msg.ConnectionID {
		m.jobList.Upsert(jobToListItem(job))
	}

	if m.currMod == JobDetail && m.selectedJob == msg.JobID {
		m.jobDet.SetDetail(buildJobDetail(job))
	}
}

func (m *Model) handleJobOutput(msg jobmsgs.OutputMsg) {
	job, ok := m.jobIndex[msg.JobID]
	if !ok {
		return
	}

	switch msg.Stream {
	case jobmsgs.StreamStdout:
		job.Stdout = append(job.Stdout, msg.Data)
	case jobmsgs.StreamStderr:
		job.Stderr = append(job.Stderr, msg.Data)
	}
	job.LastUpdatedAt = time.Now()

	if m.currMod == Jobs && m.active == msg.ConnectionID {
		m.jobList.Upsert(jobToListItem(job))
	}

	if m.currMod == JobDetail && m.selectedJob == msg.JobID {
		m.jobDet.SetDetail(buildJobDetail(job))
	}
}

func (m *Model) handleJobCompleted(msg jobmsgs.CompletedMsg) {
	job, ok := m.jobIndex[msg.JobID]
	if !ok {
		return
	}

	job.ReturnCode = &msg.ReturnCode
	now := time.Now()
	job.CompletedAt = &now
	job.LastUpdatedAt = now
	job.Cancel = nil

	if m.currMod == Jobs && m.active == msg.ConnectionID {
		m.jobList.Upsert(jobToListItem(job))
	}

	if m.currMod == JobDetail && m.selectedJob == msg.JobID {
		m.jobDet.SetDetail(buildJobDetail(job))
	}
}

func jobStatusString(job *Job) string {
	if job.ReturnCode == nil {
		return string(JobStatusRunning)
	}
	if *job.ReturnCode == -2 {
		return "cancelled"
	}
	if *job.ReturnCode == -3 {
		return "skipped"
	}
	return fmt.Sprintf("%s (%d)", JobStatusCompleted, *job.ReturnCode)
}

func buildJobListItems(jobs []*Job) []joblist.Item {
	items := make([]joblist.Item, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, jobToListItem(job))
	}
	return items
}

func jobToListItem(job *Job) joblist.Item {
	return joblist.Item{
		JobID:        job.ID,
		ConnectionID: job.ConnectionID,
		Command:      job.Command,
		Status:       jobStatusString(job),
		LastUpdated:  job.LastUpdatedAt,
		ReturnCode:   job.ReturnCode,
	}
}

func buildJobDetail(job *Job) jobdetail.Detail {
	return jobdetail.Detail{
		JobID:      job.ID,
		Command:    job.Command,
		Status:     jobStatusString(job),
		ReturnCode: job.ReturnCode,
		Stdout:     strings.Join(job.Stdout, ""),
		Stderr:     strings.Join(job.Stderr, ""),
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

func (m *Model) startCommandBatch(conn Connection, batch *cmdimporter.CommandBatch) {
	if conn.comcon == nil {
		go func() {
			NotificationChan <- cmdimporter.StatusMsg{Message: "Connection not ready for import", IsError: true}
		}()
		return
	}
	go func() {
		var (
			prevFuture *commandFuture
			prevCmd    *cmdimporter.CommandDefinition
		)
		for idx := range batch.Commands {
			cmd := &batch.Commands[idx]

			if prevCmd != nil && !prevCmd.Async && prevFuture != nil {
				prevCode := prevFuture.Wait()
				if !prevCmd.Expect.Evaluate(prevCode) {
					m.skipCommand(conn, *cmd, prevCmd.Name, prevCode)
					return
				}
			}

			jobID := uuid.New()
			ctx, cancel := context.WithCancel(context.Background())
			if cmd.TimeoutSeconds != nil && *cmd.TimeoutSeconds > 0 {
				timeoutCtx, timeoutCancel := context.WithTimeout(ctx, time.Duration(*cmd.TimeoutSeconds)*time.Second)
				ctx = timeoutCtx
				origCancel := cancel
				cancel = func() {
					origCancel()
					timeoutCancel()
				}
			}

			NotificationChan <- jobmsgs.StartedMsg{JobID: jobID, ConnectionID: conn.conid, Command: cmd.Name, Source: jobmsgs.SourceImport, Cancel: cancel}
			commandLine := formatExecCommand(cmd.Exec)
			appendHistory(conn.conid, fmt.Sprintf("[import] %s\n", commandLine))
			NotificationChan <- jobmsgs.OutputMsg{JobID: jobID, ConnectionID: conn.conid, Data: "$ " + commandLine + "\n", Stream: jobmsgs.StreamStdout}

			future := newCommandFuture()
			cmdCopy := *cmd
			run := func() {
				code := m.executeImportCommand(ctx, cancel, conn, jobID, cmdCopy)
				future.resolve(code)
			}

			if cmd.Async {
				go run()
				prevFuture = nil
				prevCmd = nil
			} else {
				run()
				prevFuture = future
				prevCmd = cmd
			}
		}
		NotificationChan <- cmdimporter.StatusMsg{Message: "Import sequencing finished", IsError: false}
	}()
}

func (m *Model) executeImportCommand(ctx context.Context, cancel context.CancelFunc, conn Connection, jobID uuid.UUID, cmd cmdimporter.CommandDefinition) int32 {
	defer cancel()
	argLine := buildArgLine(cmd.Exec)
	stdoutCb := func(data []byte) {
		NotificationChan <- jobmsgs.OutputMsg{JobID: jobID, ConnectionID: conn.conid, Data: string(data), Stream: jobmsgs.StreamStdout}
	}
	stderrCb := func(data []byte) {
		NotificationChan <- jobmsgs.OutputMsg{JobID: jobID, ConnectionID: conn.conid, Data: string(data), Stream: jobmsgs.StreamStderr}
	}
	retCode, err := commoncommands.ExecuteCommand(ctx, "exec", argLine, conn.comcon, commoncommands.Callbacks{
		Stdout: stdoutCb,
		Stderr: stderrCb,
	})
	var finalCode int32
	if err != nil {
		if errors.Is(err, context.Canceled) {
			finalCode = -2
			NotificationChan <- jobmsgs.OutputMsg{JobID: jobID, ConnectionID: conn.conid, Data: "Command cancelled\n", Stream: jobmsgs.StreamStderr}
		} else {
			finalCode = -1
			NotificationChan <- jobmsgs.OutputMsg{JobID: jobID, ConnectionID: conn.conid, Data: err.Error() + "\n", Stream: jobmsgs.StreamStderr}
		}
	} else {
		finalCode = retCode
	}
	NotificationChan <- jobmsgs.CompletedMsg{JobID: jobID, ConnectionID: conn.conid, ReturnCode: finalCode}
	return finalCode
}

func (m *Model) skipCommand(conn Connection, cmd cmdimporter.CommandDefinition, prevName string, prevReturn int32) {
	jobID := uuid.New()
	NotificationChan <- jobmsgs.StartedMsg{JobID: jobID, ConnectionID: conn.conid, Command: cmd.Name, Source: jobmsgs.SourceImport, Cancel: nil}
	if prevName == "" {
		prevName = "previous command"
	}
	msg := fmt.Sprintf("Skipping command %s; %s exit code %d did not satisfy expectation.\n", cmd.Name, prevName, prevReturn)
	NotificationChan <- jobmsgs.OutputMsg{JobID: jobID, ConnectionID: conn.conid, Data: msg, Stream: jobmsgs.StreamStderr}
	NotificationChan <- jobmsgs.CompletedMsg{JobID: jobID, ConnectionID: conn.conid, ReturnCode: -3}
	appendHistory(conn.conid, fmt.Sprintf("[import skip] %s (prev rc %d)\n", cmd.Name, prevReturn))
	go func() {
		NotificationChan <- cmdimporter.StatusMsg{Message: fmt.Sprintf("Import halted at %s due to expectation failure", prevName), IsError: true}
	}()
}

func buildArgLine(exec cmdimporter.ExecSpec) string {
	parts := []string{exec.Command}
	for _, arg := range exec.Args {
		if strings.ContainsAny(arg, " \t\"") {
			parts = append(parts, strconv.Quote(arg))
		} else {
			parts = append(parts, arg)
		}
	}
	return strings.Join(parts, " ")
}

func formatExecCommand(exec cmdimporter.ExecSpec) string {
	return buildArgLine(exec)
}

func (m *Model) findConnection(id uuid.UUID) *Connection {
	for idx := range clients {
		if clients[idx].conid == id {
			return &clients[idx]
		}
	}
	return nil
}

func dispatchRunAll(rawCommand string) {
	cmd, args := splitCommandInput(rawCommand)
	for _, conn := range clients {
		conn := conn
		jobID := uuid.New()
		ctx, cancel := context.WithCancel(context.Background())
		NotificationChan <- jobmsgs.StartedMsg{JobID: jobID, ConnectionID: conn.conid, Command: rawCommand, Source: jobmsgs.SourceRunAll, Cancel: cancel}
		go runCommandForConnection(conn, rawCommand, cmd, args, jobID, ctx, cancel)
	}
}

func runCommandForConnection(conn Connection, rawCommand, command, args string, jobID uuid.UUID, ctx context.Context, cancel context.CancelFunc) {
	defer cancel()
	if conn.comcon == nil {
		NotificationChan <- runAllOutputMsg{ID: conn.conid, Output: "Connection not ready\n", IsError: true}
		NotificationChan <- jobmsgs.OutputMsg{JobID: jobID, ConnectionID: conn.conid, Data: "Connection not ready\n", Stream: jobmsgs.StreamStderr}
		NotificationChan <- jobmsgs.CompletedMsg{JobID: jobID, ConnectionID: conn.conid, ReturnCode: -1}
		NotificationChan <- runAllResultMsg{ID: conn.conid, ReturnCode: -1}
		return
	}

	if command == "" {
		NotificationChan <- runAllOutputMsg{ID: conn.conid, Output: "Invalid command\n", IsError: true}
		NotificationChan <- jobmsgs.OutputMsg{JobID: jobID, ConnectionID: conn.conid, Data: "Invalid command\n", Stream: jobmsgs.StreamStderr}
		NotificationChan <- jobmsgs.CompletedMsg{JobID: jobID, ConnectionID: conn.conid, ReturnCode: -1}
		NotificationChan <- runAllResultMsg{ID: conn.conid, ReturnCode: -1}
		return
	}

	NotificationChan <- runAllStartMsg{ID: conn.conid, Command: rawCommand}

	retCode, err := commoncommands.ExecuteCommand(ctx, command, args, conn.comcon, commoncommands.Callbacks{
		Stdout: func(data []byte) {
			NotificationChan <- runAllOutputMsg{ID: conn.conid, Output: string(data)}
			NotificationChan <- jobmsgs.OutputMsg{JobID: jobID, ConnectionID: conn.conid, Data: string(data), Stream: jobmsgs.StreamStdout}
		},
		Stderr: func(data []byte) {
			NotificationChan <- runAllOutputMsg{ID: conn.conid, Output: string(data), IsError: true}
			NotificationChan <- jobmsgs.OutputMsg{JobID: jobID, ConnectionID: conn.conid, Data: string(data), Stream: jobmsgs.StreamStderr}
		},
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			msg := "Command cancelled\n"
			NotificationChan <- runAllOutputMsg{ID: conn.conid, Output: msg, IsError: true}
			NotificationChan <- jobmsgs.OutputMsg{JobID: jobID, ConnectionID: conn.conid, Data: msg, Stream: jobmsgs.StreamStderr}
			NotificationChan <- jobmsgs.CompletedMsg{JobID: jobID, ConnectionID: conn.conid, ReturnCode: -2}
			NotificationChan <- runAllResultMsg{ID: conn.conid, ReturnCode: -2}
			return
		}
		NotificationChan <- runAllOutputMsg{ID: conn.conid, Output: err.Error() + "\n", IsError: true}
		NotificationChan <- jobmsgs.OutputMsg{JobID: jobID, ConnectionID: conn.conid, Data: err.Error() + "\n", Stream: jobmsgs.StreamStderr}
		NotificationChan <- jobmsgs.CompletedMsg{JobID: jobID, ConnectionID: conn.conid, ReturnCode: -1}
		NotificationChan <- runAllResultMsg{ID: conn.conid, ReturnCode: -1}
		return
	}

	NotificationChan <- jobmsgs.CompletedMsg{JobID: jobID, ConnectionID: conn.conid, ReturnCode: retCode}
	NotificationChan <- runAllResultMsg{ID: conn.conid, ReturnCode: retCode}
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

func (m *Model) cancelJobsForConnection(connID uuid.UUID) {
	jobs := m.jobsByConnection[connID]
	for _, job := range jobs {
		if job.ReturnCode == nil && job.Cancel != nil {
			job.Cancel()
			job.Cancel = nil
		}
	}
}

func (m *Model) cancelJob(jobID uuid.UUID) {
	job, ok := m.jobIndex[jobID]
	if !ok || job.ReturnCode != nil || job.Cancel == nil {
		return
	}
	job.Cancel()
	job.Cancel = nil
}

func (m *Model) connectionExists(id uuid.UUID) bool {
	for _, conn := range clients {
		if conn.conid == id {
			return true
		}
	}
	return false
}
