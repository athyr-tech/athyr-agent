package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoad_ValidConfig(t *testing.T) {
	yaml := `
agent:
  name: test-agent
  description: A test agent
  model: gpt-4
  instructions: |
    You are a helpful assistant.
  topics:
    subscribe:
      - input.topic
    publish:
      - output.topic
`
	cfg, err := Load([]byte(yaml))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Agent.Name != "test-agent" {
		t.Errorf("Name = %v, want test-agent", cfg.Agent.Name)
	}
	if cfg.Agent.Model != "gpt-4" {
		t.Errorf("Model = %v, want gpt-4", cfg.Agent.Model)
	}
	if len(cfg.Agent.Topics.Subscribe) != 1 {
		t.Errorf("Subscribe topics = %v, want 1", len(cfg.Agent.Topics.Subscribe))
	}
	if len(cfg.Agent.Topics.Publish) != 1 {
		t.Errorf("Publish topics = %v, want 1", len(cfg.Agent.Topics.Publish))
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	yaml := `
agent:
  name: [invalid yaml
`
	_, err := Load([]byte(yaml))
	if err == nil {
		t.Fatal("Load() expected error for invalid YAML")
	}
}

func TestValidate_MissingName(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Model: "gpt-4",
			Topics: TopicsConfig{
				Subscribe: []string{"topic"},
				Publish:   []string{"topic"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for missing name")
	}
	if !strings.Contains(err.Error(), "agent.name is required") {
		t.Errorf("error = %v, want to contain 'agent.name is required'", err)
	}
}

func TestValidate_MissingModel(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name: "test",
			Topics: TopicsConfig{
				Subscribe: []string{"topic"},
				Publish:   []string{"topic"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for missing model")
	}
	if !strings.Contains(err.Error(), "agent.model is required") {
		t.Errorf("error = %v, want to contain 'agent.model is required'", err)
	}
}

func TestValidate_MissingTopics(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:  "test",
			Model: "gpt-4",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for missing topics")
	}
	if !strings.Contains(err.Error(), "subscribe") {
		t.Errorf("error = %v, want to contain 'subscribe'", err)
	}
	if !strings.Contains(err.Error(), "publish") {
		t.Errorf("error = %v, want to contain 'publish'", err)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:         "test",
			Model:        "gpt-4",
			Instructions: "Be helpful",
			Topics: TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Validate() unexpected error = %v", err)
	}
}

func TestLoad_WithMemory(t *testing.T) {
	yaml := `
agent:
  name: test-agent
  model: gpt-4
  topics:
    subscribe: [input]
    publish: [output]
  memory:
    enabled: true
    session_prefix: test
    ttl: 24h
`
	cfg, err := Load([]byte(yaml))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Agent.Memory.Enabled {
		t.Error("Memory.Enabled = false, want true")
	}
	if cfg.Agent.Memory.SessionPrefix != "test" {
		t.Errorf("Memory.SessionPrefix = %v, want test", cfg.Agent.Memory.SessionPrefix)
	}
	if cfg.Agent.Memory.TTL != "24h" {
		t.Errorf("Memory.TTL = %v, want 24h", cfg.Agent.Memory.TTL)
	}
}

func TestLoad_WithMCPServers(t *testing.T) {
	yaml := `
agent:
  name: test-agent
  model: gpt-4
  topics:
    subscribe: [input]
    publish: [output]
  mcp:
    servers:
      - name: filesystem
        command: ["npx", "@modelcontextprotocol/server-filesystem", "/tmp"]
      - name: github
        command: ["npx", "@modelcontextprotocol/server-github"]
        env:
          GITHUB_TOKEN: "test-token"
`
	cfg, err := Load([]byte(yaml))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Agent.MCP.Servers) != 2 {
		t.Fatalf("MCP.Servers length = %v, want 2", len(cfg.Agent.MCP.Servers))
	}

	fs := cfg.Agent.MCP.Servers[0]
	if fs.Name != "filesystem" {
		t.Errorf("Server[0].Name = %v, want filesystem", fs.Name)
	}
	if len(fs.Command) != 3 {
		t.Errorf("Server[0].Command length = %v, want 3", len(fs.Command))
	}

	gh := cfg.Agent.MCP.Servers[1]
	if gh.Name != "github" {
		t.Errorf("Server[1].Name = %v, want github", gh.Name)
	}
	if gh.Env["GITHUB_TOKEN"] != "test-token" {
		t.Errorf("Server[1].Env[GITHUB_TOKEN] = %v, want test-token", gh.Env["GITHUB_TOKEN"])
	}
}

func TestValidate_MCPServerWithoutName(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
			MCP: MCPConfig{
				Servers: []MCPServerConfig{
					{Command: []string{"some-command"}},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for MCP server without name")
	}
	if !strings.Contains(err.Error(), "mcp.servers[0].name is required") {
		t.Errorf("error = %v, want to contain 'mcp.servers[0].name is required'", err)
	}
}

func TestValidate_MCPServerWithoutCommandOrURL(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
			MCP: MCPConfig{
				Servers: []MCPServerConfig{
					{Name: "test-server"},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for MCP server without command or url")
	}
	if !strings.Contains(err.Error(), "must specify either command or url") {
		t.Errorf("error = %v, want to contain 'must specify either command or url'", err)
	}
}

func TestValidate_MCPServerWithBothCommandAndURL(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
			MCP: MCPConfig{
				Servers: []MCPServerConfig{
					{
						Name:    "test-server",
						Command: []string{"some-command"},
						URL:     "http://localhost:8080/mcp",
					},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for MCP server with both command and url")
	}
	if !strings.Contains(err.Error(), "not both") {
		t.Errorf("error = %v, want to contain 'not both'", err)
	}
}

func TestValidate_MCPServerWithURLOnly(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
			MCP: MCPConfig{
				Servers: []MCPServerConfig{
					{
						Name: "remote-server",
						URL:  "http://mcpx:8080/mcp",
					},
				},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Validate() unexpected error for URL-only MCP server: %v", err)
	}
}

func TestLoad_WithRoutes(t *testing.T) {
	yaml := `
agent:
  name: classifier
  model: gpt-4
  topics:
    subscribe: [ticket.new]
    publish: [ticket.unknown]
    routes:
      - topic: ticket.billing
        description: Payment issues, invoices, refunds
      - topic: ticket.technical
        description: Bugs, errors, crashes
`
	cfg, err := Load([]byte(yaml))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Agent.Topics.Routes) != 2 {
		t.Fatalf("Routes length = %v, want 2", len(cfg.Agent.Topics.Routes))
	}
	if cfg.Agent.Topics.Routes[0].Topic != "ticket.billing" {
		t.Errorf("Routes[0].Topic = %v, want ticket.billing", cfg.Agent.Topics.Routes[0].Topic)
	}
	if cfg.Agent.Topics.Routes[0].Description != "Payment issues, invoices, refunds" {
		t.Errorf("Routes[0].Description = %v, want 'Payment issues...'", cfg.Agent.Topics.Routes[0].Description)
	}
}

func TestValidate_RouteWithoutTopic(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
				Routes: []RouteConfig{
					{Description: "Missing topic"},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for route without topic")
	}
	if !strings.Contains(err.Error(), "routes[0].topic is required") {
		t.Errorf("error = %v, want to contain 'routes[0].topic is required'", err)
	}
}

func TestValidate_RouteWithoutDescription(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
				Routes: []RouteConfig{
					{Topic: "some.topic"},
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for route without description")
	}
	if !strings.Contains(err.Error(), "routes[0].description is required") {
		t.Errorf("error = %v, want to contain 'routes[0].description is required'", err)
	}
}

func TestTopicsConfig_HasRoutes(t *testing.T) {
	// No routes
	tc := TopicsConfig{
		Subscribe: []string{"input"},
		Publish:   []string{"output"},
	}
	if tc.HasRoutes() {
		t.Error("HasRoutes() = true, want false")
	}

	// With routes
	tc.Routes = []RouteConfig{{Topic: "a", Description: "b"}}
	if !tc.HasRoutes() {
		t.Error("HasRoutes() = false, want true")
	}
}

func TestTopicsConfig_IsValidRoute(t *testing.T) {
	tc := TopicsConfig{
		Routes: []RouteConfig{
			{Topic: "ticket.billing", Description: "Billing"},
			{Topic: "ticket.technical", Description: "Technical"},
		},
	}

	if !tc.IsValidRoute("ticket.billing") {
		t.Error("IsValidRoute(ticket.billing) = false, want true")
	}
	if !tc.IsValidRoute("ticket.technical") {
		t.Error("IsValidRoute(ticket.technical) = false, want true")
	}
	if tc.IsValidRoute("ticket.unknown") {
		t.Error("IsValidRoute(ticket.unknown) = true, want false")
	}
}

func TestTopicsConfig_BuildRoutingPrompt(t *testing.T) {
	// No routes - empty prompt
	tc := TopicsConfig{}
	if tc.BuildRoutingPrompt() != "" {
		t.Error("BuildRoutingPrompt() should be empty when no routes")
	}

	// With routes - should contain route info
	tc.Routes = []RouteConfig{
		{Topic: "ticket.billing", Description: "Payment issues"},
		{Topic: "ticket.technical", Description: "Bugs and errors"},
	}
	prompt := tc.BuildRoutingPrompt()

	if !strings.Contains(prompt, "ticket.billing") {
		t.Error("Prompt should contain topic name")
	}
	if !strings.Contains(prompt, "Payment issues") {
		t.Error("Prompt should contain description")
	}
	if !strings.Contains(prompt, "route_to") {
		t.Error("Prompt should mention route_to field")
	}
}

func TestLoad_WithMemoryProfile(t *testing.T) {
	yaml := `
agent:
  name: test-agent
  model: gpt-4
  topics:
    subscribe: [input]
    publish: [output]
  memory:
    enabled: true
    profile:
      type: rolling_window
      max_tokens: 8192
      summarization_threshold: 6000
`
	cfg, err := Load([]byte(yaml))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Agent.Memory.Enabled {
		t.Error("Memory.Enabled = false, want true")
	}
	if cfg.Agent.Memory.Profile.Type != "rolling_window" {
		t.Errorf("Memory.Profile.Type = %v, want rolling_window", cfg.Agent.Memory.Profile.Type)
	}
	if cfg.Agent.Memory.Profile.MaxTokens != 8192 {
		t.Errorf("Memory.Profile.MaxTokens = %v, want 8192", cfg.Agent.Memory.Profile.MaxTokens)
	}
	if cfg.Agent.Memory.Profile.SummarizationThreshold != 6000 {
		t.Errorf("Memory.Profile.SummarizationThreshold = %v, want 6000", cfg.Agent.Memory.Profile.SummarizationThreshold)
	}
}

func TestMemoryConfig_GetProfile_WithDefaults(t *testing.T) {
	// Test that GetProfile returns defaults when profile is not set
	cfg := MemoryConfig{
		Enabled: true,
	}

	profile := cfg.GetProfile()

	if profile.Type != "rolling_window" {
		t.Errorf("Profile.Type = %v, want rolling_window", profile.Type)
	}
	if profile.MaxTokens != 4096 {
		t.Errorf("Profile.MaxTokens = %v, want 4096", profile.MaxTokens)
	}
	if profile.SummarizationThreshold != 3000 {
		t.Errorf("Profile.SummarizationThreshold = %v, want 3000", profile.SummarizationThreshold)
	}
}

func TestMemoryConfig_GetProfile_WithCustomValues(t *testing.T) {
	cfg := MemoryConfig{
		Enabled: true,
		Profile: SessionProfileConfig{
			Type:                   "custom",
			MaxTokens:              16384,
			SummarizationThreshold: 12000,
		},
	}

	profile := cfg.GetProfile()

	if profile.Type != "custom" {
		t.Errorf("Profile.Type = %v, want custom", profile.Type)
	}
	if profile.MaxTokens != 16384 {
		t.Errorf("Profile.MaxTokens = %v, want 16384", profile.MaxTokens)
	}
	if profile.SummarizationThreshold != 12000 {
		t.Errorf("Profile.SummarizationThreshold = %v, want 12000", profile.SummarizationThreshold)
	}
}

func TestLoad_WithMCPServerURL(t *testing.T) {
	yaml := `
agent:
  name: test-agent
  model: gpt-4
  topics:
    subscribe: [input]
    publish: [output]
  mcp:
    servers:
      - name: remote-api
        url: http://mcpx:8080/mcp
`
	cfg, err := Load([]byte(yaml))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Agent.MCP.Servers) != 1 {
		t.Fatalf("MCP.Servers length = %v, want 1", len(cfg.Agent.MCP.Servers))
	}

	srv := cfg.Agent.MCP.Servers[0]
	if srv.Name != "remote-api" {
		t.Errorf("Server.Name = %v, want remote-api", srv.Name)
	}
	if srv.URL != "http://mcpx:8080/mcp" {
		t.Errorf("Server.URL = %v, want http://mcpx:8080/mcp", srv.URL)
	}
	if len(srv.Command) != 0 {
		t.Errorf("Server.Command = %v, want empty", srv.Command)
	}
}

func TestLoad_WithConnection(t *testing.T) {
	yaml := `
agent:
  name: test-agent
  model: gpt-4
  topics:
    subscribe: [input]
    publish: [output]
  connection:
    timeout: 30s
    max_retries: 5
    base_backoff: 2s
    max_backoff: 1m
`
	cfg, err := Load([]byte(yaml))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Agent.Connection.Timeout != "30s" {
		t.Errorf("Connection.Timeout = %v, want 30s", cfg.Agent.Connection.Timeout)
	}
	if cfg.Agent.Connection.MaxRetries != 5 {
		t.Errorf("Connection.MaxRetries = %v, want 5", cfg.Agent.Connection.MaxRetries)
	}
	if cfg.Agent.Connection.BaseBackoff != "2s" {
		t.Errorf("Connection.BaseBackoff = %v, want 2s", cfg.Agent.Connection.BaseBackoff)
	}
	if cfg.Agent.Connection.MaxBackoff != "1m" {
		t.Errorf("Connection.MaxBackoff = %v, want 1m", cfg.Agent.Connection.MaxBackoff)
	}
}

func TestConnectionConfig_GetOptions_Defaults(t *testing.T) {
	cfg := ConnectionConfig{}

	opts, err := cfg.GetOptions()
	if err != nil {
		t.Fatalf("GetOptions() error = %v", err)
	}

	// Check defaults match SDK defaults
	if opts.RequestTimeout != 60*time.Second {
		t.Errorf("RequestTimeout = %v, want 60s", opts.RequestTimeout)
	}
	if opts.MaxRetries != 0 {
		t.Errorf("MaxRetries = %v, want 0 (infinite)", opts.MaxRetries)
	}
	if opts.BaseBackoff != 1*time.Second {
		t.Errorf("BaseBackoff = %v, want 1s", opts.BaseBackoff)
	}
	if opts.MaxBackoff != 30*time.Second {
		t.Errorf("MaxBackoff = %v, want 30s", opts.MaxBackoff)
	}
}

func TestConnectionConfig_GetOptions_Custom(t *testing.T) {
	cfg := ConnectionConfig{
		Timeout:     "120s",
		MaxRetries:  10,
		BaseBackoff: "5s",
		MaxBackoff:  "2m",
	}

	opts, err := cfg.GetOptions()
	if err != nil {
		t.Fatalf("GetOptions() error = %v", err)
	}

	if opts.RequestTimeout != 120*time.Second {
		t.Errorf("RequestTimeout = %v, want 120s", opts.RequestTimeout)
	}
	if opts.MaxRetries != 10 {
		t.Errorf("MaxRetries = %v, want 10", opts.MaxRetries)
	}
	if opts.BaseBackoff != 5*time.Second {
		t.Errorf("BaseBackoff = %v, want 5s", opts.BaseBackoff)
	}
	if opts.MaxBackoff != 2*time.Minute {
		t.Errorf("MaxBackoff = %v, want 2m", opts.MaxBackoff)
	}
}

func TestConnectionConfig_GetOptions_InvalidDuration(t *testing.T) {
	cfg := ConnectionConfig{
		Timeout: "invalid",
	}

	_, err := cfg.GetOptions()
	if err == nil {
		t.Fatal("GetOptions() expected error for invalid duration")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("error = %v, want to contain 'timeout'", err)
	}
}

func TestConnectionConfig_GetOptions_NegativeDuration(t *testing.T) {
	cfg := ConnectionConfig{
		Timeout: "-5s",
	}

	_, err := cfg.GetOptions()
	if err == nil {
		t.Fatal("GetOptions() expected error for negative duration")
	}
	if !strings.Contains(err.Error(), "negative") || !strings.Contains(err.Error(), "timeout") {
		t.Errorf("error = %v, want to contain 'negative' and 'timeout'", err)
	}
}

func TestLoad_WithPlugins(t *testing.T) {
	yaml := `
agent:
  name: test-agent
  model: gpt-4
  plugins:
    - name: my-watcher
      file: ./plugins/file-watcher.lua
      config:
        path: /var/log/app/
        interval: 10
    - name: my-slack
      file: ./plugins/slack.lua
      restrict:
        - fs
      config:
        webhook: https://hooks.slack.com/xxx
  topics:
    subscribe:
      - input
      - my-watcher
    publish:
      - output
      - my-slack
`
	cfg, err := Load([]byte(yaml))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Agent.Plugins) != 2 {
		t.Fatalf("Plugins length = %v, want 2", len(cfg.Agent.Plugins))
	}

	w := cfg.Agent.Plugins[0]
	if w.Name != "my-watcher" {
		t.Errorf("Plugin[0].Name = %v, want my-watcher", w.Name)
	}
	if w.File != "./plugins/file-watcher.lua" {
		t.Errorf("Plugin[0].File = %v, want ./plugins/file-watcher.lua", w.File)
	}
	if w.Config["path"] != "/var/log/app/" {
		t.Errorf("Plugin[0].Config[path] = %v, want /var/log/app/", w.Config["path"])
	}

	s := cfg.Agent.Plugins[1]
	if len(s.Restrict) != 1 || s.Restrict[0] != "fs" {
		t.Errorf("Plugin[1].Restrict = %v, want [fs]", s.Restrict)
	}
}

func TestValidate_PluginWithoutName(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
			Plugins: []PluginConfig{
				{File: "./plugin.lua"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for plugin without name")
	}
	if !strings.Contains(err.Error(), "plugins[0].name is required") {
		t.Errorf("error = %v, want to contain 'plugins[0].name is required'", err)
	}
}

func TestValidate_PluginWithoutFile(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
			Plugins: []PluginConfig{
				{Name: "test-plugin"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for plugin without file")
	}
	if !strings.Contains(err.Error(), "plugins[0].file is required") {
		t.Errorf("error = %v, want to contain 'plugins[0].file is required'", err)
	}
}

func TestValidate_DuplicatePluginName(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
			Plugins: []PluginConfig{
				{Name: "dup", File: "./a.lua"},
				{Name: "dup", File: "./b.lua"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for duplicate plugin name")
	}
	if !strings.Contains(err.Error(), "duplicate plugin name") {
		t.Errorf("error = %v, want to contain 'duplicate plugin name'", err)
	}
}

func TestValidate_PluginReferencedInSubscribeButNotDefined(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Topics: TopicsConfig{
				Subscribe: []string{"nonexistent-plugin"},
				Publish:   []string{"output"},
			},
			Plugins: []PluginConfig{
				{Name: "my-plugin", File: "./plugin.lua"},
			},
		},
	}

	// Should pass — can't distinguish Athyr topics from plugin names at config level.
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Validate() unexpected error = %v", err)
	}
}

func TestValidate_NoTopicsWithPlugins(t *testing.T) {
	cfg := &Config{
		Agent: AgentConfig{
			Name:  "test",
			Model: "gpt-4",
			Plugins: []PluginConfig{
				{Name: "my-trigger", File: "./trigger.lua"},
			},
			Topics: TopicsConfig{
				Subscribe: []string{"my-trigger"},
				Publish:   []string{"my-trigger"},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Validate() unexpected error = %v", err)
	}
}
