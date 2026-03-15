# [FIX]: Security Hardening — CORS, IP Allowlist, Ingress Session Validation

**Target Repo:** `srat`  **Status:** 📅 Planned  **Issue Link:** _TBD_

## 🎯 Objective

Address three related security findings identified in `docs/FUTURE_IMPROVEMENTS.md`: (1) CORS wildcard combined with `AllowCredentials: true` violates the CORS spec and may be exploitable; (2) the HA middleware IP allowlist is hardcoded to Supervisor defaults and breaks on non-standard Docker networks; (3) the ingress session validation against the Supervisor API is entirely commented out, leaving the service trusting any request from the allowed IP range.

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

- [ ] Task 1: Fix CORS — when `AddonMode=true`, set `AllowedOrigins` to the HA ingress origin; keep wildcard only in dev mode
- [ ] Task 2: Fix IP allowlist — read the allowed Supervisor network CIDR/IPs from `ContextState` or `SUPERVISOR_NETWORK` env var; fall back to `172.30.32.2`, `127.0.0.1`
- [ ] Task 3: Re-enable ingress session validation — uncomment and validate the `gocache`-backed Supervisor API call; ensure it does not block health-check or non-ingress endpoints
- [ ] Task 4: Add unit tests for the updated CORS middleware (addon mode vs dev mode)
- [ ] Task 5: Add unit tests for the HA middleware IP allowlist (standard network, custom network, localhost)
- [ ] Task 6: Add unit tests for the ingress session validation (valid session, expired session, Supervisor unreachable fallback)
- [ ] Task 7: Manual/integration smoke test — confirm the web UI still loads via HA ingress after changes
- [ ] Task 8: Documentation — add a note to `docs/HOME_ASSISTANT_INTEGRATION.md` about the ingress security model

## 🧠 Implementation Notes (Copilot Context)

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

- [ ] `backend/src/server/http_server.go:45-52` — CORS `AllowOriginFunc` + `AllowCredentials`
- [ ] `backend/src/server/ha_middleware.go:73` — hardcoded `172.30.32.2`, `127.0.0.1`
- [ ] `backend/src/server/ha_middleware.go:28-66` — commented-out ingress session validation
- [ ] `docs/FUTURE_IMPROVEMENTS.md` — "Security and Stability Findings" section (remove once fixed)
