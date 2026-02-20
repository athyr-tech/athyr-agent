package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBridge_FSRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	sb, err := NewSandbox(nil)
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	RegisterBridge(sb)

	err = sb.DoString(`
		local fs = require("fs")
		result = fs.read("` + path + `")
	`)
	if err != nil {
		t.Fatalf("DoString() error = %v", err)
	}

	result := sb.L.GetGlobal("result")
	if result.String() != "hello world" {
		t.Errorf("fs.read result = %v, want 'hello world'", result.String())
	}
}

func TestBridge_FSWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	sb, err := NewSandbox(nil)
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	RegisterBridge(sb)

	err = sb.DoString(`
		local fs = require("fs")
		fs.write("` + path + `", "written from lua")
	`)
	if err != nil {
		t.Fatalf("DoString() error = %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "written from lua" {
		t.Errorf("file content = %v, want 'written from lua'", string(data))
	}
}

func TestBridge_FSList(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)

	sb, err := NewSandbox(nil)
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	RegisterBridge(sb)

	err = sb.DoString(`
		local fs = require("fs")
		local files = fs.list("` + dir + `")
		count = #files
	`)
	if err != nil {
		t.Fatalf("DoString() error = %v", err)
	}

	count := sb.L.GetGlobal("count")
	if count.String() != "2" {
		t.Errorf("fs.list count = %v, want 2", count.String())
	}
}

func TestBridge_FSRestricted(t *testing.T) {
	sb, err := NewSandbox([]string{"fs.write"})
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	RegisterBridge(sb)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello"), 0644)

	// fs.read should work
	err = sb.DoString(`
		local fs = require("fs")
		result = fs.read("` + path + `")
	`)
	if err != nil {
		t.Fatalf("fs.read should work: %v", err)
	}

	// fs.write should be blocked
	err = sb.DoString(`
		local fs = require("fs")
		fs.write("` + filepath.Join(dir, "out.txt") + `", "blocked")
	`)
	if err == nil {
		t.Error("fs.write should be restricted")
	}
}

func TestBridge_JSON(t *testing.T) {
	sb, err := NewSandbox(nil)
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	RegisterBridge(sb)

	err = sb.DoString(`
		local json = require("json")
		local encoded = json.encode({name = "test", count = 42})
		local decoded = json.decode(encoded)
		result_name = decoded.name
		result_count = decoded.count
	`)
	if err != nil {
		t.Fatalf("DoString() error = %v", err)
	}

	name := sb.L.GetGlobal("result_name")
	if name.String() != "test" {
		t.Errorf("json roundtrip name = %v, want 'test'", name.String())
	}
}

func TestBridge_GoIntToLuaNumber(t *testing.T) {
	sb, err := NewSandbox(nil)
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	RegisterBridge(sb)

	// Simulate YAML config with int values (YAML unmarshals numbers as int, not float64)
	tbl := goMapToLuaTable(sb.L, map[string]any{
		"interval": 5,       // int
		"retries":  int64(3), // int64
		"ratio":    1.5,     // float64
	})
	sb.L.SetGlobal("config", tbl)

	err = sb.DoString(`
		result_interval = config.interval + 0
		result_retries = config.retries + 0
		result_ratio = config.ratio + 0
	`)
	if err != nil {
		t.Fatalf("DoString() error = %v", err)
	}

	interval := sb.L.GetGlobal("result_interval")
	if interval.String() != "5" {
		t.Errorf("int config value = %v, want 5", interval.String())
	}
	retries := sb.L.GetGlobal("result_retries")
	if retries.String() != "3" {
		t.Errorf("int64 config value = %v, want 3", retries.String())
	}
	ratio := sb.L.GetGlobal("result_ratio")
	if ratio.String() != "1.5" {
		t.Errorf("float64 config value = %v, want 1.5", ratio.String())
	}
}

func TestBridge_Sleep(t *testing.T) {
	sb, err := NewSandbox(nil)
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	RegisterBridge(sb)

	err = sb.DoString(`sleep(0.01)`)
	if err != nil {
		t.Errorf("sleep() error = %v", err)
	}
}
