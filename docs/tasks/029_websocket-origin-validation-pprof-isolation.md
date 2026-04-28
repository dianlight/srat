<!-- DOCTOC SKIP -->

# [FIX]: WebSocket Origin Validation and pprof Route Isolation

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** _None — discovered in security review 2026-04-28_

## 🎯 Objective

Close two related security gaps in the HTTP/WebSocket boundary layer:

1. The Gorilla WebSocket upgrader accepts connections from **any origin**, allowing cross-site WebSocket hijacking.
2. The `/debug/pprof/` route prefix is registered unconditionally in the production router; if `net/http/pprof` were ever imported outside the build-tag file, heap dumps and goroutine stacks would be publicly accessible.

> *Context: The CORS layer already has the same wildcard-origin issue (tracked in [004]); this task handles the WebSocket-specific gap and the pprof route placement. It should be implemented alongside or immediately after [004].*

## 🛠️ Technical Specifications

- **Inputs:** `apiCtx.SecureMode`, `r.Header.Get("Origin")`, the HA Supervisor ingress origin (derivable from `ContextState` or environment)
- **Outputs:** WebSocket upgrade rejected with HTTP 403 for untrusted origins in production; pprof route only compiled in when `//go:build pprof` is active
- **Dependencies:** `backend/src/api/ws.go`, `backend/src/server/http_server.go`, `backend/src/server/pprof.go`

## 📝 Task List

- [ ] Task 1: Add `allowedOrigins []string` helper to `server/` package that reads from `ContextState` (same source as CORS fix in [004])
- [ ] Task 2: Replace `CheckOrigin: func(*http.Request) bool { return true }` in `NewWebSocketBroker` with a call to the helper; in non-`SecureMode` keep permissive
- [ ] Task 3: Add unit test: WebSocket upgrade must return 403 when `SecureMode=true` and `Origin` does not match the allowed list
- [ ] Task 4: Move `router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)` from `http_server.go:97` into `server/pprof.go` (behind `//go:build pprof`) so the route only exists in pprof builds
- [ ] Task 5: Add a build-tag-gated integration test confirming the pprof route returns 404 in production builds and 200 in pprof builds
- [ ] Task 6: Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-SEC-02 and B-SEC-05 resolved

## 🧠 Implementation Notes

```go
// In NewWebSocketBroker (api/ws.go)
upgrader := websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        if !p.State.SecureMode {
            return true // dev / non-addon mode
        }
        origin := r.Header.Get("Origin")
        return isAllowedOrigin(origin, p.State.AllowedOrigins)
    },
}
```

`isAllowedOrigin` should compare scheme+host only (strip path), and treat an empty `Origin` header as allowed (same-origin WebSocket connections from the HA ingress panel).

For the pprof fix:
- Keep `server/pprof.go` exactly as-is (the `//go:build pprof` file that imports `net/http/pprof`)
- Move the router line into a new function `RegisterPprofHandler(r *mux.Router)` in that same file
- Call `RegisterPprofHandler(router)` from `NewMuxRouter` only when the function is available (build-tag gating)

## 🔗 Code References & TODOs

- [ ] `TODO: backend/src/api/ws.go:52-57` — replace permissive CheckOrigin
- [ ] `TODO: backend/src/server/http_server.go:97` — move pprof route behind build tag
- [ ] Related: [004] Security Hardening for CORS counterpart
