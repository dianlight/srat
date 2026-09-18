<!-- DOCTOC SKIP -->

# [FEATURE]: OAuth Instance Registration Hardening

**Target Repo:** `srat` (oauth_broker)
**Status:** ✅ Completed
**Type:** FEATURE
**Issue Link:** TBD

## 🎯 Objective

Harden OAuth broker callback security via HA instance registration with TTL. The broker exposes a new registration endpoint that binds `instance_id` → `redirect_url` for 1h. Subsequent `POST /v1/start` requires `instance_id` (hard fail 400 if missing) and validates the requested `srat_callback_url` with exact match against the registered URL. The provider callback `GET /v1/callback` becomes an HTML page (broker-owned) that validates the bound redirect via `instance_id`, shows multilingual ok/ko (en+it, Accept-Language negotiation) inspired by https://my.home-assistant.io/redirect/oauth, and then redirects to the HA instance.

> _Context for Copilot: Current flow stores arbitrary srat_callback_url per session with no instance binding. New system adds instance binding with TTL to prevent open-redirect and session hijacking. instance_id is client-provided (no broker fallback). Same BROKER_API_TOKEN auth. Strict exact match on redirect_url._

## 🛠️ Technical Specifications

- **Inputs:** `POST /v1/instances/register {instance_id, redirect_url}` (auth Bearer), `POST /v1/start {provider, srat_callback_url, instance_id}` (auth Bearer), `GET /v1/callback?code&state` (browser redirect from provider)
- **Outputs:** Registration → `{instance_id, redirect_url, expires_at}`; Start → 400/403 on validation fail else `{auth_url, session_id}`; Callback → HTML page (en/it) with auto-redirect or error
- **Dependencies:** `oauth_broker/src/app.ts`, `oauth_broker/src/config.ts`, `oauth_broker/src/types.ts`, `oauth_broker/src/session.ts` (new InstanceStore), D1/KV stores, `oauth_broker/migrations/*`, `oauth_broker/tests/*`

## 📝 Task List

- [x] Task 1: Instance registration store (Memory/KV/D1 + D1 migration 0002, TTL 1h, exact-match validation helpers)
- [x] Task 2: New endpoint POST /v1/instances/register (Bearer auth, client-provided instance_id required, exact redirect validation, rate limiting)
- [x] Task 3: Harden POST /v1/start (instance_id required hard 400, exact redirect match against registered instance, 403/410 handling)
- [x] Task 4: Convert GET /v1/callback to broker-owned HTML page with i18n (en/it) ok/ko and validated redirect (inspired by my.home-assistant.io)
- [x] Task 5: Tests + docs (broker tests ≥70%, update oauth_broker/README and CLOUD_STORAGE_OAUTH)

## 🧠 Implementation Notes (Copilot Context)

- Client-provided instance_id: no broker fallback; if missing or empty → 400. Validate instance_id format (uuid/v4 or non-empty string ≤128 chars).
- redirect_url validation: reuse isValidSratCallbackUrl + MAX_CALLBACK_URL_LENGTH + isAllowedSratCallbackUrl allowlist; store normalized (trim, no trailing slash logic? keep exact string for exact match).
- TTL: 3600s (1h). Store expires_at = now + TTL. On get, check expiry lazily (like MemorySessionStore). D1: expires_at INTEGER (unix sec).
- POST /v1/start: require instance_id, lookup instance store, if not found/expired → 410 or 403 with i18n-safe error; if srat_callback_url !== registered redirect_url (exact string equality after normalization) → 403 "redirect_url mismatch for instance". Store instanceId in SessionRecord (extend type).
- GET /v1/callback: after token exchange, render HTML instead of immediate 302. Use instance binding to resolve redirect (already in session). Page: simple HTML with CSP, no external assets, inline CSS, localized strings via Accept-Language parsing (en default, it if primary tag starts with it). Auto-redirect via meta refresh 2s + JS location.href + manual link button. On error (session/instance missing, exchange failure) render error page with localized message, no redirect.
- Same BROKER_API_TOKEN for registration endpoint via requireBearer helper.
- Rate limits: reuse RATE_LIMITS map, add entry for /v1/instances/register (20/min like /v1/start).
- Security headers preserved on HTML responses (CSP frame-ancestors none, etc.).
- Keep PKCE flow unchanged; SessionRecord extension must include codeVerifier + instanceId.

## 🔗 Code References & TODOs

- [x] `oauth_broker/src/session.ts` - add InstanceRecord, InstanceStore interface, MemoryInstanceStore/KVInstanceStore/D1InstanceStore
- [x] `oauth_broker/src/types.ts` - add InstanceRecord types
- [x] `oauth_broker/src/config.ts` - INSTANCE_TTL_SECONDS=3600 + getInstanceTtlSeconds + MAX_INSTANCE_ID_LENGTH
- [x] `oauth_broker/src/app.ts` - new route + hardened start + HTML callback (en/it, Accept-Language, exact match)
- [x] `oauth_broker/src/i18n.ts` - new i18n catalog + renderHtmlPage (my.home-assistant.io inspired)
- [x] `oauth_broker/src/index.ts` - wire InstanceStore for Workers/Node (D1/KV/Memory)
- [x] `oauth_broker/migrations/0002_create_instances.sql` - D1 table for instances
- [x] `oauth_broker/tests/*` - updated broker/security/contract tests for instance flow, HTML callback, TTL 1h

Verification: `mise run //oauth_broker:test:ci` 80/80 passed, coverage 75.81% (≥70% gate), `bun tsc --noEmit` clean.
