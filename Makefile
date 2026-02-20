.PHONY: build test clean install validate send send-session docker-up docker-down docker-logs help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X github.com/athyr-tech/athyr-agent/internal/cli.Version=$(VERSION) \
                     -X github.com/athyr-tech/athyr-agent/internal/cli.GitCommit=$(COMMIT) \
                     -X github.com/athyr-tech/athyr-agent/internal/cli.BuildDate=$(DATE)"

build:
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/athyr-agent ./cmd/athyr-agent

test:
	go test -v ./...

clean:
	rm -rf bin/
	go clean

install:
	go install $(LDFLAGS) ./cmd/athyr-agent

# Development helpers
validate:
	./bin/athyr-agent validate examples/summarizer.yaml --verbose

# Default server address
SERVER ?= localhost:9090

# ─────────────────────────────────────────────────────────────────────────────
# Run Agents - TUI Mode (Recommended)
# ─────────────────────────────────────────────────────────────────────────────
run-simple-tui: build
	./bin/athyr-agent run examples/simple-test.yaml --server $(SERVER) --insecure --tui

run-summarizer-tui: build
	./bin/athyr-agent run examples/summarizer.yaml --server $(SERVER) --insecure --tui

run-memory-tui: build
	./bin/athyr-agent run examples/memory-chat.yaml --server $(SERVER) --insecure --tui

run-mcp-tui: build
	./bin/athyr-agent run examples/mcp-tools.yaml --server $(SERVER) --insecure --tui

# Demo agents (run each in a separate terminal)
run-classifier-tui: build
	./bin/athyr-agent run examples/demo/classifier.yaml --server $(SERVER) --insecure --tui

run-billing-tui: build
	./bin/athyr-agent run examples/demo/billing-agent.yaml --server $(SERVER) --insecure --tui

run-tech-tui: build
	./bin/athyr-agent run examples/demo/tech-agent.yaml --server $(SERVER) --insecure --tui

# ─────────────────────────────────────────────────────────────────────────────
# Run Agents - Terminal Mode
# ─────────────────────────────────────────────────────────────────────────────
run-simple: build
	./bin/athyr-agent run examples/simple-test.yaml --server $(SERVER) --insecure --verbose

run-summarizer: build
	./bin/athyr-agent run examples/summarizer.yaml --server $(SERVER) --insecure --verbose

run-memory: build
	./bin/athyr-agent run examples/memory-chat.yaml --server $(SERVER) --insecure --verbose

run-mcp: build
	./bin/athyr-agent run examples/mcp-tools.yaml --server $(SERVER) --insecure --verbose

run-classifier: build
	./bin/athyr-agent run examples/demo/classifier.yaml --server $(SERVER) --insecure --verbose

run-billing: build
	./bin/athyr-agent run examples/demo/billing-agent.yaml --server $(SERVER) --insecure --verbose

run-tech: build
	./bin/athyr-agent run examples/demo/tech-agent.yaml --server $(SERVER) --insecure --verbose

# ─────────────────────────────────────────────────────────────────────────────
# Send Messages
# ─────────────────────────────────────────────────────────────────────────────
# Usage: make send MSG="Hello!" [SUBJECT=test.input]
SUBJECT ?= test.input
send:
ifndef MSG
	$(error MSG is required. Usage: make send MSG="Hello!")
endif
	@./examples/scripts/send-message.sh "$(SUBJECT)" "$(MSG)"

# Usage: make send-session MSG="Hello!" SESSION=user-123
send-session:
ifndef MSG
	$(error MSG is required. Usage: make send-session MSG="Hello!" SESSION=user-1)
endif
ifndef SESSION
	$(error SESSION is required. Usage: make send-session MSG="Hello!" SESSION=user-1)
endif
	@./examples/scripts/send-message.sh "chat.input" "$(MSG)" "$(SESSION)"

# ─────────────────────────────────────────────────────────────────────────────
# Docker (local Athyr server)
# ─────────────────────────────────────────────────────────────────────────────
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

# ─────────────────────────────────────────────────────────────────────────────
# Help
# ─────────────────────────────────────────────────────────────────────────────
help:
	@echo "athyr-agent - YAML-driven agent runner"
	@echo ""
	@echo "TUI Mode (Recommended):"
	@echo "  make run-simple-tui      Basic test agent"
	@echo "  make run-summarizer-tui  Document summarizer"
	@echo "  make run-memory-tui      Memory-enabled chat"
	@echo "  make run-mcp-tui         Research with MCP tools"
	@echo ""
	@echo "Multi-Agent Demo (run each in separate terminal):"
	@echo "  make run-classifier-tui  Routes tickets to specialists"
	@echo "  make run-billing-tui     Billing specialist"
	@echo "  make run-tech-tui        Technical specialist"
	@echo ""
	@echo "Terminal Mode:"
	@echo "  make run-simple          Basic test agent"
	@echo "  make send MSG=\"Hello\"    Send message to running agent"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up           Start local Athyr server"
	@echo "  make docker-down         Stop local Athyr server"
	@echo "  make docker-logs         Tail Athyr server logs"
	@echo ""
	@echo "Build:"
	@echo "  make build               Build the binary to bin/"
	@echo "  make install             Install to GOPATH/bin"
	@echo "  make test                Run all tests"
	@echo "  make clean               Remove build artifacts"
	@echo "  make validate            Validate example YAML"
