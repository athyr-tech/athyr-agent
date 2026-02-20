package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/athyr-tech/athyr-agent/internal/config"
	"github.com/athyr-tech/athyr-agent/internal/runner"
	"github.com/athyr-tech/athyr-agent/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	insecure  bool
	useTUI    bool
	quiet     bool
	logFormat string
)

var runCmd = &cobra.Command{
	Use:   "run <file>",
	Short: "Run an agent from a YAML file",
	Long: `Run an agent defined in a YAML file.

The agent will connect to the Athyr server, subscribe to topics,
and process messages through the configured LLM.

Log Levels:
  --quiet      Only show errors
  --verbose    Show debug details (default: INFO level)

Log Format:
  --log-format=json   JSON lines for log aggregation
  --log-format=text   Human-readable key=value (default)

Example:
  athyr-agent run agent.yaml
  athyr-agent run agent.yaml --server localhost:9090
  athyr-agent run agent.yaml --tui
  athyr-agent run agent.yaml --verbose
  athyr-agent run agent.yaml --quiet --log-format=json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filepath := args[0]

		// Validate mutually exclusive flags
		if quiet && viper.GetBool("verbose") {
			return fmt.Errorf("--quiet and --verbose are mutually exclusive")
		}

		// Set up log level
		logLevel := slog.LevelInfo
		if quiet {
			logLevel = slog.LevelError
		} else if viper.GetBool("verbose") {
			logLevel = slog.LevelDebug
		}

		// Load and validate config
		cfg, err := config.LoadFile(filepath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}

		if useTUI {
			return runWithTUI(cfg, logLevel)
		}
		return runHeadless(cfg, logLevel)
	},
}

func init() {
	runCmd.Flags().BoolVar(&insecure, "insecure", false, "disable TLS (for development)")
	runCmd.Flags().BoolVar(&useTUI, "tui", false, "run with interactive terminal UI")
	runCmd.Flags().BoolVar(&quiet, "quiet", false, "only show errors (mutually exclusive with --verbose)")
	runCmd.Flags().StringVar(&logFormat, "log-format", "text", "log output format: text or json")
	rootCmd.AddCommand(runCmd)
}

// runHeadless runs the agent without TUI (original behavior).
func runHeadless(cfg *config.Config, logLevel slog.Level) error {
	// Configure log handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: logLevel}
	if logFormat == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}
	logger := slog.New(handler)

	// Collect MCP server names for logging
	var mcpServers []string
	for _, srv := range cfg.Agent.MCP.Servers {
		mcpServers = append(mcpServers, srv.Name)
	}

	logger.Info("agent started",
		"name", cfg.Agent.Name,
		"model", cfg.Agent.Model,
		"topics", cfg.Agent.Topics.Subscribe,
		"mcp_servers", mcpServers,
	)

	// Create runner
	r, err := runner.New(cfg, runner.Options{
		ServerAddr: viper.GetString("server"),
		Insecure:   insecure,
		Logger:     logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	// Run the agent
	return r.Run(ctx)
}

// runWithTUI runs the agent with the interactive terminal UI.
func runWithTUI(cfg *config.Config, logLevel slog.Level) error {
	// Create event bus for communication between runner and TUI
	eventBus := runner.NewEventBus(100)
	defer eventBus.Close()

	// Create TUI logger that emits to event bus
	logger := tui.NewTUILogger(eventBus, logLevel)

	logger.Info("loaded agent config",
		"name", cfg.Agent.Name,
		"model", cfg.Agent.Model,
	)

	// Create runner with event bus
	r, err := runner.New(cfg, runner.Options{
		ServerAddr: viper.GetString("server"),
		Insecure:   insecure,
		Logger:     logger,
		EventBus:   eventBus,
	})
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Create TUI
	tuiApp, err := tui.New(tui.Options{
		Config:     cfg,
		EventBus:   eventBus,
		ServerAddr: viper.GetString("server"),
	})
	if err != nil {
		return fmt.Errorf("failed to create TUI: %w", err)
	}

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to collect errors from goroutines
	errCh := make(chan error, 2)

	// Start the runner in a goroutine
	go func() {
		if err := r.Run(ctx); err != nil {
			errCh <- fmt.Errorf("runner error: %w", err)
		} else {
			errCh <- nil
		}
	}()

	// Set up chat and messaging handlers after runner connects
	// Poll until handler is available or context is cancelled
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if r.Handler() != nil {
					tuiApp.SetChatHandler(&chatHandlerAdapter{handler: r.Handler()})
					tuiApp.SetMessagingHandler(&messagingHandlerAdapter{
						handler: r.Handler(),
						tuiSend: tuiApp.Send,
					})
					return
				}
			}
		}
	}()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		tuiApp.Quit()
	}()

	// Run TUI (blocks until user quits)
	if err := tuiApp.Run(); err != nil {
		cancel() // Cancel runner context
		return fmt.Errorf("TUI error: %w", err)
	}

	// TUI exited normally, cancel runner
	cancel()

	// Wait for runner to finish (with timeout)
	select {
	case err := <-errCh:
		if err != nil && err.Error() != "runner error: context canceled" {
			return err
		}
	default:
		// Runner might not have finished yet, that's ok
	}

	return nil
}

// chatHandlerAdapter adapts the MessageHandler to the tui.ChatHandler interface.
type chatHandlerAdapter struct {
	handler *runner.MessageHandler
}

func (a *chatHandlerAdapter) DirectChat(content string) (string, string, int, error) {
	return a.handler.DirectChat(content)
}

// messagingHandlerAdapter adapts the MessageHandler to the tui.MessagingHandler interface.
type messagingHandlerAdapter struct {
	handler *runner.MessageHandler
	tuiSend func(msg tea.Msg) // Function to send messages to TUI
}

func (a *messagingHandlerAdapter) PublishMessage(topic string, data []byte) error {
	return a.handler.PublishMessage(topic, data)
}

func (a *messagingHandlerAdapter) RequestMessage(topic string, data []byte) ([]byte, error) {
	return a.handler.RequestMessage(topic, data)
}

func (a *messagingHandlerAdapter) WatchTopic(topic string, _ tui.WatchCallback) error {
	// Create callback that sends messages directly to TUI via Send
	callback := func(timestamp time.Time, content string) {
		if a.tuiSend != nil {
			a.tuiSend(tui.WatchMessageMsg{
				Timestamp: timestamp,
				Content:   content,
			})
		}
	}
	return a.handler.WatchTopic(topic, callback)
}

func (a *messagingHandlerAdapter) StopWatching() error {
	return a.handler.StopWatching()
}

func (a *messagingHandlerAdapter) WatchingTopic() string {
	return a.handler.WatchingTopic()
}
