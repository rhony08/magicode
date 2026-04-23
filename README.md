# OpenCode CLI (Go Implementation)

A high-performance, memory-efficient CLI coding assistant written in Go.

## Overview

This is a Go recreation of the [OpenCode](https://github.com/opencode-ai/opencode) TypeScript CLI, focusing on:

- **Memory efficiency**: Bounded caches, proper cleanup, explicit resource management
- **Performance**: Compiled binary, instant startup, no runtime overhead
- **Simplicity**: Cleaner architecture without Effect.ts complexity
- **Maintainability**: Explicit resource lifecycle management

### Memory Improvements

| Metric | TypeScript | Go Target |
|--------|------------|-----------|
| Virtual memory (startup) | 71GB+ | <100MB |
| RSS (steady state) | Growing | <500MB |
| Instance cache | Unlimited | Max 10 instances |
| LSP file tracking | Unlimited | Max 50 files |
| PTY sessions | Unlimited | Max 5 concurrent |

## Features

- **Multi-provider support**: Anthropic, OpenAI (more planned)
- **Interactive TUI**: Bubble Tea-based terminal interface
- **Tool execution**: Bash, Read, Write, Edit, Grep, Glob, WebFetch
- **LSP integration**: Diagnostics from TypeScript, Python, Go servers
- **PTY sessions**: Terminal sessions with bounded buffers
- **Session management**: Create, fork, archive sessions
- **Config management**: JSON/JSONC config files with merging
- **HTTP server**: Fiber-based API for remote access

## Installation

### Build from source

```bash
make deps
make build
```

The binary will be created in `./bin/opencode`.

### Install to system

```bash
make install
```

## Usage

```bash
# Start interactive TUI
opencode

# Start in a specific directory
opencode -d /path/to/project

# Run HTTP server
opencode serve

# List available providers
opencode providers

# List available models
opencode models

# Manage sessions
opencode session list
opencode session create
opencode session delete <id>

# Manage configuration
opencode config show
opencode config set model anthropic/claude-sonnet-4-5

# Debug commands
opencode debug version
opencode debug paths
```

## Configuration

Configuration is stored in JSON/JSONC files:

- **Global**: `~/.config/opencode/opencode.json`
- **Project**: `<project>/.opencode/opencode.json` or `<project>/opencode.json`

Example configuration:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "anthropic/claude-sonnet-4-5",
  "small_model": "anthropic/claude-3-5-haiku",
  "default_agent": "build",
  "username": "developer",
  "provider": {
    "anthropic": {
      "env": ["ANTHROPIC_API_KEY"]
    },
    "openai": {
      "env": ["OPENAI_API_KEY"]
    }
  }
}
```

## Architecture

### Key Components

| Component | Location | Purpose |
|-----------|----------|---------|
| CLI | `cmd/opencode/commands/` | Cobra command handlers |
| Instance | `internal/instance/` | Project context with bounded cache |
| Bus | `internal/bus/` | Channel-based pub/sub with cleanup |
| Config | `internal/config/` | JSON/JSONC configuration |
| Provider | `internal/provider/` | AI provider abstraction |
| Session | `internal/session/` | Conversation management |
| Storage | `internal/storage/` | SQLite persistence |
| LSP | `internal/lsp/` | Language server client |
| PTY | `internal/pty/` | Terminal sessions |
| TUI | `internal/tui/` | Bubble Tea interface |
| Tools | `internal/tool/` | Tool implementations |

### Memory Leak Solutions

The Go implementation addresses all memory leaks identified in the TypeScript version:

| Leak Source | Solution |
|-------------|----------|
| GlobalBus EventEmitter | Channel-based bus with bounded subscribers |
| BusEvent Registry | BoundedMap with Clear(), instance-scoped |
| InstanceState Cache | LRU cache with max capacity (10), TTL |
| LSP File Storage | BoundedMap (50 files), don't store full text |
| Provider SDKs | Lazy loading, shared HTTP connection pool |
| PTY Buffers | Buffer pooling (sync.Pool), max 5 sessions |
| OAuth Transports | Timer-based auto cleanup (10 min timeout) |
| Sync Registry | Instance-scoped by default, not global |

## Development

### Requirements

- Go 1.23+
- Make (optional)

### Run tests

```bash
make test
```

### Run with coverage

```bash
make test-coverage
```

### Lint

```bash
make lint
```

### Format

```bash
make fmt
```

## Project Status

This is **Phase 1** (Core Infrastructure) of the implementation.

| Phase | Status | Components |
|-------|--------|------------|
| 1 | ✅ Done | CLI, Logging, Global paths, Config, Instance, Bus |
| 2 | 🚧 Pending | Storage, Session, Sync |
| 3 | 📋 Planned | AI Providers (Anthropic, OpenAI) |
| 4 | 📋 Planned | Tools, Agent, Permission |
| 5 | 📋 Planned | LSP Integration |
| 6 | 📋 Planned | TUI (Bubble Tea) |
| 7 | 📋 Planned | PTY & Terminal |
| 8 | 📋 Planned | HTTP Server & API |
| 9 | 📋 Planned | Polish & Testing |

## License

MIT License - See [LICENSE](LICENSE) for details.

## Related

- [Original TypeScript Implementation](https://github.com/opencode-ai/opencode)
- [Memory Leak Review](../MEMORY_LEAK_REVIEW.md)
- [Go Recreation Plan](../GOLANG_RECREATION_PLAN.md)