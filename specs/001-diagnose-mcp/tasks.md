# Tasks: diagnose-mcp Proxy Server

**Input**: Design documents from `/specs/001-diagnose-mcp/`
**Prerequisites**: plan.md, spec.md, data-model.md, contracts/

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

**Tests**: Following TDD (NON-NEGOTIABLE per constitution) - all tests must be written FIRST and FAIL before implementation.

## Format: `- [ ] [ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Initialize Go module with `go mod init github.com/shizhMSFT/diagnose-mcp` in repository root
- [x] T002 [P] Create project directory structure (cmd/diagnose-mcp/, internal/, pkg/mcp/, tests/)
- [x] T003 [P] Add dependencies: `go get github.com/gorilla/websocket@v1.5.1 github.com/fsnotify/fsnotify@v1.7.0`
- [x] T004 [P] Create .golangci.yml linter configuration in repository root
- [x] T005 [P] Create Makefile with targets: build, test, lint, clean, run in repository root
- [ ] T006 [P] Create go.work file if needed for local development workspace

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T007 [P] Contract test: CLI flag parsing in tests/contract/cli_interface_test.go
- [x] T008 [P] Contract test: MCP protocol message validation in tests/contract/mcp_protocol_test.go
- [x] T009 [P] Contract test: Log output format validation in tests/contract/log_output_test.go
- [x] T010 [P] Define MCP protocol types (JSON-RPC structures) in pkg/mcp/types.go
- [x] T011 [P] Implement MCP message parser (JSON unmarshaling, type detection) in pkg/mcp/parser.go
- [x] T012 [P] Define configuration types (SessionConfig, ConnectionType) in internal/config/config.go
- [x] T013 [P] Implement CLI flag parsing and validation in internal/config/config.go
- [x] T014 [P] Define logger types (LogEntry, LogLevel, OutputFormat) in internal/logger/types.go
- [x] T015 [P] Implement structured logger with text formatter in internal/logger/logger.go
- [x] T016 [P] Implement structured logger with JSON formatter in internal/logger/formatter.go
- [x] T017 Define ProxySession entity in internal/proxy/session.go
- [x] T018 Define MCPMessage entity in internal/proxy/message.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Proxy Local MCP Server with Request Logging (Priority: P1) 🎯 MVP

**Goal**: Transparent stdio proxy that intercepts and logs all MCP messages (requests, responses, notifications, progress updates) with environment variable pass-through

**Independent Test**: Run `diagnose-mcp ./test-mcp-server`, send tool call request, verify request/response logged to stdout and server operates normally

**Entities Needed**: MCPMessage, ProxySession, LogEntry

### Tests for User Story 1 (TDD - Write FIRST) ⚠️

> **NON-NEGOTIABLE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T019 [P] [US1] Unit test: Local server process spawning in tests/unit/proxy/local_test.go
- [x] T020 [P] [US1] Unit test: Stdio message interception in tests/unit/proxy/message_test.go
- [x] T021 [P] [US1] Unit test: Environment variable pass-through in tests/unit/proxy/local_test.go
- [x] T022 [P] [US1] Integration test: Full local proxy flow with mock server in tests/integration/local_proxy_test.go

### Implementation for User Story 1

- [ ] T023 [P] [US1] Implement local server process spawning (os/exec) in internal/proxy/local.go
- [ ] T024 [P] [US1] Implement stdio pipe setup (stdin/stdout) in internal/proxy/local.go
- [ ] T025 [US1] Implement message interception and forwarding logic in internal/proxy/message.go (depends on T023, T024)
- [ ] T026 [US1] Implement environment variable pass-through in internal/proxy/local.go
- [ ] T027 [US1] Implement MCP message logging integration (LogEntry creation) in internal/proxy/proxy.go
- [ ] T028 [US1] Implement signal handling (SIGTERM/SIGINT) for graceful shutdown in cmd/diagnose-mcp/main.go
- [ ] T029 [US1] Implement main CLI entry point with local mode support in cmd/diagnose-mcp/main.go
- [ ] T030 [US1] Implement error handling and stderr separation in internal/proxy/proxy.go
- [ ] T031 [US1] Add session statistics tracking (message count, bytes) in internal/proxy/session.go

**Checkpoint**: At this point, User Story 1 should be fully functional - can proxy local MCP servers with complete logging

---

## Phase 4: User Story 2 - Proxy Remote MCP Server (Priority: P2)

**Goal**: Extend proxy to support remote MCP servers via HTTP/WebSocket transport

**Independent Test**: Run `diagnose-mcp --remote ws://localhost:8080/mcp`, send request, verify logging without requiring local server

**Entities Needed**: MCPMessage, ProxySession, LogEntry (reuse from US1)

### Tests for User Story 2 (TDD - Write FIRST) ⚠️

- [ ] T032 [P] [US2] Unit test: WebSocket connection establishment in tests/unit/proxy/remote_test.go
- [ ] T033 [P] [US2] Unit test: HTTP connection handling in tests/unit/proxy/remote_test.go
- [ ] T034 [P] [US2] Unit test: Network error handling and retries in tests/unit/proxy/remote_test.go
- [ ] T035 [P] [US2] Integration test: Full remote proxy flow with mock WebSocket server in tests/integration/remote_proxy_test.go

### Implementation for User Story 2

- [ ] T036 [P] [US2] Implement WebSocket client connection (gorilla/websocket) in internal/proxy/remote.go
- [ ] T037 [P] [US2] Implement HTTP connection handling in internal/proxy/remote.go
- [ ] T038 [US2] Implement remote message forwarding (WebSocket frames) in internal/proxy/remote.go
- [ ] T039 [US2] Add --remote flag parsing and URL validation in internal/config/validation.go
- [ ] T040 [US2] Integrate remote mode into main CLI in cmd/diagnose-mcp/main.go
- [ ] T041 [US2] Implement connection error handling and logging in internal/proxy/remote.go
- [ ] T042 [US2] Add network timeout configuration (30s connect, 5m read) in internal/config/config.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently - local and remote proxying complete

---

## Phase 5: User Story 3 - Monitor External Files During Session (Priority: P3)

**Goal**: Monitor text files for changes (creation, line appends, deletion) and log events alongside MCP traffic

**Independent Test**: Run `diagnose-mcp --watch /tmp/test.log ./server`, append lines to test.log, verify file events appear in output without MCP traffic

**Entities Needed**: FileWatch, LogEntry, ProxySession (for config)

### Tests for User Story 3 (TDD - Write FIRST) ⚠️

- [ ] T043 [P] [US3] Unit test: File creation detection in tests/unit/watcher/events_test.go
- [ ] T044 [P] [US3] Unit test: Line append detection in tests/unit/watcher/events_test.go
- [ ] T045 [P] [US3] Unit test: File deletion detection in tests/unit/watcher/events_test.go
- [ ] T046 [P] [US3] Unit test: File state tracking (size, line count) in tests/unit/watcher/state_test.go
- [ ] T047 [P] [US3] Integration test: File watching with real filesystem in tests/integration/file_watch_test.go

### Implementation for User Story 3

- [ ] T048 [P] [US3] Define FileWatch entity in internal/watcher/state.go
- [ ] T049 [P] [US3] Define FileEvent types and structures in internal/watcher/events.go
- [ ] T050 [US3] Implement fsnotify watcher initialization in internal/watcher/watcher.go (depends on T048, T049)
- [ ] T051 [US3] Implement file event detection (created, modified, deleted) in internal/watcher/events.go
- [ ] T052 [US3] Implement line count tracking for text files in internal/watcher/state.go
- [ ] T053 [US3] Implement file event logging integration in internal/watcher/watcher.go
- [ ] T054 [US3] Add --watch flag parsing (multiple allowed) in internal/config/config.go
- [ ] T055 [US3] Integrate file watching into proxy session in internal/proxy/proxy.go
- [ ] T056 [US3] Implement chronological event interleaving (MCP + file events) in internal/logger/logger.go

**Checkpoint**: All user stories (P1, P2, P3) are now independently functional and can be used together

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T057 [P] Implement benchmark test for message passthrough latency in tests/benchmark/proxy_bench_test.go
- [ ] T058 [P] Implement benchmark test for logging overhead in tests/benchmark/logger_bench_test.go
- [ ] T059 [P] Add memory profiling test for 1-hour sessions in tests/benchmark/memory_bench_test.go
- [ ] T060 [P] Verify ≥80% test coverage with `go test -cover` across all packages
- [ ] T061 [P] Add godoc comments to all public functions in pkg/mcp/, internal/proxy/, internal/logger/, internal/watcher/
- [ ] T062 [P] Verify cyclomatic complexity ≤10 with gocyclo or similar tool
- [ ] T063 [P] Run golangci-lint and fix all issues (gofmt, go vet, staticcheck)
- [ ] T064 Verify performance targets: <10ms p95 latency, <100MB memory, <500ms startup
- [ ] T065 Verify all constitution gates passed (TDD, coverage, performance, documentation)
- [ ] T066 [P] Add error message clarity improvements with remediation steps
- [ ] T067 [P] Validate quickstart.md examples work end-to-end
- [ ] T068 [P] Add --help message implementation in cmd/diagnose-mcp/main.go
- [ ] T069 [P] Add --version flag implementation in cmd/diagnose-mcp/main.go
- [ ] T070 Security audit: Input validation, no arbitrary code execution, safe path handling

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - **BLOCKS all user stories**
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Reuses MCPMessage/LogEntry from US1 but independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Adds FileWatch entity, independently testable

### Within Each User Story

**TDD Workflow (NON-NEGOTIABLE)**:
1. Write tests FIRST (contract, unit, integration)
2. Run tests - they MUST FAIL
3. Get user/constitution approval on test coverage
4. Implement minimum code to pass tests
5. Refactor while keeping tests green

**Implementation Order**:
- Tests before implementation (Red-Green-Refactor)
- Entity definitions before services
- Services before CLI integration
- Core implementation before error handling
- Story complete before moving to next priority

### Parallel Opportunities

**Phase 1 (Setup)**: Tasks T002-T006 can all run in parallel

**Phase 2 (Foundational)**: 
- Tests T007-T009 can run in parallel
- Type definitions T010-T018 can run in parallel after tests fail

**Within User Story 1**:
- Tests T019-T022 can run in parallel
- Implementation T023-T024 can run in parallel after tests fail
- T025 depends on T023 and T024 completing

**Within User Story 2**:
- Tests T032-T035 can run in parallel
- Implementation T036-T037 can run in parallel after tests fail

**Within User Story 3**:
- Tests T043-T047 can run in parallel
- Implementation T048-T049 can run in parallel after tests fail
- T050 depends on T048 and T049 completing

**Phase 6 (Polish)**:
- Benchmark tests T057-T059 can run in parallel
- Documentation T061, lint T063, help/version T068-T069 can run in parallel

**Cross-Story Parallel**:
- Once Foundational (Phase 2) completes:
  - One developer can work on User Story 1
  - Another developer can work on User Story 2
  - A third can work on User Story 3
  - All stories are independent and can merge separately

---

## Parallel Execution Examples

### Phase 2: Foundational (After Setup)

```bash
# Launch all contract tests together:
go test ./tests/contract/cli_interface_test.go    # T007
go test ./tests/contract/mcp_protocol_test.go     # T008
go test ./tests/contract/log_output_test.go       # T009

# Launch all type definitions together (after tests fail):
# Edit pkg/mcp/types.go                           # T010
# Edit pkg/mcp/parser.go                          # T011
# Edit internal/config/config.go                  # T012, T013
# Edit internal/logger/types.go                   # T014
# Edit internal/logger/logger.go                  # T015
# Edit internal/logger/formatter.go               # T016
```

### User Story 1: Local Proxy (After Foundational)

```bash
# Launch all tests together:
go test ./tests/unit/proxy/local_test.go          # T019, T021
go test ./tests/unit/proxy/message_test.go        # T020
go test ./tests/integration/local_proxy_test.go   # T022

# Launch parallel implementations (after tests fail):
# Edit internal/proxy/local.go                    # T023, T024, T026
# Edit internal/proxy/message.go                  # T025 (depends on T023, T024)
```

### User Story 2: Remote Proxy (Independent of US1)

```bash
# Launch all tests together:
go test ./tests/unit/proxy/remote_test.go         # T032, T033, T034
go test ./tests/integration/remote_proxy_test.go  # T035

# Launch parallel implementations (after tests fail):
# Edit internal/proxy/remote.go                   # T036, T037, T038
# Edit internal/config/validation.go              # T039
```

### User Story 3: File Watching (Independent of US1 and US2)

```bash
# Launch all tests together:
go test ./tests/unit/watcher/events_test.go       # T043, T044, T045
go test ./tests/unit/watcher/state_test.go        # T046
go test ./tests/integration/file_watch_test.go    # T047

# Launch parallel implementations (after tests fail):
# Edit internal/watcher/state.go                  # T048
# Edit internal/watcher/events.go                 # T049
# Edit internal/watcher/watcher.go                # T050 (depends on T048, T049)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete **Phase 1: Setup** (T001-T006)
2. Complete **Phase 2: Foundational** (T007-T018) - **CRITICAL - blocks all stories**
3. Complete **Phase 3: User Story 1** (T019-T031) - Write tests FIRST, fail, then implement
4. **STOP and VALIDATE**: 
   - Run all User Story 1 tests
   - Manually test: `diagnose-mcp ./test-server`
   - Verify all MCP messages logged
   - Verify environment variables passed through
   - Check constitution gates (TDD ✅, coverage ≥80%, latency <10ms)
5. **Demo/Deploy MVP**: User Story 1 is production-ready local proxy

**At this point**: Users can diagnose local MCP servers with full request/response logging

### Incremental Delivery

1. **Foundation** (Setup + Foundational) → Build infrastructure ready ✅
2. **MVP** (User Story 1) → Test independently → Deploy/Demo (local proxy working!)
3. **Extension** (User Story 2) → Test independently → Deploy/Demo (remote proxy added!)
4. **Enhancement** (User Story 3) → Test independently → Deploy/Demo (file watching added!)
5. **Polish** (Phase 6) → Performance benchmarks, documentation, final quality gates

Each story adds value without breaking previous stories - **truly incremental delivery**

### Parallel Team Strategy

With multiple developers (after Foundational Phase):

1. **Team completes Setup + Foundational together** (critical path)
2. **Once Foundational is done** (T018 complete):
   - **Developer A**: User Story 1 (T019-T031) - Local proxy
   - **Developer B**: User Story 2 (T032-T042) - Remote proxy
   - **Developer C**: User Story 3 (T043-T056) - File watching
3. **Stories complete independently**, merge separately, integrate cleanly
4. **Team reconvenes** for Phase 6 (Polish & Performance validation)

**Benefits**:
- Maximum parallelization after foundational work
- Each developer owns a complete, testable feature
- Merge conflicts minimized (different files)
- Can ship P1 alone if P2/P3 aren't ready

---

## TDD Workflow Checklist (Per Constitution)

For EVERY implementation task, follow this strict sequence:

1. ✅ **Write test** - Create failing test that describes desired behavior
2. ✅ **Run test** - Verify test FAILS with expected error
3. ✅ **Get approval** - Show test to reviewer/constitution gate
4. ✅ **Implement** - Write minimum code to make test pass
5. ✅ **Run test** - Verify test now PASSES
6. ✅ **Refactor** - Improve code while keeping tests green
7. ✅ **Coverage check** - Verify ≥80% coverage maintained
8. ✅ **Commit** - Commit test + implementation together

**Example for T019 (Unit test: Local server process spawning)**:

```go
// Step 1: Write test in tests/unit/proxy/local_test.go
func TestLocalProxy_SpawnsChildProcess(t *testing.T) {
    proxy := NewLocalProxy("./test-server", []string{"--arg1"})
    err := proxy.Start()
    require.NoError(t, err)
    assert.True(t, proxy.IsRunning())
}

// Step 2: Run test - should FAIL (local.go doesn't exist yet)
// Step 3: Get approval - review test with team
// Step 4: Implement in internal/proxy/local.go
// Step 5: Run test - should PASS now
// Step 6: Refactor - clean up code
// Step 7: go test -cover - verify ≥80%
// Step 8: git commit -m "feat: local proxy process spawning"
```

---

## Constitution Compliance Tracking

### TDD Enforcement (NON-NEGOTIABLE)

- [ ] **T007-T009**: Foundational contract tests written FIRST
- [ ] **T019-T022**: User Story 1 tests written FIRST
- [ ] **T032-T035**: User Story 2 tests written FIRST
- [ ] **T043-T047**: User Story 3 tests written FIRST
- [ ] **All tests FAIL** before implementation starts
- [ ] **Red-Green-Refactor** cycle followed for every task

### Performance Gates

- [ ] **T057**: Latency benchmark <10ms p95 (constitution requirement)
- [ ] **T059**: Memory footprint <100MB for 1-hour sessions (constitution requirement)
- [ ] **T064**: Startup time <500ms (constitution requirement)

### Code Quality Gates

- [ ] **T060**: Test coverage ≥80% (constitution requirement)
- [ ] **T061**: All public functions documented (constitution requirement)
- [ ] **T062**: Cyclomatic complexity ≤10 (constitution requirement)
- [ ] **T063**: Linting passed (gofmt, go vet, staticcheck)

### User Experience Gates

- [ ] **T066**: Error messages actionable with remediation
- [ ] **T067**: Quickstart examples validated end-to-end
- [ ] **T068-T069**: Help and version flags implemented

**Final Gate**: Task T065 validates ALL constitution requirements before release

---

## Task Summary

- **Total Tasks**: 70
- **Setup**: 6 tasks (T001-T006)
- **Foundational**: 12 tasks (T007-T018) - **BLOCKS all stories**
- **User Story 1 (P1)**: 13 tasks (T019-T031) - MVP
- **User Story 2 (P2)**: 11 tasks (T032-T042) 
- **User Story 3 (P3)**: 14 tasks (T043-T056)
- **Polish**: 14 tasks (T057-T070)

**Parallel Opportunities**: 35+ tasks can run in parallel (50% of total workload)

**MVP Scope** (minimum shippable product):
- Phase 1: Setup (6 tasks)
- Phase 2: Foundational (12 tasks) 
- Phase 3: User Story 1 (13 tasks)
- **Total: 31 tasks** for working local MCP proxy with full logging

**Suggested First Milestone**: Complete MVP (T001-T031), validate constitution gates, ship User Story 1

---

## Notes

- **[P] markers**: Tasks with different file targets, no sequential dependencies
- **[Story] labels**: Map task to specific user story for traceability
- **TDD is NON-NEGOTIABLE**: Tests written first, must fail, then implement
- **Each user story is independently testable**: Can ship P1 alone, add P2/P3 incrementally
- **Constitution gates enforced**: Coverage ≥80%, latency <10ms, memory <100MB
- **Commit frequently**: After each task or logical group
- **Stop at checkpoints**: Validate story independently before moving forward
- **Avoid**: Vague tasks, same-file conflicts, cross-story dependencies that break independence
