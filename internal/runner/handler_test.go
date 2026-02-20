package runner

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/athyr-tech/athyr-agent/internal/config"

	"github.com/athyr-tech/athyr-sdk-go/pkg/athyr"
)

// mockAgent implements athyr.Agent for testing
type mockAgent struct {
	completeFunc func(ctx context.Context, req athyr.CompletionRequest) (*athyr.CompletionResponse, error)
	publishFunc  func(ctx context.Context, subject string, data []byte) error
	published    []publishCall
}

type publishCall struct {
	Subject string
	Data    []byte
}

func (m *mockAgent) Complete(ctx context.Context, req athyr.CompletionRequest) (*athyr.CompletionResponse, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, req)
	}
	return &athyr.CompletionResponse{Content: "mock response"}, nil
}

func (m *mockAgent) Publish(ctx context.Context, subject string, data []byte) error {
	m.published = append(m.published, publishCall{Subject: subject, Data: data})
	if m.publishFunc != nil {
		return m.publishFunc(ctx, subject, data)
	}
	return nil
}

// Stub implementations for other interface methods
func (m *mockAgent) Connect(ctx context.Context) error                        { return nil }
func (m *mockAgent) Close() error                                             { return nil }
func (m *mockAgent) AgentID() string                                          { return "mock-agent" }
func (m *mockAgent) Connected() bool                                          { return true }
func (m *mockAgent) State() athyr.ConnectionState                             { return athyr.StateConnected }
func (m *mockAgent) Subscribe(ctx context.Context, subject string, handler athyr.MessageHandler) (athyr.Subscription, error) {
	return nil, nil
}
func (m *mockAgent) QueueSubscribe(ctx context.Context, subject, queue string, handler athyr.MessageHandler) (athyr.Subscription, error) {
	return nil, nil
}
func (m *mockAgent) Request(ctx context.Context, subject string, data []byte) ([]byte, error) {
	return nil, nil
}
func (m *mockAgent) CompleteStream(ctx context.Context, req athyr.CompletionRequest, handler athyr.StreamHandler) error {
	return nil
}
func (m *mockAgent) Models(ctx context.Context) ([]athyr.Model, error) { return nil, nil }
func (m *mockAgent) CreateSession(ctx context.Context, profile athyr.SessionProfile, systemPrompt string) (*athyr.Session, error) {
	// Return a session with a generated ID
	return &athyr.Session{ID: "server-sess-" + profile.Type}, nil
}
func (m *mockAgent) GetSession(ctx context.Context, sessionID string) (*athyr.Session, error) {
	return nil, nil
}
func (m *mockAgent) DeleteSession(ctx context.Context, sessionID string) error { return nil }
func (m *mockAgent) AddHint(ctx context.Context, sessionID, hint string) error { return nil }
func (m *mockAgent) KV(bucket string) athyr.KVBucket                           { return nil }

func TestHandler_IncludesToolsInRequest(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: config.TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
		},
	}

	var capturedReq athyr.CompletionRequest
	agent := &mockAgent{
		completeFunc: func(ctx context.Context, req athyr.CompletionRequest) (*athyr.CompletionResponse, error) {
			capturedReq = req
			return &athyr.CompletionResponse{Content: "done"}, nil
		},
	}

	// Create MCP manager with a registered tool
	mcpMgr := NewMCPManager(nil)
	mcpMgr.RegisterTool("test-server", athyr.Tool{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  json.RawMessage(`{"type": "object"}`),
	})

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := newMessageHandler(cfg, agent, logger, mcpMgr, nil, nil)

	handler.Handle(athyr.SubscribeMessage{
		Subject: "input",
		Data:    []byte("hello"),
	})

	// Verify tools were included in request
	if len(capturedReq.Tools) != 1 {
		t.Fatalf("Request.Tools = %d, want 1", len(capturedReq.Tools))
	}
	if capturedReq.Tools[0].Name != "test_tool" {
		t.Errorf("Tool.Name = %v, want test_tool", capturedReq.Tools[0].Name)
	}
}

func TestHandler_ExecutesToolCalls(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: config.TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
		},
	}

	callCount := 0
	agent := &mockAgent{
		completeFunc: func(ctx context.Context, req athyr.CompletionRequest) (*athyr.CompletionResponse, error) {
			callCount++
			if callCount == 1 {
				// First call: return a tool call
				return &athyr.CompletionResponse{
					Content:      "",
					FinishReason: "tool_calls",
					ToolCalls: []athyr.ToolCall{
						{
							ID:        "call_1",
							Name:      "test_tool",
							Arguments: json.RawMessage(`{"input": "test"}`),
						},
					},
				}, nil
			}
			// Second call: return final response
			// Verify the tool result message is present
			hasToolResult := false
			for _, msg := range req.Messages {
				if msg.Role == "tool" && msg.ToolCallID == "call_1" {
					hasToolResult = true
					if msg.Content != "tool result" {
						t.Errorf("Tool result content = %v, want 'tool result'", msg.Content)
					}
				}
			}
			if !hasToolResult {
				t.Error("Expected tool result message in second request")
			}
			return &athyr.CompletionResponse{Content: "final answer"}, nil
		},
	}

	// Create MCP manager with a mock tool executor
	mcpMgr := NewMCPManager(nil)
	mcpMgr.RegisterTool("test-server", athyr.Tool{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  json.RawMessage(`{"type": "object"}`),
	})
	// Set up a mock tool executor that returns a fixed result
	mcpMgr.SetToolExecutor(func(ctx context.Context, name string, args json.RawMessage) (string, error) {
		return "tool result", nil
	})

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := newMessageHandler(cfg, agent, logger, mcpMgr, nil, nil)

	handler.Handle(athyr.SubscribeMessage{
		Subject: "input",
		Data:    []byte("use the tool"),
	})

	// Verify Complete was called twice (initial + after tool result)
	if callCount != 2 {
		t.Errorf("Complete called %d times, want 2", callCount)
	}

	// Verify response was published
	if len(agent.published) == 0 {
		t.Fatal("Expected response to be published")
	}
}

func TestHandler_LimitsToolIterations(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: config.TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
		},
	}

	callCount := 0
	agent := &mockAgent{
		completeFunc: func(ctx context.Context, req athyr.CompletionRequest) (*athyr.CompletionResponse, error) {
			callCount++
			// Always return tool calls (infinite loop scenario)
			return &athyr.CompletionResponse{
				Content:      "",
				FinishReason: "tool_calls",
				ToolCalls: []athyr.ToolCall{
					{
						ID:        "call_" + string(rune('0'+callCount)),
						Name:      "test_tool",
						Arguments: json.RawMessage(`{}`),
					},
				},
			}, nil
		},
	}

	mcpMgr := NewMCPManager(nil)
	mcpMgr.RegisterTool("test-server", athyr.Tool{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  json.RawMessage(`{"type": "object"}`),
	})
	mcpMgr.SetToolExecutor(func(ctx context.Context, name string, args json.RawMessage) (string, error) {
		return "result", nil
	})

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := newMessageHandler(cfg, agent, logger, mcpMgr, nil, nil)

	handler.Handle(athyr.SubscribeMessage{
		Subject: "input",
		Data:    []byte("loop forever"),
	})

	// Verify we stopped after max iterations (should be 10)
	if callCount > 10 {
		t.Errorf("Complete called %d times, should stop at 10", callCount)
	}
}

func TestHandler_ExtractsSessionIDFromMessage(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: config.TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
			Memory: config.MemoryConfig{
				Enabled: true,
			},
		},
	}

	var capturedReq athyr.CompletionRequest
	agent := &mockAgent{
		completeFunc: func(ctx context.Context, req athyr.CompletionRequest) (*athyr.CompletionResponse, error) {
			capturedReq = req
			return &athyr.CompletionResponse{Content: "response"}, nil
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := newMessageHandler(cfg, agent, logger, nil, nil, nil)

	// Send message with session_id in JSON payload
	msgData, _ := json.Marshal(map[string]any{
		"session_id": "sess-123",
		"content":    "hello",
	})

	handler.Handle(athyr.SubscribeMessage{
		Subject: "input",
		Data:    msgData,
	})

	// Verify server session ID was passed in request (agent creates session on server)
	// The mock returns "server-sess-rolling_window" as the session ID
	if capturedReq.SessionID != "server-sess-rolling_window" {
		t.Errorf("SessionID = %v, want server-sess-rolling_window", capturedReq.SessionID)
	}
	if !capturedReq.IncludeMemory {
		t.Error("IncludeMemory = false, want true")
	}
}

func TestHandler_WithoutMemoryDisabled(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: config.TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
			Memory: config.MemoryConfig{
				Enabled: false, // Memory disabled
			},
		},
	}

	var capturedReq athyr.CompletionRequest
	agent := &mockAgent{
		completeFunc: func(ctx context.Context, req athyr.CompletionRequest) (*athyr.CompletionResponse, error) {
			capturedReq = req
			return &athyr.CompletionResponse{Content: "response"}, nil
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := newMessageHandler(cfg, agent, logger, nil, nil, nil)

	// Send message with session_id, but memory is disabled
	msgData, _ := json.Marshal(map[string]any{
		"session_id": "sess-123",
		"content":    "hello",
	})

	handler.Handle(athyr.SubscribeMessage{
		Subject: "input",
		Data:    msgData,
	})

	// Session ID should NOT be passed when memory is disabled
	if capturedReq.SessionID != "" {
		t.Errorf("SessionID = %v, want empty (memory disabled)", capturedReq.SessionID)
	}
	if capturedReq.IncludeMemory {
		t.Error("IncludeMemory = true, want false (memory disabled)")
	}
}

func TestHandler_PlainTextMessageWithMemory(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: config.TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
			Memory: config.MemoryConfig{
				Enabled: true,
			},
		},
	}

	var capturedReq athyr.CompletionRequest
	agent := &mockAgent{
		completeFunc: func(ctx context.Context, req athyr.CompletionRequest) (*athyr.CompletionResponse, error) {
			capturedReq = req
			return &athyr.CompletionResponse{Content: "response"}, nil
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := newMessageHandler(cfg, agent, logger, nil, nil, nil)

	// Send plain text (not JSON) - no session ID available
	handler.Handle(athyr.SubscribeMessage{
		Subject: "input",
		Data:    []byte("plain text message"),
	})

	// No session ID should be set for plain text
	if capturedReq.SessionID != "" {
		t.Errorf("SessionID = %v, want empty for plain text", capturedReq.SessionID)
	}
	// But IncludeMemory should still be false since no session ID
	if capturedReq.IncludeMemory {
		t.Error("IncludeMemory = true, want false (no session ID)")
	}
}

func TestExtractRouteFrom(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "valid route_to",
			content: `{"route_to": "ticket.billing", "content": "classified"}`,
			want:    "ticket.billing",
		},
		{
			name:    "no route_to field",
			content: `{"content": "just some content"}`,
			want:    "",
		},
		{
			name:    "empty route_to",
			content: `{"route_to": "", "content": "empty route"}`,
			want:    "",
		},
		{
			name:    "not JSON",
			content: "This is just plain text",
			want:    "",
		},
		{
			name:    "invalid JSON",
			content: `{"route_to": broken`,
			want:    "",
		},
		{
			name: "markdown wrapped json",
			content: "```json\n{\"route_to\": \"ticket.technical\", \"category\": \"technical\"}\n```",
			want: "ticket.technical",
		},
		{
			name: "markdown wrapped without language",
			content: "```\n{\"route_to\": \"ticket.billing\"}\n```",
			want: "ticket.billing",
		},
		{
			name: "markdown with extra whitespace",
			content: "  ```json\n  {\"route_to\": \"ticket.billing\"}  \n```  ",
			want: "ticket.billing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRouteFrom(tt.content)
			if got != tt.want {
				t.Errorf("extractRouteFrom() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_DynamicRouting(t *testing.T) {
	// Track what was published
	var publishedTopics []string

	agent := &mockAgent{
		completeFunc: func(ctx context.Context, req athyr.CompletionRequest) (*athyr.CompletionResponse, error) {
			// Return JSON with route_to
			return &athyr.CompletionResponse{
				Content:      `{"route_to": "ticket.billing", "category": "billing", "summary": "test"}`,
				Model:        "gpt-4",
				FinishReason: "stop",
			}, nil
		},
		publishFunc: func(ctx context.Context, subject string, data []byte) error {
			publishedTopics = append(publishedTopics, subject)
			return nil
		},
	}

	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:         "classifier",
			Model:        "gpt-4",
			Instructions: "Classify tickets",
			Topics: config.TopicsConfig{
				Subscribe: []string{"ticket.new"},
				Publish:   []string{"ticket.unknown"}, // Default
				Routes: []config.RouteConfig{
					{Topic: "ticket.billing", Description: "Billing issues"},
					{Topic: "ticket.technical", Description: "Tech issues"},
				},
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := newMessageHandler(cfg, agent, logger, nil, nil, nil)

	// Send a message
	handler.Handle(athyr.SubscribeMessage{
		Subject: "ticket.new",
		Data:    []byte("My bill is wrong"),
	})

	// Should have published to ticket.billing (the routed topic), not ticket.unknown (default)
	if len(publishedTopics) != 1 {
		t.Fatalf("Expected 1 publish, got %d: %v", len(publishedTopics), publishedTopics)
	}
	if publishedTopics[0] != "ticket.billing" {
		t.Errorf("Published to %v, want ticket.billing", publishedTopics[0])
	}
}

func TestHandler_InvalidRouteUsesDefault(t *testing.T) {
	var publishedTopics []string

	agent := &mockAgent{
		completeFunc: func(ctx context.Context, req athyr.CompletionRequest) (*athyr.CompletionResponse, error) {
			// Return JSON with invalid route_to
			return &athyr.CompletionResponse{
				Content:      `{"route_to": "ticket.invalid", "content": "test"}`,
				Model:        "gpt-4",
				FinishReason: "stop",
			}, nil
		},
		publishFunc: func(ctx context.Context, subject string, data []byte) error {
			publishedTopics = append(publishedTopics, subject)
			return nil
		},
	}

	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:         "classifier",
			Model:        "gpt-4",
			Instructions: "Classify tickets",
			Topics: config.TopicsConfig{
				Subscribe: []string{"ticket.new"},
				Publish:   []string{"ticket.unknown"},
				Routes: []config.RouteConfig{
					{Topic: "ticket.billing", Description: "Billing"},
				},
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	handler := newMessageHandler(cfg, agent, logger, nil, nil, nil)

	handler.Handle(athyr.SubscribeMessage{
		Subject: "ticket.new",
		Data:    []byte("Test message"),
	})

	// Should fall back to default publish topic
	if len(publishedTopics) != 1 {
		t.Fatalf("Expected 1 publish, got %d", len(publishedTopics))
	}
	if publishedTopics[0] != "ticket.unknown" {
		t.Errorf("Published to %v, want ticket.unknown (default)", publishedTopics[0])
	}
}
