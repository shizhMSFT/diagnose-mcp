<!--
SYNC IMPACT REPORT
==================
Version Change: Initial → 1.0.0
Constitution Type: MAJOR (Initial ratification)

Modified Principles:
- NEW: Code Quality & Maintainability
- NEW: Testing Standards (NON-NEGOTIABLE)
- NEW: User Experience Consistency
- NEW: Performance & Reliability

Added Sections:
- Core Principles (4 principles)
- Performance Standards
- Development Workflow & Quality Gates
- Governance

Templates Status:
✅ plan-template.md - Aligned with constitution principles
✅ spec-template.md - Aligned with user story requirements
✅ tasks-template.md - Aligned with testing discipline

Follow-up TODOs:
- None (all placeholders filled)

Commit Message:
docs: ratify constitution v1.0.0 (code quality, testing, UX, performance principles)
-->

# diagnose-mcp Constitution

## Core Principles

### I. Code Quality & Maintainability

**MUST requirements:**
- All code MUST follow consistent style guidelines enforced by automated linting and formatting tools
- Functions and modules MUST have single, well-defined responsibilities (Single Responsibility Principle)
- Public APIs MUST include comprehensive documentation with usage examples
- Code complexity MUST be justified; cyclomatic complexity >10 requires explicit rationale
- No dead code, commented-out blocks, or unused imports permitted in production branches

**Rationale:** As a diagnostic proxy server, code clarity directly impacts debuggability. When users troubleshoot MCP server issues, our codebase itself must not introduce confusion. Clean, maintainable code ensures rapid issue resolution and confident feature additions.

### II. Testing Standards (NON-NEGOTIABLE)

**Test-Driven Development (TDD) workflow:**
1. Write failing tests that capture acceptance criteria
2. Obtain user/stakeholder approval of test scenarios
3. Verify tests fail for the right reasons
4. Implement minimum code to pass tests
5. Refactor while maintaining green tests

**Coverage requirements:**
- Unit tests MUST achieve ≥80% line coverage for business logic
- All public API endpoints MUST have contract tests
- Integration tests MUST cover MCP protocol interactions
- Edge cases and error paths MUST have explicit test coverage

**Test quality standards:**
- Tests MUST be independent, repeatable, and deterministic
- Test names MUST describe behavior, not implementation (e.g., `test_logs_request_when_verbose_enabled` not `test_log_function`)
- Flaky tests are treated as critical bugs and MUST be fixed or removed within one sprint

**Rationale:** Diagnostic tools require extreme reliability. TDD ensures our proxy correctly handles MCP protocol edge cases. Comprehensive testing prevents regression when adding diagnostic features and builds confidence for refactoring.

### III. User Experience Consistency

**Interface design requirements:**
- Command-line interface MUST follow POSIX conventions (stdin/stdout/stderr separation)
- Output formats MUST support both human-readable and machine-parsable modes (JSON, structured logs)
- Error messages MUST be actionable, including suggested remediation steps
- Configuration MUST use standard formats (JSON, YAML, TOML) with schema validation
- Progress indicators MUST be shown for operations >2 seconds

**Documentation standards:**
- All features MUST include quickstart guides with real-world examples
- Breaking changes MUST be documented with migration guides
- Configuration options MUST be documented with valid value ranges and defaults

**Rationale:** Users adopt diagnostic tools when they integrate seamlessly into workflows. Consistent interfaces reduce cognitive load. Clear error messages transform debugging sessions from frustrating to productive.

### IV. Performance & Reliability

**Performance targets:**
- Proxy overhead MUST be <10ms p95 latency for request/response passthrough
- Memory footprint MUST remain <100MB for typical diagnostic sessions
- Startup time MUST be <500ms for responsive CLI experience
- Long-running diagnostic sessions (>1 hour) MUST NOT leak memory

**Reliability requirements:**
- Proxy MUST NOT crash on malformed MCP messages; log and sanitize instead
- Network timeouts MUST be configurable with sensible defaults (30s connect, 5m read)
- All external dependencies MUST have retry logic with exponential backoff
- Diagnostic features MUST degrade gracefully; core proxying continues even if logging fails

**Observability:**
- Structured logging MUST be used throughout (JSON format preferred)
- All MCP protocol violations MUST be logged with message context
- Performance metrics MUST be exposed (request counts, latencies, error rates)

**Rationale:** Diagnostic tools that degrade performance or reliability are abandoned. Our proxy must be invisible to MCP communication while providing deep visibility. Observability ensures we can debug our own diagnostic tool.

## Performance Standards

**Benchmarking discipline:**
- All performance-critical paths MUST have benchmark tests
- Performance regressions >15% require investigation before merge
- Benchmarks MUST run in CI on representative workloads

**Resource management:**
- File handles and network connections MUST be explicitly closed (use context managers)
- Caching strategies MUST have TTL and size limits
- Async I/O MUST be used for network operations to prevent blocking

**Scalability:**
- Design MUST support diagnosing multiple concurrent MCP connections
- Configuration MUST allow rate limiting to prevent resource exhaustion

## Development Workflow & Quality Gates

**Branch strategy:**
- Feature development occurs on branches named `###-feature-name`
- Main branch MUST always pass all tests and quality checks
- No direct commits to main; all changes via pull requests

**Pre-commit requirements:**
- All tests MUST pass (unit, integration, contract)
- Linting and formatting MUST be clean (zero warnings)
- Type checking MUST pass (if using typed language)
- No security vulnerabilities in dependencies (automated scanning)

**Code review gates:**
- All PRs MUST have ≥1 approval from core maintainer
- Constitution compliance MUST be verified (testing standards, documentation, performance)
- Breaking changes MUST be explicitly flagged and justified
- New dependencies MUST be justified for size/license/maintenance status

**Definition of Done:**
- Feature implemented with passing tests (TDD workflow)
- Documentation updated (README, API docs, quickstart guides)
- Performance benchmarks added if applicable
- Integration tests added for MCP protocol interactions
- Changelog updated with user-facing changes

## Governance

**Authority:**
- This Constitution supersedes ad-hoc practices, undocumented conventions, and individual preferences
- Technical decisions MUST align with Core Principles; deviations require explicit justification and amendment

**Amendment process:**
1. Propose amendment with rationale in GitHub issue
2. Document impact on existing codebase and templates
3. Obtain consensus from core maintainers (≥2 approvals)
4. Update version per semantic versioning rules below
5. Create migration plan if affecting existing code
6. Update dependent templates (.specify/templates/*) for consistency

**Versioning policy:**
- **MAJOR**: Backward-incompatible changes (removing principles, changing non-negotiable requirements)
- **MINOR**: Additive changes (new principles, expanded guidance)
- **PATCH**: Clarifications, typo fixes, non-semantic refinements

**Compliance:**
- All pull requests MUST verify alignment with this Constitution
- Constitution violations MUST be justified in PR description or rejected
- Complexity exceptions MUST be documented in plan.md with mitigation strategies
- Annual constitution review to ensure relevance and remove obsolete constraints

**Runtime guidance:**
- For implementation specifics beyond this Constitution, consult `.specify/templates/` and `.github/prompts/`
- When Constitution is silent on a topic, favor simplicity, testability, and user experience

**Version**: 1.0.0 | **Ratified**: 2025-12-02 | **Last Amended**: 2025-12-02
