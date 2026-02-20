package tui

import (
	"context"
	"log/slog"

	"github.com/athyr-tech/athyr-agent/internal/runner"
)

// LogHandler is an slog.Handler that emits LogEvents to an EventBus.
// It also forwards logs to an underlying handler if provided.
type LogHandler struct {
	eventBus    runner.EventBus
	underlying  slog.Handler
	level       slog.Level
	attrs       []slog.Attr
	groupPrefix string
}

// NewLogHandler creates a new LogHandler that emits to the given EventBus.
// If underlying is provided, logs are also forwarded there.
func NewLogHandler(eventBus runner.EventBus, underlying slog.Handler, level slog.Level) *LogHandler {
	return &LogHandler{
		eventBus:   eventBus,
		underlying: underlying,
		level:      level,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *LogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle processes a log record.
func (h *LogHandler) Handle(ctx context.Context, r slog.Record) error {
	// Convert slog level to our LogLevel
	var level runner.LogLevel
	switch {
	case r.Level < slog.LevelInfo:
		level = runner.LogLevelDebug
	case r.Level < slog.LevelWarn:
		level = runner.LogLevelInfo
	case r.Level < slog.LevelError:
		level = runner.LogLevelWarn
	default:
		level = runner.LogLevelError
	}

	// Collect attributes
	attrs := make(map[string]any)
	for _, a := range h.attrs {
		key := a.Key
		if h.groupPrefix != "" {
			key = h.groupPrefix + "." + key
		}
		attrs[key] = a.Value.Any()
	}
	r.Attrs(func(a slog.Attr) bool {
		key := a.Key
		if h.groupPrefix != "" {
			key = h.groupPrefix + "." + key
		}
		attrs[key] = a.Value.Any()
		return true
	})

	// Emit event
	if h.eventBus != nil {
		h.eventBus.Send(runner.LogEvent{
			Time:    r.Time,
			Level:   level,
			Message: r.Message,
			Attrs:   attrs,
		})
	}

	// Forward to underlying handler if present
	if h.underlying != nil {
		return h.underlying.Handle(ctx, r)
	}

	return nil
}

// WithAttrs returns a new handler with the given attributes.
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	newAttrs = append(newAttrs, attrs...)

	var underlying slog.Handler
	if h.underlying != nil {
		underlying = h.underlying.WithAttrs(attrs)
	}

	return &LogHandler{
		eventBus:    h.eventBus,
		underlying:  underlying,
		level:       h.level,
		attrs:       newAttrs,
		groupPrefix: h.groupPrefix,
	}
}

// WithGroup returns a new handler with the given group name.
func (h *LogHandler) WithGroup(name string) slog.Handler {
	prefix := name
	if h.groupPrefix != "" {
		prefix = h.groupPrefix + "." + name
	}

	var underlying slog.Handler
	if h.underlying != nil {
		underlying = h.underlying.WithGroup(name)
	}

	return &LogHandler{
		eventBus:    h.eventBus,
		underlying:  underlying,
		level:       h.level,
		attrs:       h.attrs,
		groupPrefix: prefix,
	}
}

// NewTUILogger creates a logger that sends logs to both the TUI and stderr.
func NewTUILogger(eventBus runner.EventBus, level slog.Level) *slog.Logger {
	// We don't need a text handler in TUI mode - the TUI handles display
	handler := NewLogHandler(eventBus, nil, level)
	return slog.New(handler)
}

// NewTUILoggerWithFallback creates a logger that sends logs to the TUI
// and also to a text handler for debugging.
func NewTUILoggerWithFallback(eventBus runner.EventBus, level slog.Level, fallback slog.Handler) *slog.Logger {
	handler := NewLogHandler(eventBus, fallback, level)
	return slog.New(handler)
}

// Ensure LogHandler implements slog.Handler at compile time.
var _ slog.Handler = (*LogHandler)(nil)
