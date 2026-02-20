package plugin

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/athyr-tech/athyr-agent/internal/config"
)

func TestIntegration_SourceToDestination(t *testing.T) {
	dir := t.TempDir()

	// Source plugin: emits a fixed message
	srcLua := `
function subscribe(config, callback)
	callback("test-event-data")
end
`
	srcPath := filepath.Join(dir, "source.lua")
	os.WriteFile(srcPath, []byte(srcLua), 0644)

	// Destination plugin: writes to file
	outPath := filepath.Join(dir, "output.txt")
	dstLua := `
function publish(config, data)
	local fs = require("fs")
	fs.write(config.output, data)
end
`
	dstPath := filepath.Join(dir, "dest.lua")
	os.WriteFile(dstPath, []byte(dstLua), 0644)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	mgr := NewManager(logger)

	mgr.LoadPlugin(config.PluginConfig{
		Name: "src",
		File: srcPath,
	})
	mgr.LoadPlugin(config.PluginConfig{
		Name: "dst",
		File: dstPath,
		Config: map[string]any{
			"output": outPath,
		},
	})

	// Wire: source callback publishes to destination
	received := make(chan string, 1)
	mgr.StartSubscribe("src", func(data string) {
		received <- data
	})

	select {
	case data := <-received:
		err := mgr.Publish("dst", data)
		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for source")
	}

	content, _ := os.ReadFile(outPath)
	if string(content) != "test-event-data" {
		t.Errorf("output = %v, want 'test-event-data'", string(content))
	}

	mgr.Close()
}

func TestIntegration_RestrictedPlugin(t *testing.T) {
	dir := t.TempDir()

	luaCode := `
function publish(config, data)
	local http = require("http")
	http.post("http://localhost:1234", data)
end
`
	luaPath := filepath.Join(dir, "restricted.lua")
	os.WriteFile(luaPath, []byte(luaCode), 0644)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	mgr := NewManager(logger)

	mgr.LoadPlugin(config.PluginConfig{
		Name:     "restricted",
		File:     luaPath,
		Restrict: []string{"http"},
	})

	err := mgr.Publish("restricted", "test")
	if err == nil {
		t.Error("Publish() should fail when http is restricted")
	}
}
