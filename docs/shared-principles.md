<!-- DOCTOC SKIP -->

# Shared Development Principles

These core principles appear across all SRAT instruction files and guide all development activities.

## Respect Existing Code

The established architecture, style, and patterns of the codebase define your approach. Always follow them.

**Application:**

- Go: Follow go.instructions.md patterns (services, interfaces, error handling)
- TypeScript: Follow typescript-7-es2022.instructions.md (strict types, async patterns)
- ReactJS: Follow reactjs.instructions.md (hooks, testing, composition)
- Python: Follow python.instructions.md (type hints, Home Assistant patterns)

**When in doubt:** Read the file header first but file specific rules override general principles.

## Error Handling

**Core pattern:** Add context when propagating errors, choose between logging OR returning errors (not both), handle at the appropriate level.

**Language-specific:**

- **Go:** Use `fmt.Errorf("%w", err)` for context, `errors.Is()` / `errors.AsType[T]()` for matching
- **TypeScript:** Use try/catch with structured errors, log via project utilities (not console)
- **Python:** Validate input early, raise specific exceptions, log with context
- **React:** Implement Error Boundaries, surface user-facing errors via notification pattern

**Test perspective:** Always test both success and error paths.

## Testing Lifecycle (Common to All Languages)

### Setup and Teardown

- **Before each test (SetupTest/beforeEach):** Initialize dependencies, configure mocks, set fixtures
- **After each test (TearDownTest/afterEach):** Cancel contexts, wait on goroutines/timers, clean up resources
- **Critical order for Go:** Cancel context BEFORE waiting on WaitGroup (prevents deadlock)

### Test Naming

- Use descriptive names reflecting the behavior being tested, not implementation
- Go: `TestMethodName_Scenario_Result` (e.g., `TestGetUser_WithValidID_ReturnsUser`)
- TypeScript/JavaScript: `describe("ComponentName", () => test("should...", ...))`
- Python: `test_method_scenario` or `test_method_should_behavior`

### Test Structure

- One primary test case per test (focus on one behavior)
- Use table-driven tests for multiple scenarios (Go, Python)
- Use subtests (Go: `t.Run()`, JavaScript: `test.describe()`)
- Keep tests independent; no shared state between tests

### Mocking Strategy

- Mock external dependencies (APIs, databases, services)
- Never mock the service/component under test
- Use type-safe mocks (mockio in Go, jest.mock in TS, unittest.mock in Python)
- Set expectations clearly; verify calls when behavior depends on dependency interaction

### Assertions

- **Require** for critical preconditions (stops test on failure)
- **Assert** for non-critical checks (continues test)
- Test both positive and negative cases

## Code Quality

### Maintainability

- Write clear, readable code that other developers can easily understand
- Avoid clever or complex solutions when straightforward approaches will do
- Keep functions focused; extract helpers when logic branches grow
- Add comments that capture intent, not what's obvious from code

### Comments & Documentation

- Prioritize self-documenting code through clear naming and structure
- Comment only when necessary: complex logic, business rules, non-obvious decisions
- Start public API documentation with the symbol name
- Keep documentation close to code; update when code changes

### Performance

- Profile before optimizing; focus on algorithmic improvements first
- Minimize allocations in hot paths; reuse objects when appropriate
- Lazy-load heavy dependencies; defer expensive work until needed
- Batch or debounce high-frequency events

## Security Practices

- **Input validation:** Validate all external input with schema validators or type guards
- **Secrets:** Never hardcode secrets; load from secure storage; rotate regularly
- **Injection prevention:** Use parameterized queries, prepared statements, no shell interpolation
- **Encoding:** Encode untrusted content before rendering; use framework escaping
- **Dependencies:** Patch promptly; monitor advisories; audit regularly

## Documentation & Comments

- Respect existing documentation; keep it current with code changes
- Use specific examples showing expected output when helpful
- Avoid vague language ("simply", "obviously", etc.)
- Use SRAT terminology consistently (see project glossary)

---

## By Language & Context

For language-specific instantiations of these principles, refer to:

- **Go:** `.github/instructions/go.instructions.md`
- **TypeScript:** `.opencode/instructions/typescript-7-es2022.instructions.md`
- **ReactJS:** `.github/instructions/reactjs.instructions.md`
- **Python:** `.github/instructions/python.instructions.md`
- **Testing (Go):** `.github/instructions/backend_test.instructions.md`
- **Testing (TypeScript):** `.github/instructions/fontend_test.instructions.md`

## Using These Principles

When writing or reviewing code:

1. **Respect existing code** - Follow established patterns; read file headers first
2. **Handle errors properly** - Add context, choose logging OR return, handle at right level
3. **Test thoroughly** - Setup/teardown properly, name clearly, test both cases, keep independent
4. **Maintain quality** - Clear code, focused functions, helpful comments, profile before optimizing
5. **Secure by default** - Validate input, protect secrets, prevent injection, audit dependencies

If a principle and a language-specific instruction conflict, the language-specific instruction wins.
