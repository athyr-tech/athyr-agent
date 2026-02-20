package plugin

import (
	"testing"
)

func TestNewSandbox(t *testing.T) {
	sb, err := NewSandbox(nil)
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	err = sb.DoString(`x = 1 + 1`)
	if err != nil {
		t.Errorf("DoString() error = %v", err)
	}
}

func TestSandbox_NativeLibsRemoved(t *testing.T) {
	sb, err := NewSandbox(nil)
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	err = sb.DoString(`os.execute("echo hello")`)
	if err == nil {
		t.Error("os.execute should not be available")
	}

	err = sb.DoString(`io.open("test.txt")`)
	if err == nil {
		t.Error("io library should not be available")
	}

	err = sb.DoString(`debug.getinfo(1)`)
	if err == nil {
		t.Error("debug library should not be available")
	}
}

func TestSandbox_RestrictionCheck(t *testing.T) {
	sb, err := NewSandbox([]string{"fs", "http.post"})
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	if !sb.IsRestricted("fs.read") {
		t.Error("fs.read should be restricted when fs is restricted")
	}
	if !sb.IsRestricted("fs.write") {
		t.Error("fs.write should be restricted when fs is restricted")
	}
	if !sb.IsRestricted("http.post") {
		t.Error("http.post should be restricted")
	}
	if sb.IsRestricted("http.get") {
		t.Error("http.get should NOT be restricted")
	}
	if sb.IsRestricted("json.encode") {
		t.Error("json.encode should NOT be restricted")
	}
}

func TestSandbox_NoRestrictions(t *testing.T) {
	sb, err := NewSandbox(nil)
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	if sb.IsRestricted("fs.read") {
		t.Error("nothing should be restricted with nil restrict list")
	}
	if sb.IsRestricted("http.post") {
		t.Error("nothing should be restricted with nil restrict list")
	}
}
