package tui

import (
	"fmt"

	"github.com/athyr-tech/athyr-agent/internal/config"
	"github.com/athyr-tech/athyr-agent/internal/runner"

	tea "github.com/charmbracelet/bubbletea"
)

// TUI wraps the Bubble Tea program and provides a high-level interface.
type TUI struct {
	program  *tea.Program
	model    *Model
	eventBus runner.EventBus
}

// Options configures the TUI.
type Options struct {
	Config           *config.Config
	EventBus         runner.EventBus
	ChatHandler      ChatHandler
	MessagingHandler MessagingHandler
	ServerAddr       string // Athyr server address
}

// New creates a new TUI instance.
func New(opts Options) (*TUI, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if opts.EventBus == nil {
		return nil, fmt.Errorf("event bus is required")
	}

	model := NewModel(opts.Config, opts.EventBus, opts.ServerAddr)
	if opts.ChatHandler != nil {
		model.SetChatHandler(opts.ChatHandler)
	}
	if opts.MessagingHandler != nil {
		model.SetMessagingHandler(opts.MessagingHandler)
	}

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Set the program reference on the model for async message sending
	model.SetProgram(program)

	return &TUI{
		program:  program,
		model:    &model,
		eventBus: opts.EventBus,
	}, nil
}

// Run starts the TUI and blocks until it exits.
func (t *TUI) Run() error {
	_, err := t.program.Run()
	return err
}

// Quit gracefully quits the TUI.
func (t *TUI) Quit() {
	t.program.Quit()
}

// Send sends a message to the TUI program.
// This can be used to inject events from outside.
func (t *TUI) Send(msg tea.Msg) {
	t.program.Send(msg)
}

// SetChatHandler sets the chat handler after creation.
// This sends a message through Bubble Tea's event loop to ensure
// the handler is set on the actual model instance being used.
func (t *TUI) SetChatHandler(h ChatHandler) {
	t.program.Send(SetChatHandlerMsg{Handler: h})
}

// SetMessagingHandler sets the messaging handler after creation.
// This sends a message through Bubble Tea's event loop to ensure
// the handler is set on the actual model instance being used.
func (t *TUI) SetMessagingHandler(h MessagingHandler) {
	t.program.Send(SetMessagingHandlerMsg{Handler: h})
}
