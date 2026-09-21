# [FIX]: Security Hardening — CORS, IP Allowlist, Ingress Session Validation, WS Origin, pprof Isolation

**Target Repo:** `srat`  **Status:** ✅ Completed  **Issue Link:** [dianlight/srat#1213](https://github.com/dianlight/srat/issues/1213)

## 🎯 Objective

Address five related security findings identified in `docs/FUTURE_IMPROVEMENTS.md` and the 2026-04-28 security review: (1) CORS wildcard combined with `AllowCredentials: true` violates the CORS spec and may be exploitable; (2) the HA middleware IP allowlist is hardcoded to Supervisor defaults and breaks on non-standard Docker networks; (3) the ingress session validation against the Supervisor API is entirely commented out, leaving the service trusting any request from the allowed IP range; (4) the Gorilla WebSocket upgrader accepts connections from **any origin**, allowing cross-site WebSocket hijacking (merged from 029); (5) the `/debug/pprof/` route prefix is registered unconditionally in the production router (merged from 029).

> _Context for Copilot: All three findings are in `backend/src/server/`. The CORS issue is in `http_server.go:45-52`. The IP allowlist and ingress session stubs are in `ha_middleware.go`. The Supervisor token and network details are available via `ContextState` which is already injected into the server._

## 🛠️ Technical Specifications

- **Inputs:**
  - `ContextState.AddonMode` — whether the server is running as a HA addon
  - `ContextState` / addon config — ingress origin, allowed IP range from Supervisor
  - Supervisor API — `/ingress/session` endpoint for session cookie validation

- **Outputs:**
  - CORS: restrict `AllowedOrigins` to HA ingress origin in addon mode; dev mode remains permissive
  - IP allowlist: read allowed IPs from `ContextState` or env var; fall back to current defaults
  - Ingress session: re-enable cookie validation with caching (existing `gocache` implementation)

- **Dependencies:**
  - `backend/src/server/http_server.go` — CORS configuration (line 45–52)
  - `backend/src/server/ha_middleware.go` — IP allowlist (line 73), ingress session validation (lines 28–66)
  - `backend/src/config/` — `ContextState` struct for runtime config access
  - `backend/src/homeassistant/` — Supervisor API client (`ingressClient`)

## 📝 Task List

- [x] Task 0 (prerequisite): Extend `dto.ContextState` with `IngressOrigin string`, `AllowedOrigins []string`, `SupervisorAllowedIPs []string` (`dto.ParseCommaList` helper, 100% covered); flags `--ingress-origin/--allowed-origins/--supervisor-allowed-ips` with `SRAT_INGRESS_ORIGIN`/`SRAT_ALLOWED_ORIGINS`/`SUPERVISOR_NETWORK` env defaults in both `srat-server` and `srat-cli`. Note: no separate `AddonMode` field — existing `SecureMode` (`-addon` flag) already is the addon-mode signal.
- [x] Task 1: Fix CORS — when `AddonMode=true`, set `AllowedOrigins` to the HA ingress origin; keep wildcard only in dev mode
- [x] Task 2: Fix IP allowlist — read the allowed Supervisor network CIDR/IPs from `ContextState` or `SUPERVISOR_NETWORK` env var; fall back to `172.30.32.2`, `127.0.0.1`
- [x] Task 3 (documented wont-do): ~~Re-enable ingress session validation~~ — live HAOS spike 2026-09-21 proved `POST /ingress/validate_session` 401s with a valid addon token (`@require_home_assistant`); Supervisor proxy already 401s invalid cookies, so addon-side re-validation is not viable unless Supervisor grants the capability.
- [x] Task 4: Add unit tests for the updated CORS middleware (addon mode vs dev mode)
- [x] Task 5: Add unit tests for the HA middleware IP allowlist (standard network, custom network, localhost)
- [x] Task 6 (documented wont-do): ~~Add unit tests for the ingress session validation~~ — moot without Task 3; coverage comes from the IP-allowlist/middleware tests instead.
- [x] Task 7: Manual/integration smoke test — confirm the web UI still loads via HA ingress after changes (verified 2026-09-21: build 2026.9.10-dev.1 deployed via develop watcher; ingress UI renders Dashboard with live volumes/WS data, screenshot `.playwright-cli/page-2026-09-21T05-31-24-309Z.png`; /api/health 200; 0 ERROR/FATAL/panic in addon log. Note: lab runs leave_front_door_open=true so SecureMode=false by design; pprof 200 live is expected, dev builds ship --pprof)
- [x] Task 8: Documentation — add a note to `docs/HOME_ASSISTANT_INTEGRATION.md` about the ingress security model
- [x] Task 9 (merged from 029): Add `allowedOrigins []string` helper to `server/` package that reads from `ContextState` (same source as CORS fix above)
- [x] Task 10 (merged from 029): Replace `CheckOrigin: func(*http.Request) bool { return true }` in `NewWebSocketBroker` with a call to the helper; in non-`SecureMode` keep permissive
- [x] Task 11 (merged from 029): Add unit test: WebSocket upgrade must return 403 when `SecureMode=true` and `Origin` does not match the allowed list
- [x] Task 12 (merged from 029): Move `router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)` from `http_server.go:97` into `server/pprof.go` (behind `//go:build pprof`) so the route only exists in pprof builds
- [x] Task 13 (merged from 029): Add a build-tag-gated integration test confirming the pprof route returns 404 in production builds and 200 in pprof builds
- [x] Task 14 (merged from 029): Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-SEC-02 and B-SEC-05 resolved

## 🧠 Implementation Notes (Copilot Context)

### Progress 2026-09-20 (approved scope)

- Implemented Tasks 1, 2, 4, 5, 8–14; Tasks 3, 6, 7 left open.
- Task 3 deferred: `POST /ingress/validate_session` requires HA user auth the addon token lacks; Supervisor proxy already 401s invalid cookies. Spike-test on live HAOS before re-enabling; documented in `ha_middleware.go`.
- CORS/WS use exact-match allowlist (`IngressOrigin` + `AllowedOrigins`), fail-closed in `SecureMode`, permissive in dev; empty WS `Origin` allowed for non-browser HA component.
- IP allowlist merges `127.0.0.0/8` + `172.30.32.0/23` with `SupervisorAllowedIPs` (IP or CIDR); `net.SplitHostPort` parsing fixes IPv6.
- pprof route only in `//go:build pprof` builds via `RegisterPprof`; prod returns 404.
- Coverage: `server` 90.5%, `api` 78.1%; `NewHTTPServer` 72.4% via lifecycle test; full `mise run //backend:test` green; pprof-tagged test green.

### Spike result 2026-09-21 — Task 3 (validate_session from addon)

- Live HAOS (`local_sambanas2`, Supervisor 2026.09.3): `POST http://supervisor/ingress/validate_session` with the addon `SUPERVISOR_TOKEN` and a dummy session returns `401 Unauthorized`.
- Control probe with the same token: `GET /supervisor/info` returns `200` (token valid); `GET /ingress/info` returns `404`.
- Conclusion: the endpoint requires HA user auth (`@require_home_assistant`); the addon token lacks permission, as predicted. Addon-side re-validation is NOT viable — keep relying on the Supervisor proxy (it 401s invalid `ingress_session` cookies itself) plus IP allowlist and header stripping. Task 3 stays open as documented-wont-do unless Supervisor grants addons a session-validation capability.

### Completion 2026-09-21

- All implementable tasks done (1, 2, 4, 5, 7–14); Tasks 3 and 6 closed as documented wont-do (see spike result above).
- Verified live: build 2026.9.10-dev.1 via develop watcher, ingress UI smoke passed, full `mise run //backend:test` green, docs-validate clean.

### Official Supervisor behavior (verified 2026-09-20 against home-assistant/supervisor `api/ingress.py`)

- `POST /ingress/validate_session` exists and takes `{session}`, BUT it is decorated with `@require_home_assistant` — the caller must present Home Assistant auth, not an addon `SUPERVISOR_TOKEN`. Our own `ingress_ci_test.go:92` already skips with "Addons don't have the right permissions". So Task 3 (addon-side re-validation) may 401 on HAOS: spike-test it first on a live Supervisor before building on it. Fallback if blocked: rely on the Supervisor proxy (it 401s invalid `ingress_session` cookies itself in `handler` before forwarding) plus the `X-Remote-User-Id` header it sets.
- `_init_header` STRIPS client-supplied `X-Remote-User-Id`/`-Name` and re-sets them from session data — but only when the session carries user data. Header spoofing via ingress is impossible; direct-to-port calls can still spoof, hence IP allowlist + SecureMode remain necessary. The `homeassistant` fallback in the middleware covers headerless internal traffic.
- `validate_session` EXTENDS session validity on every call — the 30s cache in Task 3 is required, not optional, or every request keeps sessions alive forever.
- CORS/WS origin: no Supervisor endpoint returns the HA frontend origin directly; Tasks 1/9 need an explicit origin-discovery decision (addon config, `SUPERVISOR_*` env, or HA Core API) — do not hardcode.
- Supervisor proxies WebSockets too (`_handle_websocket`, session validated first), so the WS 403 check in Task 11 only fires for direct (non-ingress) connections — that is exactly the case it must catch.

### CORS fix

```go
// http_server.go
if state.AddonMode {
    corsConfig.AllowOrigins = []string{state.IngressOrigin}
} else {
    corsConfig.AllowOriginFunc = func(origin string) bool { return true }
}
// Never combine wildcard with AllowCredentials: true
```

### IP allowlist fix

```go
// ha_middleware.go
allowedIPs := state.SupervisorAllowedIPs
if len(allowedIPs) == 0 {
    allowedIPs = []string{"172.30.32.2", "127.0.0.1"}
}
// Also read from env: SRAT_ALLOWED_IPS (comma-separated) for non-addon deployments
```

### Ingress session validation

The existing commented-out block in `ha_middleware.go:28-66` uses `ingressClient` and `gocache`.
- Re-enable it, ensuring the cache TTL is short enough (e.g., 30 s) to detect session expiry but long enough to avoid per-request overhead.
- Endpoints that must bypass session validation: `/api/health`, `/metrics`, any liveness probe paths.

### Testing approach

- Use `humatest` for HTTP handler tests.
- Inject a mock `ingressClient` that returns 200/401 to cover valid/expired session paths.
- Use table-driven tests for IP allowlist to cover all address variants.

## 🔗 Code References & TODOs

- [x] `backend/src/server/http_server.go:45-52` — CORS `AllowOriginFunc` + `AllowCredentials`
- [x] `backend/src/server/ha_middleware.go:73` — hardcoded `172.30.32.2`, `127.0.0.1`
- [x] `backend/src/server/ha_middleware.go:28-66` — commented-out ingress session validation (kept out by decision; rationale in `ha_middleware.go` + spike note above)
- [x] `docs/FUTURE_IMPROVEMENTS.md` — "Security and Stability Findings" section (remove once fixed)
- [x] `backend/src/api/ws.go:52-57` — replace permissive CheckOrigin (merged from 029)
- [x] `backend/src/server/http_server.go:97` — move pprof route behind build tag (merged from 029)
