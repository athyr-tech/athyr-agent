package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/athyr-tech/athyr-agent/internal/config"
	"github.com/athyr-tech/athyr-agent/internal/plugin"

	"github.com/athyr-tech/athyr-sdk-go/pkg/athyr"
	"github.com/google/uuid"
)

const maxToolIterations = 10

// WatchCallback is called when a message is received on a watched topic.
type WatchCallback func(timestamp time.Time, content string)

// MessageHandler processes incoming messages through the LLM.
type MessageHandler struct {
	cfg      *config.Config
	agent    athyr.Agent
	logger   *slog.Logger
	mcp      *MCPManager
	plugins  *plugin.Manager
	eventBus EventBus
	sessions map[string]string // user session ID -> server session ID

	// Watch subscription state
	watchSub   athyr.Subscription
	watchTopic string
}

func newMessageHandler(cfg *config.Config, agent athyr.Agent, logger *slog.Logger, mcp *MCPManager, plugins *plugin.Manager, eventBus EventBus) *MessageHandler {
	return &MessageHandler{
		cfg:      cfg,
		agent:    agent,
		logger:   logger,
		mcp:      mcp,
		plugins:  plugins,
		eventBus: eventBus,
		sessions: make(map[string]string),
	}
}

// emitEvent sends an event to the EventBus if one is configured.
func (h *MessageHandler) emitEvent(event Event) {
	if h.eventBus != nil {
		h.eventBus.Send(event)
	}
}

// IncomingMessage represents a structured message with optional session info.
type IncomingMessage struct {
	SessionID string `json:"session_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

// parseMessage extracts session ID and content from incoming data.
// If data is JSON with session_id field, extracts it; otherwise treats as plain text.
func parseMessage(data []byte) (sessionID, content string) {
	var msg IncomingMessage
	if err := json.Unmarshal(data, &msg); err == nil && msg.Content != "" {
		return msg.SessionID, msg.Content
	}
	// Plain text or malformed JSON - use raw data as content
	return "", string(data)
}

// Handle processes a single incoming message.
func (h *MessageHandler) Handle(msg athyr.SubscribeMessage) {
	// Generate trace_id for correlating all logs for this request
	traceID := uuid.New().String()[:8] // Short ID for readability
	startTime := time.Now()

	h.logger.Info("message received",
		"trace_id", traceID,
		"topic", msg.Subject,
		"size_bytes", len(msg.Data),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Parse message to extract session ID and content
	userSessionID, content := parseMessage(msg.Data)

	// Emit incoming message event
	h.emitEvent(MessageEvent{
		Time:      time.Now(),
		Direction: MessageIncoming,
		Topic:     msg.Subject,
		Content:   content,
	})

	// Resolve session ID - create session if needed and get server-side ID
	var serverSessionID string
	if h.cfg.Agent.Memory.Enabled && userSessionID != "" {
		serverSessionID = h.ensureSession(ctx, userSessionID)
	}

	// Build messages for completion
	messages := []athyr.Message{}

	// Add system instructions if configured
	if h.cfg.Agent.Instructions != "" {
		systemPrompt := h.cfg.Agent.Instructions

		// Append routing instructions if routes are configured
		if h.cfg.Agent.Topics.HasRoutes() {
			systemPrompt += h.cfg.Agent.Topics.BuildRoutingPrompt()
		}

		messages = append(messages, athyr.Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Add the incoming message as user content
	messages = append(messages, athyr.Message{
		Role:    "user",
		Content: content,
	})

	// Get available tools from MCP manager
	var tools []athyr.Tool
	if h.mcp != nil {
		tools = h.mcp.GetAthyrTools()
		h.logger.Debug("tools available",
			"trace_id", traceID,
			"count", len(tools),
		)
	}

	// Tool-calling loop
	var resp *athyr.CompletionResponse
	for i := 0; i < maxToolIterations; i++ {
		// Create completion request
		req := athyr.CompletionRequest{
			Model:    h.cfg.Agent.Model,
			Messages: messages,
			Tools:    tools,
			Config: athyr.CompletionConfig{
				Temperature: 0.7,
				MaxTokens:   2048,
			},
		}
		if len(tools) > 0 {
			req.ToolChoice = "auto"
		}

		// Add session context if memory is enabled and session ID is provided
		if h.cfg.Agent.Memory.Enabled && serverSessionID != "" {
			req.SessionID = serverSessionID
			req.IncludeMemory = true
			h.logger.Info("using session memory", "user_session_id", userSessionID, "server_session_id", serverSessionID)
		}

		// Execute LLM completion
		llmStart := time.Now()
		h.logger.Debug("llm request",
			"trace_id", traceID,
			"model", req.Model,
			"iteration", i+1,
		)

		var err error
		resp, err = h.agent.Complete(ctx, req)
		llmLatency := time.Since(llmStart)

		if err != nil {
			h.logger.Error("llm failed",
				"trace_id", traceID,
				"error", err.Error(),
				"model", req.Model,
				"latency_ms", llmLatency.Milliseconds(),
			)
			return
		}

		h.logger.Info("llm completed",
			"trace_id", traceID,
			"model", resp.Model,
			"tokens_in", resp.Usage.PromptTokens,
			"tokens_out", resp.Usage.CompletionTokens,
			"latency_ms", llmLatency.Milliseconds(),
		)

		// If no tool calls, we're done
		if len(resp.ToolCalls) == 0 {
			break
		}

		h.logger.Debug("executing tool calls",
			"trace_id", traceID,
			"count", len(resp.ToolCalls),
		)

		// Add assistant's tool request to history
		messages = append(messages, athyr.Message{
			Role:      "assistant",
			ToolCalls: resp.ToolCalls,
		})

		// Execute each tool call
		for _, call := range resp.ToolCalls {
			// Emit tool started event
			toolStart := time.Now()
			argsStr := string(call.Arguments)
			h.emitEvent(ToolEvent{
				Time:   toolStart,
				Status: ToolStarted,
				Name:   call.Name,
				Args:   argsStr,
			})

			result, err := h.executeToolCall(ctx, call)
			toolDuration := time.Since(toolStart)

			if err != nil {
				h.logger.Error("tool failed",
					"trace_id", traceID,
					"tool", call.Name,
					"error", err.Error(),
					"latency_ms", toolDuration.Milliseconds(),
				)
				h.emitEvent(ToolEvent{
					Time:     time.Now(),
					Status:   ToolFailed,
					Name:     call.Name,
					Args:     argsStr,
					Error:    err,
					Duration: toolDuration,
				})
				result = fmt.Sprintf(`{"error": "%s"}`, err.Error())
			} else {
				h.logger.Info("tool executed",
					"trace_id", traceID,
					"tool", call.Name,
					"server", h.mcp.GetServerForTool(call.Name),
					"latency_ms", toolDuration.Milliseconds(),
					"success", true,
				)
				h.emitEvent(ToolEvent{
					Time:     time.Now(),
					Status:   ToolCompleted,
					Name:     call.Name,
					Args:     argsStr,
					Result:   result,
					Duration: toolDuration,
				})
			}

			// Add tool result to messages
			messages = append(messages, athyr.Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    result,
			})
		}
	}

	if resp == nil {
		h.logger.Error("no response after tool loop",
			"trace_id", traceID,
			"topic", msg.Subject,
		)
		return
	}

	// Check for dynamic routing in LLM response
	routeTo := extractRouteFrom(resp.Content)
	if routeTo != "" && h.cfg.Agent.Topics.IsValidRoute(routeTo) {
		h.logger.Debug("routing response",
			"trace_id", traceID,
			"route_to", routeTo,
		)
	} else if routeTo != "" {
		h.logger.Warn("invalid route_to, using default publish",
			"trace_id", traceID,
			"route_to", routeTo,
		)
		routeTo = "" // Reset to use default
	}

	// Publish response to configured output topics
	response := Response{
		Content:      resp.Content,
		Model:        resp.Model,
		SourceTopic:  msg.Subject,
		Tokens:       resp.Usage.TotalTokens,
		FinishReason: resp.FinishReason,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		h.logger.Error("failed to marshal response", "error", err)
		return
	}

	// Determine target topics
	var targetTopics []string
	if routeTo != "" {
		// Dynamic routing - publish to specified route only
		targetTopics = []string{routeTo}
	} else {
		// Default - publish to all configured output topics
		targetTopics = h.cfg.Agent.Topics.Publish
	}

	for _, topic := range targetTopics {
		var pubErr error
		if h.plugins != nil && h.plugins.IsPlugin(topic) {
			// Plugin destination: publish via plugin manager
			pubErr = h.plugins.Publish(topic, resp.Content)
		} else {
			// Athyr topic: publish via SDK agent
			pubErr = h.agent.Publish(ctx, topic, responseData)
		}

		if pubErr != nil {
			h.logger.Error("message send failed",
				"trace_id", traceID,
				"topic", topic,
				"error", pubErr.Error(),
			)
		} else {
			h.logger.Info("message sent",
				"trace_id", traceID,
				"topic", topic,
				"size_bytes", len(responseData),
			)
			// Emit outgoing message event
			h.emitEvent(MessageEvent{
				Time:      time.Now(),
				Direction: MessageOutgoing,
				Topic:     topic,
				Content:   resp.Content,
				Model:     resp.Model,
				Tokens:    resp.Usage.TotalTokens,
			})
		}
	}

	// If there's a reply subject (request/reply pattern), respond directly
	if msg.Reply != "" {
		if err := h.agent.Publish(ctx, msg.Reply, responseData); err != nil {
			h.logger.Error("reply failed",
				"trace_id", traceID,
				"reply", msg.Reply,
				"error", err.Error(),
			)
		}
	}

	// Log request completion with total duration
	h.logger.Debug("request completed",
		"trace_id", traceID,
		"total_ms", time.Since(startTime).Milliseconds(),
	)
}

// executeToolCall executes a single tool call via the MCP manager.
func (h *MessageHandler) executeToolCall(ctx context.Context, call athyr.ToolCall) (string, error) {
	if h.mcp == nil {
		return "", fmt.Errorf("no MCP manager configured")
	}
	return h.mcp.CallTool(ctx, call.Name, call.Arguments)
}

// ensureSession creates a session if it doesn't exist and returns the server session ID.
func (h *MessageHandler) ensureSession(ctx context.Context, userSessionID string) string {
	// Check if we already have a mapping
	if serverID, ok := h.sessions[userSessionID]; ok {
		return serverID
	}

	// Create new session on the server
	h.logger.Info("creating session", "user_session_id", userSessionID)

	profile := h.cfg.Agent.Memory.GetProfile()
	session, err := h.agent.CreateSession(ctx, athyr.SessionProfile{
		Type:                   profile.Type,
		MaxTokens:              profile.MaxTokens,
		SummarizationThreshold: profile.SummarizationThreshold,
	}, h.cfg.Agent.Instructions)

	if err != nil {
		h.logger.Error("failed to create session", "user_session_id", userSessionID, "error", err)
		return ""
	}

	// Store the mapping
	h.sessions[userSessionID] = session.ID
	h.logger.Info("session created", "user_session_id", userSessionID, "server_session_id", session.ID)

	return session.ID
}

// DirectChat sends a message directly to the LLM and returns the response.
// This is used by the TUI for interactive chat without going through pub/sub.
func (h *MessageHandler) DirectChat(content string) (response string, model string, tokens int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Build messages for completion
	messages := []athyr.Message{}

	// Add system instructions if configured
	if h.cfg.Agent.Instructions != "" {
		messages = append(messages, athyr.Message{
			Role:    "system",
			Content: h.cfg.Agent.Instructions,
		})
	}

	// Add the user message
	messages = append(messages, athyr.Message{
		Role:    "user",
		Content: content,
	})

	// Get available tools from MCP manager
	var tools []athyr.Tool
	if h.mcp != nil {
		tools = h.mcp.GetAthyrTools()
	}

	// Tool-calling loop
	var resp *athyr.CompletionResponse
	for i := 0; i < maxToolIterations; i++ {
		// Create completion request
		req := athyr.CompletionRequest{
			Model:    h.cfg.Agent.Model,
			Messages: messages,
			Tools:    tools,
			Config: athyr.CompletionConfig{
				Temperature: 0.7,
				MaxTokens:   2048,
			},
		}
		if len(tools) > 0 {
			req.ToolChoice = "auto"
		}

		// Execute LLM completion
		resp, err = h.agent.Complete(ctx, req)
		if err != nil {
			return "", "", 0, fmt.Errorf("completion failed: %w", err)
		}

		// If no tool calls, we're done
		if len(resp.ToolCalls) == 0 {
			break
		}

		// Add assistant's tool request to history
		messages = append(messages, athyr.Message{
			Role:      "assistant",
			ToolCalls: resp.ToolCalls,
		})

		// Execute each tool call
		for _, call := range resp.ToolCalls {
			toolStart := time.Now()
			argsStr := string(call.Arguments)
			h.emitEvent(ToolEvent{
				Time:   toolStart,
				Status: ToolStarted,
				Name:   call.Name,
				Args:   argsStr,
			})

			result, execErr := h.executeToolCall(ctx, call)
			toolDuration := time.Since(toolStart)

			if execErr != nil {
				h.emitEvent(ToolEvent{
					Time:     time.Now(),
					Status:   ToolFailed,
					Name:     call.Name,
					Args:     argsStr,
					Error:    execErr,
					Duration: toolDuration,
				})
				result = fmt.Sprintf(`{"error": "%s"}`, execErr.Error())
			} else {
				h.emitEvent(ToolEvent{
					Time:     time.Now(),
					Status:   ToolCompleted,
					Name:     call.Name,
					Args:     argsStr,
					Result:   result,
					Duration: toolDuration,
				})
			}

			// Add tool result to messages
			messages = append(messages, athyr.Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    result,
			})
		}
	}

	if resp == nil {
		return "", "", 0, fmt.Errorf("no response from LLM")
	}

	return resp.Content, resp.Model, resp.Usage.TotalTokens, nil
}

// Response is the structure published to output topics.
type Response struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	SourceTopic  string `json:"source_topic"`
	Tokens       int    `json:"tokens"`
	FinishReason string `json:"finish_reason"`
}

// routeResponse is used to parse the route_to field from LLM JSON output.
type routeResponse struct {
	RouteTo string `json:"route_to"`
}

// extractRouteFrom attempts to extract a route_to field from JSON content.
// Handles both raw JSON and markdown-wrapped JSON (```json ... ```).
// Returns empty string if not found or content is not valid JSON.
func extractRouteFrom(content string) string {
	// Try to extract JSON from markdown code blocks if present
	jsonContent := extractJSONFromMarkdown(content)

	var r routeResponse
	if err := json.Unmarshal([]byte(jsonContent), &r); err != nil {
		return ""
	}
	return r.RouteTo
}

// PublishMessage sends a message to a topic (fire-and-forget).
// This is used by the TUI to send messages without waiting for a response.
func (h *MessageHandler) PublishMessage(topic string, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h.logger.Debug("publishing message", "topic", topic, "size", len(data))

	// Emit outgoing message event for TUI visibility
	h.emitEvent(MessageEvent{
		Time:      time.Now(),
		Direction: MessageOutgoing,
		Topic:     topic,
		Content:   string(data),
	})

	return h.agent.Publish(ctx, topic, data)
}

// RequestMessage sends a message to a topic and waits for a reply.
// This uses the request/reply pattern with a 30 second timeout.
func (h *MessageHandler) RequestMessage(topic string, data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	h.logger.Debug("sending request", "topic", topic, "size", len(data))

	// Emit outgoing message event
	h.emitEvent(MessageEvent{
		Time:      time.Now(),
		Direction: MessageOutgoing,
		Topic:     topic,
		Content:   string(data),
	})

	resp, err := h.agent.Request(ctx, topic, data)
	if err != nil {
		return nil, err
	}

	// Emit incoming response event
	h.emitEvent(MessageEvent{
		Time:      time.Now(),
		Direction: MessageIncoming,
		Topic:     topic + ".reply",
		Content:   string(resp),
	})

	return resp, nil
}

// extractJSONFromMarkdown extracts JSON from markdown code blocks.
// If no code block is found, returns the original content.
func extractJSONFromMarkdown(content string) string {
	// Look for ```json or ``` followed by JSON
	content = strings.TrimSpace(content)

	// Check for ```json ... ``` pattern
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
		return strings.TrimSpace(content)
	}

	// Check for ``` ... ``` pattern (generic code block)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
		return strings.TrimSpace(content)
	}

	return content
}

// WatchTopic subscribes to a topic and calls the callback for each message.
// Only one topic can be watched at a time; calling this again will stop the previous watch.
func (h *MessageHandler) WatchTopic(topic string, callback WatchCallback) error {
	// Stop existing watch if any
	if err := h.StopWatching(); err != nil {
		h.logger.Warn("failed to stop previous watch", "error", err)
	}

	ctx := context.Background()

	h.logger.Debug("starting watch", "topic", topic)

	// Subscribe to the topic
	sub, err := h.agent.Subscribe(ctx, topic, func(msg athyr.SubscribeMessage) {
		callback(time.Now(), string(msg.Data))
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", topic, err)
	}

	h.watchSub = sub
	h.watchTopic = topic

	h.logger.Info("watching topic", "topic", topic)
	return nil
}

// StopWatching stops the current watch subscription if any.
func (h *MessageHandler) StopWatching() error {
	if h.watchSub == nil {
		return nil
	}

	h.logger.Debug("stopping watch", "topic", h.watchTopic)

	err := h.watchSub.Unsubscribe()
	h.watchSub = nil
	h.watchTopic = ""

	if err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}
	return nil
}

// WatchingTopic returns the currently watched topic, or empty string if not watching.
func (h *MessageHandler) WatchingTopic() string {
	return h.watchTopic
}
