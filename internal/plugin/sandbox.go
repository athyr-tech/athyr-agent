package plugin

import (
	"fmt"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// Sandbox wraps a Lua state with restricted library access.
type Sandbox struct {
	L        *lua.LState
	restrict []string // e.g., ["fs", "http.post"]
}

// NewSandbox creates a sandboxed Lua state.
// restrict is a list of API names to block (e.g., "fs", "http.post").
// Pass nil for no restrictions.
func NewSandbox(restrict []string) (*Sandbox, error) {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})

	// Open only safe built-in libraries
	for _, pair := range []struct {
		name string
		fn   lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenPackage},
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
		{lua.OsLibName, lua.OpenOs},
	} {
		if err := L.CallByParam(lua.P{
			Fn:      L.NewFunction(pair.fn),
			NRet:    0,
			Protect: true,
		}, lua.LString(pair.name)); err != nil {
			L.Close()
			return nil, fmt.Errorf("failed to open library %s: %w", pair.name, err)
		}
	}

	// Remove dangerous functions from base library
	L.SetGlobal("dofile", lua.LNil)
	L.SetGlobal("loadfile", lua.LNil)

	// Remove dangerous os functions (keep os.time, os.date, os.clock, os.difftime)
	if osLib := L.GetGlobal("os"); osLib != lua.LNil {
		if osTbl, ok := osLib.(*lua.LTable); ok {
			osTbl.RawSetString("execute", lua.LNil)
			osTbl.RawSetString("exit", lua.LNil)
			osTbl.RawSetString("getenv", lua.LNil)
			osTbl.RawSetString("remove", lua.LNil)
			osTbl.RawSetString("rename", lua.LNil)
			osTbl.RawSetString("setlocale", lua.LNil)
			osTbl.RawSetString("tmpname", lua.LNil)
		}
	}

	return &Sandbox{
		L:        L,
		restrict: restrict,
	}, nil
}

// IsRestricted checks if a given API call is blocked by the restriction list.
// Supports both module-level ("fs") and function-level ("fs.write") restrictions.
func (s *Sandbox) IsRestricted(api string) bool {
	if len(s.restrict) == 0 {
		return false
	}

	for _, r := range s.restrict {
		// Exact match (e.g., "http.post" matches "http.post")
		if r == api {
			return true
		}
		// Module-level match (e.g., "fs" matches "fs.read", "fs.write", etc.)
		if !strings.Contains(r, ".") && strings.HasPrefix(api, r+".") {
			return true
		}
	}
	return false
}

// DoString executes a Lua string.
func (s *Sandbox) DoString(code string) error {
	return s.L.DoString(code)
}

// DoFile executes a Lua file.
func (s *Sandbox) DoFile(path string) error {
	return s.L.DoFile(path)
}

// Close shuts down the Lua state.
func (s *Sandbox) Close() {
	s.L.Close()
}
