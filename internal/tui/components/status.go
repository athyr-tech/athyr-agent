package components

import (
	"fmt"
	"strings"

	"github.com/athyr-tech/athyr-agent/internal/tui/styles"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// RouteInfo holds a routing destination.
type RouteInfo struct {
	Topic       string
	Description string
}

// MemoryInfo holds memory/session configuration.
type MemoryInfo struct {
	Enabled       bool
	SessionPrefix string
	TTL           string
	ProfileType   string
	MaxTokens     int
}

// AgentInfo holds configuration details about the agent.
type AgentInfo struct {
	Name       string
	Model      string
	Server     string // Athyr server address
	Subscribe  []string
	Publish    []string
	Routes     []RouteInfo
	MCPServers []string
	Memory     MemoryInfo
}

// Status displays the connection status panel.
type Status struct {
	info        AgentInfo
	agentID     string
	connected   bool
	errorMsg    string
	totalTokens int
	width       int
	height      int
	viewport    viewport.Model
	ready       bool
}

// NewStatus creates a new Status component.
func NewStatus(info AgentInfo) Status {
	return Status{
		info: info,
	}
}

// SetSize updates the component size.
func (s *Status) SetSize(w, h int) {
	s.width = w
	s.height = h

	// Content inside panel: total height - panel overhead (4 lines)
	// Then subtract title (2 lines: title + blank line after)
	contentHeight := h - styles.PanelVerticalOverhead
	viewportHeight := contentHeight - 2
	if viewportHeight < 3 {
		viewportHeight = 3
	}

	// Viewport width: panel width - panel horizontal overhead
	viewportWidth := w - styles.PanelHorizontalOverhead

	if s.ready {
		s.viewport.Width = viewportWidth
		s.viewport.Height = viewportHeight
	} else {
		s.viewport = viewport.New(viewportWidth, viewportHeight)
		s.ready = true
	}
	s.updateContent()
}

// SetConnected updates the connection status.
func (s *Status) SetConnected(connected bool) {
	s.connected = connected
	if connected {
		s.errorMsg = ""
	}
	s.updateContent()
}

// SetAgentID updates the agent ID.
func (s *Status) SetAgentID(id string) {
	s.agentID = id
	s.updateContent()
}

// SetError sets an error message.
func (s *Status) SetError(msg string) {
	s.errorMsg = msg
	s.updateContent()
}

// AddTokens adds tokens to the total count.
func (s *Status) AddTokens(count int) {
	s.totalTokens += count
	s.updateContent()
}

// TotalTokens returns the total token count.
func (s Status) TotalTokens() int {
	return s.totalTokens
}

// Connected returns the current connection status.
func (s Status) Connected() bool {
	return s.connected
}

// AgentID returns the current agent ID.
func (s Status) AgentID() string {
	return s.agentID
}

// updateContent rebuilds the viewport content.
func (s *Status) updateContent() {
	if !s.ready {
		return
	}

	var b strings.Builder

	// Connection status
	var statusIcon, statusText string
	var statusStyle lipgloss.Style

	if s.connected {
		statusIcon = "●"
		statusText = "Connected"
		statusStyle = styles.Connected
	} else {
		statusIcon = "○"
		statusText = "Disconnected"
		statusStyle = styles.Disconnected
	}

	b.WriteString(fmt.Sprintf("%s %s\n\n", statusStyle.Render(statusIcon), statusStyle.Render(statusText)))

	// Agent info
	labelStyle := styles.Muted
	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Agent:"), s.info.Name))

	if s.agentID != "" {
		// Truncate long IDs
		id := s.agentID
		if len(id) > 20 {
			id = id[:17] + "..."
		}
		b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("ID:"), id))
	}

	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Athyr:"), s.info.Server))
	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Model:"), s.info.Model))

	// Topics
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Subscribe:") + "\n")
	for _, topic := range s.info.Subscribe {
		b.WriteString(fmt.Sprintf("  %s\n", styles.MessageTopic.Render(topic)))
	}

	b.WriteString(labelStyle.Render("Publish:") + "\n")
	for _, topic := range s.info.Publish {
		b.WriteString(fmt.Sprintf("  %s\n", styles.MessageTopic.Render(topic)))
	}

	// Routes (if any)
	if len(s.info.Routes) > 0 {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Routes:") + "\n")
		for _, route := range s.info.Routes {
			b.WriteString(fmt.Sprintf("  %s\n", styles.MessageTopic.Render(route.Topic)))
			if route.Description != "" {
				desc := route.Description
				maxLen := s.width - 12
				if maxLen > 0 && len(desc) > maxLen {
					desc = desc[:maxLen-3] + "..."
				}
				b.WriteString(fmt.Sprintf("    %s\n", styles.Muted.Render(desc)))
			}
		}
	}

	// MCP Servers (if any)
	if len(s.info.MCPServers) > 0 {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("MCP Servers:") + "\n")
		for _, server := range s.info.MCPServers {
			b.WriteString(fmt.Sprintf("  %s\n", server))
		}
	}

	// Memory/Session info
	b.WriteString("\n")
	if s.info.Memory.Enabled {
		b.WriteString(fmt.Sprintf("%s %s\n", styles.Connected.Render("●"), labelStyle.Render("Memory Enabled")))
		if s.info.Memory.SessionPrefix != "" {
			b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Prefix:"), s.info.Memory.SessionPrefix))
		}
		if s.info.Memory.TTL != "" {
			b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("TTL:"), s.info.Memory.TTL))
		}
		if s.info.Memory.MaxTokens > 0 {
			b.WriteString(fmt.Sprintf("  %s %d\n", labelStyle.Render("Max Tokens:"), s.info.Memory.MaxTokens))
		}
	} else {
		b.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("○"), labelStyle.Render("Memory Disabled")))
	}

	// Token counter
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s %d\n", labelStyle.Render("Total Tokens:"), s.totalTokens))

	// Error message
	if s.errorMsg != "" {
		// Truncate long errors
		errMsg := s.errorMsg
		maxLen := s.width - 10
		if maxLen > 0 && len(errMsg) > maxLen {
			errMsg = errMsg[:maxLen-3] + "..."
		}
		b.WriteString(fmt.Sprintf("\n%s", styles.LogError.Render("Error: "+errMsg)))
	}

	s.viewport.SetContent(b.String())
}

// Content returns the inner content without panel wrapper.
func (s Status) Content() string {
	return s.viewport.View()
}

// View renders the status panel filling exactly width × height.
func (s Status) View() string {
	title := styles.PanelTitle.Render("Agent Info")
	content := title + "\n" + s.viewport.View()
	return styles.Panel.Width(s.width).Height(s.height).Render(content)
}
