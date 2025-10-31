package joblist

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

var NotificationChan chan tea.Msg

type keyMap struct {
	Select key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Select}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Select},
	}
}

var keys = keyMap{
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "Open job detail"),
	),
}

type Item struct {
	JobID        uuid.UUID
	ConnectionID uuid.UUID
	Command      string
	Status       string
	LastUpdated  time.Time
	ReturnCode   *int32
}

func (i Item) Title() string {
	return i.Command
}

func (i Item) Description() string {
	rc := "pending"
	if i.ReturnCode != nil {
		rc = fmt.Sprintf("return %d", *i.ReturnCode)
	}
	return fmt.Sprintf("%s • %s", i.Status, rc)
}

func (i Item) FilterValue() string {
	return i.Command
}

type Model struct {
	keys      keyMap
	Help      help.Model
	List      list.Model
	connID    uuid.UUID
	items     []Item
	lastWidth int
}

type JobSelected struct {
	JobID uuid.UUID
}

func New(notif chan tea.Msg) Model {
	NotificationChan = notif

	h := help.New()
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Jobs"

	return Model{
		keys:  keys,
		Help:  h,
		List:  l,
		items: []Item{},
	}
}

func (m *Model) SetConnection(id uuid.UUID) {
	m.connID = id
}

func (m *Model) Reset() {
	m.items = []Item{}
	m.List = list.New([]list.Item{}, list.NewDefaultDelegate(), m.lastWidth, 0)
	m.List.Title = "Jobs"
}

func (m *Model) SetJobs(items []Item) {
	m.items = items
	listItems := make([]list.Item, 0, len(items))
	for _, it := range items {
		listItems = append(listItems, it)
	}
	m.List.SetItems(listItems)
}

func (m *Model) Upsert(job Item) {
	if job.ConnectionID != m.connID {
		return
	}

	for idx, existing := range m.items {
		if existing.JobID == job.JobID {
			m.items[idx] = job
			m.List.SetItem(idx, job)
			return
		}
	}

	m.items = append(m.items, job)
	m.List.InsertItem(len(m.items)-1, job)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Select):
			if len(m.items) == 0 || m.List.Index() < 0 || m.List.Index() >= len(m.items) {
				return m, nil
			}
			go func(jobID uuid.UUID) {
				NotificationChan <- JobSelected{JobID: jobID}
			}(m.items[m.List.Index()].JobID)
		}
	case tea.WindowSizeMsg:
		m.lastWidth = msg.Width
		m.List.SetSize(msg.Width, msg.Height-lipgloss.Height(m.Help.View(m.keys)))
	}

	var hCmd, lCmd tea.Cmd
	m.Help, hCmd = m.Help.Update(msg)
	m.List, lCmd = m.List.Update(msg)

	return m, tea.Batch(hCmd, lCmd)
}

func (m Model) View() string {
	return m.List.View() + "\n" + m.Help.View(m.keys)
}
