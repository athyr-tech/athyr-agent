package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents a complete agent YAML file.
type Config struct {
	Agent AgentConfig `yaml:"agent"`
}

// AgentConfig defines the agent's configuration.
type AgentConfig struct {
	Name         string           `yaml:"name"`
	Description  string           `yaml:"description"`
	Model        string           `yaml:"model"`
	Instructions string           `yaml:"instructions"`
	Plugins      []PluginConfig   `yaml:"plugins,omitempty"`
	Topics       TopicsConfig     `yaml:"topics"`
	Memory       MemoryConfig     `yaml:"memory,omitempty"`
	MCP          MCPConfig        `yaml:"mcp,omitempty"`
	Connection   ConnectionConfig `yaml:"connection,omitempty"`
}

// PluginConfig defines a Lua plugin.
type PluginConfig struct {
	Name     string         `yaml:"name"`
	File     string         `yaml:"file"`
	Restrict []string       `yaml:"restrict,omitempty"`
	Config   map[string]any `yaml:"config,omitempty"`
}

// TopicsConfig defines pub/sub topics.
type TopicsConfig struct {
	Subscribe []string      `yaml:"subscribe"`
	Publish   []string      `yaml:"publish"`
	Routes    []RouteConfig `yaml:"routes,omitempty"`
}

// RouteConfig defines a dynamic routing destination.
// The LLM can route messages to these topics based on its analysis.
type RouteConfig struct {
	Topic       string `yaml:"topic"`
	Description string `yaml:"description"`
}

// HasRoutes returns true if dynamic routes are configured.
func (t *TopicsConfig) HasRoutes() bool {
	return len(t.Routes) > 0
}

// BuildRoutingPrompt creates instructions for the LLM about available routes.
func (t *TopicsConfig) BuildRoutingPrompt() string {
	if !t.HasRoutes() {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## Routing Instructions\n")
	sb.WriteString("Based on your analysis, route this message to the appropriate destination.\n")
	sb.WriteString("Include a `route_to` field in your JSON response with one of these topics:\n\n")

	for _, route := range t.Routes {
		sb.WriteString("- `")
		sb.WriteString(route.Topic)
		sb.WriteString("`: ")
		sb.WriteString(route.Description)
		sb.WriteString("\n")
	}

	sb.WriteString("\nExample response format:\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"route_to\": \"<topic>\",\n")
	sb.WriteString("  \"content\": \"your analysis or response\"\n")
	sb.WriteString("}\n")
	sb.WriteString("```")

	return sb.String()
}

// IsValidRoute checks if a topic is in the allowed routes.
func (t *TopicsConfig) IsValidRoute(topic string) bool {
	for _, route := range t.Routes {
		if route.Topic == topic {
			return true
		}
	}
	return false
}

// MemoryConfig defines optional memory/session settings.
type MemoryConfig struct {
	Enabled       bool                 `yaml:"enabled"`
	SessionPrefix string               `yaml:"session_prefix,omitempty"`
	TTL           string               `yaml:"ttl,omitempty"` // Duration string like "1h", "24h"
	Profile       SessionProfileConfig `yaml:"profile,omitempty"`
}

// SessionProfileConfig defines session memory behavior.
type SessionProfileConfig struct {
	Type                   string `yaml:"type,omitempty"`                    // "rolling_window"
	MaxTokens              int    `yaml:"max_tokens,omitempty"`              // Max tokens in memory
	SummarizationThreshold int    `yaml:"summarization_threshold,omitempty"` // When to trigger summarization
}

// GetProfile returns the session profile with defaults applied.
func (m *MemoryConfig) GetProfile() SessionProfileConfig {
	profile := m.Profile

	// Apply defaults
	if profile.Type == "" {
		profile.Type = "rolling_window"
	}
	if profile.MaxTokens == 0 {
		profile.MaxTokens = 4096
	}
	if profile.SummarizationThreshold == 0 {
		profile.SummarizationThreshold = 3000
	}

	return profile
}

// MCPConfig defines MCP server connections.
type MCPConfig struct {
	Servers []MCPServerConfig `yaml:"servers,omitempty"`
}

// MCPServerConfig defines an MCP server to connect to.
// Exactly one of Command or URL must be specified:
//   - Command: spawns a local subprocess (stdio transport)
//   - URL: connects to a remote server (Streamable HTTP transport)
type MCPServerConfig struct {
	Name    string            `yaml:"name"`
	Command []string          `yaml:"command,omitempty"`
	URL     string            `yaml:"url,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// ConnectionConfig defines SDK connection options.
type ConnectionConfig struct {
	Timeout     string `yaml:"timeout,omitempty"`      // Request timeout (e.g., "60s", "2m")
	MaxRetries  int    `yaml:"max_retries,omitempty"`  // Max reconnection retries (0 = infinite)
	BaseBackoff string `yaml:"base_backoff,omitempty"` // Initial backoff (e.g., "1s")
	MaxBackoff  string `yaml:"max_backoff,omitempty"`  // Max backoff (e.g., "30s")
}

// ConnectionOptions holds parsed connection settings.
type ConnectionOptions struct {
	RequestTimeout time.Duration
	MaxRetries     int
	BaseBackoff    time.Duration
	MaxBackoff     time.Duration
}

// GetOptions parses the connection config and returns options with defaults.
func (c *ConnectionConfig) GetOptions() (ConnectionOptions, error) {
	opts := ConnectionOptions{
		RequestTimeout: 60 * time.Second,
		MaxRetries:     0, // infinite
		BaseBackoff:    1 * time.Second,
		MaxBackoff:     30 * time.Second,
	}

	// Parse timeout
	if c.Timeout != "" {
		d, err := time.ParseDuration(c.Timeout)
		if err != nil {
			return opts, fmt.Errorf("invalid connection.timeout: %w", err)
		}
		if d < 0 {
			return opts, fmt.Errorf("connection.timeout cannot be negative: %s", c.Timeout)
		}
		opts.RequestTimeout = d
	}

	// Max retries (no parsing needed, just use directly)
	if c.MaxRetries != 0 {
		opts.MaxRetries = c.MaxRetries
	}

	// Parse base backoff
	if c.BaseBackoff != "" {
		d, err := time.ParseDuration(c.BaseBackoff)
		if err != nil {
			return opts, fmt.Errorf("invalid connection.base_backoff: %w", err)
		}
		if d < 0 {
			return opts, fmt.Errorf("connection.base_backoff cannot be negative: %s", c.BaseBackoff)
		}
		opts.BaseBackoff = d
	}

	// Parse max backoff
	if c.MaxBackoff != "" {
		d, err := time.ParseDuration(c.MaxBackoff)
		if err != nil {
			return opts, fmt.Errorf("invalid connection.max_backoff: %w", err)
		}
		if d < 0 {
			return opts, fmt.Errorf("connection.max_backoff cannot be negative: %s", c.MaxBackoff)
		}
		opts.MaxBackoff = d
	}

	return opts, nil
}

// LoadFile loads and parses a YAML config file.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return Load(data)
}

// Load parses YAML data into a Config.
func Load(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	return &cfg, nil
}

// Validate checks that the config contains all required fields.
func (c *Config) Validate() error {
	var errs []error

	if c.Agent.Name == "" {
		errs = append(errs, errors.New("agent.name is required"))
	}

	if c.Agent.Model == "" {
		errs = append(errs, errors.New("agent.model is required"))
	}

	if len(c.Agent.Topics.Subscribe) == 0 {
		errs = append(errs, errors.New("agent.topics.subscribe must have at least one topic"))
	}

	if len(c.Agent.Topics.Publish) == 0 {
		errs = append(errs, errors.New("agent.topics.publish must have at least one topic"))
	}

	// Validate route definitions
	for i, route := range c.Agent.Topics.Routes {
		if route.Topic == "" {
			errs = append(errs, fmt.Errorf("agent.topics.routes[%d].topic is required", i))
		}
		if route.Description == "" {
			errs = append(errs, fmt.Errorf("agent.topics.routes[%d].description is required", i))
		}
	}

	// Validate plugin definitions
	pluginNames := make(map[string]bool)
	for i, plugin := range c.Agent.Plugins {
		if plugin.Name == "" {
			errs = append(errs, fmt.Errorf("agent.plugins[%d].name is required", i))
		}
		if plugin.File == "" {
			errs = append(errs, fmt.Errorf("agent.plugins[%d].file is required", i))
		}
		if plugin.Name != "" {
			if pluginNames[plugin.Name] {
				errs = append(errs, fmt.Errorf("agent.plugins[%d]: duplicate plugin name %q", i, plugin.Name))
			}
			pluginNames[plugin.Name] = true
		}
	}

	// Validate MCP server definitions
	for i, srv := range c.Agent.MCP.Servers {
		if srv.Name == "" {
			errs = append(errs, fmt.Errorf("agent.mcp.servers[%d].name is required", i))
		}
		hasCommand := len(srv.Command) > 0
		hasURL := srv.URL != ""
		if hasCommand && hasURL {
			errs = append(errs, fmt.Errorf("agent.mcp.servers[%d] must specify either command or url, not both", i))
		}
		if !hasCommand && !hasURL {
			errs = append(errs, fmt.Errorf("agent.mcp.servers[%d] must specify either command or url", i))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
