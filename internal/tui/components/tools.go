package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/athyr-tech/athyr-agent/internal/tui/styles"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ToolStatus indicates the state of a tool execution.
type ToolStatus int

const (
	ToolStarted ToolStatus = iota
	ToolCompleted
	ToolFailed
)

// ToolExecution represents a tool call and its result.
type ToolExecution struct {
	Time     time.Time
	Status   ToolStatus
	Name     string
	Args     string
	Result   string
	Error    error
	Duration time.Duration
}

// AvailableTool represents an available MCP tool.
type AvailableTool struct {
	Name        string
	Description string
	Server      string
}

// Tools displays available tools and execution history in a split view.
type Tools struct {
	available  []AvailableTool
	executions []ToolExecution

	// Left panel: available tools
	leftViewport viewport.Model
	leftWidth    int

	// Right panel: execution history
	rightViewport viewport.Model
	rightWidth    int

	// Layout
	width      int
	height     int
	ready      bool
	focusRight bool // which panel has focus for scrolling
}

// NewTools creates a new Tools component.
func NewTools() Tools {
	return Tools{
		available:  make([]AvailableTool, 0),
		executions: make([]ToolExecution, 0),
		focusRight: true, // Default focus on execution history
	}
}

// SetSize updates the component size.
func (t *Tools) SetSize(w, h int) {
	t.width = w
	t.height = h

	// Split width: reduce width to fit within bounds
	usableWidth := w - 2
	t.leftWidth = usableWidth / 2
	t.rightWidth = usableWidth - t.leftWidth

	// Content inside panel: total height - panel overhead (4 lines)
	// Then subtract title (2 lines: title + blank line after)
	contentHeight := h - styles.PanelVerticalOverhead
	viewportHeight := contentHeight - 2
	if viewportHeight < 3 {
		viewportHeight = 3
	}

	// Viewport width: panel width - panel horizontal overhead
	leftViewportWidth := t.leftWidth - styles.PanelHorizontalOverhead
	rightViewportWidth := t.rightWidth - styles.PanelHorizontalOverhead

	if t.ready {
		t.leftViewport.Width = leftViewportWidth
		t.leftViewport.Height = viewportHeight
		t.rightViewport.Width = rightViewportWidth
		t.rightViewport.Height = viewportHeight
	} else {
		t.leftViewport = viewport.New(leftViewportWidth, viewportHeight)
		t.rightViewport = viewport.New(rightViewportWidth, viewportHeight)
		t.ready = true
	}

	t.updateLeftContent()
	t.updateRightContent()
}

// SetAvailableTools sets the list of available tools.
func (t *Tools) SetAvailableTools(tools []AvailableTool) {
	t.available = tools
	t.updateLeftContent()
}

// AddEvent adds a new tool execution event.
func (t *Tools) AddEvent(exec ToolExecution) {
	// If this is an update (completed/failed), find and update the existing entry
	if exec.Status != ToolStarted {
		for i := len(t.executions) - 1; i >= 0; i-- {
			if t.executions[i].Name == exec.Name && t.executions[i].Status == ToolStarted {
				t.executions[i] = exec
				t.updateRightContent()
				return
			}
		}
	}
	// Otherwise add as new entry
	t.executions = append(t.executions, exec)
	// Keep only last 100 entries
	if len(t.executions) > 100 {
		t.executions = t.executions[len(t.executions)-100:]
	}
	t.updateRightContent()
	t.rightViewport.GotoBottom()
}

// Update handles key messages for scrolling.
func (t Tools) Update(msg tea.Msg) (Tools, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			t.focusRight = false
			return t, nil
		case "right", "l":
			t.focusRight = true
			return t, nil
		}
	}

	// Update the focused viewport
	if t.focusRight {
		t.rightViewport, cmd = t.rightViewport.Update(msg)
	} else {
		t.leftViewport, cmd = t.leftViewport.Update(msg)
	}

	return t, cmd
}

// updateLeftContent rebuilds the available tools panel content.
func (t *Tools) updateLeftContent() {
	if !t.ready {
		return
	}

	var lines []string

	if len(t.available) == 0 {
		lines = append(lines, styles.Muted.Render("No MCP tools configured"))
	} else {
		lines = append(lines, styles.Muted.Render(fmt.Sprintf("%d tools available", len(t.available))))
		lines = append(lines, "")

		for _, tool := range t.available {
			// Tool name
			lines = append(lines, styles.ToolName.Render(tool.Name))

			// Server
			lines = append(lines, styles.Muted.Render("  "+tool.Server))

			// Description (word wrapped)
			if tool.Description != "" {
				desc := tool.Description
				maxLen := t.leftWidth - 8
				if maxLen > 0 && len(desc) > maxLen {
					desc = desc[:maxLen-3] + "..."
				}
				lines = append(lines, styles.Muted.Render("  "+desc))
			}
			lines = append(lines, "") // Blank line between tools
		}
	}

	t.leftViewport.SetContent(strings.Join(lines, "\n"))
}

// updateRightContent rebuilds the execution history panel content.
func (t *Tools) updateRightContent() {
	if !t.ready {
		return
	}

	var lines []string

	if len(t.executions) == 0 {
		lines = append(lines, styles.Muted.Render("No tool executions yet"))
		lines = append(lines, "")
		lines = append(lines, styles.Muted.Render("Tool calls will appear here"))
		lines = append(lines, styles.Muted.Render("when the LLM uses tools."))
	} else {
		for _, exec := range t.executions {
			lines = append(lines, t.formatExecution(exec)...)
			lines = append(lines, "") // Blank line between executions
		}
	}

	t.rightViewport.SetContent(strings.Join(lines, "\n"))
}

// formatExecution formats a single tool execution for display.
func (t Tools) formatExecution(exec ToolExecution) []string {
	var lines []string

	timestamp := styles.MessageTimestamp.Render(exec.Time.Format("15:04:05"))

	var statusStyle lipgloss.Style
	var statusIcon string
	switch exec.Status {
	case ToolStarted:
		statusStyle = styles.ToolRunning
		statusIcon = "●"
	case ToolCompleted:
		statusStyle = styles.ToolSuccess
		statusIcon = "✓"
	case ToolFailed:
		statusStyle = styles.ToolFailed
		statusIcon = "✗"
	}

	toolName := styles.ToolName.Render(exec.Name)
	status := statusStyle.Render(statusIcon)

	// First line: timestamp, status, tool name
	header := fmt.Sprintf("%s %s %s", timestamp, status, toolName)
	if exec.Duration > 0 {
		header += styles.Muted.Render(fmt.Sprintf(" %s", exec.Duration.Round(time.Millisecond)))
	}
	lines = append(lines, header)

	// Args (truncated)
	if exec.Args != "" {
		args := exec.Args
		maxLen := t.rightWidth - 10
		if maxLen > 0 && len(args) > maxLen {
			args = args[:maxLen-3] + "..."
		}
		lines = append(lines, styles.Muted.Render("  "+args))
	}

	// Result or error (truncated)
	if exec.Status == ToolCompleted && exec.Result != "" {
		result := strings.ReplaceAll(exec.Result, "\n", " ")
		maxLen := t.rightWidth - 10
		if maxLen > 0 && len(result) > maxLen {
			result = result[:maxLen-3] + "..."
		}
		lines = append(lines, styles.ToolSuccess.Render("  ✓ ")+result)
	} else if exec.Status == ToolFailed && exec.Error != nil {
		errMsg := exec.Error.Error()
		maxLen := t.rightWidth - 10
		if maxLen > 0 && len(errMsg) > maxLen {
			errMsg = errMsg[:maxLen-3] + "..."
		}
		lines = append(lines, styles.ToolFailed.Render("  ✗ "+errMsg))
	}

	return lines
}

// View renders the tools panel filling exactly width × height.
func (t Tools) View() string {
	// Left panel: Available Tools
	leftTitle := "Available Tools"
	if !t.focusRight {
		leftTitle = "● " + leftTitle
	}
	leftPanel := styles.Panel.Width(t.leftWidth).Height(t.height).Render(
		styles.PanelTitle.Render(leftTitle) + "\n" + t.leftViewport.View(),
	)

	// Right panel: Execution History
	rightTitle := "Execution History"
	if t.focusRight {
		rightTitle = "● " + rightTitle
	}
	rightPanel := styles.Panel.Width(t.rightWidth).Height(t.height).Render(
		styles.PanelTitle.Render(rightTitle) + "\n" + t.rightViewport.View(),
	)

	// Join panels directly - no gap (borders provide separation)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

// LeftTitle returns the title for the left panel.
func (t Tools) LeftTitle() string {
	title := "Available Tools"
	if !t.focusRight {
		title = "● " + title
	}
	return title
}

// RightTitle returns the title for the right panel.
func (t Tools) RightTitle() string {
	title := "Execution History"
	if t.focusRight {
		title = "● " + title
	}
	return title
}

// LeftContent returns the content for the left panel (viewport content).
func (t Tools) LeftContent() string {
	return t.leftViewport.View()
}

// RightContent returns the content for the right panel (viewport content).
func (t Tools) RightContent() string {
	return t.rightViewport.View()
}
