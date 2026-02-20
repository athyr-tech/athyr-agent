package runner

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/athyr-tech/athyr-agent/internal/config"

	"github.com/athyr-tech/athyr-sdk-go/pkg/athyr"
)

func TestMCPManager_GetAthyrTools_Empty(t *testing.T) {
	mgr := NewMCPManager(nil)

	tools := mgr.GetAthyrTools()
	if len(tools) != 0 {
		t.Errorf("GetAthyrTools() = %v tools, want 0", len(tools))
	}
}

func TestMCPManager_GetAthyrTools_ReturnsRegisteredTools(t *testing.T) {
	mgr := NewMCPManager(nil)

	// Manually register a tool for testing (simulates what Start() would do)
	mgr.RegisterTool("test-server", athyr.Tool{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}}}`),
	})

	tools := mgr.GetAthyrTools()
	if len(tools) != 1 {
		t.Fatalf("GetAthyrTools() = %v tools, want 1", len(tools))
	}

	if tools[0].Name != "test_tool" {
		t.Errorf("Tool.Name = %v, want test_tool", tools[0].Name)
	}
	if tools[0].Description != "A test tool" {
		t.Errorf("Tool.Description = %v, want 'A test tool'", tools[0].Description)
	}
}

func TestMCPManager_CallTool_UnknownTool(t *testing.T) {
	mgr := NewMCPManager(nil)

	_, err := mgr.CallTool(context.Background(), "unknown_tool", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("CallTool() expected error for unknown tool")
	}
}

func TestMCPManager_NoServersConfigured(t *testing.T) {
	cfg := &config.MCPConfig{
		Servers: []config.MCPServerConfig{},
	}

	mgr := NewMCPManager(nil)
	err := mgr.Start(context.Background(), cfg.Servers)
	if err != nil {
		t.Fatalf("Start() with empty servers should not error, got: %v", err)
	}

	tools := mgr.GetAthyrTools()
	if len(tools) != 0 {
		t.Errorf("GetAthyrTools() = %v tools, want 0", len(tools))
	}
}
