<!-- DOCTOC SKIP -->

# [FIX]: Production Logging Safety — Body Logging and Secret Sanitization

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** _None — discovered in security review 2026-04-28_

## 🎯 Objective

Prevent Samba user passwords and the HA mount password from being written to structured logs. Currently `WithRequestBody: true` and `WithResponseBody: true` in `http_server.go` cause the full raw JSON body of every `POST /user`, `PUT /user/{username}`, `PUT /useradmin`, and `PUT /settings` request to be logged **before Huma deserializes it** — bypassing the `logfusc.Secret` wrapper protection on `dto.User.Password` and `dto.Settings.HASmbPassword`.

> *The `logfusc.Secret` type correctly masks values in Go struct printing. This task closes the gap at the HTTP logging layer.*

## 🛠️ Technical Specifications

- **Inputs:** `sloghttp.Config` in `NewHTTPServer`, optional `SRAT_LOG_BODIES` env var
- **Outputs:** No credential data in structured logs under normal operation; opt-in body logging for debugging
- **Dependencies:** `backend/src/server/http_server.go`, `backend/src/dto/user.go`, `backend/src/dto/settings.go`

## 📝 Task List

- [ ] Task 1: Set `WithRequestBody: false` and `WithResponseBody: false` in `sloghttp.Config` in `NewHTTPServer`
- [ ] Task 2: Add an opt-in flag: if `os.Getenv("SRAT_LOG_BODIES") == "true"`, re-enable body logging (useful for development debugging)
- [ ] Task 3: Audit all `slog.*` callsites in `api/` and `service/` for any remaining direct logging of `dto.User`, `dto.Settings`, or `dto.ContextState` fields that are not wrapped in `logfusc.Secret` — fix any found
- [ ] Task 4: Add a test that POST /user with a password body does **not** produce a log entry containing the password string (use a `slog.Handler` interceptor in the test)
- [ ] Task 5: Document the `SRAT_LOG_BODIES` flag in `docs/SETTINGS_DOCUMENTATION.md`
- [ ] Task 6: Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-SEC-06 and B-SEC-03 (partial) resolved

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

## 🔗 Code References & TODOs

- [ ] `TODO: backend/src/server/http_server.go:35-40` — disable body logging by default
- [ ] Related: B-SEC-06 in `docs/SECURITY_OPTIMIZATION_REVIEW.md`
