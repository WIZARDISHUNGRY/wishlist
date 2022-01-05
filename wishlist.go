package wishlist

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gliderlabs/ssh"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

var enter = key.NewBinding(
	key.WithKeys("enter"),
	key.WithHelp("Enter", "SSH"),
)

func newListing(endpoints []*Endpoint, s ssh.Session) tea.Model {
	var items []list.Item
	for _, endpoint := range endpoints {
		if endpoint.Valid() {
			items = append(items, endpoint)
		}
	}
	l := list.NewModel(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Directory Listing"
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{enter}
	}
	return model{
		list:      l,
		endpoints: endpoints,
		session:   s,
	}
}

type model struct {
	list      list.Model
	endpoints []*Endpoint
	session   ssh.Session
}

func (i *Endpoint) Title() string       { return i.Name }
func (i *Endpoint) Description() string { return fmt.Sprintf("ssh://%s", i.Address) }
func (i *Endpoint) FilterValue() string { return i.Name }

func (m model) Init() tea.Cmd {
	return nil
}

type connectMsg struct {
	sess ssh.Session
	name string
	addr string
}

func connectCmd(sess ssh.Session, name, addr string) tea.Cmd {
	return func() tea.Msg {
		return connectMsg{
			sess: sess,
			name: name,
			addr: addr,
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, enter) {
			e := m.list.SelectedItem().(*Endpoint)
			return noopModel{}, connectCmd(m.session, e.Name, e.Address)
		}
	case tea.WindowSizeMsg:
		top, right, bottom, left := docStyle.GetMargin()
		m.list.SetSize(msg.Width-left-right, msg.Height-top-bottom)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

func toAddress(listen string, port int64) string {
	return fmt.Sprintf("%s:%d", listen, port)
}

type noopModel struct{}

func (noopModel) Init() tea.Cmd { return nil }
func (m noopModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case connectMsg:
		addr := msg.addr
		sess := msg.sess
		name := msg.name
		log.Println("connecting to", addr)
		if err := connect(sess, addr); err != nil {
			fmt.Fprintln(sess, err.Error())
			sess.Exit(1)
			return m, nil
		}
		log.Printf("finished connection to %q (%s)", name, addr)
		fmt.Fprintf(sess, "Closed connection to %q (%s)\n", name, addr)
		sess.Exit(0)
		return m, nil
	}
	log.Println("noop msg:", msg)
	return m, nil
}
func (noopModel) View() string { return "" }
