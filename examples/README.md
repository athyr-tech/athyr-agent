# Examples

## Prerequisites

Start an Athyr server before running any example:

```bash
# With Ollama (local LLM — default)
docker run --rm -p 8080:8080 -p 9090:9090 ghcr.io/athyr-tech/athyr:latest

# With OpenRouter (cloud models)
OPENROUTER_API_KEY=sk-or-... make docker-up
```

Verify: `curl http://localhost:8080/healthz` → `{"status":"healthy"}`

## Quick Start with TUI (Recommended)

```bash
make run-simple-tui
```

Then use the **Messaging tab** (press `3`) to send messages directly!

## TUI Tabs Overview

| Tab | Key | What it shows |
|-----|-----|---------------|
| Dashboard | 1 | Agent status, message flow, routing |
| Chat | 2 | Direct conversation with agent's LLM |
| Messaging | 3 | Send messages to any topic |
| Logs | 4 | Real-time logs and errors |
| Tools | 5 | MCP tool executions |

## Sending Messages via TUI

1. Press `3` to go to Messaging tab
2. Press `i` to focus input
3. Select topic from dropdown or type custom
4. Enter your message (JSON validation shown)
5. Press `Ctrl+S` to send
6. Switch to Dashboard (`1`) to see the response

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `1-5` | Switch tabs |
| `?` | Help overlay |
| `i` | Focus input |
| `Esc` | Unfocus |
| `Ctrl+S` | Send message |
| `q` | Quit |

---

## Available Examples

### Basic Examples

| Example | Description | Run Command |
|---------|-------------|-------------|
| **simple-test** | Minimal agent for testing | `make run-simple-tui` |
| **summarizer** | Document summarization | `make run-summarizer-tui` |

### Feature Examples

| Example | Description | Run Command |
|---------|-------------|-------------|
| **memory-chat** | Multi-turn conversations with memory | `make run-memory-tui` |
| **mcp-tools** | Research assistant with academic tools | `make run-mcp-tui` |

### Multi-Agent Demo (Support Workflow)

Run each agent in a separate terminal:

| Agent | Description | Run Command |
|-------|-------------|-------------|
| **Classifier** | Routes tickets to specialists | `make run-classifier-tui` |
| **Billing** | Handles billing issues | `make run-billing-tui` |
| **Tech** | Handles technical issues | `make run-tech-tui` |

---

## Example Details

### Simple Test Agent

Basic connectivity testing without tools.

```bash
make run-simple-tui
```

**Test via TUI:**
- Topic: `test.input`
- Message: `Hello, how are you?`

### Summarizer Agent

Summarizes documents with structured output.

```bash
make run-summarizer-tui
```

**Test via TUI:**
- Topic: `documents.new`
- Message: Paste any document or article text

### Memory Chat Agent

Multi-turn conversation with session memory.

```bash
make run-memory-tui
```

**Test via TUI:**
- Topic: `chat.input`
- Message (JSON): `{"session_id": "user-123", "content": "My name is Alice"}`
- Follow-up: `{"session_id": "user-123", "content": "What is my name?"}`

The agent remembers context within the same session.

### MCP Tools Agent

Research assistant with academic database access.

```bash
make run-mcp-tui
```

**Prerequisites:**
- Docker Desktop with MCP enabled (`docker mcp gateway run`)

**Test via TUI:**
- Topic: `research.query`
- Message: `What are the latest papers on transformer architectures?`
- Watch: Tools tab (`5`) to see MCP tool executions

### Multi-Agent Demo

Support ticket routing with 3 agents. Run each in a separate terminal:

```bash
# Terminal 1: Classifier (entry point)
make run-classifier-tui

# Terminal 2: Billing specialist
make run-billing-tui

# Terminal 3: Tech specialist
make run-tech-tui
```

**Flow:**
```
ticket.new → [Classifier] → ticket.billing   → [Billing Agent] → ticket.response
                          → ticket.technical → [Tech Agent]    → ticket.response
```

**Test:** Send a message to `ticket.new` via the Classifier's Messaging tab. Watch it get routed to the appropriate specialist.

---

## Terminal Mode (Alternative)

For non-interactive use or scripting:

```bash
# Terminal 1: Start an agent
make run-simple

# Terminal 2: Send a message (agent ID auto-detected)
make send MSG="Hello, how are you?"
```

### Available Make Targets

| Target | Description |
|--------|-------------|
| `make run-simple` | Start basic agent (no tools) |
| `make run-summarizer` | Start document summarizer |
| `make run-memory` | Start memory-enabled agent |
| `make run-mcp` | Start agent with MCP tools |
| `make send MSG="..."` | Send message to running agent |
| `make send-session MSG="..." SESSION=id` | Send message with session ID |
| `make docker-up` | Start local Athyr server |

---

## How Features Work

### Tool Execution (MCP)

1. **Discovery**: On startup, agent connects to MCP servers and discovers tools
2. **Completion**: When a message arrives, tools are passed to the LLM
3. **Tool Call**: If the LLM decides to use a tool, it returns a tool call request
4. **Execution**: Agent executes the tool via MCP protocol
5. **Loop**: Tool results are added to conversation and sent back to LLM
6. **Response**: LLM generates final response incorporating tool results

Watch the Tools tab (`5`) in TUI to see this in real-time.

### Memory (Session Persistence)

1. **Session ID**: Messages include `session_id` in JSON payload
2. **Server-Side**: Athyr server stores conversation history per session
3. **Injection**: History is automatically injected into LLM context
4. **Summarization**: Old messages are summarized when threshold is reached

### Dynamic Routing

Agents can route messages to different topics based on content analysis:

```yaml
topics:
  routes:
    - topic: ticket.billing
      description: Payment issues, invoices, refunds
    - topic: ticket.technical
      description: Bugs, errors, crashes
```

The classifier agent demonstrates this pattern.

---

## Connection Options

Configure SDK connection settings for production deployments:

```yaml
agent:
  name: my-agent
  model: google/gemini-2.5-flash-lite
  topics:
    subscribe: [input]
    publish: [output]

  # Connection options (all optional, shown with defaults)
  connection:
    timeout: 60s       # Request timeout
    max_retries: 0     # Reconnection retries (0 = infinite)
    base_backoff: 1s   # Initial backoff between retries
    max_backoff: 30s   # Maximum backoff duration
```
