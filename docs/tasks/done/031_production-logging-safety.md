<!-- DOCTOC SKIP -->

# [FIX]: Production Logging Safety — Body Logging and Secret Sanitization

**Target Repo:** `srat`
**Status:** ✅ Complete
**Issue Link:** _None — discovered in security review 2026-04-28_

## 🎯 Objective

Prevent Samba user passwords and the HA mount password from being written to structured logs. Currently `WithRequestBody: true` and `WithResponseBody: true` in `http_server.go` cause the full raw JSON body of every `POST /user`, `PUT /user/{username}`, `PUT /useradmin`, and `PUT /settings` request to be logged **before Huma deserializes it** — bypassing the `logfusc.Secret` wrapper protection on `dto.User.Password` and `dto.Settings.HASmbPassword`.

> *The `logfusc.Secret` type correctly masks values in Go struct printing. This task closes the gap at the HTTP logging layer.*

## 🛠️ Technical Specifications

- **Inputs:** `sloghttp.Config` in `NewHTTPServer`, optional `SRAT_LOG_BODIES` env var
- **Outputs:** No credential data in structured logs under normal operation; opt-in body logging for debugging
- **Dependencies:** `backend/src/server/http_server.go`, `backend/src/dto/user.go`, `backend/src/dto/settings.go`

## 📝 Task List

- [x] Task 1: Set `WithRequestBody: false` and `WithResponseBody: false` in `sloghttp.Config` in `NewHTTPServer`
  > ✅ **Done** — both flags now follow `logBodies` (default `false`).
- [x] Task 2: Add an opt-in flag: if `os.Getenv("SRAT_LOG_BODIES") == "true"`, re-enable body logging (useful for development debugging)
  > ✅ **Done** — `logBodies := os.Getenv("SRAT_LOG_BODIES") == "true"` in `NewHTTPServer` (same `== "true"` convention as `SRAT_MOCK`). `WithRequestHeader`/`WithResponseHeader` left untouched (out of scope; revisit if Task 3 audit finds header leaks).
- [x] Task 3: Audit all `slog.*` callsites in `api/` and `service/` for any remaining direct logging of `dto.User`, `dto.Settings`, or `dto.ContextState` fields that are not wrapped in `logfusc.Secret` — fix any found
  > ✅ **Done 2026-09-21** — audit result: `api/` logs only scalar identifiers (usernames, share/disk/partition/problem keys); `service/` logs only `user.Username`. `dto.Secret`/`logfusc.Secret` mask under `%v`/`%+v`/`%#v`/slog-Any via `String`/`GoString` (+ null JSON), and the broadcaster relays only path/hash for AppConfig. One genuine leak found and fixed: `DirtyDataService` logged the whole `AppConfigUpdateRequest` (arbitrary user-controlled options map, may contain the addon password) at Debug — now logs sorted option names only via `appConfigOptionKeys` (`dirty_data_service.go`). Covered by `TestSetDirtyAppConfigDoesNotLogOptionValues` + `TestAppConfigOptionKeys` (helper 100%).
- [x] Task 4: Add a test that POST /user with a password body does **not** produce a log entry containing the password string (use a `slog.Handler` interceptor in the test)
  > ✅ **Done** — `TestNewHTTPServerRequestBodyLogging` (`server/http_server_test.go`): boots the real `NewHTTPServer` + sloghttp chain with an echo stub on POST /user, captures `slog.Default` at TRACE into a mutex-guarded buffer, asserts the secret is absent by default and present with `SRAT_LOG_BODIES=true`. Failing-proof: with body flags hardcoded to `true` the default subtest fails. `NewHTTPServer` coverage 78.1% (≥70% gate).
- [x] Task 5: Document the `SRAT_LOG_BODIES` flag in `docs/SETTINGS_DOCUMENTATION.md`
  > ✅ **Done** — new `### Environment Variables` subsection under Implementation Details (`SRAT_LOG_BODIES` + related `SRAT_*` vars).
- [x] Task 6: Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-SEC-06 and B-SEC-03 (partial) resolved
  > ✅ **Done** — B-SEC-06 marked ✅ RESOLVED with guard-test references; B-SEC-03 annotated as partially mitigated (log side channel closed, spoofing vector still owned by Task 004).

## 🧠 Implementation Notes

```go
// In NewHTTPServer (server/http_server.go)
logBodies := os.Getenv("SRAT_LOG_BODIES") == "true"
handler := sloghttp.NewWithConfig(slog.Default(), sloghttp.Config{
    DefaultLevel:       tlog.LevelTrace,
    WithRequestBody:    logBodies,
    WithResponseBody:   logBodies,
    WithRequestHeader:  true,
    WithResponseHeader: false, // avoid leaking Set-Cookie or Authorization echoes
    WithUserAgent:      true,
    WithRequestID:      true,
    WithSpanID:         true,
    WithTraceID:        true,
})(sloghttp.Recovery(mux))
```

`WithRequestHeader: true` is acceptable because headers only contain the `X-Remote-User-Id` (not a credential). `WithResponseHeader: false` avoids echoing Set-Cookie headers.

### Phase 2026-09-21 (Task 4)

- Test is `TestNewHTTPServerRequestBodyLogging`, not a `slog.Handler` interceptor: it captures the real `slog.Default` output, which is what `sloghttp.NewWithConfig(slog.Default(), …)` reads at construction.
- sloghttp only captures bodies actually read through its wrapper (`dump.go` `bodyReader`): the stub must `io.ReadAll(r.Body)` like a real JSON handler, otherwise the test passes vacuously. Discovered via a false-green run; fixed with an echo stub (also covers the response-body path).
- Completion signal is `status=200` (response group, printed after body attrs) polled with `require.Eventually` — polling for the path alone raced partially flushed lines.
- `stubLifecycle` drives only the OnStart hook: invoking OnStop after `srv.Close()` would double-`Done` the WaitGroup (`wg.Go` auto-Done + explicit `Done` in OnStop) and panic. Suspected pre-existing imbalance, left untouched (out of scope; candidate follow-up for task 033).
- Full `mise run //backend:test` green (total 46.7%); server package `go test -race` clean; test rerun ×5 stable.

### Phase 2026-09-21 (Task 3)

- `go test -race ./service/` shows DATA RACE warnings in `TestConcurrentEventPropagation` and `TestVolumeServiceTestSuite/TestBootWarmupFailureDoesNotFailAppStart` (unlocked `dataDirtyTracker` vs `startTimer`). Reproduced on clean HEAD via a throwaway worktree — pre-existing, unrelated to this change; the sanctioned suite (no `-race`) is green. Candidate follow-up (mutex on tracker or task 033 scope).
- Full `mise run //backend:test` green (total 46.7%) with the fix.

### Phase 2026-09-21 (Task 1+2)

- `mise run //backend:test` green (total 46.6%), after generating `src/config/metadata_constants.go` via `mise run //backend:metadata --version 0.0.0` (gitignored prerequisite; the `undefined: MetadataJSON` build failure is pre-existing on fresh checkouts, unrelated to this change).
- Coverage note (superseded by Task 4): `NewHTTPServer` is now at 78.1% via `TestNewHTTPServerRequestBodyLogging`.

## 🔗 Code References & TODOs

- [x] `TODO: backend/src/server/http_server.go:35-40` — disable body logging by default (done, env-gated)
- [ ] Related: B-SEC-06 in `docs/SECURITY_OPTIMIZATION_REVIEW.md`
