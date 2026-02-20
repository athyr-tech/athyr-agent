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

// LogLevel represents log severity.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// LogEntry represents a single log entry.
type LogEntry struct {
	Time    time.Time
	Level   LogLevel
	Message string
	Attrs   map[string]any
}

// Logs displays a scrollable list of log entries.
type Logs struct {
	entries  []LogEntry
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

// NewLogs creates a new Logs component.
func NewLogs() Logs {
	return Logs{
		entries: make([]LogEntry, 0),
	}
}

// SetSize updates the component size.
// w and h are the total panel dimensions.
func (l *Logs) SetSize(w, h int) {
	l.width = w
	l.height = h

	// Content inside panel: total height - panel overhead (4 lines)
	// Then subtract title (2 lines: title + blank line after)
	contentHeight := h - styles.PanelVerticalOverhead
	viewportHeight := contentHeight - 2
	if viewportHeight < 3 {
		viewportHeight = 3
	}

	// Viewport width: panel width - panel horizontal overhead
	viewportWidth := w - styles.PanelHorizontalOverhead

	if l.ready {
		l.viewport.Width = viewportWidth
		l.viewport.Height = viewportHeight
	} else {
		l.viewport = viewport.New(viewportWidth, viewportHeight)
		l.ready = true
	}
	l.updateContent()
}

// AddLog adds a new log entry.
func (l *Logs) AddLog(entry LogEntry) {
	l.entries = append(l.entries, entry)
	// Keep only last 1000 entries
	if len(l.entries) > 1000 {
		l.entries = l.entries[len(l.entries)-1000:]
	}
	l.updateContent()
	// Auto-scroll to bottom
	l.viewport.GotoBottom()
}

// Update handles key messages for scrolling.
func (l Logs) Update(msg tea.Msg) (Logs, tea.Cmd) {
	var cmd tea.Cmd
	l.viewport, cmd = l.viewport.Update(msg)
	return l, cmd
}

// updateContent rebuilds the viewport content from log entries.
func (l *Logs) updateContent() {
	if !l.ready {
		return
	}

	var lines []string
	for _, entry := range l.entries {
		line := l.formatEntry(entry)
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		lines = append(lines, styles.Muted.Render("No logs yet..."))
	}

	l.viewport.SetContent(strings.Join(lines, "\n"))
}

// formatEntry formats a single log entry for display.
func (l Logs) formatEntry(entry LogEntry) string {
	timestamp := styles.MessageTimestamp.Render(entry.Time.Format("15:04:05"))

	var levelStyle lipgloss.Style
	var levelStr string
	switch entry.Level {
	case LogLevelDebug:
		levelStyle = styles.LogDebug
		levelStr = "DBG"
	case LogLevelInfo:
		levelStyle = styles.LogInfo
		levelStr = "INF"
	case LogLevelWarn:
		levelStyle = styles.LogWarn
		levelStr = "WRN"
	case LogLevelError:
		levelStyle = styles.LogError
		levelStr = "ERR"
	}

	level := levelStyle.Render(levelStr)

	// Format message
	msg := entry.Message

	// Truncate if too long
	maxLen := l.width - 25
	if maxLen > 0 && len(msg) > maxLen {
		msg = msg[:maxLen-3] + "..."
	}

	result := fmt.Sprintf("%s %s %s", timestamp, level, msg)

	// Add key attributes if present
	if len(entry.Attrs) > 0 {
		var attrs []string
		for k, v := range entry.Attrs {
			attrs = append(attrs, fmt.Sprintf("%s=%v", k, v))
		}
		if len(attrs) > 0 {
			attrStr := strings.Join(attrs, " ")
			// Truncate attrs if too long
			if len(attrStr) > 40 {
				attrStr = attrStr[:37] + "..."
			}
			result += " " + styles.Muted.Render(attrStr)
		}
	}

	return result
}

// Content returns the inner content without panel wrapper.
func (l Logs) Content() string {
	return l.viewport.View()
}

// View renders the logs panel filling exactly width × height.
func (l Logs) View() string {
	title := styles.PanelTitle.Render("Logs")
	content := title + "\n" + l.viewport.View()
	return styles.Panel.Width(l.width).Height(l.height).Render(content)
}
