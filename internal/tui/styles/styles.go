package styles

import "github.com/charmbracelet/lipgloss"

// Colors used throughout the TUI.
var (
	ColorPrimary    = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary  = lipgloss.Color("#10B981") // Green
	ColorAccent     = lipgloss.Color("#F59E0B") // Amber
	ColorError      = lipgloss.Color("#EF4444") // Red
	ColorMuted      = lipgloss.Color("#6B7280") // Gray
	ColorBackground = lipgloss.Color("#1F2937") // Dark gray
	ColorForeground = lipgloss.Color("#F9FAFB") // Light gray
	ColorBorder     = lipgloss.Color("#374151") // Gray border
)

// Status indicator styles.
var (
	Connected = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	Disconnected = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	Pending = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)
)

// Tab styles.
var (
	TabActive = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 2).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(ColorPrimary)

	TabInactive = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 2)

	TabBar = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		BorderBottom(true).
		MarginBottom(1)
)

// Panel styles.
var (
	Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	PanelTitle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			MarginBottom(1)

	PanelFocused = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)
)

// Header styles.
var (
	Header = lipgloss.NewStyle().
		Foreground(ColorForeground).
		Background(ColorBackground).
		Bold(true).
		Padding(0, 1)

	HeaderTitle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	HeaderStatus = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// Footer (help bar) styles.
var (
	Footer = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Padding(0, 1)

	FooterKey = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	FooterDesc = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// Muted is a style for muted/secondary text.
var Muted = lipgloss.NewStyle().Foreground(ColorMuted)

// Message styles.
var (
	MessageIncoming = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	MessageOutgoing = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	MessageTimestamp = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Width(10)

	MessageTopic = lipgloss.NewStyle().
			Foreground(ColorAccent)
)

// Log styles by level.
var (
	LogDebug = lipgloss.NewStyle().
			Foreground(ColorMuted)

	LogInfo = lipgloss.NewStyle().
		Foreground(ColorForeground)

	LogWarn = lipgloss.NewStyle().
		Foreground(ColorAccent)

	LogError = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)
)

// Tool styles.
var (
	ToolName = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	ToolRunning = lipgloss.NewStyle().
			Foreground(ColorAccent)

	ToolSuccess = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	ToolFailed = lipgloss.NewStyle().
			Foreground(ColorError)
)

// Chat styles.
var (
	ChatUser = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	ChatAssistant = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	ChatInput = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	ChatInputFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)
)

// Spinner style.
var Spinner = lipgloss.NewStyle().
	Foreground(ColorAccent)

// PanelOverhead returns the total lines/chars consumed by Panel borders and padding.
// Panel has Border (1 char each side) + Padding (1 line top/bottom, 2 chars left/right)
const (
	PanelVerticalOverhead   = 4 // 2 (border) + 2 (padding top/bottom)
	PanelHorizontalOverhead = 6 // 2 (border) + 4 (padding left/right)
)
