package plugin

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/athyr-tech/athyr-agent/internal/config"

	lua "github.com/yuin/gopher-lua"
)

// pluginState holds the loaded state for a single plugin.
type pluginState struct {
	sandbox *Sandbox
	config  map[string]any
}

// Manager manages Lua plugin loading, subscribe, and publish lifecycle.
type Manager struct {
	logger  *slog.Logger
	plugins map[string]*pluginState // name → state
	mu      sync.RWMutex
}

// NewManager creates a new plugin manager.
func NewManager(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		logger:  logger,
		plugins: make(map[string]*pluginState),
	}
}

// LoadPlugin creates a sandbox, loads the Lua file, and registers bridge modules.
func (m *Manager) LoadPlugin(cfg config.PluginConfig) error {
	sb, err := NewSandbox(cfg.Restrict)
	if err != nil {
		return fmt.Errorf("failed to create sandbox for plugin %s: %w", cfg.Name, err)
	}

	RegisterBridge(sb)

	if err := sb.DoFile(cfg.File); err != nil {
		sb.Close()
		return fmt.Errorf("failed to load plugin %s from %s: %w", cfg.Name, cfg.File, err)
	}

	m.mu.Lock()
	m.plugins[cfg.Name] = &pluginState{
		sandbox: sb,
		config:  cfg.Config,
	}
	m.mu.Unlock()

	m.logger.Info("loaded plugin", "name", cfg.Name, "file", cfg.File)
	return nil
}

// HasPlugin returns true if a plugin with the given name is loaded.
func (m *Manager) HasPlugin(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.plugins[name]
	return ok
}

// IsPlugin is an alias for HasPlugin, used by the runner to distinguish plugins from Athyr topics.
func (m *Manager) IsPlugin(name string) bool {
	return m.HasPlugin(name)
}

// StartSubscribe calls the plugin's Lua subscribe(config, callback) function in a goroutine.
func (m *Manager) StartSubscribe(name string, callback func(string)) error {
	m.mu.RLock()
	ps, ok := m.plugins[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	L := ps.sandbox.L

	// Get the subscribe function
	fn := L.GetGlobal("subscribe")
	if fn == lua.LNil {
		return fmt.Errorf("plugin %s does not define a subscribe function", name)
	}

	// Build config table
	configTbl := goMapToLuaTable(L, ps.config)

	// Build callback function
	cbFn := L.NewFunction(func(L *lua.LState) int {
		data := L.CheckString(1)
		callback(data)
		return 0
	})

	// Run subscribe in a goroutine (it may loop forever).
	// When Close() is called, L.Close() frees the Lua state, causing a panic
	// in the running goroutine. We recover from it gracefully.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.logger.Debug("subscribe goroutine recovered", "plugin", name, "panic", r)
			}
		}()
		if err := L.CallByParam(lua.P{
			Fn:      fn,
			NRet:    0,
			Protect: true,
		}, configTbl, cbFn); err != nil {
			m.logger.Debug("subscribe goroutine ended", "plugin", name, "error", err.Error())
		}
	}()

	m.logger.Info("started subscribe", "plugin", name)
	return nil
}

// Publish calls the plugin's Lua publish(config, data) function synchronously.
func (m *Manager) Publish(name string, data string) error {
	m.mu.RLock()
	ps, ok := m.plugins[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	L := ps.sandbox.L

	fn := L.GetGlobal("publish")
	if fn == lua.LNil {
		return fmt.Errorf("plugin %s does not define a publish function", name)
	}

	configTbl := goMapToLuaTable(L, ps.config)

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    0,
		Protect: true,
	}, configTbl, lua.LString(data)); err != nil {
		return fmt.Errorf("plugin %s publish failed: %w", name, err)
	}

	return nil
}

// Close shuts down all plugin Lua states.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, ps := range m.plugins {
		ps.sandbox.Close()
		m.logger.Debug("closed plugin", "name", name)
	}
	m.plugins = make(map[string]*pluginState)
	return nil
}

// goMapToLuaTable converts a Go map[string]any to a Lua table.
func goMapToLuaTable(L *lua.LState, m map[string]any) *lua.LTable {
	tbl := L.NewTable()
	if m == nil {
		return tbl
	}
	for k, v := range m {
		L.SetField(tbl, k, goValueToLua(L, v))
	}
	return tbl
}
