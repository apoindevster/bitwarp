package connlist

import (
	"math"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)
var NotificationChan chan tea.Msg

type keyMap struct {
	AddConn  key.Binding
	DelConn  key.Binding
	Interact key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.AddConn, k.DelConn, k.Interact}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.AddConn, k.DelConn, k.Interact}, // first column
	}
}

var keys = keyMap{
	AddConn: key.NewBinding(
		key.WithKeys("c", "C"),
		key.WithHelp("c/C", "Add a Connection"),
	),
	DelConn: key.NewBinding(
		key.WithKeys("d", "D"),
		key.WithHelp("d/D", "Delete a Connection"),
	),
	Interact: key.NewBinding(
		key.WithKeys("i", "enter"),
		key.WithHelp("i/enter", "Interact with current Connection"),
	),
}

type Item struct {
	T, Desc string
}

// The following Types are the possible custom tea.Msg types
// objects of these types get propagated back up to NotificationChan
type NewConnReq struct {
	Item Item
}
type DelConnReq struct {
	Id int
}
type InteractConnReq struct {
	Id int
}

// End

func (i Item) Title() string       { return i.T }
func (i Item) Description() string { return i.Desc }
func (i Item) FilterValue() string { return i.T }

type Model struct {
	keys  keyMap
	Help  help.Model
	List  list.Model
	Items []Item
}

func New(notif chan tea.Msg) Model {
	NotificationChan = notif

	h := help.New()
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)

	return Model{
		keys: keys,
		Help: h,
		List: l,
	}
}

func AddItemReq() {
	NotificationChan <- NewConnReq{}
}

func DeleteItem(idx int) {
	NotificationChan <- DelConnReq{Id: idx}
}

func Interact(idx int) {
	NotificationChan <- InteractConnReq{Id: idx}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.AddConn):
			go AddItemReq()
		case key.Matches(msg, m.keys.DelConn):
			go DeleteItem(m.List.GlobalIndex())
		case key.Matches(msg, m.keys.Interact):
			go Interact(m.List.GlobalIndex())
		}
	case tea.WindowSizeMsg:
		m.List.SetSize(msg.Width, msg.Height-lipgloss.Height(m.Help.View(m.keys)))
	case NewConnReq:
		m.List.InsertItem(math.MaxInt32, msg.Item)
	case DelConnReq:
		m.List.RemoveItem(msg.Id)
	}

	help, hCmd := m.Help.Update(msg)
	l, lCmd := m.List.Update(msg)

	m.Help = help
	m.List = l

	return m, tea.Batch(
		hCmd,
		lCmd,
	)
}

func (m Model) View() string {
	return m.List.View() + "\n" + m.Help.View(m.keys)
}
