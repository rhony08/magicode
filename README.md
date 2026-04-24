# MagiCode CLI

A high-performance, memory-efficient AI-powered coding assistant written in Go.

## Overview

MagiCode is a Go recreation of the [OpenCode](https://github.com/opencode-ai/opencode) TypeScript CLI, focusing on:

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

The binary will be created in `./magicode`.

### Install to system

```bash
make install
```

## Usage

```bash
# Start interactive TUI
magicode

# Start in a specific directory
magicode -d /path/to/project

# Run HTTP server
magicode serve

# List available providers
magicode providers

# List available models
magicode models

# Manage sessions
magicode session list
magicode session create
magicode session delete <id>

# Manage configuration
magicode config show
magicode config set model anthropic/claude-sonnet-4-5

# Show path configuration
magicode paths

# Debug commands
magicode debug version
magicode debug paths
```

## Path Configuration

MagiCode uses the XDG Base Directory Specification:

| Type | Default Path |
|------|--------------|
| Data | `~/.local/share/magicode` |
| Config | `~/.config/magicode` |
| State | `~/.local/state/magicode` |
| Cache | `~/.cache/magicode` |

### Custom Paths

Override paths with command-line flags:

```bash
# Custom data directory
magicode --data-dir /custom/data

# Custom config directory
magicode --config-dir /custom/config

# Custom database file
magicode --database /custom/magicode.db

# Custom log file
magicode --log-file /custom/magicode.log
```

## Backward Compatibility with OpenCode

MagiCode provides full backward compatibility with OpenCode. You can seamlessly switch from OpenCode to MagiCode without losing any data.

### Quick Migration

Use the `--use-opencode` flag to use OpenCode's paths:

```bash
# Use OpenCode paths for this session
magicode --use-opencode
```

This will use:
- Data: `~/.local/share/opencode`
- Config: `~/.config/opencode`
- State: `~/.local/state/opencode`
- Cache: `~/.cache/opencode`

### Config File Compatibility

MagiCode reads both `magicode.json` and `opencode.json` config files:

```bash
# These are all valid config files:
~/.config/magicode/magicode.json
~/.config/magicode/magicode.jsonc
~/.config/opencode/opencode.json    # Backward compatible
~/.config/opencode/opencode.jsonc  # Backward compatible
```

### Aliasing for Drop-in Replacement

Create an alias in your shell configuration:

```bash
# ~/.bashrc or ~/.zshrc
alias opencode='magicode --use-opencode'
```

Or create a symlink:

```bash
# After 'make install'
ln -s $(which magicode) /usr/local/bin/opencode
```

### Session & Database Compatibility

The database schema is fully compatible with OpenCode:
- Sessions, messages, and projects are stored identically
- You can switch between OpenCode and MagiCode without migration
- All your conversation history is preserved

## Configuration

Configuration is stored in JSON/JSONC files:

- **Global**: `~/.config/magicode/magicode.json` (or `~/.config/opencode/opencode.json`)
- **Project**: `<project>/.magicode/magicode.json` or `<project>/magicode.json`

Example configuration:

```json
{
  "$schema": "https://magicode.ai/config.json",
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
| CLI | `cmd/magicode/commands/` | Cobra command handlers |
| Instance | `internal/instance/` | Project context with bounded cache |
| Bus | `internal/bus/` | Channel-based pub/sub with cleanup |
| Config | `internal/config/` | JSON/JSONC configuration |
| Provider | `internal/provider/` | AI provider abstraction |
| Session | `internal/session/` | Conversation management |
| Storage | `internal/database/` | SQLite persistence |
| LSP | `internal/lsp/` | Language server client |
| PTY | `internal/pty/` | Terminal sessions |
| TUI | `internal/tui/` | Bubble Tea interface |
| Tools | `internal/tool/` | Tool implementations |
| Server | `internal/server/` | Fiber HTTP server |

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
# or
go test ./... -timeout 60s
```

### Run with coverage

```bash
make test-coverage
# or
go test ./... -coverprofile=coverage.out
```

### Build

```bash
make build
# or
go build -o magicode ./cmd/magicode
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

| Phase | Status | Components |
|-------|--------|------------|
| 1 | ✅ Done | CLI, Logging, Global paths, Config, Instance, Bus |
| 2 | ✅ Done | Database, Session, Migration |
| 3 | ✅ Done | AI Providers (Anthropic, OpenAI) |
| 4 | ✅ Done | Tools, Permission |
| 5 | ✅ Done | LSP Integration |
| 6 | ✅ Done | TUI (Bubble Tea) |
| 7 | ✅ Done | PTY & Terminal |
| 8 | ✅ Done | HTTP Server & API |
| 9 | 🚧 In Progress | Polish & Testing |

## License

MIT License - See [LICENSE](LICENSE) for details.

## Related

- [Original TypeScript Implementation](https://github.com/opencode-ai/opencode)
- [Memory Leak Review](../MEMORY_LEAK_REVIEW.md)
- [Go Recreation Plan](../GOLANG_RECREATION_PLAN.md)