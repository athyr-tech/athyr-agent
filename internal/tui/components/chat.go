package components

import (
	"strings"
	"time"

	"github.com/athyr-tech/athyr-agent/internal/tui/styles"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ChatMessage represents a message in the chat history.
type ChatMessage struct {
	Time    time.Time
	Role    string // "user", "assistant", "error"
	Content string
}

// Chat provides an interactive chat interface.
type Chat struct {
	messages []ChatMessage
	textarea textarea.Model
	viewport viewport.Model
	width    int
	height   int
	ready    bool
	focused  bool
	sending  bool
}

// NewChat creates a new Chat component.
func NewChat() Chat {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.ShowLineNumbers = false
	ta.CharLimit = 2000
	ta.SetHeight(3)

	return Chat{
		messages: make([]ChatMessage, 0),
		textarea: ta,
		focused:  false, // Start unfocused so tab navigation works
	}
}

// Init initializes the chat component.
func (c Chat) Init() tea.Cmd {
	return textarea.Blink
}

// SetSize updates the component size.
// w and h are the panel content dimensions (excluding panel borders/padding).
func (c *Chat) SetSize(w, h int) {
	c.width = w
	c.height = h

	// Content inside panel: total height - panel overhead (4 lines)
	contentHeight := h - styles.PanelVerticalOverhead
	// Input area: 3 lines for textarea + 2 for input borders
	inputHeight := 5
	// Title takes 1 line + margin
	titleHeight := 2
	viewportHeight := contentHeight - inputHeight - titleHeight

	if viewportHeight < 3 {
		viewportHeight = 3
	}

	// Viewport/input width: panel width - panel horizontal overhead
	contentWidth := w - styles.PanelHorizontalOverhead
	c.textarea.SetWidth(contentWidth)

	if c.ready {
		c.viewport.Width = contentWidth
		c.viewport.Height = viewportHeight
	} else {
		c.viewport = viewport.New(contentWidth, viewportHeight)
		c.ready = true
	}

	c.updateContent()
}

// Focused returns whether the input is focused.
func (c Chat) Focused() bool {
	return c.focused
}

// Focus focuses the input.
func (c *Chat) Focus() {
	c.focused = true
	c.textarea.Focus()
}

// Blur unfocuses the input.
func (c *Chat) Blur() {
	c.focused = false
	c.textarea.Blur()
}

// Value returns the current input value.
func (c Chat) Value() string {
	return strings.TrimSpace(c.textarea.Value())
}

// ClearInput clears the input textarea.
func (c *Chat) ClearInput() {
	c.textarea.Reset()
}

// SetSending sets the sending state (shows spinner).
func (c *Chat) SetSending(sending bool) {
	c.sending = sending
}

// AddUserMessage adds a user message to the history.
func (c *Chat) AddUserMessage(content string) {
	c.messages = append(c.messages, ChatMessage{
		Time:    time.Now(),
		Role:    "user",
		Content: content,
	})
	c.updateContent()
	c.viewport.GotoBottom()
}

// AddAssistantMessage adds an assistant message to the history.
func (c *Chat) AddAssistantMessage(content string) {
	c.messages = append(c.messages, ChatMessage{
		Time:    time.Now(),
		Role:    "assistant",
		Content: content,
	})
	c.updateContent()
	c.viewport.GotoBottom()
}

// AddErrorMessage adds an error message to the history.
func (c *Chat) AddErrorMessage(content string) {
	c.messages = append(c.messages, ChatMessage{
		Time:    time.Now(),
		Role:    "error",
		Content: content,
	})
	c.updateContent()
	c.viewport.GotoBottom()
}

// Update handles input.
func (c Chat) Update(msg tea.Msg) (Chat, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if c.focused {
				c.Blur()
				return c, nil
			}
		case "i":
			if !c.focused {
				c.Focus()
				return c, nil
			}
		case "enter":
			// Don't process Enter here - let model.go handle sending
			if c.focused && c.Value() != "" {
				return c, nil
			}
		}

		// Pass key to textarea if focused
		if c.focused {
			var cmd tea.Cmd
			c.textarea, cmd = c.textarea.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			// Handle viewport scrolling when not focused on input
			var cmd tea.Cmd
			c.viewport, cmd = c.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return c, tea.Batch(cmds...)
}

// updateContent rebuilds the viewport content from messages.
func (c *Chat) updateContent() {
	if !c.ready {
		return
	}

	var lines []string
	for _, msg := range c.messages {
		lines = append(lines, c.formatMessage(msg))
		lines = append(lines, "") // Blank line between messages
	}

	if len(lines) == 0 {
		lines = append(lines, styles.Muted.Render("Start a conversation..."))
	}

	if c.sending {
		lines = append(lines, styles.Pending.Render("● Thinking..."))
	}

	c.viewport.SetContent(strings.Join(lines, "\n"))
}

// formatMessage formats a chat message for display.
func (c Chat) formatMessage(msg ChatMessage) string {
	var roleStyle lipgloss.Style
	var roleLabel string

	switch msg.Role {
	case "user":
		roleStyle = styles.ChatUser
		roleLabel = "You"
	case "assistant":
		roleStyle = styles.ChatAssistant
		roleLabel = "Assistant"
	case "error":
		roleStyle = styles.LogError
		roleLabel = "Error"
	}

	timestamp := styles.MessageTimestamp.Render(msg.Time.Format("15:04:05"))
	header := timestamp + " " + roleStyle.Render(roleLabel+":")

	// Wrap content to fit width
	content := msg.Content
	maxWidth := c.width - 8
	if maxWidth > 0 {
		content = wordWrap(content, maxWidth)
	}

	return header + "\n" + content
}

// View renders the chat interface.
func (c Chat) View() string {
	// Chat history viewport
	historyView := c.viewport.View()

	// Input area with border
	var inputStyle lipgloss.Style
	if c.focused {
		inputStyle = styles.ChatInputFocused
	} else {
		inputStyle = styles.ChatInput
	}
	inputView := inputStyle.Width(c.width - 6).Render(c.textarea.View())

	// Build content: title + viewport + input
	title := styles.PanelTitle.Render("Chat")
	content := title + "\n" + historyView + "\n" + inputView

	// Use Height() to ensure consistent footer positioning across all tabs
	return styles.Panel.Width(c.width).Height(c.height).Render(content)
}

// wordWrap wraps text to fit within maxWidth.
func wordWrap(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		words := strings.Fields(line)
		if len(words) == 0 {
			continue
		}

		currentLine := words[0]
		for _, word := range words[1:] {
			if len(currentLine)+1+len(word) > maxWidth {
				result.WriteString(currentLine)
				result.WriteString("\n")
				currentLine = word
			} else {
				currentLine += " " + word
			}
		}
		result.WriteString(currentLine)
	}

	return result.String()
}
