# Implementation Plan: diagnose-mcp Proxy Server

**Branch**: `001-diagnose-mcp` | **Date**: 2025-12-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-diagnose-mcp/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Build a transparent MCP protocol proxy server that intercepts and logs all MCP messages (requests, responses, notifications, progress updates) between clients and servers. Supports both local stdio-based and remote HTTP/WebSocket MCP servers. Includes file monitoring capability to correlate MCP traffic with external log files. Technical approach: Go 1.25.4 CLI application with goroutines for concurrent message handling, polling-based file watching with fspoll-go, structured logging with JSON/text output formats.

## Technical Context

**Language/Version**: Go 1.25.4  
**Primary Dependencies**: 
- `github.com/gorilla/websocket` - WebSocket client for remote MCP servers
- `github.com/shizhMSFT/fspoll-go` - Polling-based file system watching
- Standard library: `os/exec`, `encoding/json`, `log/slog`, `net/http`, `io`, `bufio`, `context`, `os/signal`

**Storage**: N/A (stateless proxy, logs to stdout)  
**Testing**: Go testing framework (`testing` package), table-driven tests, test coverage with `go test -cover`  
**Target Platform**: Cross-platform CLI (Linux, macOS, Windows)  
**Project Type**: Single project (CLI application)  
**Performance Goals**: 
- <10ms p95 latency for message passthrough (constitution requirement)
- <100ms logging overhead per message
- <500ms startup time
- Startup time <500ms

**Constraints**: 
- <10ms p95 latency overhead (constitution)
- <100MB memory footprint for 1-hour sessions (constitution)
- Must not modify MCP protocol messages
- Cross-platform compatibility (Windows/Unix process/signal handling)

**Scale/Scope**: 
- Single MCP client-server pair per invocation
- Support up to 100 concurrent file watches
- Handle MCP messages up to 10MB (reasonable JSON-RPC limit)
- Long-running sessions (1+ hours) without memory leaks

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Code Quality & Maintainability
- [ ] **GATE**: All public functions documented with usage examples
- [ ] **GATE**: Cyclomatic complexity ≤10 for all functions (or justified in Complexity Tracking)
- [ ] **GATE**: Automated linting configured (`gofmt`, `go vet`, `staticcheck`)
- [ ] **GATE**: No dead code, commented-out blocks, or unused imports
- [x] **PASS**: Single Responsibility Principle - modules separated by concern (proxy, logging, file watching, CLI)

**Status**: ⚠️ GATES NOT YET VERIFIED - Check after implementation

### II. Testing Standards (NON-NEGOTIABLE)
- [ ] **GATE**: TDD workflow followed - tests written before implementation
- [ ] **GATE**: Unit test coverage ≥80% for business logic
- [ ] **GATE**: Contract tests for MCP protocol message handling
- [ ] **GATE**: Integration tests for stdio/WebSocket proxying
- [ ] **GATE**: Edge case tests for malformed messages, crashes, signal handling
- [ ] **GATE**: Tests are independent, repeatable, deterministic
- [ ] **GATE**: Test names describe behavior (e.g., `TestProxyForwardsProgressNotifications`)

**Status**: ❌ CRITICAL - Must implement TDD from start

### III. User Experience Consistency
- [x] **PASS**: CLI follows POSIX conventions (stdin for MCP, stdout for logs, stderr for errors)
- [x] **PASS**: Supports both human-readable and JSON output formats
- [ ] **GATE**: Error messages are actionable with remediation steps
- [ ] **GATE**: Configuration validated with clear error messages
- [ ] **GATE**: Quickstart guide with real-world examples

**Status**: ⚠️ PARTIALLY MET - Need to implement error handling and docs

### IV. Performance & Reliability
- [ ] **GATE**: <10ms p95 latency verified by benchmarks
- [ ] **GATE**: <100MB memory footprint verified for 1-hour sessions
- [ ] **GATE**: <500ms startup time verified
- [ ] **GATE**: Graceful degradation: proxy continues if logging fails
- [ ] **GATE**: No panics on malformed MCP messages (recover and log)
- [ ] **GATE**: Structured logging (slog) with JSON output
- [ ] **GATE**: Benchmark tests for message forwarding

**Status**: ❌ CRITICAL - Must verify all performance targets

### Performance Standards
- [ ] **GATE**: Benchmark tests in place for message passthrough
- [ ] **GATE**: CI configured to detect >15% performance regressions
- [ ] **GATE**: Context-based cancellation for goroutines (no leaks)
- [ ] **GATE**: File handles and connections explicitly closed (defer patterns)

**Status**: ❌ NOT YET IMPLEMENTED

**Overall Constitution Compliance**: ❌ **BLOCKED** - Cannot proceed to Phase 0 research until TDD commitment established and performance benchmarking planned

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/
└── diagnose-mcp/
    └── main.go              # CLI entry point, flag parsing, signal handling

internal/
├── proxy/
│   ├── proxy.go             # Core proxy orchestration
│   ├── local.go             # Local stdio-based MCP server handling
│   ├── remote.go            # Remote HTTP/WebSocket MCP server handling
│   └── message.go           # MCP message interception and forwarding
├── logger/
│   ├── logger.go            # Structured logging (slog)
│   ├── formatter.go         # Human-readable vs JSON output
│   └── types.go             # Log entry structures
├── watcher/
│   └── watcher.go           # File watching using fspoll-go with direct logging
└── config/
    ├── config.go            # Configuration parsing from CLI flags
    └── validation.go        # Input validation

pkg/
└── mcp/
    ├── types.go             # MCP protocol message types
    ├── parser.go            # JSON-RPC message parsing
    └── validator.go         # Protocol conformance checking (optional)

tests/
├── contract/
│   ├── mcp_protocol_test.go     # Contract tests for MCP message handling
│   └── cli_interface_test.go   # CLI flag parsing and output format tests
├── integration/
│   ├── local_proxy_test.go     # E2E tests with mock stdio MCP server
│   ├── remote_proxy_test.go    # E2E tests with mock HTTP/WebSocket server
│   └── file_watch_test.go      # File monitoring integration tests
└── unit/
    ├── proxy/
    │   └── *_test.go            # Unit tests for proxy logic
    ├── logger/
    │   └── *_test.go            # Unit tests for logging logic
    └── watcher/
        └── *_test.go            # Unit tests for file watching logic

go.mod                       # Go module definition
go.sum                       # Dependency checksums
.golangci.yml                # Linter configuration
Makefile                     # Build, test, lint targets
```

**Structure Decision**: Selected Go standard project layout (Option 1: Single project). Uses `cmd/` for entry points, `internal/` for private application code (prevents external imports), `pkg/` for reusable MCP protocol types (could be extracted to library), `tests/` with clear separation by test type. This structure supports TDD workflow with isolated, testable modules.

## Complexity Tracking

> **No constitution violations detected - section left empty as per template guidance**

All design choices align with constitution principles:
- Single project structure (no unnecessary complexity)
- Standard Go project layout (widely accepted convention)
- Minimal dependencies (gorilla/websocket, fspoll-go - both well-maintained)
- Direct implementation without abstraction layers (YAGNI principle)

---

## Phase 0: Research & Unknowns Resolution

**Status**: ✅ **COMPLETE** (see [research.md](research.md))

**Key Decisions**:
- MCP Protocol: JSON-RPC 2.0 with newline-delimited stdio transport
- WebSocket: `gorilla/websocket` v1.5.1 (mature, battle-tested)
- File Watching: `github.com/shizhMSFT/fspoll-go` v0.1.0 (polling-based, cross-platform reliable)
- Logging: `log/slog` standard library (sufficient performance, native)
- Process Management: `os/exec` standard library
- Signal Handling: `os/signal` with context cancellation

All NEEDS CLARIFICATION items resolved. Technology stack validated.

---

## Phase 1: Design & Contracts

**Status**: ✅ **COMPLETE**

### Deliverables Generated

1. **[data-model.md](data-model.md)** - Entity definitions
   - MCPMessage: Immutable value object for protocol messages
   - ProxySession: Session state and configuration
   - FileWatch: Monitored file state and event history
   - LogEntry: Diagnostic log line with formatting

2. **[contracts/mcp-protocol.md](contracts/mcp-protocol.md)** - MCP message schemas
   - JSON-RPC 2.0 structure (requests, responses, notifications, progress)
   - Transport framing (stdio, HTTP, WebSocket)
   - Message type detection algorithm
   - Validation rules

3. **[contracts/cli-interface.md](contracts/cli-interface.md)** - CLI specification
   - Command syntax (local vs remote modes)
   - Flags: `--remote`, `--watch`, `--verbose`, `--json`
   - Exit codes (0-4, 130, 143)
   - Usage examples

4. **[contracts/log-output.md](contracts/log-output.md)** - Log format specification
   - Text format (human-readable with timestamps)
   - JSON format (machine-parsable)
   - Field specifications for all log types
   - Verbosity levels (summary vs verbose)

5. **[quickstart.md](quickstart.md)** - User guide
   - Installation instructions
   - Quick start examples (local, remote, file watching)
   - Common use cases (debugging, profiling, protocol learning)
   - Troubleshooting tips

### Constitution Re-Check

**After Phase 1 Design**:
- ✅ User Experience: CLI interface follows POSIX, supports JSON/text
- ✅ Observability: Structured logging designed with slog
- ✅ Simplicity: No over-engineering, direct implementation
- ⚠️ Testing: TDD must be enforced during implementation (Phase 2)
- ⚠️ Performance: Benchmarks must be implemented with code (Phase 2)

---

## Next Steps

**Command to proceed**: `/speckit.tasks`

The `/speckit.tasks` command will generate `tasks.md` based on this plan, spec, and design documents. Tasks will be organized by user story (P1, P2, P3) to enable independent, incremental implementation.

**Before starting tasks**:
1. ✅ Commit this plan and all Phase 0/1 documents
2. ✅ Review constitution gates - understand TDD requirement
3. ✅ Set up Go development environment (Go 1.25.4, linter, editor)
4. ⚠️ Prepare to write tests FIRST before any implementation

---

## Plan Status Summary

| Phase | Status | Artifacts |
|-------|--------|-----------|
| **Setup** | ✅ Complete | Technical Context, Constitution Check, Project Structure |
| **Phase 0: Research** | ✅ Complete | [research.md](research.md) - All tech decisions made |
| **Phase 1: Design** | ✅ Complete | [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md) |
| **Phase 2: Tasks** | ⏸️ Ready | Run `/speckit.tasks` to generate task breakdown |

---

**Implementation Plan Complete** - Ready for task generation and TDD implementation workflow.

**Key Reminders**:
- 🔴 **TDD is NON-NEGOTIABLE** - Write tests first, see them fail, then implement
- ⚡ **Performance gates** - <10ms latency, <100MB memory, <500ms startup
- 📊 **80% test coverage** - Constitution requirement for business logic
- 🎯 **User stories are MVP slices** - P1 alone should be valuable and testable
