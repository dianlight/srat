<!-- DOCTOC SKIP -->

# [REFACTOR]: Backend Code Quality — Anti-patterns & Instruction Violations

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:**
**PR Link:** https://github.com/dianlight/srat/pull/600

## 🎯 Objective

Eliminate a cluster of recurring Go anti-patterns in the backend: `interface{}` usage that should be `any` (Go 1.18+), the old `wg.Add(1)` + `go func() { defer wg.Done() }()` pattern that should be `wg.Go()` (Go 1.25+), the `errors.As(err, &target)` pattern that should use `errors.AsType[T](err)` (Go 1.26+), highly duplicated stdout/stderr consumption boilerplate in all filesystem adapters, context-less `slog.*` calls where a `context.Context` is in scope, and two empty marker interfaces that serve no purpose.

## 🛠️ Technical Specifications

- **Inputs:** Backend Go source files in `backend/src/`
- **Outputs:** Idiomatic Go 1.26 code with reduced duplication and no regressions
- **Dependencies:** Go standard library (`sync`, `errors`, `log/slog`), `gitlab.com/tozd/go/errors`

## 📝 Task List

- [ ] Task 1: Replace all `interface{}` with `any` in non-generated backend source files (`addons_service.go`, `ntfs_adapter.go`, `addon_config_watcher_service.go`, `problem_ha_bridge.go`)
- [ ] Task 2: Migrate `errors.As(err, &e)` calls in `unixsamba/unixsamba.go` to `errors.AsType[T](err)` pattern (5 occurrences)
- [ ] Task 3: Replace `wg.Add(1)` + `go func() { defer wg.Done() }()` with `wg.Go(func() { ... })` across all filesystem adapters (60 occurrences)
- [ ] Task 4: Extract duplicated stdout/stderr channel-consumption loop from filesystem adapters into a shared helper on `baseAdapter` (or as a package-level helper) — affected adapters: ext4, xfs, f2fs, exfat, gfs2, hfsplus, ntfs, btrfs, vfat, reiserfs, zfs
- [ ] Task 5: Fix context-less `slog.Debug/Error/Warn` calls in `filesystem_service.go` and `server_process_service.go` to use `slog.DebugContext` / `slog.ErrorContext` / `slog.WarnContext` where a `context.Context` is available
- [ ] Task 6: Address `FIXME` in `broadcaster_service.go:202` — `go broker.sendToHomeAssistant(msg)` should be wired as a proper broadcast listener, not a naked goroutine
- [ ] Task 7: Remove or add methods to empty marker interfaces `AddonConfigWatcherServiceInterface` (`addon_config_watcher_service.go:29`) and `ProblemHABridgeInterface` (`problem_ha_bridge.go:16`) — if they are just marker types, document the intent; if they are stubs, complete them
- [ ] Task 8: Unit testing — run affected package tests first, then `go test ./...` from `backend/src` with zero new failures
- [ ] Task 9: Integration — run `mise run //backend:build` and `mise run //backend:test` to confirm no regressions
- [ ] Task 10: Capture lessons learned and update documentation
- [x] Task 11: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)

### Task 1 — `interface{}` → `any`

Files and locations:
- `backend/src/service/addons_service.go:277,337,393` — function parameters and return types using `map[string]interface{}`; change to `map[string]any`
- `backend/src/service/filesystem/ntfs_adapter.go:346,348,365,367` — local `make(map[string]interface{})` calls; change to `map[string]any`
- `backend/src/service/addon_config_watcher_service.go:29` — `type AddonConfigWatcherServiceInterface interface{}` → `type AddonConfigWatcherServiceInterface any` or, if it's a genuine empty interface marker, add a comment explaining why it exists
- `backend/src/service/problem_ha_bridge.go:16` — same as above

> Do NOT edit `*.gen.go` files or files under `homeassistant/` — those are vendored/generated.

### Task 2 — `errors.As` → `errors.AsType[T]`

File: `backend/src/unixsamba/unixsamba.go`, lines 363, 417, 460, 538, 604

Pattern to replace:
```go
var e *SomeErrorType
if errors.As(err, &e) {
    // use e
}
```

Go 1.26 replacement:
```go
if e, ok := errors.AsType[*SomeErrorType](err); ok {
    // use e
}
```

This is type-safe, avoids pre-declaring the target variable, and is the canonical Go 1.26+ pattern per project instructions.

### Task 3 — `wg.Add/Done` → `wg.Go`

Affected files (60 goroutine sites total): `ext4_adapter.go`, `xfs_adapter.go`, `f2fs_adapter.go`, `exfat_adapter.go`, `gfs2_adapter.go`, `hfsplus_adapter.go`, `ntfs_adapter.go`, `btrfs_adapter.go`, `vfat_adapter.go`, `reiserfs_adapter.go`, `zfs_adapter.go`, and `commandexec/runner.go`.

Pattern to replace:
```go
var wg sync.WaitGroup
wg.Add(2)
go func() {
    defer wg.Done()
    for line := range stdoutChan { ... }
}()
go func() {
    defer wg.Done()
    for line := range stderrChan { ... }
}()
wg.Wait()
```

Go 1.25+ replacement:
```go
var wg sync.WaitGroup
wg.Go(func() {
    for line := range stdoutChan { ... }
})
wg.Go(func() {
    for line := range stderrChan { ... }
})
wg.Wait()
```

### Task 4 — Extract duplicated channel-consumption helper

Every filesystem adapter that calls `executeCommandWithProgress` or `executeCommandWithProgressAndInput` contains an almost-identical block:

```go
var outputLines []string
var errorLines []string
var wg sync.WaitGroup
wg.Go(func() {
    for line := range stdoutChan {
        outputLines = append(outputLines, line)
        if progress != nil { notes = append(notes, line); progress("running", pct, notes) }
    }
})
wg.Go(func() {
    for line := range stderrChan {
        errorLines = append(errorLines, line)
        if progress != nil { notes = append(notes, "ERROR: "+line); progress("running", pct, notes) }
    }
})
wg.Wait()
```

Extract this into a helper on `baseAdapter` (or a package-level function):

```go
// drainCommandOutput reads stdout and stderr channels concurrently,
// appending to outputLines / errorLines and calling progress on each line.
func drainCommandOutput(
    stdoutChan, stderrChan <-chan string,
    progress dto.ProgressCallback,
    pct int,
    notes *[]string,
) (outputLines, errorLines []string) {
    var wg sync.WaitGroup
    wg.Go(func() {
        for line := range stdoutChan {
            outputLines = append(outputLines, line)
            if progress != nil { *notes = append(*notes, line); progress("running", pct, *notes) }
        }
    })
    wg.Go(func() {
        for line := range stderrChan {
            errorLines = append(errorLines, line)
            if progress != nil { *notes = append(*notes, "ERROR: "+line); progress("running", pct, *notes) }
        }
    })
    wg.Wait()
    return
}
```

Add tests for the new helper in `base_adapter_test.go`.

### Task 5 — Context-aware slog

File: `backend/src/service/filesystem_service.go`, lines 334, 377, 389, 395, 398, 400, 433, 474, 490, 494

Functions like `ResolveLinuxFsModule`, `MountFlagsToSyscallFlagAndData`, `FsTypeFromDevice` receive a `ctx context.Context` parameter. All `slog.Debug(...)`, `slog.Error(...)`, `slog.Warn(...)` calls in these functions must be changed to `slog.DebugContext(ctx, ...)`, `slog.ErrorContext(ctx, ...)`, `slog.WarnContext(ctx, ...)`.

File: `backend/src/service/server_process_service.go`, lines 67, 72, 100 — same pattern.

### Task 6 — FIXME in broadcaster_service.go

File: `backend/src/service/broadcaster_service.go:202`

```go
go broker.sendToHomeAssistant(msg) // FIXME: put as broadcast listener
```

This fires a bare goroutine without any lifecycle management or wait group tracking. Per the event architecture, HA-bound messages should go through the `events.EventBusInterface` subscription, not an ad-hoc goroutine. Investigate if `sendToHomeAssistant` should subscribe to the event bus at startup via a broadcast listener; replace the naked goroutine accordingly and track it in the service context.

### Task 7 — Empty marker interfaces

`AddonConfigWatcherServiceInterface interface{}` and `ProblemHABridgeInterface interface{}` are empty. In Go, an empty `interface{}` / `any` as a named type adds no type safety. Either:
- Add the methods that callers expect (if this is an unfinished stub), or
- Replace with `type AddonConfigWatcherServiceInterface = any` with a doc comment explaining the intent, or
- Remove and use concrete types directly

## 🔗 Code References & TODOs

- [ ] `backend/src/service/addons_service.go:277,337,393` — `interface{}` → `any`
- [ ] `backend/src/service/filesystem/ntfs_adapter.go:346,348,365,367` — `interface{}` → `any`
- [ ] `backend/src/service/addon_config_watcher_service.go:29` — empty interface
- [ ] `backend/src/service/problem_ha_bridge.go:16` — empty interface
- [ ] `backend/src/unixsamba/unixsamba.go:363,417,460,538,604` — `errors.As` → `errors.AsType[T]`
- [ ] `backend/src/service/filesystem/` (all adapters) — `wg.Add/Done` → `wg.Go` (60 sites)
- [ ] `backend/src/service/filesystem/` (all adapters) — extract `drainCommandOutput` helper
- [ ] `backend/src/service/filesystem_service.go:334,377,389,395,398,400,433,474,490,494` — context-less slog
- [ ] `backend/src/service/server_process_service.go:67,72,100` — context-less slog
- [ ] `backend/src/service/broadcaster_service.go:202` — FIXME naked goroutine
