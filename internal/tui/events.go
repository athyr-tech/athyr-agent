package tui

import (
	"time"

	"github.com/athyr-tech/athyr-agent/internal/runner"
)

// EventMsg wraps a runner.Event for Bubble Tea's message system.
type EventMsg struct {
	Event runner.Event
}

// ChatResponseMsg is sent when a direct chat response is received.
type ChatResponseMsg struct {
	Content string
	Model   string
	Tokens  int
	Error   error
}

// WindowSizeMsg wraps terminal size updates.
type WindowSizeMsg struct {
	Width  int
	Height int
}

// SetChatHandlerMsg is sent to set the chat handler from outside the event loop.
type SetChatHandlerMsg struct {
	Handler ChatHandler
}

// SetMessagingHandlerMsg is sent to set the messaging handler from outside the event loop.
type SetMessagingHandlerMsg struct {
	Handler MessagingHandler
}

// MessagingResponseMsg is sent when a messaging request response is received.
type MessagingResponseMsg struct {
	Response []byte
	Error    error
}

// WatchMessageMsg is sent when a message is received on a watched topic.
type WatchMessageMsg struct {
	Timestamp time.Time
	Content   string
}

// WatchStatusMsg is sent when watch status changes (started, stopped, error).
type WatchStatusMsg struct {
	Topic string // Empty if stopped
	Error error  // Non-nil if failed
}
