package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Dashboard displays the agent status and message flow side by side.
type Dashboard struct {
	status   Status
	messages Messages
	width    int
	height   int
	ready    bool
}

// NewDashboard creates a new Dashboard component.
func NewDashboard(info AgentInfo) Dashboard {
	return Dashboard{
		status:   NewStatus(info),
		messages: NewMessages(),
	}
}

// SetSize updates the component size.
func (d *Dashboard) SetSize(w, h int) {
	d.width = w
	d.height = h
	d.ready = true

	// Split: two panels - reduce width to fit within bounds
	usableWidth := w - 2
	leftWidth := usableWidth / 2
	rightWidth := usableWidth - leftWidth

	d.status.SetSize(leftWidth, h)
	d.messages.SetSize(rightWidth, h)
}

// Update handles messages for the dashboard.
func (d Dashboard) Update(msg tea.Msg) (Dashboard, tea.Cmd) {
	var cmds []tea.Cmd

	// Update messages component (for scrolling)
	var cmd tea.Cmd
	d.messages, cmd = d.messages.Update(msg)
	cmds = append(cmds, cmd)

	return d, tea.Batch(cmds...)
}

// View renders the dashboard filling exactly width × height.
func (d Dashboard) View() string {
	if !d.ready {
		return "Loading..."
	}

	// Join panels directly - gap is built into width calculation
	return lipgloss.JoinHorizontal(lipgloss.Top, d.status.View(), d.messages.View())
}

// Status accessors - delegate to internal status component

// SetConnected updates the connection status.
func (d *Dashboard) SetConnected(connected bool) {
	d.status.SetConnected(connected)
}

// SetAgentID updates the agent ID.
func (d *Dashboard) SetAgentID(id string) {
	d.status.SetAgentID(id)
}

// SetError sets an error message.
func (d *Dashboard) SetError(msg string) {
	d.status.SetError(msg)
}

// AddTokens adds tokens to the total count.
func (d *Dashboard) AddTokens(count int) {
	d.status.AddTokens(count)
}

// TotalTokens returns the total token count.
func (d Dashboard) TotalTokens() int {
	return d.status.TotalTokens()
}

// Connected returns the current connection status.
func (d Dashboard) Connected() bool {
	return d.status.Connected()
}

// AgentID returns the current agent ID.
func (d Dashboard) AgentID() string {
	return d.status.AgentID()
}

// Messages accessors - delegate to internal messages component

// AddMessage adds a new message to the log.
func (d *Dashboard) AddMessage(msg Message) {
	d.messages.AddMessage(msg)
}
