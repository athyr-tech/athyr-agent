package runner

import (
	"log/slog"
	"os"
	"testing"

	"github.com/athyr-tech/athyr-agent/internal/config"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:  "test-agent",
			Model: "gpt-4",
			Topics: config.TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	r, err := New(cfg, Options{
		ServerAddr: "localhost:9090",
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if r.cfg != cfg {
		t.Error("Runner config mismatch")
	}
	if r.opts.ServerAddr != "localhost:9090" {
		t.Errorf("ServerAddr = %v, want localhost:9090", r.opts.ServerAddr)
	}
}

func TestNew_DefaultLogger(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Name:  "test-agent",
			Model: "gpt-4",
			Topics: config.TopicsConfig{
				Subscribe: []string{"input"},
				Publish:   []string{"output"},
			},
		},
	}

	r, err := New(cfg, Options{
		ServerAddr: "localhost:9090",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if r.logger == nil {
		t.Error("Logger should not be nil")
	}
}
