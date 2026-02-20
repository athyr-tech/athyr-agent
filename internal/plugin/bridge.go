package plugin

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// RegisterBridge registers all Go bridge modules (fs, http, json, sleep, log) into the sandbox.
func RegisterBridge(sb *Sandbox) {
	registerFS(sb)
	registerHTTP(sb)
	registerJSON(sb)
	registerSleep(sb)
	registerLog(sb)
}

// --- fs module ---

func registerFS(sb *Sandbox) {
	sb.L.PreloadModule("fs", func(L *lua.LState) int {
		mod := L.NewTable()
		L.SetField(mod, "read", L.NewFunction(sb.luaFSRead))
		L.SetField(mod, "write", L.NewFunction(sb.luaFSWrite))
		L.SetField(mod, "list", L.NewFunction(sb.luaFSList))
		L.Push(mod)
		return 1
	})
}

func (sb *Sandbox) luaFSRead(L *lua.LState) int {
	if sb.IsRestricted("fs.read") {
		L.ArgError(1, "fs.read is restricted for this plugin")
		return 0
	}
	path := L.CheckString(1)
	data, err := os.ReadFile(path)
	if err != nil {
		L.ArgError(1, err.Error())
		return 0
	}
	L.Push(lua.LString(string(data)))
	return 1
}

func (sb *Sandbox) luaFSWrite(L *lua.LState) int {
	if sb.IsRestricted("fs.write") {
		L.ArgError(1, "fs.write is restricted for this plugin")
		return 0
	}
	path := L.CheckString(1)
	data := L.CheckString(2)
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		L.ArgError(1, err.Error())
		return 0
	}
	return 0
}

func (sb *Sandbox) luaFSList(L *lua.LState) int {
	if sb.IsRestricted("fs.list") {
		L.ArgError(1, "fs.list is restricted for this plugin")
		return 0
	}
	path := L.CheckString(1)
	entries, err := os.ReadDir(path)
	if err != nil {
		L.ArgError(1, err.Error())
		return 0
	}
	tbl := L.NewTable()
	for _, entry := range entries {
		tbl.Append(lua.LString(entry.Name()))
	}
	L.Push(tbl)
	return 1
}

// --- json module ---

func registerJSON(sb *Sandbox) {
	sb.L.PreloadModule("json", func(L *lua.LState) int {
		mod := L.NewTable()
		L.SetField(mod, "encode", L.NewFunction(sb.luaJSONEncode))
		L.SetField(mod, "decode", L.NewFunction(sb.luaJSONDecode))
		L.Push(mod)
		return 1
	})
}

func (sb *Sandbox) luaJSONEncode(L *lua.LState) int {
	if sb.IsRestricted("json.encode") {
		L.ArgError(1, "json.encode is restricted for this plugin")
		return 0
	}
	val := L.CheckAny(1)
	goVal := luaValueToGo(val)
	data, err := json.Marshal(goVal)
	if err != nil {
		L.ArgError(1, err.Error())
		return 0
	}
	L.Push(lua.LString(string(data)))
	return 1
}

func (sb *Sandbox) luaJSONDecode(L *lua.LState) int {
	if sb.IsRestricted("json.decode") {
		L.ArgError(1, "json.decode is restricted for this plugin")
		return 0
	}
	str := L.CheckString(1)
	var goVal any
	if err := json.Unmarshal([]byte(str), &goVal); err != nil {
		L.ArgError(1, err.Error())
		return 0
	}
	L.Push(goValueToLua(L, goVal))
	return 1
}

// --- sleep global ---

func registerSleep(sb *Sandbox) {
	sb.L.SetGlobal("sleep", sb.L.NewFunction(func(L *lua.LState) int {
		seconds := L.CheckNumber(1)
		time.Sleep(time.Duration(float64(seconds) * float64(time.Second)))
		return 0
	}))
}

// --- log global ---

func registerLog(sb *Sandbox) {
	sb.L.SetGlobal("log", sb.L.NewFunction(func(L *lua.LState) int {
		level := L.CheckString(1)
		msg := L.CheckString(2)
		switch level {
		case "debug":
			slog.Debug(msg)
		case "info":
			slog.Info(msg)
		case "warn":
			slog.Warn(msg)
		case "error":
			slog.Error(msg)
		default:
			slog.Info(msg)
		}
		return 0
	}))
}

// --- Lua ↔ Go value conversion ---

func luaValueToGo(val lua.LValue) any {
	switch v := val.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	case *lua.LTable:
		// Check if it's an array (sequential integer keys starting at 1)
		maxN := v.MaxN()
		if maxN > 0 {
			arr := make([]any, 0, maxN)
			for i := 1; i <= maxN; i++ {
				arr = append(arr, luaValueToGo(v.RawGetInt(i)))
			}
			return arr
		}
		// Otherwise treat as map
		m := make(map[string]any)
		v.ForEach(func(key, value lua.LValue) {
			if ks, ok := key.(lua.LString); ok {
				m[string(ks)] = luaValueToGo(value)
			}
		})
		return m
	default:
		return val.String()
	}
}

func goValueToLua(L *lua.LState, val any) lua.LValue {
	switch v := val.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(v)
	case int:
		return lua.LNumber(float64(v))
	case int64:
		return lua.LNumber(float64(v))
	case float64:
		return lua.LNumber(v)
	case string:
		return lua.LString(v)
	case []any:
		tbl := L.NewTable()
		for _, item := range v {
			tbl.Append(goValueToLua(L, item))
		}
		return tbl
	case map[string]any:
		tbl := L.NewTable()
		for k, item := range v {
			L.SetField(tbl, k, goValueToLua(L, item))
		}
		return tbl
	default:
		return lua.LString("")
	}
}
