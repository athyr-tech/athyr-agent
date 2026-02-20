package runner

import "time"

// EventType identifies the kind of event.
type EventType int

const (
	EventTypeStatus EventType = iota
	EventTypeMessage
	EventTypeTool
	EventTypeLog
)

// Event is the base interface for all events emitted by the runner.
type Event interface {
	Type() EventType
	Timestamp() time.Time
}

// StatusEvent indicates a change in connection status.
type StatusEvent struct {
	Time      time.Time
	Connected bool
	AgentID   string
	AgentName string
	Error     error
}

func (e StatusEvent) Type() EventType      { return EventTypeStatus }
func (e StatusEvent) Timestamp() time.Time { return e.Time }

// MessageDirection indicates whether a message is incoming or outgoing.
type MessageDirection int

const (
	MessageIncoming MessageDirection = iota
	MessageOutgoing
)

// MessageEvent represents a message sent or received.
type MessageEvent struct {
	Time      time.Time
	Direction MessageDirection
	Topic     string
	Content   string
	Model     string
	Tokens    int
}

func (e MessageEvent) Type() EventType      { return EventTypeMessage }
func (e MessageEvent) Timestamp() time.Time { return e.Time }

// ToolStatus indicates the state of a tool execution.
type ToolStatus int

const (
	ToolStarted ToolStatus = iota
	ToolCompleted
	ToolFailed
)

// ToolEvent represents a tool execution.
type ToolEvent struct {
	Time     time.Time
	Status   ToolStatus
	Name     string
	Args     string
	Result   string
	Error    error
	Duration time.Duration
}

func (e ToolEvent) Type() EventType      { return EventTypeTool }
func (e ToolEvent) Timestamp() time.Time { return e.Time }

// ToolInfo describes an available tool.
type ToolInfo struct {
	Name        string
	Description string
	Server      string // MCP server name
}

// ToolsAvailableEvent is emitted when MCP tools are discovered.
type ToolsAvailableEvent struct {
	Time  time.Time
	Tools []ToolInfo
}

func (e ToolsAvailableEvent) Type() EventType      { return EventTypeTool }
func (e ToolsAvailableEvent) Timestamp() time.Time { return e.Time }

// LogLevel mirrors slog levels for the TUI.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// LogEvent represents a log entry.
type LogEvent struct {
	Time    time.Time
	Level   LogLevel
	Message string
	Attrs   map[string]any
}

func (e LogEvent) Type() EventType      { return EventTypeLog }
func (e LogEvent) Timestamp() time.Time { return e.Time }

// EventBus receives events from the runner for TUI consumption.
type EventBus interface {
	// Send sends an event to the bus. Non-blocking; drops if buffer full.
	Send(event Event)
	// Events returns the channel to receive events from.
	Events() <-chan Event
	// Close closes the event bus.
	Close()
}

// ChannelEventBus is the default EventBus implementation using a buffered channel.
type ChannelEventBus struct {
	ch     chan Event
	closed bool
}

// NewEventBus creates a new ChannelEventBus with the given buffer size.
func NewEventBus(bufferSize int) *ChannelEventBus {
	return &ChannelEventBus{
		ch: make(chan Event, bufferSize),
	}
}

// Send sends an event to the bus. Non-blocking; drops if buffer full.
func (b *ChannelEventBus) Send(event Event) {
	if b.closed {
		return
	}
	select {
	case b.ch <- event:
	default:
		// Buffer full, drop event to avoid blocking
	}
}

// Events returns the channel to receive events from.
func (b *ChannelEventBus) Events() <-chan Event {
	return b.ch
}

// Close closes the event bus.
func (b *ChannelEventBus) Close() {
	if !b.closed {
		b.closed = true
		close(b.ch)
	}
}
