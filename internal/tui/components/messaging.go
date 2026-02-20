package components

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/athyr-tech/athyr-agent/internal/tui/styles"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MessagingMode represents the message sending mode.
type MessagingMode int

const (
	ModePublish MessagingMode = iota // Fire-and-forget
	ModeRequest                      // Wait for reply
)

// WatchMessage represents a message received on a watched topic.
type WatchMessage struct {
	Timestamp time.Time
	Content   string
}

// maxWatchMessages is the maximum number of messages to buffer.
const maxWatchMessages = 50

// Messaging provides an interface to send messages to topics.
type Messaging struct {
	// Mode
	mode MessagingMode

	// Topic selection
	configuredTopics []string // from config (subscribe + publish + routes)
	topicInput       textinput.Model
	showDropdown     bool
	dropdownIdx      int

	// Message input
	messageInput textarea.Model

	// Response (for request mode)
	response    string
	responseErr error
	sending     bool

	// Success feedback (for publish mode)
	lastSendTime    time.Time
	lastSendSuccess bool

	// Focus state
	focusedField int  // 0=send topic, 1=send message, 2=watch topic, -1=none
	focused      bool // whether component has focus at all
	width        int
	height       int
	ready        bool

	// Watch state (right panel)
	watchTopicInput   textinput.Model
	watchShowDropdown bool
	watchDropdownIdx  int
	watchActive       bool
	watchTopic        string // Currently watched topic
	watchError        error
	watchMessages     []WatchMessage // Buffer (max 50)
}

// NewMessaging creates a new Messaging component.
func NewMessaging(topics []string) Messaging {
	ti := textinput.New()
	ti.Placeholder = "Select or type topic..."
	ti.Prompt = "" // Remove default "> " prompt
	ti.CharLimit = 256
	ti.Width = 40

	ta := textarea.New()
	ta.Placeholder = `{"key": "value"}`
	ta.Prompt = "" // Remove default prompt
	ta.ShowLineNumbers = false
	ta.CharLimit = 4000
	ta.SetHeight(5)

	// Watch topic input
	wti := textinput.New()
	wti.Placeholder = "Topic to watch..."
	wti.Prompt = ""
	wti.CharLimit = 256
	wti.Width = 40

	return Messaging{
		mode:             ModePublish,
		configuredTopics: topics,
		topicInput:       ti,
		messageInput:     ta,
		focusedField:     0,     // Start on send topic
		focused:          false, // Start unfocused so global keys work
		watchTopicInput:  wti,
		watchMessages:    make([]WatchMessage, 0, maxWatchMessages),
	}
}

// Init initializes the component.
func (m Messaging) Init() tea.Cmd {
	return nil
}

// SetSize updates the component size.
func (m *Messaging) SetSize(w, h int) {
	m.width = w
	m.height = h

	// Split view: each panel gets half width minus gap
	// Total width = panel1 + gap(2) + panel2
	// Each panel content width accounts for border/padding
	panelWidth := (w - 2) / 2 // 2 for gap between panels
	inputContentWidth := panelWidth - 8
	if inputContentWidth < 20 {
		inputContentWidth = 20
	}

	// Send panel inputs
	m.topicInput.Width = inputContentWidth

	// Message textarea: leave room for mode selector, topic, status
	// Mode: 3, Topic label+input: 4, Status: 2, Panel padding: 4
	messageHeight := h - 18
	if messageHeight < 3 {
		messageHeight = 3
	}
	m.messageInput.SetWidth(inputContentWidth)
	m.messageInput.SetHeight(messageHeight)

	// Watch panel input
	m.watchTopicInput.Width = inputContentWidth

	m.ready = true
}

// Focus focuses the component on the topic field.
func (m *Messaging) Focus() {
	m.focused = true
	m.focusedField = 0
	m.topicInput.Focus()
	m.messageInput.Blur()
}

// Blur blurs all inputs.
func (m *Messaging) Blur() {
	m.focused = false
	m.focusedField = -1
	m.topicInput.Blur()
	m.messageInput.Blur()
	m.watchTopicInput.Blur()
	m.showDropdown = false
	m.watchShowDropdown = false
}

// Focused returns whether the component has focus.
func (m Messaging) Focused() bool {
	return m.focused
}

// Topic returns the current topic value.
func (m Messaging) Topic() string {
	return strings.TrimSpace(m.topicInput.Value())
}

// Message returns the current message content.
func (m Messaging) Message() string {
	return strings.TrimSpace(m.messageInput.Value())
}

// Mode returns the current messaging mode.
func (m Messaging) Mode() MessagingMode {
	return m.mode
}

// IsSending returns whether a message is currently being sent.
func (m Messaging) IsSending() bool {
	return m.sending
}

// SetSending sets the sending state.
func (m *Messaging) SetSending(sending bool) {
	m.sending = sending
}

// SetResponse sets the response from a request.
func (m *Messaging) SetResponse(response string, err error) {
	m.response = response
	m.responseErr = err
	// Also track success for publish mode feedback
	if err == nil {
		m.lastSendSuccess = true
		m.lastSendTime = time.Now()
	} else {
		m.lastSendSuccess = false
	}
}

// ClearInputs clears all inputs, response, and watch state.
func (m *Messaging) ClearInputs() {
	m.topicInput.SetValue("")
	m.messageInput.SetValue("")
	m.response = ""
	m.responseErr = nil
	m.lastSendSuccess = false
	// Note: Watch state is cleared separately via ClearWatch()
	// This allows the model to also stop the subscription
}

// WatchTopicValue returns the current watch topic input value.
func (m Messaging) WatchTopicValue() string {
	return strings.TrimSpace(m.watchTopicInput.Value())
}

// IsWatchActive returns whether watching is active.
func (m Messaging) IsWatchActive() bool {
	return m.watchActive
}

// WatchedTopic returns the currently watched topic (empty if not watching).
func (m Messaging) WatchedTopic() string {
	return m.watchTopic
}

// SetWatchStatus updates the watch state.
func (m *Messaging) SetWatchStatus(topic string, err error) {
	if err != nil {
		m.watchError = err
		m.watchActive = false
		m.watchTopic = ""
	} else if topic == "" {
		// Stopped watching
		m.watchActive = false
		m.watchTopic = ""
		m.watchError = nil
	} else {
		// Started watching
		m.watchActive = true
		m.watchTopic = topic
		m.watchError = nil
	}
}

// AddWatchMessage adds a message to the watch buffer.
func (m *Messaging) AddWatchMessage(msg WatchMessage) {
	m.watchMessages = append(m.watchMessages, msg)
	// Trim to max size
	if len(m.watchMessages) > maxWatchMessages {
		m.watchMessages = m.watchMessages[len(m.watchMessages)-maxWatchMessages:]
	}
}

// ClearWatch clears watch state and messages.
func (m *Messaging) ClearWatch() {
	m.watchTopicInput.SetValue("")
	m.watchActive = false
	m.watchTopic = ""
	m.watchError = nil
	m.watchMessages = m.watchMessages[:0]
	m.watchShowDropdown = false
}

// isValidJSON checks if the message content is valid JSON.
func (m Messaging) isValidJSON() bool {
	msg := m.Message()
	if msg == "" {
		return true // Empty is fine
	}
	var js json.RawMessage
	return json.Unmarshal([]byte(msg), &js) == nil
}

// showSentFeedback returns true if we should show "Sent!" feedback.
func (m Messaging) showSentFeedback() bool {
	if !m.lastSendSuccess {
		return false
	}
	// Show for 3 seconds after sending
	return time.Since(m.lastSendTime) < 3*time.Second
}

// matchingTopics returns topics that match the current send topic input.
func (m Messaging) matchingTopics() []string {
	input := strings.ToLower(m.topicInput.Value())
	if input == "" {
		return m.configuredTopics
	}

	var matches []string
	for _, t := range m.configuredTopics {
		if strings.Contains(strings.ToLower(t), input) {
			matches = append(matches, t)
		}
	}
	return matches
}

// matchingWatchTopics returns topics that match the current watch topic input.
func (m Messaging) matchingWatchTopics() []string {
	input := strings.ToLower(m.watchTopicInput.Value())
	if input == "" {
		return m.configuredTopics
	}

	var matches []string
	for _, t := range m.configuredTopics {
		if strings.Contains(strings.ToLower(t), input) {
			matches = append(matches, t)
		}
	}
	return matches
}

// Update handles input.
func (m Messaging) Update(msg tea.Msg) (Messaging, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Focus on 'i' when not focused
		if key == "i" && !m.focused {
			m.Focus()
			return m, nil
		}

		// Mode toggle (only when not typing in message field)
		if key == "m" && !m.messageInput.Focused() && m.focusedField != 2 {
			if m.mode == ModePublish {
				m.mode = ModeRequest
			} else {
				m.mode = ModePublish
			}
			return m, nil
		}

		// Unfocus on Esc (consistent with Chat)
		if key == "esc" {
			if m.showDropdown {
				m.showDropdown = false
				return m, nil
			}
			if m.watchShowDropdown {
				m.watchShowDropdown = false
				return m, nil
			}
			m.Blur()
			return m, nil
		}

		// Clear inputs with Ctrl+L
		if key == "ctrl+l" {
			m.ClearInputs()
			return m, nil
		}

		// If not focused, don't process other keys
		if !m.focused {
			return m, nil
		}

		// Left/Right arrow keys to switch between panels (like Tools tab)
		// Only when not in message textarea (which needs arrow keys for editing)
		if !m.messageInput.Focused() {
			switch key {
			case "left", "h":
				// Switch to send panel (field 0 = send topic)
				if m.focusedField == 2 {
					m.focusedField = 0
					m.updateFieldFocus()
					m.watchShowDropdown = false
				}
				return m, nil
			case "right", "l":
				// Switch to watch panel (field 2 = watch topic)
				if m.focusedField != 2 {
					m.focusedField = 2
					m.updateFieldFocus()
					m.showDropdown = false
				}
				return m, nil
			}
		}

		// Tab/Ctrl+N to switch fields (cycles through 3 fields)
		if key == "ctrl+n" || (key == "tab" && !m.messageInput.Focused()) {
			m.focusedField = (m.focusedField + 1) % 3
			m.updateFieldFocus()
			m.showDropdown = false
			m.watchShowDropdown = false
			return m, nil
		}

		// Dropdown navigation for send topic
		if m.showDropdown && m.focusedField == 0 {
			matches := m.matchingTopics()
			switch key {
			case "down", "ctrl+j":
				if m.dropdownIdx < len(matches)-1 {
					m.dropdownIdx++
				}
				return m, nil
			case "up", "ctrl+k":
				if m.dropdownIdx > 0 {
					m.dropdownIdx--
				}
				return m, nil
			case "enter":
				if len(matches) > 0 && m.dropdownIdx < len(matches) {
					m.topicInput.SetValue(matches[m.dropdownIdx])
					m.showDropdown = false
					// Move to message field
					m.focusedField = 1
					m.updateFieldFocus()
				}
				return m, nil
			}
		}

		// Dropdown navigation for watch topic
		if m.watchShowDropdown && m.focusedField == 2 {
			matches := m.matchingWatchTopics()
			switch key {
			case "down", "ctrl+j":
				if m.watchDropdownIdx < len(matches)-1 {
					m.watchDropdownIdx++
				}
				return m, nil
			case "up", "ctrl+k":
				if m.watchDropdownIdx > 0 {
					m.watchDropdownIdx--
				}
				return m, nil
			case "enter":
				if len(matches) > 0 && m.watchDropdownIdx < len(matches) {
					m.watchTopicInput.SetValue(matches[m.watchDropdownIdx])
					m.watchShowDropdown = false
				}
				return m, nil
			}
		}

		// Handle send topic input
		if m.focusedField == 0 {
			var cmd tea.Cmd
			m.topicInput, cmd = m.topicInput.Update(msg)
			cmds = append(cmds, cmd)

			// Show dropdown when typing
			if key != "esc" && len(m.matchingTopics()) > 0 {
				m.showDropdown = true
				m.dropdownIdx = 0
			}
		}

		// Handle message input
		if m.focusedField == 1 {
			var cmd tea.Cmd
			m.messageInput, cmd = m.messageInput.Update(msg)
			cmds = append(cmds, cmd)
		}

		// Handle watch topic input
		if m.focusedField == 2 {
			var cmd tea.Cmd
			m.watchTopicInput, cmd = m.watchTopicInput.Update(msg)
			cmds = append(cmds, cmd)

			// Show dropdown when typing
			if key != "esc" && len(m.matchingWatchTopics()) > 0 {
				m.watchShowDropdown = true
				m.watchDropdownIdx = 0
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// updateFieldFocus updates the focus state of all input fields based on focusedField.
func (m *Messaging) updateFieldFocus() {
	m.topicInput.Blur()
	m.messageInput.Blur()
	m.watchTopicInput.Blur()

	switch m.focusedField {
	case 0:
		m.topicInput.Focus()
	case 1:
		m.messageInput.Focus()
	case 2:
		m.watchTopicInput.Focus()
	}
}

// View renders the messaging interface filling exactly width × height.
func (m Messaging) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Split: two panels - reduce width to fit within bounds
	usableWidth := m.width - 2
	leftWidth := usableWidth / 2
	rightWidth := usableWidth - leftWidth

	// Content width = panel width - border(2) - padding(4)
	leftContentWidth := leftWidth - styles.PanelHorizontalOverhead
	rightContentWidth := rightWidth - styles.PanelHorizontalOverhead
	if leftContentWidth < 20 {
		leftContentWidth = 20
	}
	if rightContentWidth < 20 {
		rightContentWidth = 20
	}

	// Content height = panel height - border(2) - padding(2)
	contentHeight := m.height - styles.PanelVerticalOverhead
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Build panels with content
	leftPanel := m.renderSendPanel(leftContentWidth)
	rightPanel := m.renderWatchPanel(rightContentWidth)

	// Pad content to fill height
	leftPanel = padToHeight(leftPanel, contentHeight)
	rightPanel = padToHeight(rightPanel, contentHeight)

	// Wrap in Panel style
	leftStyled := styles.Panel.Width(leftWidth).Height(m.height).Render(leftPanel)
	rightStyled := styles.Panel.Width(rightWidth).Height(m.height).Render(rightPanel)

	// Join panels directly - no gap
	return lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, rightStyled)
}

// padToHeight pads content with empty lines to reach target height.
func padToHeight(content string, targetHeight int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < targetHeight {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// LeftTitle returns the title for the left (Send) panel.
func (m Messaging) LeftTitle() string {
	title := "Send Message"
	if m.focusedField == 0 || m.focusedField == 1 {
		title = "● " + title
	}
	return title
}

// RightTitle returns the title for the right (Watch) panel.
func (m Messaging) RightTitle() string {
	title := "Watch Topic"
	if m.focusedField == 2 {
		title = "● " + title
	}
	return title
}

// LeftContent returns the content for the left (Send) panel without title.
func (m Messaging) LeftContent() string {
	// Panel width from model's split: (m.width)/2 - 1
	// Content inside panel: subtract 6 for panel border/padding
	panelWidth := m.width/2 - 1
	contentWidth := panelWidth - 6
	if contentWidth < 20 {
		contentWidth = 20
	}
	return m.renderSendPanelContent(contentWidth)
}

// RightContent returns the content for the right (Watch) panel without title.
func (m Messaging) RightContent() string {
	// Panel width from model's split: (m.width)/2 - 1
	// Content inside panel: subtract 6 for panel border/padding
	panelWidth := m.width/2 - 1
	contentWidth := panelWidth - 6
	if contentWidth < 20 {
		contentWidth = 20
	}
	return m.renderWatchPanelContent(contentWidth)
}

// renderSendPanel renders the left panel for sending messages (with title).
func (m Messaging) renderSendPanel(contentWidth int) string {
	var b strings.Builder

	// Title with focus indicator
	title := "Send Message"
	if m.focusedField == 0 || m.focusedField == 1 {
		title = "● " + title
	}
	b.WriteString(styles.PanelTitle.Render(title))
	b.WriteString("\n\n")

	b.WriteString(m.renderSendPanelContent(contentWidth))
	return b.String()
}

// renderSendPanelContent renders the left panel content without title.
func (m Messaging) renderSendPanelContent(contentWidth int) string {
	var b strings.Builder

	// Mode selector
	publishLabel := "Publish"
	requestLabel := "Request"
	if m.mode == ModePublish {
		publishLabel = styles.Connected.Render("[*] " + publishLabel)
		requestLabel = styles.Muted.Render("[ ] " + requestLabel)
	} else {
		publishLabel = styles.Muted.Render("[ ] " + publishLabel)
		requestLabel = styles.Connected.Render("[*] " + requestLabel)
	}
	b.WriteString("Mode: " + publishLabel + "  " + requestLabel)
	b.WriteString(styles.Muted.Render("  ('m')"))
	b.WriteString("\n\n")

	// Topic input
	topicLabel := "Topic:"
	if m.focusedField == 0 {
		topicLabel = styles.ChatUser.Render(topicLabel)
	}
	b.WriteString(topicLabel + "\n")

	topicStyle := styles.ChatInput
	if m.focusedField == 0 {
		topicStyle = styles.ChatInputFocused
	}
	b.WriteString(topicStyle.Width(contentWidth).Render(m.topicInput.View()))
	b.WriteString("\n")

	// Topic dropdown
	if m.showDropdown && m.focusedField == 0 {
		matches := m.matchingTopics()
		if len(matches) > 0 {
			var dropdownLines []string
			maxShow := 5
			if len(matches) < maxShow {
				maxShow = len(matches)
			}
			for i := 0; i < maxShow; i++ {
				topic := matches[i]
				prefix := "    "
				if i == m.dropdownIdx {
					prefix = styles.Connected.Render("  > ")
					topic = styles.FooterKey.Render(topic)
				} else {
					topic = styles.Muted.Render(topic)
				}
				dropdownLines = append(dropdownLines, prefix+topic)
			}
			if len(matches) > maxShow {
				dropdownLines = append(dropdownLines, styles.Muted.Render(fmt.Sprintf("    +%d more", len(matches)-maxShow)))
			}
			b.WriteString(strings.Join(dropdownLines, "\n"))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	// Message input with JSON validation indicator
	msgLabel := "Message:"
	if m.focusedField == 1 {
		msgLabel = styles.ChatUser.Render(msgLabel)
	}
	if m.Message() != "" {
		if m.isValidJSON() {
			msgLabel += styles.Connected.Render(" [JSON]")
		} else {
			msgLabel += styles.LogWarn.Render(" [text]")
		}
	}
	b.WriteString(msgLabel + "\n")

	msgStyle := styles.ChatInput
	if m.focusedField == 1 {
		msgStyle = styles.ChatInputFocused
	}
	b.WriteString(msgStyle.Width(contentWidth).Render(m.messageInput.View()))
	b.WriteString("\n\n")

	// Send status / button hint
	if m.sending {
		b.WriteString(styles.Pending.Render("* Sending..."))
	} else if m.showSentFeedback() && m.mode == ModePublish {
		b.WriteString(styles.Connected.Render("* Sent!"))
	} else {
		hint := "Ctrl+S to send"
		if m.Topic() == "" || m.Message() == "" {
			hint = styles.Muted.Render("Enter topic and message")
		} else if !m.focused {
			hint = styles.Muted.Render("'i' to focus, Ctrl+S send")
		}
		b.WriteString(hint)
	}

	// Response section (Request mode only)
	if m.mode == ModeRequest {
		b.WriteString("\n\n")
		b.WriteString(styles.Muted.Render("-- Response --"))
		b.WriteString("\n")
		if m.responseErr != nil {
			b.WriteString(styles.LogError.Render("Error: " + m.responseErr.Error()))
		} else if m.response != "" {
			respLines := strings.Split(m.response, "\n")
			for i, line := range respLines {
				if i > 5 {
					b.WriteString(styles.Muted.Render("... (truncated)"))
					break
				}
				b.WriteString(styles.MessageIncoming.Render(line) + "\n")
			}
		} else {
			b.WriteString(styles.Muted.Render("No response yet"))
		}
	}

	return b.String()
}

// renderWatchPanel renders the right panel for watching topics (with title).
func (m Messaging) renderWatchPanel(contentWidth int) string {
	var b strings.Builder

	// Title with focus indicator
	title := "Watch Topic"
	if m.focusedField == 2 {
		title = "● " + title
	}
	b.WriteString(styles.PanelTitle.Render(title))
	b.WriteString("\n\n")

	b.WriteString(m.renderWatchPanelContent(contentWidth))
	return b.String()
}

// renderWatchPanelContent renders the right panel content without title.
func (m Messaging) renderWatchPanelContent(contentWidth int) string {
	var b strings.Builder

	// Watch topic input
	topicLabel := "Topic:"
	if m.focusedField == 2 {
		topicLabel = styles.ChatUser.Render(topicLabel)
	}
	b.WriteString(topicLabel + "\n")

	topicStyle := styles.ChatInput
	if m.focusedField == 2 {
		topicStyle = styles.ChatInputFocused
	}
	b.WriteString(topicStyle.Width(contentWidth).Render(m.watchTopicInput.View()))
	b.WriteString("\n")

	// Watch topic dropdown
	if m.watchShowDropdown && m.focusedField == 2 {
		matches := m.matchingWatchTopics()
		if len(matches) > 0 {
			var dropdownLines []string
			maxShow := 5
			if len(matches) < maxShow {
				maxShow = len(matches)
			}
			for i := 0; i < maxShow; i++ {
				topic := matches[i]
				prefix := "    "
				if i == m.watchDropdownIdx {
					prefix = styles.Connected.Render("  > ")
					topic = styles.FooterKey.Render(topic)
				} else {
					topic = styles.Muted.Render(topic)
				}
				dropdownLines = append(dropdownLines, prefix+topic)
			}
			if len(matches) > maxShow {
				dropdownLines = append(dropdownLines, styles.Muted.Render(fmt.Sprintf("    +%d more", len(matches)-maxShow)))
			}
			b.WriteString(strings.Join(dropdownLines, "\n"))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	// Watch status
	b.WriteString("Status: ")
	if m.watchError != nil {
		b.WriteString(styles.LogError.Render("✗ Error: " + m.watchError.Error()))
	} else if m.watchActive {
		b.WriteString(styles.Connected.Render("● Watching"))
		b.WriteString(styles.Muted.Render(" [Ctrl+W: stop]"))
	} else {
		b.WriteString(styles.Muted.Render("○ Not watching"))
		if m.WatchTopicValue() != "" {
			b.WriteString(styles.Muted.Render(" [Ctrl+W: start]"))
		}
	}
	b.WriteString("\n\n")

	// Messages header
	b.WriteString(styles.Muted.Render("-- Messages --"))
	b.WriteString("\n")

	// Message list
	if len(m.watchMessages) == 0 {
		b.WriteString(styles.Muted.Render("No messages yet"))
	} else {
		// Show messages (newest at bottom, show last N that fit)
		maxLines := 10
		startIdx := 0
		if len(m.watchMessages) > maxLines {
			startIdx = len(m.watchMessages) - maxLines
		}
		for i := startIdx; i < len(m.watchMessages); i++ {
			msg := m.watchMessages[i]
			ts := msg.Timestamp.Format("15:04:05")
			// Truncate content if too long
			content := msg.Content
			maxContentLen := contentWidth - 12 // timestamp + spacing
			if len(content) > maxContentLen {
				content = content[:maxContentLen-3] + "..."
			}
			b.WriteString(styles.Muted.Render(ts) + "  " + styles.MessageIncoming.Render(content) + "\n")
		}
	}

	// Message count
	b.WriteString("\n")
	b.WriteString(styles.Muted.Render(fmt.Sprintf("%d messages received", len(m.watchMessages))))

	return b.String()
}

// Help returns help text for the messaging tab.
func (m Messaging) Help() string {
	if !m.focused {
		return styles.FooterKey.Render("i") + styles.FooterDesc.Render(": focus")
	}
	return styles.FooterKey.Render("←/→") + styles.FooterDesc.Render(": panel") + "  " +
		styles.FooterKey.Render("m") + styles.FooterDesc.Render(": mode") + "  " +
		styles.FooterKey.Render("Ctrl+S") + styles.FooterDesc.Render(": send") + "  " +
		styles.FooterKey.Render("Ctrl+W") + styles.FooterDesc.Render(": watch") + "  " +
		styles.FooterKey.Render("Ctrl+L") + styles.FooterDesc.Render(": clear") + "  " +
		styles.FooterKey.Render("Esc") + styles.FooterDesc.Render(": unfocus")
}
