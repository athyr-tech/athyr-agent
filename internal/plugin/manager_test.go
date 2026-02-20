package plugin

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/athyr-tech/athyr-agent/internal/config"
)

func TestManager_LoadPlugin(t *testing.T) {
	dir := t.TempDir()

	luaCode := `
function subscribe(config, callback)
	callback("hello from plugin")
end
`
	luaPath := filepath.Join(dir, "test-plugin.lua")
	os.WriteFile(luaPath, []byte(luaCode), 0644)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	mgr := NewManager(logger)

	cfg := config.PluginConfig{
		Name: "test-plugin",
		File: luaPath,
	}

	err := mgr.LoadPlugin(cfg)
	if err != nil {
		t.Fatalf("LoadPlugin() error = %v", err)
	}

	if !mgr.HasPlugin("test-plugin") {
		t.Error("HasPlugin() = false, want true")
	}
}

func TestManager_SubscribePlugin(t *testing.T) {
	dir := t.TempDir()

	luaCode := `
function subscribe(config, callback)
	callback("event-data")
end
`
	luaPath := filepath.Join(dir, "source.lua")
	os.WriteFile(luaPath, []byte(luaCode), 0644)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	mgr := NewManager(logger)

	cfg := config.PluginConfig{
		Name: "source",
		File: luaPath,
	}

	mgr.LoadPlugin(cfg)

	received := make(chan string, 1)
	err := mgr.StartSubscribe("source", func(data string) {
		received <- data
	})
	if err != nil {
		t.Fatalf("StartSubscribe() error = %v", err)
	}

	select {
	case data := <-received:
		if data != "event-data" {
			t.Errorf("received = %v, want 'event-data'", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for subscribe callback")
	}
}

func TestManager_PublishPlugin(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.txt")

	luaCode := `
function publish(config, data)
	local fs = require("fs")
	fs.write(config.path, data)
end
`
	luaPath := filepath.Join(dir, "dest.lua")
	os.WriteFile(luaPath, []byte(luaCode), 0644)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	mgr := NewManager(logger)

	cfg := config.PluginConfig{
		Name: "dest",
		File: luaPath,
		Config: map[string]any{
			"path": outPath,
		},
	}

	mgr.LoadPlugin(cfg)

	err := mgr.Publish("dest", "response data")
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	data, _ := os.ReadFile(outPath)
	if string(data) != "response data" {
		t.Errorf("output file = %v, want 'response data'", string(data))
	}
}

func TestManager_PluginNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	mgr := NewManager(logger)

	err := mgr.StartSubscribe("nonexistent", func(data string) {})
	if err == nil {
		t.Error("StartSubscribe() expected error for nonexistent plugin")
	}

	err = mgr.Publish("nonexistent", "data")
	if err == nil {
		t.Error("Publish() expected error for nonexistent plugin")
	}
}

func TestManager_Close(t *testing.T) {
	dir := t.TempDir()

	luaCode := `
function subscribe(config, callback)
	while true do
		sleep(0.1)
	end
end
`
	luaPath := filepath.Join(dir, "loop.lua")
	os.WriteFile(luaPath, []byte(luaCode), 0644)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	mgr := NewManager(logger)

	cfg := config.PluginConfig{
		Name: "loop",
		File: luaPath,
	}

	mgr.LoadPlugin(cfg)
	mgr.StartSubscribe("loop", func(data string) {})

	// Let the goroutine start and enter the loop
	time.Sleep(50 * time.Millisecond)

	// Close should not hang
	err := mgr.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
