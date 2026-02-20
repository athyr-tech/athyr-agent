package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"

	"github.com/athyr-tech/athyr-agent/internal/config"

	"github.com/athyr-tech/athyr-sdk-go/pkg/athyr"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolExecutor is a function that executes a tool call.
type ToolExecutor func(ctx context.Context, name string, args json.RawMessage) (string, error)

// MCPManager manages MCP server connections and tool execution.
type MCPManager struct {
	logger       *slog.Logger
	client       *mcp.Client
	sessions     map[string]*mcp.ClientSession // server name → session
	tools        map[string]athyr.Tool         // tool name → tool definition
	toolSrc      map[string]string             // tool name → server name
	mu           sync.RWMutex
	toolExecutor ToolExecutor // optional override for testing
}

// NewMCPManager creates a new MCP manager.
func NewMCPManager(logger *slog.Logger) *MCPManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &MCPManager{
		logger:   logger,
		client:   mcp.NewClient(&mcp.Implementation{Name: "athyr-agent", Version: "1.0.0"}, nil),
		sessions: make(map[string]*mcp.ClientSession),
		tools:    make(map[string]athyr.Tool),
		toolSrc:  make(map[string]string),
	}
}

// Start connects to all configured MCP servers and discovers their tools.
func (m *MCPManager) Start(ctx context.Context, servers []config.MCPServerConfig) error {
	for _, srv := range servers {
		if err := m.connectServer(ctx, srv); err != nil {
			return fmt.Errorf("failed to connect to MCP server %s: %w", srv.Name, err)
		}
	}
	return nil
}

// connectServer connects to a single MCP server and discovers its tools.
func (m *MCPManager) connectServer(ctx context.Context, srv config.MCPServerConfig) error {
	var (
		transport mcp.Transport
		session   *mcp.ClientSession
		err       error
	)

	if srv.URL != "" {
		m.logger.Info("connecting to MCP server via HTTP", "name", srv.Name, "url", srv.URL)
		transport = &mcp.StreamableClientTransport{Endpoint: srv.URL}
	} else {
		m.logger.Info("connecting to MCP server via stdio", "name", srv.Name, "command", srv.Command)

		cmd := exec.Command(srv.Command[0], srv.Command[1:]...)
		if len(srv.Env) > 0 {
			for k, v := range srv.Env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
		}
		transport = &mcp.CommandTransport{Command: cmd}
	}

	session, err = m.client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	m.mu.Lock()
	m.sessions[srv.Name] = session
	m.mu.Unlock()

	// Discover tools
	if err := m.discoverTools(ctx, srv.Name, session); err != nil {
		return fmt.Errorf("tool discovery failed: %w", err)
	}

	m.logger.Info("connected to MCP server", "name", srv.Name, "tools", len(m.tools))
	return nil
}

// discoverTools queries tools from an MCP server and registers them.
func (m *MCPManager) discoverTools(ctx context.Context, serverName string, session *mcp.ClientSession) error {
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			return err
		}

		// Convert MCP tool to athyr.Tool
		athyrTool := m.convertTool(tool)

		m.mu.Lock()
		m.tools[tool.Name] = athyrTool
		m.toolSrc[tool.Name] = serverName
		m.mu.Unlock()

		m.logger.Debug("discovered tool", "name", tool.Name, "server", serverName)
	}
	return nil
}

// convertTool converts an MCP tool to an athyr.Tool.
func (m *MCPManager) convertTool(tool *mcp.Tool) athyr.Tool {
	var params json.RawMessage
	if tool.InputSchema != nil {
		// InputSchema is typically map[string]any, marshal it to JSON
		data, err := json.Marshal(tool.InputSchema)
		if err == nil {
			params = data
		}
	}

	return athyr.Tool{
		Name:        tool.Name,
		Description: tool.Description,
		Parameters:  params,
	}
}

// RegisterTool manually registers a tool (useful for testing or shell tools).
func (m *MCPManager) RegisterTool(serverName string, tool athyr.Tool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[tool.Name] = tool
	m.toolSrc[tool.Name] = serverName
}

// SetToolExecutor sets a custom tool executor (useful for testing).
func (m *MCPManager) SetToolExecutor(executor ToolExecutor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolExecutor = executor
}

// GetAthyrTools returns all discovered tools in athyr.Tool format.
func (m *MCPManager) GetAthyrTools() []athyr.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]athyr.Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetServerForTool returns the server name that provides a given tool.
func (m *MCPManager) GetServerForTool(toolName string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.toolSrc[toolName]
}

// GetToolsInfo returns tool information including which server they came from.
func (m *MCPManager) GetToolsInfo() []ToolInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]ToolInfo, 0, len(m.tools))
	for name, tool := range m.tools {
		infos = append(infos, ToolInfo{
			Name:        name,
			Description: tool.Description,
			Server:      m.toolSrc[name],
		})
	}
	return infos
}

// CallTool executes a tool call and returns the result.
func (m *MCPManager) CallTool(ctx context.Context, name string, args json.RawMessage) (string, error) {
	m.mu.RLock()
	executor := m.toolExecutor
	serverName, ok := m.toolSrc[name]
	if !ok {
		m.mu.RUnlock()
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	session := m.sessions[serverName]
	m.mu.RUnlock()

	// Use custom executor if set (for testing)
	if executor != nil {
		return executor(ctx, name, args)
	}

	if session == nil {
		return "", fmt.Errorf("no session for server: %s", serverName)
	}

	// Parse arguments
	var arguments map[string]any
	if len(args) > 0 {
		if err := json.Unmarshal(args, &arguments); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}

	// Call the tool
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		return "", fmt.Errorf("tool call failed: %w", err)
	}

	// Extract text content from result
	return m.extractContent(result), nil
}

// extractContent extracts text from a CallToolResult.
func (m *MCPManager) extractContent(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}

	var text string
	for _, content := range result.Content {
		if tc, ok := content.(*mcp.TextContent); ok {
			if text != "" {
				text += "\n"
			}
			text += tc.Text
		}
	}
	return text
}

// Close shuts down all MCP server connections.
func (m *MCPManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, session := range m.sessions {
		if err := session.Close(); err != nil {
			m.logger.Error("failed to close MCP session", "name", name, "error", err)
		}
	}
	m.sessions = make(map[string]*mcp.ClientSession)
	return nil
}
