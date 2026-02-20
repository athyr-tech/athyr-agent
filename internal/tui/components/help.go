package components

import (
	"strings"

	"github.com/athyr-tech/athyr-agent/internal/tui/styles"

	"github.com/charmbracelet/lipgloss"
)

// Help displays a keyboard shortcuts overlay.
type Help struct {
	width  int
	height int
}

// NewHelp creates a new Help component.
func NewHelp() Help {
	return Help{}
}

// SetSize updates the component size.
func (h *Help) SetSize(w, ht int) {
	h.width = w
	h.height = ht
}

// View renders the help overlay.
func (h Help) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorPrimary).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Foreground(styles.ColorAccent).
		Bold(true).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(styles.ColorAccent).
		Width(12)

	descStyle := lipgloss.NewStyle().
		Foreground(styles.ColorForeground)

	var b strings.Builder

	b.WriteString(titleStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n")

	// Global
	b.WriteString(sectionStyle.Render("Global"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("q") + descStyle.Render("Quit application") + "\n")
	b.WriteString(keyStyle.Render("Ctrl+C") + descStyle.Render("Force quit") + "\n")
	b.WriteString(keyStyle.Render("?") + descStyle.Render("Toggle this help") + "\n")

	// Navigation
	b.WriteString(sectionStyle.Render("Navigation"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("Tab") + descStyle.Render("Next tab") + "\n")
	b.WriteString(keyStyle.Render("Shift+Tab") + descStyle.Render("Previous tab") + "\n")
	b.WriteString(keyStyle.Render("1-5") + descStyle.Render("Jump to tab") + "\n")

	// Scrolling
	b.WriteString(sectionStyle.Render("Scrolling"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("↑/k") + descStyle.Render("Scroll up") + "\n")
	b.WriteString(keyStyle.Render("↓/j") + descStyle.Render("Scroll down") + "\n")
	b.WriteString(keyStyle.Render("PgUp") + descStyle.Render("Page up") + "\n")
	b.WriteString(keyStyle.Render("PgDn") + descStyle.Render("Page down") + "\n")

	// Chat
	b.WriteString(sectionStyle.Render("Chat Tab"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("i") + descStyle.Render("Focus input") + "\n")
	b.WriteString(keyStyle.Render("Esc") + descStyle.Render("Unfocus input") + "\n")
	b.WriteString(keyStyle.Render("Enter") + descStyle.Render("Send message") + "\n")

	// Messaging
	b.WriteString(sectionStyle.Render("Messaging Tab"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("i") + descStyle.Render("Focus input") + "\n")
	b.WriteString(keyStyle.Render("m") + descStyle.Render("Toggle Publish/Request mode") + "\n")
	b.WriteString(keyStyle.Render("Ctrl+N") + descStyle.Render("Next field") + "\n")
	b.WriteString(keyStyle.Render("Ctrl+S") + descStyle.Render("Send message") + "\n")
	b.WriteString(keyStyle.Render("Ctrl+Enter") + descStyle.Render("Send message (alt)") + "\n")
	b.WriteString(keyStyle.Render("Ctrl+L") + descStyle.Render("Clear inputs") + "\n")
	b.WriteString(keyStyle.Render("↑/↓") + descStyle.Render("Navigate topic dropdown") + "\n")
	b.WriteString(keyStyle.Render("Esc") + descStyle.Render("Unfocus") + "\n")

	// Tools
	b.WriteString(sectionStyle.Render("Tools Tab"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("←/h") + descStyle.Render("Focus left panel") + "\n")
	b.WriteString(keyStyle.Render("→/l") + descStyle.Render("Focus right panel") + "\n")

	// Footer
	b.WriteString("\n")
	b.WriteString(styles.Muted.Render("Press any key to close"))

	// Center the content in a box
	content := b.String()

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(1, 3).
		Width(50)

	box := boxStyle.Render(content)

	// Center on screen
	return lipgloss.Place(
		h.width,
		h.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}
