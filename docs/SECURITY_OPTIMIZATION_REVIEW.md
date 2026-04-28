<!-- DOCTOC SKIP -->

# SRAT Security & Optimization Review

**Date:** 2026-04-28  
**Scope:** `backend/src/` (Go 1.26) and `frontend/src/` (TypeScript / React 19)  
**Reviewer:** Architecture & Security Analysis

---

## Executive Summary

SRAT is a well-structured monorepo with solid architectural choices (Huma v2, FX DI, RTK Query, event-driven WebSocket design). The codebase follows documented patterns consistently and has reasonable test coverage. The primary risks are concentrated in the HTTP/WebSocket boundary layer, two data-correctness bugs (a mis-registered migration and an inverted loading guard), memory-growth and goroutine-leak issues in long-running subsystems, and a path-traversal surface in the upgrade and HDIdle handlers.

**Severity distribution:**

| Severity | Backend | Frontend | Total |
| -------- | ------- | -------- | ----- |
| Critical | 1       | 0        | 1     |
| High     | 9       | 5        | 14    |
| Medium   | 8       | 6        | 14    |
| Low      | 6       | 5        | 11    |

---

## Backend Findings

### Security

---

#### [B-SEC-01] CORS wildcard origin with `AllowCredentials: true`

**Severity:** High  
**File:** `backend/src/server/http_server.go:44-54`

```go
AllowOriginFunc:     func(origin string) bool { return true },
AllowCredentials:    true,
AllowPrivateNetwork: true,
```

The CORS spec prohibits wildcard origins when `Access-Control-Allow-Credentials: true` is set. Any page on the internet can make credentialed cross-origin requests to the backend and receive responses with full cookie/auth context.

**Fix:** In `SecureMode`, restrict `AllowOriginFunc` to the HA Supervisor ingress hostname or an explicit allowlist from `ContextState`. Keep the permissive policy only in non-`SecureMode` (dev).  
**Task:** [004] Security Hardening (planned).

---

#### [B-SEC-02] WebSocket upgrader accepts any origin

**Severity:** High  
**File:** `backend/src/api/ws.go:52-57`

```go
CheckOrigin: func(r *http.Request) bool {
    // Allow connections from any origin in development
    return true
},
```

Any page can open a WebSocket and receive all broadcast events (disk events, share events, settings, upgrade notifications). In combination with the permissive CORS policy, a cross-site attack can establish a persistent event stream from a victim's browser.

**Fix:** In `SecureMode`, validate the `Origin` header against the HA ingress host before upgrading.  
**Task:** [029] WebSocket Origin Validation (new).

---

#### [B-SEC-03] Ingress session validation commented out

**Severity:** High  
**File:** `backend/src/server/ha_middleware.go:43-73`

The entire `ingress_session` cookie validation block is disabled. The middleware trusts any request from `172.30.32.0/23` or `127.0.0.0/8`. A container on the same Docker network can fabricate `X-Remote-User-Id` and gain full API access.

**Fix:** Re-enable the commented-out validation using `ingressClient` with the 30-second `gocache` TTL that is already coded.  
**Task:** [004] Security Hardening (planned).

---

#### [B-SEC-04] Hardcoded trusted IP prefixes, naive IPv6 splitting

**Severity:** High  
**File:** `backend/src/server/ha_middleware.go:31-34, 75-78`

IP prefixes are hardcoded to a single Docker network. Additionally, `strings.Split(remoteAddr, ":")[0]` is wrong for IPv6 (`[::1]:1234` → `[::1]`, which `netip.ParseAddr` rejects).

**Fix:** Read allowed prefixes from `ContextState`; use `net.SplitHostPort` for address parsing.  
**Task:** [004] Security Hardening (planned).

---

#### [B-SEC-05] pprof route registered unconditionally in production router

**Severity:** Medium  
**File:** `backend/src/server/http_server.go:97`

```go
router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
```

The route exists regardless of the `pprof` build tag. Any accidental import of `net/http/pprof` would silently expose heap dumps, goroutine stacks, and CPU profiles to trusted-network callers.

**Fix:** Move this line into `server/pprof.go` (behind `//go:build pprof`).  
**Task:** [029] WebSocket Origin Validation (new).

---

#### [B-SEC-06] Request and response body logging in all modes — password leakage

**Severity:** High  
**File:** `backend/src/server/http_server.go:35-40`

```go
WithRequestBody:  true,
WithResponseBody: true,
```

Raw JSON request bodies are logged before Huma deserializes them. `dto.User.Password` (Samba passwords) and `dto.Settings.HASmbPassword` appear in plaintext in every log sink because the `logfusc.Secret` wrapper only masks Go struct printing, not raw JSON.

**Fix:** Disable body logging by default; enable only under `SRAT_LOG_BODIES=true`.  
**Task:** [031] Production Logging Safety (new).

---

#### [B-SEC-07] Hardcoded fallback credential used when password generation fails

**Severity:** High  
**File:** `backend/src/service/user_service.go:24`, `backend/src/dbom/migrations/00008*.go:23`, `backend/src/dbom/migrations/00014*.go:23`

```go
const defaultAdminPassword = "changeme!"
```

If `GenerateSecurePassword` fails (silently), the well-known string `"changeme!"` (or `"changeme"`) is committed to the Samba database. The fallback should be fatal rather than silently setting a predictable credential.

**Fix:** Make password-generation failure fatal in migrations and in the service bootstrap.  
**Task:** [004] Security Hardening (planned) — add to scope.

---

#### [B-SEC-08] ZipSlip bypass via `dest != "."` escape in upgrade handler

**Severity:** High  
**File:** `backend/src/service/upgrade_service.go:467-470`

```go
if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) && dest != "." {
    return nil, errors.Errorf("illegal file path: %s", path)
}
```

When `dest` is `"."`, the guard is skipped entirely, allowing zip entries to traverse outside the destination directory.

**Fix:** Remove the `dest != "."` special case; always verify the cleaned path prefix.  
**Task:** [035] Upgrade & HDIdle Path Traversal Fixes (new).

---

#### [B-SEC-09] Unsanitized `disk_id` path component in HDIdle handler

**Severity:** High  
**File:** `backend/src/api/hdidle_handler.go:60,80,116,220`

```go
devicePath := "/dev/disk/by-id/" + input.DiskID
```

A caller can supply `DiskID = "../../sda"` to reach `/dev/sda` or any arbitrary path. There is no `pattern:` constraint on the path parameter.

**Fix:** Add `pattern:"[a-zA-Z0-9_-]+"` to `DiskID` path parameter; validate that the constructed path starts with `/dev/disk/by-id/` after `filepath.Clean`.  
**Task:** [035] Upgrade & HDIdle Path Traversal Fixes (new).

---

#### [B-SEC-10] `http.DefaultClient` used for update downloads — no timeout

**Severity:** Medium  
**File:** `backend/src/service/upgrade_service.go:553`

```go
resp, err := http.DefaultClient.Do(req) // #nosec G704
```

`http.DefaultClient` has no timeout. A slow GitHub asset server blocks the upgrade goroutine indefinitely, holding the `sync.WaitGroup` and preventing clean shutdown.

**Fix:** Create a dedicated `http.Client{Timeout: 30*time.Minute}`.  
**Task:** [035] Upgrade & HDIdle Path Traversal Fixes (new).

---

#### [B-SEC-11] Samba passwords stored in cleartext in SQLite

**Severity:** Low  
**File:** `backend/src/dbom/samba_user.go:16`

```go
Password string
```

Any process with access to the SQLite file can extract all user passwords.

**Fix:** Store the NT hash instead of plaintext (Samba's `pdbedit` requires the NT hash anyway). Remove the plaintext field from the DBOM.

---

### Performance

---

#### [B-PERF-01] Package-level `commandResultCache` is shared across all adapter instances

**Severity:** High  
**File:** `backend/src/service/filesystem/base_adapter.go:27-29`

The global `var commandResultCache` is shared by all filesystem adapters. A `invalidateCommandResultCache()` call (e.g., after a format operation) flushes valid entries from all other adapters. Error results (e.g., `mkfs.ext4` not installed) are also cached for 30 minutes, surfacing stale errors after package installation.

**Fix:** Move the cache into `FilesystemService`, keyed per adapter type. Cache only successful results, or use a short TTL for errors (30 seconds).  
**Task:** [036] Frontend Performance — NavBar Lazy Loading and Metrics Render (see also frontend).

---

#### [B-PERF-02] Busy-wait poll in `executeWithInput` (10ms ticker)

**Severity:** Medium  
**File:** `backend/src/internal/commandexec/runner.go:232-254`

Polls the snapshots map every 10ms under lock. For 60-second operations this is ~6,000 lock acquisitions.

**Fix:** Replace with a per-execution `chan struct{}` closed when the command finishes.  
**Task:** [030] commandexec Memory Leak (new).

---

#### [B-PERF-03] `commandexec.Service.snapshots` map grows unbounded

**Severity:** High  
**File:** `backend/src/internal/commandexec/runner.go:41-97`

Completed snapshots (up to 500 lines each) are never evicted. A long-running server accumulates entries from every filesystem check, SMART test, and upgrade.

**Fix:** Add TTL-based eviction with a capped LRU fallback.  
**Task:** [030] commandexec Memory Leak (new).

---

#### [B-PERF-04] No HTTP request body size limit

**Severity:** Medium  
**File:** `backend/src/server/http_server.go`

No `http.MaxBytesReader` wrapper. A caller can submit an arbitrarily large JSON body, buffering it entirely in memory.

**Fix:** Add middleware wrapping `r.Body` with `http.MaxBytesReader(w, r.Body, 1<<20)` (1 MB default).  
**Task:** [033] Database & HTTP Safety Guards (new).

---

### Reliability

---

#### [B-REL-01] **CRITICAL** — Migration 14 registers wrong function names

**Severity:** Critical  
**File:** `backend/src/dbom/migrations/00014_sanitize_empty_HASmbPassword.go:15`

```go
func init() {
    goose.AddMigrationNoTxContext(Up00008, Down00008) // BUG: should be Up00014, Down00014
}
```

Migration 14's `init()` registers `Up00008` and `Down00008` instead of `Up00014` and `Down00014`. When goose runs migration 14, it executes migration 8's password-seeding `INSERT OR IGNORE` logic again, and migration 14's `UPDATE` logic (sanitising empty passwords) is **never run**. Any database that has empty `HASmbPassword` values will not be repaired by this migration.

**Fix:** Change the `init()` call to `goose.AddMigrationNoTxContext(Up00014, Down00014)`.  
**Task:** [034] Critical Migration Fix (new — immediate).

---

#### [B-REL-02] Goroutine leak in `ProcessWebSocketChannel`

**Severity:** High  
**File:** `backend/src/api/ws.go:318`

```go
go self.broadcaster.ProcessWebSocketChannel(wsMessageSender.SendFunc)
```

A new goroutine is spawned per WebSocket connection with no mechanism to stop it when the connection closes. Persistent HA panel sessions eventually accumulate leaked goroutines.

**Fix:** Pass a connection-scoped `context.Context` or `done chan struct{}` into `ProcessWebSocketChannel`; return from the goroutine when it is signalled.  
**Task:** [033] Database & HTTP Safety Guards (new).

---

#### [B-REL-03] Unchecked nil-pointer assertion in `main-server.go`

**Severity:** High  
**File:** `backend/src/cmd/srat-server/main-server.go:288`

```go
apiCtx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup).Wait()
```

Every other call site uses the safe `if wg, ok := ...; ok && wg != nil` form. This one is unguarded and panics if the value is absent.

**Fix:** Apply the same guard pattern.  
**Task:** [033] Database & HTTP Safety Guards (new).

---

#### [B-REL-04] `replaceDatabase` infinite recursion

**Severity:** Medium  
**File:** `backend/src/dbom/db_config.go:220-232`

`NewDB` → `replaceDatabase` → `NewDB` recurse on persistent failure. Stack overflows on a consistently corrupt/unwritable filesystem.

**Fix:** Add a depth counter; on the second attempt, `tlog.Fatal`.  
**Task:** [033] Database & HTTP Safety Guards (new).

---

#### [B-REL-05] `NewDB` returns `nil` silently when `os.Remove` fails

**Severity:** Medium  
**File:** `backend/src/dbom/db_config.go:225-228`

`return nil` from `replaceDatabase` propagates a nil `*gorm.DB` into every FX-injected service, causing opaque nil-pointer panics later.

**Fix:** Replace `return nil` with `tlog.Fatal(...)`.  
**Task:** [033] Database & HTTP Safety Guards (new).

---

#### [B-REL-06] Debounce timer race in `watchForDevelopUpdates`

**Severity:** Medium  
**File:** `backend/src/service/upgrade_service.go:251-256`

`time.Timer.Stop()` returns `false` when the callback is already running. A new timer is created concurrently, resulting in two `selfupdate.Apply` calls racing on the same binary file.

**Fix:** After `Stop()` returns false, drain the channel and use a mutex around the install call.  
**Task:** [035] Upgrade & HDIdle Path Traversal Fixes (new).

---

#### [B-REL-07] Data race in `HDIdleService.IsRunning()`

**Severity:** Medium  
**File:** `backend/src/service/hdidle_service.go`

`IsRunning()` reads `s.stopChan` without holding the mutex, while `Start` and `Stop` write it under `s.mu.Lock()`.

**Fix:** Use `s.mu.RLock()` in `IsRunning()`, or an `atomic.Bool`.  
**Task:** [035] Upgrade & HDIdle Path Traversal Fixes (new).

---

#### [B-REL-08] `context.Background()` in GitHub release API call

**Severity:** Low  
**File:** `backend/src/service/upgrade_service.go:361`

```go
releases, _, err := self.gh.Repositories.ListReleases(context.Background(), ...)
```

Ignores the application shutdown context. The request blocks indefinitely on slow GitHub responses during shutdown.

**Fix:** Use `self.ctx` (with a timeout-based derivative).

---

---

## Frontend Findings

### Security

---

#### [F-SEC-01] Hooks called inside `try/catch` in `useRollbarTelemetry` — Rules of Hooks violation

**Severity:** High  
**File:** `frontend/src/hooks/useRollbarTelemetry.ts:24-32, 78-83`

`useRollbar()` and `useRollbarConfiguration()` are called inside `try {}` blocks (acknowledged with a `biome-ignore` suppression). Conditional hook calls corrupt React's hook order on any render where the catch path is taken, causing wrong state for all subsequent hooks in the same component and potentially across re-mounts after error boundary resets.

**Fix:** Move both hook calls unconditionally to the top level. Use null-check on the returned Rollbar instance. Provide a no-op stub context when Rollbar is unavailable.  
**Task:** [037] Frontend Data Correctness Fixes (new).

---

#### [F-SEC-02] Plaintext password comparison in frontend

**Severity:** Medium  
**File:** `frontend/src/hooks/useBaseConfigModal.ts:50-52`

```typescript
adminUser.password === "changeme!";
```

The API returns `password` in user objects; the frontend receives and compares credential material. Any XSS, Redux devtools extension, or network log captures the password value.

**Fix:** Add a `has_default_password: boolean` field to the backend User DTO response; remove the `password` field from GET responses. Never expose credentials in API reads.  
**Task:** [037] Frontend Data Correctness Fixes (new).

---

#### [F-SEC-03] `localStorage.getItem` deserialized without `try/catch`

**Severity:** Medium  
**Files:** `frontend/src/hooks/issueHooks.ts:8-10`, `frontend/src/pages/dashboard/metrics/SystemMetricsAccordion.tsx:83`

`JSON.parse(saved)` in a `useState` initializer throws synchronously if the stored value is corrupted, crashing the hook and propagating to the nearest error boundary. `issueHooks.ts` is particularly critical as it affects the Dashboard tab.

**Fix:** Wrap all `JSON.parse(localStorage.getItem(...))` calls in `try/catch` returning the default value.  
**Task:** [037] Frontend Data Correctness Fixes (new).

---

#### [F-SEC-04] `globalThis` used as a runtime configuration channel

**Severity:** Medium  
**File:** `frontend/src/store/wsApi.ts:44-46, 77-83`

`__SRAT_WS_INACTIVITY_MS` and `__SRAT_WS_RECONNECT_MS` are read from `globalThis` at cache-entry time. Any script on the same origin can set these to 0, triggering a reconnect storm.

**Fix:** Remove `globalThis`-based configuration; pass timing parameters through RTK Query endpoint configuration or React context.  
**Task:** [032] WebSocket Reconnect Resilience (new).

---

#### [F-SEC-05] WebSocket message parsing without error handling

**Severity:** Medium  
**File:** `frontend/src/store/wsApi.ts:181-200`

`JSON.parse(data)` is called without `try/catch`. A malformed or truncated frame silently corrupts Redux state.

**Fix:** Wrap `JSON.parse` in `try/catch`; discard malformed frames with a warning log.  
**Task:** [032] WebSocket Reconnect Resilience (new).

---

### Performance

---

#### [F-PERF-01] All page components instantiated eagerly in `NavBar.tsx`

**Severity:** High  
**File:** `frontend/src/components/NavBar.tsx:100-145`

`ALL_TAB_CONFIGS` is a module-level constant holding constructed JSX elements for every page. All hooks, RTK Query subscriptions, and `useEffect` calls fire at module load regardless of which tab is active. The Swagger page eagerly imports `openapi-explorer` (a large Web Component).

**Fix:** Use `React.lazy()` for each tab component; wrap `TabPanel` in `<Suspense>`.  
**Task:** [036] Frontend Performance — NavBar and Metrics Rendering (new).

---

#### [F-PERF-02] `NavBar` event handlers re-created on every heartbeat

**Severity:** High  
**File:** `frontend/src/components/NavBar.tsx:400-435`

15+ state variables, 6 `useEffect` blocks, and 8+ plain function expressions in a 848-line component. `evdata` updates every few seconds on heartbeat, causing the entire navigation chrome to re-render at that frequency. All child components receive new function references.

**Fix:** Wrap all handlers in `useCallback`; extract SSE debug overlay, UpdateButton, and ThemeToggle as separate memoized components.  
**Task:** [036] Frontend Performance — NavBar and Metrics Rendering (new).

---

#### [F-PERF-03] Metrics history uses 3 separate `useEffect`/`useState` pairs per component

**Severity:** High  
**Files:** `frontend/src/pages/dashboard/DashboardMetrics.tsx:47-101`, `NetworkHealthMetrics.tsx:31-63`, `DiskHealthMetrics.tsx:63-113`, `SystemMetricsAccordion.tsx`

Each history array is updated in its own `useEffect`, causing multiple sequential state updates and re-renders per heartbeat. `DashboardMetrics.tsx` triggers 3 re-renders per heartbeat.

**Fix:** Consolidate all history state into a single `useReducer` or one `setState` call with an object per component.  
**Task:** [036] Frontend Performance — NavBar and Metrics Rendering (new).

---

#### [F-PERF-04] `mergeCommandLines` allocates new `Set` and arrays on every WebSocket event

**Severity:** Medium  
**File:** `frontend/src/App.tsx:34-57`

Called inside `setCommandSessions` on every `command_output` event. For commands with hundreds of output lines, this creates significant GC pressure.

**Fix:** Use a `Map` keyed by `timestamp:channel:line` as the canonical data structure; only trigger re-renders when the dialog is open.  
**Task:** [032] WebSocket Reconnect Resilience (new).

---

#### [F-PERF-05] WebSocket reconnect has no exponential backoff

**Severity:** Medium  
**File:** `frontend/src/store/wsApi.ts:142-150`

Fixed 1-second reconnect delay. All connected browser tabs synchronize reconnect attempts after a HA supervisor restart.

**Fix:** Implement `delay = min(base * 2^attempt, maxDelay) + jitter(0..200ms)`; reset counter on successful open.  
**Task:** [032] WebSocket Reconnect Resilience (new).

---

### Reliability

---

#### [F-REL-01] `isLoading` uses `&&` (AND) instead of `||` (OR) in hook combiners

**Severity:** High  
**Files:** `frontend/src/hooks/healthHook.ts:48`, `frontend/src/hooks/volumeHook.ts:29`, `frontend/src/hooks/shareHook.ts:30`

```typescript
isLoading: isLoading && evloading,
```

This returns `false` (loaded) when either source finishes, even if the other is still in-flight or has errored. A REST API error makes `isLoading: false && true = false`, so consumers see "loaded" while `health` is an empty object and `error` is set — causing charts to render silently empty rather than showing an error state.

**Fix:** Change `&&` to `||`: `isLoading: isLoading || evloading`.  
**Task:** [037] Frontend Data Correctness Fixes (new).

---

#### [F-REL-02] WebSocket reconnect race: `setWsConnected(true)` after cache removal

**Severity:** Medium  
**File:** `frontend/src/store/wsApi.ts:140-156`

If `cacheEntryRemoved` resolves between timer firing and `connect()` returning, `setWsConnected(true)` updates an already-removed cache entry, triggering Immer warnings. `isStopped` is not set before `ws?.close()` in the `finally` block, so the new WebSocket from `connect()` may still open.

**Fix:** Set `isStopped = true` before `ws?.close()` in the `finally` block; guard `setWsConnected` with `if (!isStopped)`.  
**Task:** [032] WebSocket Reconnect Resilience (new).

---

#### [F-REL-03] `useIgnoredIssues` crashes tab on corrupted localStorage

**Severity:** Medium  
**File:** `frontend/src/hooks/issueHooks.ts:8-10`

`JSON.parse` without try-catch in the `useState` initializer. Corrupted storage crashes the Dashboard tab.

**Fix:** Wrap in `try/catch` returning `[]`.  
**Task:** [037] Frontend Data Correctness Fixes (new).

---

### Code Quality

---

#### [F-QUAL-01] `noImplicitAny: false` overrides `strict: true`

**Severity:** High  
**File:** `frontend/tsconfig.json:33`

The explicit `"noImplicitAny": false` override defeats `"strict": true`, silently allowing implicit `any` everywhere. Widespread `as unknown as X` casts and `[key: string]: unknown` index signatures indicate the codebase has been written to tolerate this.

**Fix:** Remove the override and enable `noImplicitAny`; fix resulting errors incrementally.  
**Task:** Low-priority; bundle with [026] MUI v9 upgrade or create a dedicated TypeScript strictness refactor task.

---

#### [F-QUAL-02] `NavBar.tsx` is an 848-line monolith mixing 8+ concerns

**Severity:** High  
**File:** `frontend/src/components/NavBar.tsx`

Tab routing, SSE debug display, update flow, tour control, report-issue dialog, dark-mode toggle, notifications, and the entire page layout are in one component. It re-renders at WebSocket heartbeat frequency.

**Fix:** Extract `SSEDebugOverlay`, `UpdateButton`, `TabPanelRenderer`, and `ThemeToggle` as separate components. Move tab-routing state to a `useTabNavigation` hook.  
**Task:** [036] Frontend Performance — NavBar and Metrics Rendering (new).

---

#### [F-QUAL-03] `SmbConfPage` refetches on every IntersectionObserver event including exit

**Severity:** Low  
**File:** `frontend/src/pages/SmbConf.tsx:23-28`

`onChange={(_inView, _entry) => smbconfig.refetch()}` fires on both enter and exit, issuing a redundant network request every time the user scrolls away from the tab.

**Fix:** Add `if (_inView) smbconfig.refetch();`.

---

---

## Cross-Cutting Concerns

### No API-level rate limiting on user/settings mutations

**Severity:** Medium  
No per-client rate limiting on `POST /user`, `PUT /user`, `PUT /settings`. Low-risk in the HA addon context (trusted IPs only), but defense-in-depth suggests a token-bucket limiter on mutation endpoints.  
**Task:** [004] Security Hardening (planned) — add to scope.

---

## Already-Planned Work (Do Not Duplicate)

| Finding                                 | Planned Task                    |
| --------------------------------------- | ------------------------------- |
| CORS wildcard + credentials             | [004] Security Hardening        |
| Ingress session validation disabled     | [004] Security Hardening        |
| Hardcoded IP allowlist                  | [004] Security Hardening        |
| `errors.As` → `errors.AsType` migration | [007] Backend Code Quality      |
| Large service file splits               | [007] Backend Code Quality      |
| HDIdle missing endpoints                | [003] HDIdle Service Completion |
| Converter interface gap                 | [006] DB and ORM Stubs          |
| Migration 00009 no-op                   | [006] DB and ORM Stubs          |

---

## New Tasks Generated

| #     | Task                                                                | Type     | Severity |
| ----- | ------------------------------------------------------------------- | -------- | -------- |
| [029] | WebSocket Origin Validation and pprof Route Isolation               | FIX      | High     |
| [030] | commandexec Snapshot Memory Leak and Busy-Wait Elimination          | FIX      | High     |
| [031] | Production Logging Safety — Body Logging and Secret Sanitization    | FIX      | High     |
| [032] | WebSocket Reconnect Resilience and Frontend Safety Guards           | FIX      | Medium   |
| [033] | Database Recovery Safety, HTTP Size Limits, and Goroutine Leak      | FIX      | Medium   |
| [034] | CRITICAL: Migration 14 Wrong Function Registration                  | FIX      | Critical |
| [035] | Upgrade & HDIdle Path Traversal, Timer Race, and Data Race          | FIX      | High     |
| [036] | Frontend Performance — NavBar Lazy Loading and Metrics Render       | REFACTOR | High     |
| [037] | Frontend Data Correctness — isLoading Bug, Hook Rules, localStorage | FIX      | High     |
