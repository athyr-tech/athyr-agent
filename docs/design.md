# athyr-agent: YAML-Driven Agent Runner

> **Design document** - General concept for a standalone YAML-driven agent runner.
> Created: 2026-01-29

## Overview

`athyr-agent` is a standalone tool that runs AI agents defined in YAML files. It connects to an Athyr server as a regular agent, maintaining Athyr's architecture principle that agents are external processes.

## Core Concept

```
┌─────────────────────────────────────────────────────────┐
│                    athyr-agent                          │
│                                                         │
│   ┌─────────────┐    ┌─────────────┐    ┌───────────┐   │
│   │ YAML Config │───▶│  Runtime    │───▶│ Athyr SDK │   │
│   │(agent.yaml) │    │  Interpreter│    │  (gRPC)   │   │
│   └─────────────┘    └─────────────┘    └───────────┘   │
│                                                         │
└─────────────────────────────────────────────────────────┘
                            │
                            │ gRPC/HTTP
                            ▼
                    ┌───────────────┐
                    │ Athyr Server  │
                    └───────────────┘
```

## Usage

```bash
# Run an agent from YAML definition
athyr-agent run agent.yaml

# Run with custom server address
athyr-agent run agent.yaml --server localhost:9090

# Validate YAML without running
athyr-agent validate agent.yaml
```

## YAML Format

```yaml
# agent.yaml
agent:
  name: summarizer
  description: Summarizes documents concisely

  # LLM configuration
  model: google/gemini-2.5-flash-lite
  instructions: |
    You are a document summarizer.
    Create concise, accurate summaries.

  # Athyr pub/sub topics
  topics:
    subscribe:
      - documents.new
      - documents.updated
    publish:
      - summaries.ready

  # Optional: MCP servers for tool access
  mcp:
    servers:
      - name: docker-gateway
        command: ["docker", "mcp", "gateway", "run"]

  # Optional: memory/session configuration
  memory:
    enabled: true
    session_prefix: summarizer
```

## Design Principles

1. **Agents stay external** — athyr-agent runs as a separate process, not inside Athyr server
2. **YAML is the portable format** — no custom binary format, just YAML
3. **Uses Go SDK internally** — full SDK capabilities (auto-reconnect, middleware, etc.)
4. **Simple by default** — basic agents need minimal config
5. **Extensible** — can add tools, memory, advanced routing

## Architecture Decisions

| Decision | Rationale |
|----------|-----------|
| Separate repo | Not part of Athyr platform or SDK |
| Go implementation | Matches Athyr ecosystem, single binary output |
| YAML over JSON | More readable for agent definitions |
| No embedded LLM | Uses Athyr's LLM Gateway |

## Scope

### In Scope
- Run agents from YAML definitions
- Connect to Athyr server
- Subscribe/publish to topics
- MCP tool integration (via MCP servers)
- Memory/session configuration

### Out of Scope (for now)
- Visual editor
- Agent marketplace
- Complex orchestration (use SDK for that)
- Hot reload (restart to change config)

## Inspired By

- **Docker cagent** — YAML agent definitions
- **Letta .af files** — Portable agent format concept

## References

- [Framework Comparison Synthesis](../../athyr/docs/research/framework-comparison-synthesis.md)
- [ADR-004: Framework Learnings](../../athyr/docs/architecture/decisions/004-framework-learnings.md)
- [Docker cagent docs](https://docs.docker.com/ai/cagent/)
