package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/athyr-tech/athyr-agent/internal/config"
	"github.com/athyr-tech/athyr-agent/internal/runner"
	"github.com/athyr-tech/athyr-agent/internal/tui/components"
	"github.com/athyr-tech/athyr-agent/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the root Bubble Tea model for the TUI.
type Model struct {
	// Configuration
	cfg      *config.Config
	eventBus runner.EventBus

	// Layout
	width  int
	height int

	// Components
	tabs      components.Tabs
	dashboard components.Dashboard
	chat      components.Chat
	messaging components.Messaging
	logs      components.Logs
	tools     components.Tools

	// State
	ready            bool
	quitting         bool
	showHelp         bool
	chatHandler      ChatHandler
	messagingHandler MessagingHandler

	// Help overlay
	help components.Help

	// Program reference for sending watch messages
	program *tea.Program
}

// ChatHandler is the interface for sending chat messages.
// This is implemented by the Runner/MessageHandler.
type ChatHandler interface {
	DirectChat(content string) (response string, model string, tokens int, err error)
}

// WatchCallback is the function signature for receiving watch messages.
type WatchCallback func(timestamp time.Time, content string)

// MessagingHandler is the interface for sending messages to topics.
// This is implemented by the Runner/MessageHandler.
type MessagingHandler interface {
	PublishMessage(topic string, data []byte) error
	RequestMessage(topic string, data []byte) ([]byte, error)
	WatchTopic(topic string, callback WatchCallback) error
	StopWatching() error
	WatchingTopic() string
}

// NewModel creates a new root Model.
func NewModel(cfg *config.Config, eventBus runner.EventBus, serverAddr string) Model {
	// Build agent info from config
	agentInfo := components.AgentInfo{
		Name:      cfg.Agent.Name,
		Model:     cfg.Agent.Model,
		Server:    serverAddr,
		Subscribe: cfg.Agent.Topics.Subscribe,
		Publish:   cfg.Agent.Topics.Publish,
	}

	// Add routes if configured
	for _, route := range cfg.Agent.Topics.Routes {
		agentInfo.Routes = append(agentInfo.Routes, components.RouteInfo{
			Topic:       route.Topic,
			Description: route.Description,
		})
	}

	// Add MCP server names if configured
	for _, srv := range cfg.Agent.MCP.Servers {
		agentInfo.MCPServers = append(agentInfo.MCPServers, srv.Name)
	}

	// Add memory configuration
	agentInfo.Memory = components.MemoryInfo{
		Enabled:       cfg.Agent.Memory.Enabled,
		SessionPrefix: cfg.Agent.Memory.SessionPrefix,
		TTL:           cfg.Agent.Memory.TTL,
		ProfileType:   cfg.Agent.Memory.GetProfile().Type,
		MaxTokens:     cfg.Agent.Memory.GetProfile().MaxTokens,
	}

	// Collect all topics for messaging component
	allTopics := make([]string, 0)
	allTopics = append(allTopics, cfg.Agent.Topics.Subscribe...)
	allTopics = append(allTopics, cfg.Agent.Topics.Publish...)
	for _, route := range cfg.Agent.Topics.Routes {
		allTopics = append(allTopics, route.Topic)
	}

	return Model{
		cfg:       cfg,
		eventBus:  eventBus,
		tabs:      components.NewTabs(components.DefaultTabs()),
		dashboard: components.NewDashboard(agentInfo),
		chat:      components.NewChat(),
		messaging: components.NewMessaging(allTopics),
		logs:      components.NewLogs(),
		tools:     components.NewTools(),
		help:      components.NewHelp(),
	}
}

// SetChatHandler sets the handler for sending chat messages.
func (m *Model) SetChatHandler(h ChatHandler) {
	m.chatHandler = h
}

// SetMessagingHandler sets the handler for sending messages to topics.
func (m *Model) SetMessagingHandler(h MessagingHandler) {
	m.messagingHandler = h
}

// SetProgram sets the tea.Program reference for sending async messages.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		listenForEvents(m.eventBus),
		m.chat.Init(),
		m.messaging.Init(),
	)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update component sizes
		m.updateLayout()

	case tea.KeyMsg:
		key := msg.String()

		// Help overlay toggle
		if key == "?" {
			m.showHelp = !m.showHelp
			return m, nil
		}

		// Any key closes help overlay
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// Tab/shift+tab always navigate tabs
		switch key {
		case "tab", "shift+tab":
			m.tabs.Update(msg)
			return m, tea.Batch(cmds...)
		}

		// Number keys navigate tabs only when no text input is focused
		isChatFocused := m.tabs.Active() == components.TabChat && m.chat.Focused()
		isMessagingFocused := m.tabs.Active() == components.TabMessaging && m.messaging.Focused()
		if !isChatFocused && !isMessagingFocused {
			switch key {
			case "1", "2", "3", "4", "5":
				m.tabs.Update(msg)
				return m, tea.Batch(cmds...)
			}
		}

		// Global quit keys (but not when typing in chat/messaging)
		switch key {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			isChatFocused := m.tabs.Active() == components.TabChat && m.chat.Focused()
			isMessagingFocused := m.tabs.Active() == components.TabMessaging && m.messaging.Focused()
			if !isChatFocused && !isMessagingFocused {
				m.quitting = true
				return m, tea.Quit
			}
		}

		// Tab-specific key handling
		switch m.tabs.Active() {
		case components.TabChat:
			// Only pass keys to chat component
			var cmd tea.Cmd
			m.chat, cmd = m.chat.Update(msg)
			cmds = append(cmds, cmd)

			// Check if user pressed Enter to send
			if key == "enter" && m.chat.Focused() {
				content := m.chat.Value()
				if content != "" {
					m.chat.ClearInput()
					m.chat.AddUserMessage(content)
					cmds = append(cmds, m.sendChatMessage(content))
				}
			}

		case components.TabLogs:
			var cmd tea.Cmd
			m.logs, cmd = m.logs.Update(msg)
			cmds = append(cmds, cmd)

		case components.TabTools:
			var cmd tea.Cmd
			m.tools, cmd = m.tools.Update(msg)
			cmds = append(cmds, cmd)

		case components.TabDashboard:
			var cmd tea.Cmd
			m.dashboard, cmd = m.dashboard.Update(msg)
			cmds = append(cmds, cmd)

		case components.TabMessaging:
			// Check if user pressed Ctrl+S or Ctrl+Enter to send
			if key == "ctrl+s" || key == "ctrl+enter" {
				topic := m.messaging.Topic()
				message := m.messaging.Message()
				if topic != "" && message != "" && !m.messaging.IsSending() {
					m.messaging.SetSending(true)
					cmds = append(cmds, m.sendMessage(topic, message))
				}
			} else if key == "ctrl+w" {
				// Toggle watch
				cmds = append(cmds, m.toggleWatch())
			} else if key == "ctrl+l" {
				// Clear all including watch
				m.messaging.ClearInputs()
				m.messaging.ClearWatch()
				if m.messagingHandler != nil {
					_ = m.messagingHandler.StopWatching()
				}
			} else {
				var cmd tea.Cmd
				m.messaging, cmd = m.messaging.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case EventMsg:
		// Handle events from the runner
		cmds = append(cmds, m.handleEvent(msg.Event))
		// Continue listening for events
		cmds = append(cmds, listenForEvents(m.eventBus))

	case ChatResponseMsg:
		// Handle chat response
		if msg.Error != nil {
			m.chat.AddErrorMessage(msg.Error.Error())
		} else {
			m.chat.AddAssistantMessage(msg.Content)
		}
		m.chat.SetSending(false)

	case SetChatHandlerMsg:
		// Set the chat handler from external source
		m.chatHandler = msg.Handler

	case SetMessagingHandlerMsg:
		// Set the messaging handler from external source
		m.messagingHandler = msg.Handler

	case MessagingResponseMsg:
		// Handle messaging response
		m.messaging.SetSending(false)
		m.messaging.SetResponse(string(msg.Response), msg.Error)

	case WatchStatusMsg:
		// Handle watch status change
		m.messaging.SetWatchStatus(msg.Topic, msg.Error)

	case WatchMessageMsg:
		// Handle incoming watch message
		m.messaging.AddWatchMessage(components.WatchMessage{
			Timestamp: msg.Timestamp,
			Content:   msg.Content,
		})
	}

	return m, tea.Batch(cmds...)
}

// handleEvent processes a runner event and updates the appropriate component.
func (m *Model) handleEvent(event runner.Event) tea.Cmd {
	switch e := event.(type) {
	case runner.StatusEvent:
		m.dashboard.SetConnected(e.Connected)
		m.dashboard.SetAgentID(e.AgentID)
		if e.Error != nil {
			m.dashboard.SetError(e.Error.Error())
		}

	case runner.MessageEvent:
		m.dashboard.AddMessage(components.Message{
			Time:      e.Time,
			Direction: components.MessageDirection(e.Direction),
			Topic:     e.Topic,
			Content:   e.Content,
			Model:     e.Model,
			Tokens:    e.Tokens,
		})
		// Track total tokens
		if e.Tokens > 0 {
			m.dashboard.AddTokens(e.Tokens)
		}

	case runner.ToolEvent:
		m.tools.AddEvent(components.ToolExecution{
			Time:     e.Time,
			Status:   components.ToolStatus(e.Status),
			Name:     e.Name,
			Args:     e.Args,
			Result:   e.Result,
			Error:    e.Error,
			Duration: e.Duration,
		})

	case runner.ToolsAvailableEvent:
		// Convert runner.ToolInfo to components.AvailableTool
		available := make([]components.AvailableTool, len(e.Tools))
		for i, t := range e.Tools {
			available[i] = components.AvailableTool{
				Name:        t.Name,
				Description: t.Description,
				Server:      t.Server,
			}
		}
		m.tools.SetAvailableTools(available)

	case runner.LogEvent:
		m.logs.AddLog(components.LogEntry{
			Time:    e.Time,
			Level:   components.LogLevel(e.Level),
			Message: e.Message,
			Attrs:   e.Attrs,
		})
	}
	return nil
}

// sendChatMessage sends a message via the chat handler asynchronously.
func (m Model) sendChatMessage(content string) tea.Cmd {
	return func() tea.Msg {
		if m.chatHandler == nil {
			return ChatResponseMsg{Error: fmt.Errorf("chat not available")}
		}
		m.chat.SetSending(true)
		response, model, tokens, err := m.chatHandler.DirectChat(content)
		return ChatResponseMsg{
			Content: response,
			Model:   model,
			Tokens:  tokens,
			Error:   err,
		}
	}
}

// sendMessage sends a message to a topic asynchronously.
func (m Model) sendMessage(topic, message string) tea.Cmd {
	return func() tea.Msg {
		if m.messagingHandler == nil {
			return MessagingResponseMsg{Error: fmt.Errorf("messaging not available")}
		}
		data := []byte(message)

		if m.messaging.Mode() == components.ModePublish {
			// Fire-and-forget
			err := m.messagingHandler.PublishMessage(topic, data)
			return MessagingResponseMsg{Error: err}
		}

		// Request mode - wait for reply
		resp, err := m.messagingHandler.RequestMessage(topic, data)
		return MessagingResponseMsg{
			Response: resp,
			Error:    err,
		}
	}
}

// toggleWatch toggles the watch subscription on the current watch topic.
func (m Model) toggleWatch() tea.Cmd {
	return func() tea.Msg {
		if m.messagingHandler == nil {
			return WatchStatusMsg{Error: fmt.Errorf("messaging not available")}
		}

		// If currently watching, stop
		if m.messaging.IsWatchActive() {
			err := m.messagingHandler.StopWatching()
			return WatchStatusMsg{Topic: "", Error: err}
		}

		// Otherwise, start watching
		topic := m.messaging.WatchTopicValue()
		if topic == "" {
			return WatchStatusMsg{Error: fmt.Errorf("no topic specified")}
		}

		// Create callback that sends messages to the TUI
		callback := func(timestamp time.Time, content string) {
			if m.program != nil {
				m.program.Send(WatchMessageMsg{
					Timestamp: timestamp,
					Content:   content,
				})
			}
		}

		err := m.messagingHandler.WatchTopic(topic, callback)
		if err != nil {
			return WatchStatusMsg{Error: err}
		}
		return WatchStatusMsg{Topic: topic, Error: nil}
	}
}

// Outer padding applied by model around all component content.
const outerPaddingH = 1 // Horizontal padding (each side)

// updateLayout updates component sizes based on current window size.
func (m *Model) updateLayout() {
	// Layout breakdown:
	// - Header: 1 line + separator line + 1 newline = 3 lines
	// - Tabs bar: 2 lines (content + border) + 1 margin = 3 lines
	// - Footer: 1 line + 1 newline before = 2 lines
	// Total chrome: 3 + 3 + 2 = 8 lines
	//
	// Model controls outer padding around component content.
	// Components receive inner dimensions and fill exactly that space.

	// Available space for components (after outer padding)
	// Chrome: header(3) + tabs(3) + footer(2) + debug line(1) = 9, plus buffer
	componentWidth := m.width - (outerPaddingH * 2)
	componentHeight := m.height - 12 // Extra buffer to prevent overflow

	if componentHeight < 5 {
		componentHeight = 5 // Minimum usable height
	}

	m.tabs.SetWidth(m.width)

	// All components get the same dimensions - they handle internal layout
	m.dashboard.SetSize(componentWidth, componentHeight)
	m.chat.SetSize(componentWidth, componentHeight)
	m.messaging.SetSize(componentWidth, componentHeight)
	m.logs.SetSize(componentWidth, componentHeight)
	m.tools.SetSize(componentWidth, componentHeight)
	m.help.SetSize(m.width, m.height)
}


// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if !m.ready {
		return "Initializing..."
	}

	// Show help overlay if active
	if m.showHelp {
		return m.help.View()
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Tabs
	b.WriteString(m.tabs.View())
	b.WriteString("\n")

	// Content - each component renders its own panel(s) via View()
	// Model wraps with outer padding for consistent spacing
	var content string
	switch m.tabs.Active() {
	case components.TabDashboard:
		content = m.dashboard.View()
	case components.TabChat:
		content = m.chat.View()
	case components.TabMessaging:
		content = m.messaging.View()
	case components.TabLogs:
		content = m.logs.View()
	case components.TabTools:
		content = m.tools.View()
	}

	// No padding - raw component output for debugging
	b.WriteString(content)

	// Footer
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

// renderHeader renders the top header bar.
func (m Model) renderHeader() string {
	title := styles.HeaderTitle.Render("athyr-agent: " + m.cfg.Agent.Name)

	statusIcon := "●"
	var statusStyle lipgloss.Style
	if m.dashboard.Connected() {
		statusStyle = styles.Connected
	} else {
		statusStyle = styles.Disconnected
	}
	status := statusStyle.Render(statusIcon)

	statusText := "Disconnected"
	if m.dashboard.Connected() {
		statusText = "Connected"
	}

	agentID := m.dashboard.AgentID()
	if len(agentID) > 12 {
		agentID = agentID[:12]
	}

	right := fmt.Sprintf("[%s %s] %s", status, statusText, styles.Muted.Render(agentID))

	// Calculate spacing
	spacing := m.width - lipgloss.Width(title) - lipgloss.Width(right) - 2
	if spacing < 0 {
		spacing = 0
	}

	headerLine := styles.Header.Render(title + strings.Repeat(" ", spacing) + right)

	// Add separator line
	separator := styles.Muted.Render(strings.Repeat("─", m.width))

	return headerLine + "\n" + separator
}

// renderFooter renders the bottom help bar.
func (m Model) renderFooter() string {
	var parts []string

	// Global shortcuts
	parts = append(parts, styles.FooterKey.Render("q")+styles.FooterDesc.Render(": quit"))

	// Tab-specific shortcuts
	switch m.tabs.Active() {
	case components.TabChat:
		if m.chat.Focused() {
			parts = append(parts, styles.FooterKey.Render("Enter")+styles.FooterDesc.Render(": send"))
			parts = append(parts, styles.FooterKey.Render("Esc")+styles.FooterDesc.Render(": unfocus"))
		} else {
			parts = append(parts, styles.FooterKey.Render("i")+styles.FooterDesc.Render(": focus input"))
			parts = append(parts, styles.FooterKey.Render("↑/↓")+styles.FooterDesc.Render(": scroll"))
		}
	case components.TabMessaging:
		if m.messaging.Focused() {
			parts = append(parts, styles.FooterKey.Render("←/→")+styles.FooterDesc.Render(": panel"))
			parts = append(parts, styles.FooterKey.Render("m")+styles.FooterDesc.Render(": mode"))
			parts = append(parts, styles.FooterKey.Render("Ctrl+S")+styles.FooterDesc.Render(": send"))
			parts = append(parts, styles.FooterKey.Render("Ctrl+W")+styles.FooterDesc.Render(": watch"))
			parts = append(parts, styles.FooterKey.Render("Esc")+styles.FooterDesc.Render(": unfocus"))
		} else {
			parts = append(parts, styles.FooterKey.Render("i")+styles.FooterDesc.Render(": focus"))
		}
	case components.TabTools:
		parts = append(parts, styles.FooterKey.Render("←/→")+styles.FooterDesc.Render(": panel"))
		parts = append(parts, styles.FooterKey.Render("↑/↓")+styles.FooterDesc.Render(": scroll"))
	case components.TabLogs, components.TabDashboard:
		parts = append(parts, styles.FooterKey.Render("↑/↓")+styles.FooterDesc.Render(": scroll"))
	}

	// Tab shortcuts
	parts = append(parts, styles.FooterKey.Render("Tab")+styles.FooterDesc.Render(": switch"))
	parts = append(parts, styles.FooterKey.Render("1-5")+styles.FooterDesc.Render(": tabs"))

	return styles.Footer.Render(strings.Join(parts, "  "))
}

// listenForEvents creates a command that waits for the next event from the bus.
func listenForEvents(eventBus runner.EventBus) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-eventBus.Events()
		if !ok {
			return nil // Channel closed
		}
		return EventMsg{Event: event}
	}
}
