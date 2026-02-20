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

func TestSandbox_SafeOsFunctions(t *testing.T) {
	sb, err := NewSandbox(nil)
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	// os.time and os.date should work
	err = sb.DoString(`t = os.time()`)
	if err != nil {
		t.Errorf("os.time() should be available: %v", err)
	}

	err = sb.DoString(`d = os.date("!%Y-%m-%dT%H:%M:%SZ")`)
	if err != nil {
		t.Errorf("os.date() should be available: %v", err)
	}

	// Dangerous os functions should be removed
	for _, fn := range []string{"execute", "exit", "getenv", "remove", "rename"} {
		err = sb.DoString(`os.` + fn + `()`)
		if err == nil {
			t.Errorf("os.%s should not be available", fn)
		}
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
