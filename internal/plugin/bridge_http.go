package plugin

import (
	"io"
	"net/http"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func init() {
	// HTTP module is registered alongside other bridge modules via RegisterBridge.
	// This file contains the implementation only.
}

func registerHTTP(sb *Sandbox) {
	sb.L.PreloadModule("http", func(L *lua.LState) int {
		mod := L.NewTable()
		L.SetField(mod, "get", L.NewFunction(sb.luaHTTPGet))
		L.SetField(mod, "post", L.NewFunction(sb.luaHTTPPost))
		L.Push(mod)
		return 1
	})
}

func (sb *Sandbox) luaHTTPGet(L *lua.LState) int {
	if sb.IsRestricted("http.get") {
		L.ArgError(1, "http.get is restricted for this plugin")
		return 0
	}
	url := L.CheckString(1)

	resp, err := http.Get(url)
	if err != nil {
		L.ArgError(1, err.Error())
		return 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		L.ArgError(1, err.Error())
		return 0
	}

	tbl := L.NewTable()
	L.SetField(tbl, "body", lua.LString(string(body)))
	L.SetField(tbl, "status", lua.LNumber(resp.StatusCode))
	L.Push(tbl)
	return 1
}

func (sb *Sandbox) luaHTTPPost(L *lua.LState) int {
	if sb.IsRestricted("http.post") {
		L.ArgError(1, "http.post is restricted for this plugin")
		return 0
	}
	url := L.CheckString(1)
	bodyStr := L.CheckString(2)
	headers := L.OptTable(3, nil)

	req, err := http.NewRequest("POST", url, strings.NewReader(bodyStr))
	if err != nil {
		L.ArgError(1, err.Error())
		return 0
	}

	// Apply headers from Lua table
	if headers != nil {
		headers.ForEach(func(key, value lua.LValue) {
			if ks, ok := key.(lua.LString); ok {
				if vs, ok := value.(lua.LString); ok {
					req.Header.Set(string(ks), string(vs))
				}
			}
		})
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		L.ArgError(1, err.Error())
		return 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		L.ArgError(1, err.Error())
		return 0
	}

	tbl := L.NewTable()
	L.SetField(tbl, "body", lua.LString(string(body)))
	L.SetField(tbl, "status", lua.LNumber(resp.StatusCode))
	L.Push(tbl)
	return 1
}
