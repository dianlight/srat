<!-- DOCTOC SKIP -->

---

description: 'Guidelines for TypeScript Development targeting TypeScript 6.0+ and ES2022 output'
applyTo: '**/\*.ts,**/\*.tsx'

---

# TypeScript Development

> These instructions assume projects are built with TypeScript 6.0+ (or newer) compiling to an ES2022 JavaScript baseline. The project uses `@typescript/native-preview` (tsgo) which is the TypeScript 7.0 (Go-based) preview compiler.

## TypeScript Version and Tooling

- **TypeScript Version**: 6.0 Beta / 7.0 Preview (tsgo)
- **Target**: ES2022 with modern ECMAScript features
- **Type Checking**: Uses `bun tsgo --noEmit` command (not regular `tsc`)
- **Migration Guide**: See `frontend/TYPESCRIPT_MIGRATION.md` for upgrade details

### TypeScript 6.0/7.0 Key Changes

**Removed Deprecated Flags** (Do not use):
- ❌ `experimentalDecorators` - Use native decorators instead
- ❌ `useDefineForClassFields: false` - ES2022+ requires default `true`
- ❌ `target: es5` - ES2015+ is the minimum
- ❌ Classic module resolution - Use `bundler` or `node`

**Enabled Strict Flags**:
- ✅ `noImplicitOverride: true` - Requires explicit `override` keyword on class methods
- 🚧 `noUncheckedIndexedAccess: true` - TODO: See migration guide for implementation plan

**Performance Benefits**:
- 20-50% faster builds with `types: []` configuration
- Better type inference and consistency

## Core Intent

- Respect the existing architecture and coding standards.
- Prefer readable, explicit solutions over clever shortcuts.
- Extend current abstractions before inventing new ones.
- Prioritize maintainability and clarity, short methods and classes, clean code.

## General Guardrails

- Target TypeScript 6.0+ / ES2022 and prefer native features over polyfills.
- Use pure ES modules; never emit `require`, `module.exports`, or CommonJS helpers.
- Rely on the project's build, lint, and test scripts unless asked otherwise.
- Note design trade-offs when intent is not obvious.
- Follow the TypeScript 6.0/7.0 migration guide when making config changes.

## Project Organization

- Follow the repository's folder and responsibility layout for new code.
- Use kebab-case filenames (e.g., `user-session.ts`, `data-service.ts`) unless told otherwise.
- Keep tests, types, and helpers near their implementation when it aids discovery.
- Reuse or extend shared utilities before adding new ones.

## Naming & Style

- Use PascalCase for classes, interfaces, enums, and type aliases; camelCase for everything else.
- Skip interface prefixes like `I`; rely on descriptive names.
- Name things for their behavior or domain meaning, not implementation.

## Formatting & Style

- Run the repository's lint/format scripts (e.g., `npm run lint`) before submitting.
- Match the project's indentation, quote style, and trailing comma rules.
- Keep functions focused; extract helpers when logic branches grow.
- Favor immutable data and pure functions when practical.

## Type System Expectations

- Avoid `any` (implicit or explicit); prefer `unknown` plus narrowing.
- Use discriminated unions for realtime events and state machines.
- Centralize shared contracts instead of duplicating shapes.
- Express intent with TypeScript utility types (e.g., `Readonly`, `Partial`, `Record`).
- With `noImplicitOverride` enabled, use `override` keyword for methods that override parent class methods.

## Class Inheritance and Override Keyword

With `noImplicitOverride: true` enabled, you **must** use the `override` keyword when overriding methods from a parent class:

```typescript
export class MyComponent extends Component<Props, State> {
  // ✅ CORRECT - override keyword required
  public override componentDidMount() {
    // ...
  }

  public override render() {
    return <div>...</div>;
  }

  // ❌ INCORRECT - missing override keyword
  // public componentDidMount() { ... }
}
```

## Async, Events & Error Handling

- Use `async/await`; wrap awaits in try/catch with structured errors.
- Guard edge cases early to avoid deep nesting.
- Send errors through the project's logging/telemetry utilities.
- Surface user-facing errors via the repository's notification pattern.
- Debounce configuration-driven updates and dispose resources deterministically.

## Architecture & Patterns

- Follow the repository's dependency injection or composition pattern; keep modules single-purpose.
- Observe existing initialization and disposal sequences when wiring into lifecycles.
- Keep transport, domain, and presentation layers decoupled with clear interfaces.
- Supply lifecycle hooks (e.g., `initialize`, `dispose`) and targeted tests when adding services.

## External Integrations

- Instantiate clients outside hot paths and inject them for testability.
- Never hardcode secrets; load them from secure sources.
- Apply retries, backoff, and cancellation to network or IO calls.
- Normalize external responses and map errors to domain shapes.

## Security Practices

- Validate and sanitize external input with schema validators or type guards.
- Avoid dynamic code execution and untrusted template rendering.
- Encode untrusted content before rendering HTML; use framework escaping or trusted types.
- Use parameterized queries or prepared statements to block injection.
- Keep secrets in secure storage, rotate them regularly, and request least-privilege scopes.
- Favor immutable flows and defensive copies for sensitive data.
- Use vetted crypto libraries only.
- Patch dependencies promptly and monitor advisories.

## Configuration & Secrets

- Reach configuration through shared helpers and validate with schemas or dedicated validators.
- Handle secrets via the project's secure storage; guard `undefined` and error states.
- Document new configuration keys and update related tests.

## UI & UX Components

- Sanitize user or external content before rendering.
- Keep UI layers thin; push heavy logic to services or state managers.
- Use messaging or events to decouple UI from business logic.

## Testing Expectations

- Add or update unit tests with the project's framework and naming style.
- Expand integration or end-to-end suites when behavior crosses modules or platform APIs.
- Run targeted test scripts for quick feedback before submitting.
- Avoid brittle timing assertions; prefer fake timers or injected clocks.
- Tests must follow patterns in `frontend/__tests__` directories.

## Performance & Reliability

- Lazy-load heavy dependencies and dispose them when done.
- Defer expensive work until users need it.
- Batch or debounce high-frequency events to reduce thrash.
- Track resource lifetimes to prevent leaks.

## Documentation & Comments

- Add JSDoc to public APIs; include `@remarks` or `@example` when helpful.
- Write comments that capture intent, and remove stale notes during refactors.
- Update architecture or design docs when introducing significant patterns.

## TypeScript Configuration Best Practices

When modifying `tsconfig.json`:

1. **Do not re-introduce deprecated flags**:
   - No `experimentalDecorators`
   - No `useDefineForClassFields: false`
   - No `target: es5` or older
   - No classic module resolution

2. **Maintain strict type checking**:
   - Keep `strict: true`
   - Keep `noImplicitOverride: true`
   - Consider enabling `noUncheckedIndexedAccess` after reviewing migration guide

3. **Performance optimizations**:
   - Keep `types: []` for faster builds
   - Use `incremental: true` for caching
   - Target ES2022 or newer for modern features

4. **Reference the migration guide**:
   - See `frontend/TYPESCRIPT_MIGRATION.md` for detailed upgrade information
   - Check TODO items before enabling additional strict flags

## Common Patterns for TypeScript 6.0+

### Using Native Decorators (Not Experimental)

```typescript
// ✅ CORRECT - Native decorators (TypeScript 6.0+)
function logged(target: any, context: ClassMethodDecoratorContext) {
  return function(...args: any[]) {
    console.log(`Calling ${String(context.name)}`);
    return target.apply(this, args);
  };
}

class MyClass {
  @logged
  myMethod() { }
}
```

### Class Field Initialization (ES2022+ Semantics)

```typescript
// ✅ CORRECT - Class fields with ES2022+ semantics
class MyComponent {
  // Field initializers run after super() call
  public state = { count: 0 };
  
  constructor() {
    // super() called first (if extending)
    // then field initializers run
    // then constructor body runs
  }
}
```

### Indexed Access Safety (Future Enhancement)

When `noUncheckedIndexedAccess` is enabled:

```typescript
// Future pattern (after migration)
const items = ['a', 'b', 'c'];
const item = items[0]; // Type: string | undefined

// Use optional chaining or null checks
const length = items[0]?.length;
const value = items[0] ?? 'default';

// Or explicit type guards
if (items[0] !== undefined) {
  const definiteValue = items[0]; // Type: string
}
```

## Migration Resources

- **Migration Guide**: `frontend/TYPESCRIPT_MIGRATION.md`
- **Implementation Summary**: `TYPESCRIPT_6_IMPLEMENTATION_SUMMARY.md`
- **Official Release Notes**: [TypeScript 6.0 Beta](https://devblogs.microsoft.com/typescript/announcing-typescript-6-0-beta/)
- **TypeScript 7.0 Discussion**: [microsoft/typescript-go](https://github.com/microsoft/typescript-go/discussions/825)
