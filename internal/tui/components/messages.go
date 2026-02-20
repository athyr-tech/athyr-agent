package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/athyr-tech/athyr-agent/internal/tui/styles"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// MessageDirection indicates whether a message is incoming or outgoing.
type MessageDirection int

const (
	MessageIncoming MessageDirection = iota
	MessageOutgoing
)

// Message represents a single message in the log.
type Message struct {
	Time      time.Time
	Direction MessageDirection
	Topic     string
	Content   string
	Model     string
	Tokens    int
}

// Messages displays a scrollable list of messages.
type Messages struct {
	messages []Message
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

// NewMessages creates a new Messages component.
func NewMessages() Messages {
	return Messages{
		messages: make([]Message, 0),
	}
}

// SetSize updates the component size.
// w and h are the total panel dimensions.
func (m *Messages) SetSize(w, h int) {
	m.width = w
	m.height = h

	// Content inside panel: total height - panel overhead (4 lines)
	// Then subtract title (2 lines: title + blank line after)
	contentHeight := h - styles.PanelVerticalOverhead
	viewportHeight := contentHeight - 2
	if viewportHeight < 3 {
		viewportHeight = 3
	}

	// Viewport width: panel width - panel horizontal overhead
	viewportWidth := w - styles.PanelHorizontalOverhead

	if m.ready {
		m.viewport.Width = viewportWidth
		m.viewport.Height = viewportHeight
	} else {
		m.viewport = viewport.New(viewportWidth, viewportHeight)
		m.ready = true
	}
	m.updateContent()
}

// AddMessage adds a new message to the log.
func (m *Messages) AddMessage(msg Message) {
	m.messages = append(m.messages, msg)
	m.updateContent()
	// Auto-scroll to bottom
	m.viewport.GotoBottom()
}

// Update handles key messages for scrolling.
func (m Messages) Update(msg tea.Msg) (Messages, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// updateContent rebuilds the viewport content from messages.
func (m *Messages) updateContent() {
	if !m.ready {
		return
	}

	var lines []string
	for _, msg := range m.messages {
		line := m.formatMessage(msg)
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		lines = append(lines, styles.Muted.Render("No messages yet..."))
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
}

// formatMessage formats a single message for display.
func (m Messages) formatMessage(msg Message) string {
	timestamp := styles.MessageTimestamp.Render(msg.Time.Format("15:04:05"))

	var dirStyle = styles.MessageIncoming
	dirIcon := "←"
	if msg.Direction == MessageOutgoing {
		dirStyle = styles.MessageOutgoing
		dirIcon = "→"
	}

	topic := styles.MessageTopic.Render(msg.Topic)

	// Truncate content if too long
	content := msg.Content
	maxLen := m.width - 30
	if maxLen > 0 && len(content) > maxLen {
		content = content[:maxLen-3] + "..."
	}

	// Escape newlines for single-line display
	content = strings.ReplaceAll(content, "\n", " ")

	result := fmt.Sprintf("%s %s %s %s",
		timestamp,
		dirStyle.Render(dirIcon),
		topic,
		content,
	)

	if msg.Tokens > 0 {
		result += styles.Muted.Render(fmt.Sprintf(" (%d tokens)", msg.Tokens))
	}

	return result
}

// Content returns the inner content without panel wrapper.
func (m Messages) Content() string {
	return m.viewport.View()
}

// View renders the messages panel filling exactly width × height.
func (m Messages) View() string {
	title := styles.PanelTitle.Render("Messages")
	content := title + "\n" + m.viewport.View()
	return styles.Panel.Width(m.width).Height(m.height).Render(content)
}
