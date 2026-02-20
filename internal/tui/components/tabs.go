package components

import (
	"strings"

	"github.com/athyr-tech/athyr-agent/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tab represents a single tab.
type Tab struct {
	Name string
	Key  string // Single character shortcut
}

// Tabs manages tab navigation.
type Tabs struct {
	tabs   []Tab
	active int
	width  int
}

// NewTabs creates a new Tabs component.
func NewTabs(tabs []Tab) Tabs {
	return Tabs{
		tabs:   tabs,
		active: 0,
	}
}

// SetWidth updates the component width.
func (t *Tabs) SetWidth(w int) {
	t.width = w
}

// Active returns the index of the active tab.
func (t Tabs) Active() int {
	return t.active
}

// ActiveTab returns the active tab.
func (t Tabs) ActiveTab() Tab {
	if t.active >= 0 && t.active < len(t.tabs) {
		return t.tabs[t.active]
	}
	return Tab{}
}

// SetActive sets the active tab by index.
func (t *Tabs) SetActive(idx int) {
	if idx >= 0 && idx < len(t.tabs) {
		t.active = idx
	}
}

// Next moves to the next tab.
func (t *Tabs) Next() {
	t.active = (t.active + 1) % len(t.tabs)
}

// Prev moves to the previous tab.
func (t *Tabs) Prev() {
	t.active--
	if t.active < 0 {
		t.active = len(t.tabs) - 1
	}
}

// Update handles key messages for tab navigation.
func (t *Tabs) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			t.Next()
		case "shift+tab":
			t.Prev()
		default:
			// Check for number keys (1-9)
			if len(msg.String()) == 1 {
				key := msg.String()
				for i, tab := range t.tabs {
					if tab.Key == key {
						t.active = i
						return nil
					}
				}
			}
		}
	}
	return nil
}

// View renders the tabs.
func (t Tabs) View() string {
	var tabs []string
	for i, tab := range t.tabs {
		var style lipgloss.Style
		if i == t.active {
			style = styles.TabActive
		} else {
			style = styles.TabInactive
		}
		tabs = append(tabs, style.Render("["+tab.Key+"] "+tab.Name))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	return styles.TabBar.Width(t.width).Render(row)
}

// DefaultTabs returns the standard TUI tabs.
func DefaultTabs() []Tab {
	return []Tab{
		{Name: "Dashboard", Key: "1"},
		{Name: "Chat", Key: "2"},
		{Name: "Messaging", Key: "3"},
		{Name: "Logs", Key: "4"},
		{Name: "Tools", Key: "5"},
	}
}

// TabIndex constants for easy reference.
const (
	TabDashboard  = 0
	TabChat       = 1
	TabMessaging  = 2
	TabLogs       = 3
	TabTools      = 4
)

// Help returns help text for tab navigation.
func (t Tabs) Help() string {
	var parts []string
	parts = append(parts, styles.FooterKey.Render("Tab")+styles.FooterDesc.Render(": switch"))
	for _, tab := range t.tabs {
		parts = append(parts, styles.FooterKey.Render(tab.Key)+styles.FooterDesc.Render(": "+strings.ToLower(tab.Name)))
	}
	return strings.Join(parts, "  ")
}
