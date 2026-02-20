package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/athyr-tech/athyr-agent/internal/config"
	"github.com/athyr-tech/athyr-agent/internal/plugin"

	"github.com/athyr-tech/athyr-sdk-go/pkg/athyr"
)

// Options configures the runner.
type Options struct {
	ServerAddr string
	Insecure   bool
	Logger     *slog.Logger
	EventBus   EventBus // Optional: for TUI mode
}

// Runner manages the agent lifecycle.
type Runner struct {
	cfg      *config.Config
	opts     Options
	logger   *slog.Logger
	agent    athyr.Agent
	mcp      *MCPManager
	plugins  *plugin.Manager
	eventBus EventBus
	handler  *MessageHandler
}

// New creates a new Runner.
func New(cfg *config.Config, opts Options) (*Runner, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	return &Runner{
		cfg:      cfg,
		opts:     opts,
		logger:   opts.Logger,
		eventBus: opts.EventBus,
	}, nil
}

// emitEvent sends an event to the EventBus if one is configured.
func (r *Runner) emitEvent(event Event) {
	if r.eventBus != nil {
		r.eventBus.Send(event)
	}
}

// Handler returns the message handler, used by the TUI for direct chat.
func (r *Runner) Handler() *MessageHandler {
	return r.handler
}

// Config returns the runner's configuration.
func (r *Runner) Config() *config.Config {
	return r.cfg
}

// AgentID returns the connected agent's ID, or empty if not connected.
func (r *Runner) AgentID() string {
	if r.agent != nil {
		return r.agent.AgentID()
	}
	return ""
}

// Run starts the agent and blocks until context is cancelled.
func (r *Runner) Run(ctx context.Context) error {
	// Parse connection options from config
	connOpts, err := r.cfg.Agent.Connection.GetOptions()
	if err != nil {
		return fmt.Errorf("invalid connection config: %w", err)
	}

	// Create SDK agent with options
	agentOpts := []athyr.AgentOption{
		athyr.WithAgentCard(athyr.AgentCard{
			Name:        r.cfg.Agent.Name,
			Description: r.cfg.Agent.Description,
			Version:     "1.0.0",
			Metadata: map[string]string{
				"runner": "athyr-agent",
				"model":  r.cfg.Agent.Model,
			},
		}),
		athyr.WithLogger(newSDKLogger(r.logger)),
		athyr.WithRequestTimeout(connOpts.RequestTimeout),
		athyr.WithAutoReconnect(connOpts.MaxRetries, connOpts.BaseBackoff),
		athyr.WithMaxBackoff(connOpts.MaxBackoff),
	}

	if r.opts.Insecure {
		agentOpts = append(agentOpts, athyr.WithInsecure())
	}

	agent, err := athyr.NewAgent(r.opts.ServerAddr, agentOpts...)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}
	r.agent = agent

	// Connect to server
	r.logger.Info("connecting to server", "addr", r.opts.ServerAddr)
	if err := agent.Connect(ctx); err != nil {
		r.emitEvent(StatusEvent{
			Time:      time.Now(),
			Connected: false,
			AgentName: r.cfg.Agent.Name,
			Error:     err,
		})
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		r.logger.Info("disconnected", "reason", "shutdown")
		r.emitEvent(StatusEvent{
			Time:      time.Now(),
			Connected: false,
			AgentID:   agent.AgentID(),
			AgentName: r.cfg.Agent.Name,
		})
		_ = agent.Close()
	}()

	r.logger.Info("connected",
		"agent_id", agent.AgentID(),
		"server", r.opts.ServerAddr,
	)
	r.emitEvent(StatusEvent{
		Time:      time.Now(),
		Connected: true,
		AgentID:   agent.AgentID(),
		AgentName: r.cfg.Agent.Name,
	})

	// Initialize MCP manager if servers are configured
	var mcpMgr *MCPManager
	if len(r.cfg.Agent.MCP.Servers) > 0 {
		mcpMgr = NewMCPManager(r.logger)
		if err := mcpMgr.Start(ctx, r.cfg.Agent.MCP.Servers); err != nil {
			return fmt.Errorf("failed to start MCP manager: %w", err)
		}
		defer mcpMgr.Close()

		tools := mcpMgr.GetAthyrTools()
		r.logger.Info("MCP tools available", "count", len(tools))

		// Emit tools available event for TUI
		r.emitEvent(ToolsAvailableEvent{
			Time:  time.Now(),
			Tools: mcpMgr.GetToolsInfo(),
		})
	}
	r.mcp = mcpMgr

	// Initialize plugin manager if plugins are configured
	var pluginMgr *plugin.Manager
	if len(r.cfg.Agent.Plugins) > 0 {
		pluginMgr = plugin.NewManager(r.logger)
		for _, p := range r.cfg.Agent.Plugins {
			if err := pluginMgr.LoadPlugin(p); err != nil {
				return fmt.Errorf("failed to load plugin %s: %w", p.Name, err)
			}
		}
		defer pluginMgr.Close()
	}
	r.plugins = pluginMgr

	// Create message handler
	handler := newMessageHandler(r.cfg, agent, r.logger, mcpMgr, pluginMgr, r.eventBus)
	r.handler = handler

	// Subscribe to configured topics
	for _, topic := range r.cfg.Agent.Topics.Subscribe {
		if pluginMgr != nil && pluginMgr.IsPlugin(topic) {
			// Plugin source: start the plugin's subscribe function
			topicCopy := topic
			r.logger.Info("starting plugin subscribe", "plugin", topicCopy)
			if err := pluginMgr.StartSubscribe(topicCopy, func(data string) {
				handler.Handle(athyr.SubscribeMessage{
					Subject: topicCopy,
					Data:    []byte(data),
				})
			}); err != nil {
				return fmt.Errorf("failed to start plugin subscribe for %s: %w", topicCopy, err)
			}
		} else {
			// Athyr topic: subscribe via SDK agent
			r.logger.Info("subscribing to topic", "topic", topic)
			_, err := agent.Subscribe(ctx, topic, handler.Handle)
			if err != nil {
				return fmt.Errorf("failed to subscribe to %s: %w", topic, err)
			}
		}
	}

	r.logger.Info("agent running",
		"name", r.cfg.Agent.Name,
		"subscriptions", r.cfg.Agent.Topics.Subscribe,
	)

	// Wait for shutdown signal
	<-ctx.Done()

	return nil
}

// sdkLogger adapts slog.Logger to the athyr.Logger interface.
type sdkLogger struct {
	logger *slog.Logger
}

func newSDKLogger(l *slog.Logger) *sdkLogger {
	return &sdkLogger{logger: l}
}

func (l *sdkLogger) Debug(msg string, keysAndValues ...any) {
	l.logger.Debug(msg, keysAndValues...)
}

func (l *sdkLogger) Info(msg string, keysAndValues ...any) {
	l.logger.Info(msg, keysAndValues...)
}

func (l *sdkLogger) Warn(msg string, keysAndValues ...any) {
	l.logger.Warn(msg, keysAndValues...)
}

func (l *sdkLogger) Error(msg string, keysAndValues ...any) {
	l.logger.Error(msg, keysAndValues...)
}
