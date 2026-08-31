# SRAT OAuth Broker

Hosted, serverless OAuth broker for SRAT cloud-sync (issue #1002). One
TypeScript (Hono) codebase deploys to **both** free targets:

- **Cloudflare Workers + KV**—edge, zero cold start, KV session store
- **Render (Node)**—free web service, in-memory store (single instance)

It implements the contract the Go client `backend/src/service/rclone/broker.go`
already consumes.

## Protocol (SRAT contract)

```text
SRAT  POST /v1/start {provider, srat_callback_url}  → {auth_url, session_id}
        broker generates PKCE code_verifier (43-char base64url, 32 random bytes)
        and S256 code_challenge = BASE64URL(SHA256(verifier)); stores verifier
        in session, embeds challenge in auth_url
browser GET auth_url?code_challenge=<S256>&code_challenge_method=S256  → provider sign-in / consent
provider GET {BROKER_PUBLIC_URL}/v1/callback?code&state  (state = session_id)
broker  exchanges code + code_verifier with ITS client secret (S256 proof),
        stores rclone-shaped token under session_id (single use, short TTL)
        and 302s browser → srat_callback_url
SRAT  GET /v1/session/{session_id}  → {token_json, account_label, client_id, client_secret}
```

- `POST /v1/start` + `GET /v1/session/{id}` require `Authorization: Bearer <BROKER_API_TOKEN>`
  (constant-time compare). `GET /v1/callback` is browser-facing, protected by
  the opaque single-use `session_id` as OAuth `state` **and** the PKCE
  `code_verifier` / `code_challenge` (S256) binding.
- `GET /v1/healthz` → `{status:"ok", providers:[...]}` (public).

See the full spec in GitHub issue #1002.

## PKCE (RFC 7636) — S256

The broker implements **Proof Key for Code Exchange (PKCE)** with `S256` for
every OAuth flow. No configuration is required — it is always on.

| Step | Function | Detail |
| --- | --- | --- |
| **Generate verifier** | `generateCodeVerifier()` in `src/app.ts` | 32 random bytes via `crypto.getRandomValues()` → 43-char `base64url` string (RFC 7636 §4.1, 43–128 chars, charset `A-Z a-z 0-9 - _ . ~`) |
| **Derive challenge** | `pkceChallengeFromVerifier(verifier)` | `BASE64URL(SHA256(ASCII(verifier)))` using `node:crypto` `createHash("sha256")` |
| **Authorize** | `buildAuthUrl(prov, publicUrl, sessionId, codeVerifier)` | Sets `code_challenge=<S256>` + `code_challenge_method=S256` on the provider `authorize_url` alongside `client_id`, `response_type=code`, `redirect_uri`, `state` |
| **Store** | `POST /v1/start` handler | Persists `codeVerifier` in `SessionRecord.codeVerifier` (memory / KV / D1) for the TTL window |
| **Exchange** | `exchangeCodeForToken(prov, code, redirectUri, fetchImpl, codeVerifier)` | Sends `code_verifier` in the `application/x-www-form-urlencoded` token request (`grant_type=authorization_code`); omitted only for legacy sessions without a verifier |
| **Verify** | Provider | Validates `code_verifier` against the earlier `code_challenge` (S256) before issuing tokens |

- **Why S256 only:** `plain` is not offered — SHA-256 prevents challenge reversal even if the authorize URL leaks in logs. Verified against the RFC 7636 test vector (`dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk` → `E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM`).
- **Compatibility:** Providers that do not support PKCE ignore the extra params; the flow still succeeds. Providers that require PKCE (e.g., Dropbox with PKCE-enforced clients, Google) now succeed where plain `authorization_code` would be rejected. No SRAT client changes are needed — the broker handles PKCE transparently.
- **Session binding:** The verifier is single-use and bound to the opaque `state` (`session_id`), so an intercepted `code` alone cannot be redeemed without the verifier stored only server-side.

## Configuration

| Variable | Required | Default | Notes |
| --- | --- | --- | --- |
| `BROKER_PUBLIC_URL` | yes | – | Externally reachable https base; used as `redirect_uri={BROKER_PUBLIC_URL}/v1/callback` |
| `BROKER_API_TOKEN` | yes (prod) | – | Shared bearer secret SRAT presents; empty requires `BROKER_DISABLE_AUTH=true` for local dev, otherwise 401 |
| `BROKER_PROVIDERS_FILE` | no | – | Path to JSON providers file (Node/Render) |
| `BROKER_PROVIDERS_JSON` | no | – | Inline JSON providers (Workers secret/KV). Merged with file if both set; env shorthand wins |
| `DROPBOX_CLIENT_ID` / `DROPBOX_CLIENT_SECRET` | no | – | Shorthand for built-in `dropbox`. Either these or an entry in providers JSON/file is required for `dropbox` |
| `PORT` | no | `8080` | Listen port for Node/Render (Workers ignores) |
| `SESSION_TTL` | no | `10m` | Completed-session fetch window. Workers KV `expirationTtl`, memory store TTL |
| `BROKER_ALLOWED_CALLBACK_PATTERNS` | no | – | Comma CSV glob allowlist for `srat_callback_url` (e.g. `https://*.srat.example/*,https://srat.example.com/*`). Empty = allow any `https:` (default, see Security notes) |
| `BROKER_DISABLE_AUTH` | no | `false` | Set `true` only for local dev; **refuses to start in production** (when `BROKER_PUBLIC_URL` looks like production) |

### Providers JSON schema

```json
{
  "dropbox": { "client_id": "…", "client_secret": "…" },
  "gdrive": {
    "client_id": "…", "client_secret": "…",
    "authorize_url": "https://accounts.google.com/o/oauth2/v2/auth",
    "token_url": "https://oauth2.googleapis.com/token",
    "scopes": ["https://www.googleapis.com/auth/drive"],
    "auth_params": { "access_type": "offline", "prompt": "consent" }
  }
}
```

Built-in defaults for `dropbox` (`authorize_url`, `token_url`, `auth_params: {token_access_type: offline}`) are merged in—only credentials are required.

## Monorepo wiring

This subproject is the **4th mise config root** alongside `frontend`, `backend`,
`custom_components`:

```toml
# /.mise.toml
[monorepo]
config_roots = ["frontend", "backend", "custom_components", "oauth_broker"]
```

Tasks (Bun):

```sh
mise run //oauth_broker:prepare        # bun install
mise run //oauth_broker:build          # tsc --noEmit
mise run //oauth_broker:test           # Bun runtime (~2× faster, no coverage)
mise run //oauth_broker:test:ci        # Node runtime + coverage (≥70% gate, CI)
mise run //oauth_broker:test:coverage  # Node + coverage locally
mise run //oauth_broker:lint
mise run //oauth_broker:format
mise run test                          # runs all 4 suites (backend+frontend+hac+broker)
```

**Coverage caveat:** Do NOT combine `--bun` with `--coverage`—`@vitest/coverage-v8`
report merging crashes under Bun (same as frontend). Use `test:ci` for coverage.

## Local dev

```sh
# Node (memory store)
BROKER_PUBLIC_URL=http://localhost:8080 \
BROKER_API_TOKEN=test-token \
DROPBOX_CLIENT_ID=xxx DROPBOX_CLIENT_SECRET=yyy \
mise run //oauth_broker:dev            # listens on :8080

# Workers
mise run //oauth_broker:dev:worker     # wrangler dev (KV preview)

curl http://localhost:8080/v1/healthz
curl -H "Authorization: Bearer test-token" -H "Content-Type: application/json" \
  -d '{"provider":"dropbox","srat_callback_url":"https://srat.example/cb"}' \
  http://localhost:8080/v1/start
```

## Deploy

The broker **reuses the SRAT version** (`YYYY.MM.PATCH` / `YYYY.MM.PATCH-dev.N` /
`YYYY.MM.PATCH-rc.N` from `build.yaml: setversion`) but deploys via its own
job `deploy-oauth-broker` to **both** platforms.

### GitHub Environments + Sentry mapping

| SRAT version suffix | Broker GitHub env | Sentry env | Workers env | Render service |
| --- | --- | --- | --- | --- |
| `-dev.*` | `broker-staging` | `development` | `staging` | staging |
| `-rc.*` | `broker-staging` | `prerelease` | `staging` | staging |
| final | `broker-production` | `production` | `production` | production |

Workflow: `.github/workflows/oauth-broker.yaml`

- `test-oauth-broker` runs on every PR/push (`mise run //oauth_broker:test:ci`).
- `test-e2e-broker-smoke` hits `BROKER_STAGING_URL/v1/healthz` + bearer smoke.
- `deploy-oauth-broker` (only on non-PR) deploys to Workers (wrangler) and
  triggers Render via deploy hooks, then smokes `healthz`.

### Cloudflare Workers

```sh
# One-time setup per env (staging / production) — D1 (preferred, atomic)
wrangler d1 create srat-oauth-broker-staging --env staging
wrangler d1 create srat-oauth-broker --env production
# Paste database_id into oauth_broker/wrangler.toml [[env.*.d1_databases]] and run:
wrangler d1 migrations apply srat-oauth-broker --env staging
wrangler d1 migrations apply srat-oauth-broker --env production
# Legacy KV (fallback, kept for zero-downtime migration):
wrangler kv namespace create OAUTH_SESSIONS --env staging
wrangler kv namespace create OAUTH_SESSIONS --env production
# Paste IDs into oauth_broker/wrangler.toml kv_namespaces

# Optional Cloudflare Rate Limiting (Both mode: in-app fallback always active)
# For stronger DDoS protection, also create a Workers Rate Limit binding:
# wrangler ratelimit create broker-ratelimit --period 60 --limit 20
# Then uncomment [[ratelimits]] in wrangler.toml and paste namespace_id.
# In-app fallback (20/min /v1/start, 30/min /v1/callback, 60/min /v1/session) still protects Render.

# Secrets per env (Workers secret store, not committed)
wrangler secret put BROKER_API_TOKEN --env staging
wrangler secret put BROKER_API_TOKEN --env production
wrangler secret put DROPBOX_CLIENT_ID --env staging
wrangler secret put DROPBOX_CLIENT_SECRET --env staging
wrangler secret put DROPBOX_CLIENT_ID --env production
wrangler secret put DROPBOX_CLIENT_SECRET --env production
# Optional generic providers
wrangler secret put BROKER_PROVIDERS_JSON --env staging
# BROKER_PUBLIC_URL is set in wrangler.toml [env.*].vars

# Deploy
mise run //oauth_broker:deploy:worker   # or npx wrangler deploy --env staging|production --cwd oauth_broker
```

`wrangler.toml` vars: `BROKER_PUBLIC_URL`, `SESSION_TTL`, `BROKER_ALLOWED_CALLBACK_PATTERNS`. `OAUTH_SESSIONS_DB` (D1) is preferred: `sessions` table (`oauth_broker/migrations/0001_create_sessions.sql`) with atomic `DELETE ... RETURNING` in `D1SessionStore.consume()`; `OAUTH_SESSIONS` (KV) remains as fallback (eventually consistent, best-effort consume). Both use `expirationTtl`/`expires_at = SESSION_TTL` seconds (minimum 60 s); only one concurrent `GET /v1/session/{id}` receives the token when D1 is bound. Rate limiting is **Both mode**: in-app sliding window (20/min `POST /v1/start`, 30/min `GET /v1/callback`, 60/min `GET /v1/session/{id}` per IP) plus optional `RATE_LIMITER` binding when configured.

### Render

`oauth_broker/render.yaml` defines the free web service (per <https://bun.com/guides/deployment/render> — Render's `Node` runtime includes Bun):

- `buildCommand: bun install && bun tsc --noEmit` (type-check at Render build; CI also runs `mise run //oauth_broker:test:ci`)
- `startCommand: bun src/index.ts` (Bun transpiles TS at start)
- `healthCheckPath: /v1/healthz`
- `envVars: PORT=10000`, `BROKER_PUBLIC_URL`, `BROKER_API_TOKEN`, `DROPBOX_*`, optionally `BROKER_PROVIDERS_JSON` / `BROKER_ALLOWED_CALLBACK_PATTERNS`

Connect the GitHub repo to Render, then create **two services** (staging + production) from `render.yaml` or set deploy hooks:

- `RENDER_DEPLOY_HOOK_STAGING` / `RENDER_DEPLOY_HOOK_PRODUCTION` in GitHub
  repo secrets → `deploy-oauth-broker` triggers them via `curl -X POST`.
- `BROKER_STAGING_URL` / `BROKER_PRODUCTION_URL` (e.g. `https://srat-oauth-broker-staging.onrender.com`)
  → healthz smoke.

The same Hono code runs on both: Workers prefers D1 (`OAUTH_SESSIONS_DB`), otherwise KV (`OAUTH_SESSIONS`) when present, otherwise falls back to `MemorySessionStore` (Render / local). No code change is needed per platform.

## Dropbox app registration

The Dropbox app's `redirect_uri` **must** be `{BROKER_PUBLIC_URL}/v1/callback`:

- Staging: `https://srat-oauth-broker-staging.lucio-tarantino.workers.dev/v1/callback` (and/or staging Render URL)
- Production: `https://srat-oauth-broker-production.lucio-tarantino.workers.dev/v1/callback` (and/or prod Render URL)

Create two Dropbox apps (or one with both URIs) at <https://www.dropbox.com/developers/apps>.
Use the app key/secret as `DROPBOX_CLIENT_ID` / `DROPBOX_CLIENT_SECRET` (or in providers JSON).

## Google Drive & Google Photos app registration

Both use Google Cloud OAuth 2.0. Register a project at <https://console.cloud.google.com>:

1. Create/select a Google Cloud project
2. Enable **Google Drive API** and **Google Photos Library API** (for Photos)
3. Go to **APIs & Services → Credentials → Create Credentials → OAuth client ID**
4. Application type: **Web application**
5. Authorized redirect URIs: add `{BROKER_PUBLIC_URL}/v1/callback` for each environment
   - Staging: `https://srat-oauth-broker-staging.lucio-tarantino.workers.dev/v1/callback`
   - Production: `https://srat-oauth-broker-production.lucio-tarantino.workers.dev/v1/callback`
6. Copy **Client ID** and **Client Secret**

**Google Photos specifics:**
- Uses the same OAuth credentials as Google Drive (same project)
- Requires additional scope: `https://www.googleapis.com/auth/photoslibrary.readonly` (or `.appendonly`, `.sharing`, `.edit`)
- The Photos Library API has stricter quota (10k requests/day default) — request increase if needed
- `photoslibrary.readonly` scope allows listing albums/media but **not** downloading original files; use `photoslibrary` (full) for download access
- Add scopes in providers JSON under `gdrive` entry:
```json
"gdrive": {
  "client_id": "...",
  "client_secret": "...",
  "scopes": [
    "https://www.googleapis.com/auth/drive",
    "https://www.googleapis.com/auth/photoslibrary.readonly"
  ]
}
```

**Built-in fallback**: rclone ships with a default Google client ID/secret, but it's shared and rate-limited. Register your own for production.

## OneDrive (Microsoft Graph) app registration

Register at <https://portal.azure.com> → **App registrations**:

1. **New registration** → name it, select "Accounts in any organizational directory and personal Microsoft accounts"
2. Redirect URI: **Web** → `{BROKER_PUBLIC_URL}/v1/callback`
3. **Certificates & secrets** → New client secret
4. **API permissions** → Add permission → Microsoft Graph → Delegated → `Files.ReadWrite.All`, `offline_access`
5. Grant admin consent if required by tenant

Use **Application (client) ID** and **Client secret** as `ONEDRIVE_CLIENT_ID` / `ONEDRIVE_CLIENT_SECRET`.

**Built-in fallback**: rclone includes a default Microsoft client ID, but register your own for production.

## iCloud (CloudKit) — different auth model

iCloud does **not** use OAuth. It uses Apple's **CloudKit** with app-specific passwords:

1. Apple Developer account required ($99/year)
2. Create a **CloudKit container** in Certificates, Identifiers & Profiles
3. Enable **CloudKit** capability for your App ID
4. User generates an **app-specific password** at <https://appleid.apple.com> → Security → App-Specific Passwords
5. rclone iCloud backend uses: Apple ID + app-specific password (not client_id/secret)

No broker registration needed — SRAT would need a different auth flow for iCloud (not currently supported by oauth_broker).

## Testing

Contract tests mirror `backend/src/service/rclone/broker_test.go` (`fakeBroker`) to
guarantee SRAT client compatibility:

- `POST /v1/start` happy path + early validation + bearer + public URL trimming + **PKCE** `code_challenge` S256 present
- `GET /v1/callback` 400/410/502 + token envelope wrapping (`expiry` RFC3339) + **PKCE** `code_verifier` forwarded (S256 proof)
- `GET /v1/session/{id}` early polling 404 without consuming + single-use 200 + `Cache-Control: no-store` + path escaping
- Providers: `BROKER_PROVIDERS_FILE` + `BROKER_PROVIDERS_JSON` merging, built-in defaults, missing file ignored
- Session stores: `MemorySessionStore` TTL + `KVSessionStore` JSON roundtrip + `codeVerifier` persistence
- **PKCE**: RFC 7636 test vector (`dBjftJeZ4CVP…` → `E9Melhoa…`), verifier 43-char length + charset, challenge S256, `buildAuthUrl` / `exchangeCodeForToken` unit coverage, end-to-end `POST /v1/start` → `GET /v1/callback` verifier round-trip
- Coverage gate ≥70% lines/functions (see `vitest.config.ts`)

```sh
mise run //oauth_broker:test:ci   # must pass before PR merge
```

SRAT addon wizard: set `SRAT_OAUTH_BROKER_URL` to the deployed `BROKER_PUBLIC_URL`
and `SRAT_OAUTH_BROKER_TOKEN` to `BROKER_API_TOKEN`; the “Hosted SRAT OAuth”
wizard option becomes available when `broker_available` is true (`GET /rclone/providers`).

## Security notes (hardened 2026-08 – Z-Audit fixes; PKCE added 2026-08)

- **PKCE S256 (RFC 7636):** Every `POST /v1/start` generates a 32-byte `code_verifier` → 43-char `base64url` and stores it in the session; `buildAuthUrl` sends `code_challenge=S256(challenge)` + `code_challenge_method=S256`; `GET /v1/callback` sends `code_verifier` in the token exchange. This binds the authorization `code` to the server-side verifier, so an intercepted `code` (e.g., via log leak or open-redirect) cannot be redeemed without the verifier. `plain` is never offered; S256 is verified against the RFC test vector. See `## PKCE` above.

- `client_secret` is sent to the provider `token_url` during code exchange and
  returned only inside the single-use `GET /v1/session/{id}` SRAT handover
  (librclone needs it bound to refresh token); never shipped in the binary.
  Responses use `Cache-Control: no-store, no-cache, must-revalidate` + `Pragma: no-cache` + `Expires: 0`.
- Sessions expire after `SESSION_TTL`, consumed on first fetch; early polling cannot
  destroy an in-flight flow. `MemorySessionStore` caps at 10 000 entries with TTL eviction and returns `429` when full (DoS guard).
- `srat_callback_url` validated as absolute `https` (loopback `http` allowed for dev) and capped at 2048 chars. By default any `https:` is accepted (SRAT instances have arbitrary domains), but **optional allowlist** `BROKER_ALLOWED_CALLBACK_PATTERNS` (comma CSV globs, e.g. `https://*.srat.example/*`) enforces `403` when set. If `BROKER_API_TOKEN` leaks, an attacker could still call `POST /v1/start` with an attacker-controlled callback and have the victim browser 302'd there after provider consent (token itself stays single-use in the broker). Mitigate by rotating `BROKER_API_TOKEN` per environment, keeping it only in `wrangler secret` / Render env / GitHub secrets, setting the allowlist, and fronting the broker with an IP allowlist if needed. See `docs/CLOUD_STORAGE_OAUTH.md` trade-off note.
- `BROKER_PUBLIC_URL` validated as absolute `https` (loopback `http` allowed for dev); misconfigured `http://` outside loopback now returns `500 BROKER_PUBLIC_URL must be an absolute https URL` instead of a later opaque `502 redirect_uri_mismatch` from the provider. Startup also fails fast if `BROKER_PUBLIC_URL` is invalid.
- Bearer auth uses `crypto.timingSafeEqual` over SHA-256 digests (fixed-length, non-ASCII safe) constant-time compare; missing `BROKER_API_TOKEN` fails closed unless `BROKER_DISABLE_AUTH=true`. **`BROKER_DISABLE_AUTH=true` is refused in production** (`BROKER_PUBLIC_URL` containing `production` or `ENV=production`) – the broker logs `BROKER_DISABLE_AUTH is not allowed in production – denying request` and returns `401`; Node/Workers startup throws `must not be enabled in production – refusing to start`.
- **Rate limiting (Both mode):** in-app sliding window per IP (`20/min POST /v1/start`, `30/min GET /v1/callback`, `60/min GET /v1/session`) always active, plus optional Cloudflare `RATE_LIMITER` binding (`wrangler ratelimit create broker-ratelimit --period 60 --limit 20` + `[[ratelimits]]` in `wrangler.toml`) when configured. Exceeded returns `429 + Retry-After:60`.
- **Security headers on every response:** `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, `Strict-Transport-Security: max-age=63072000; includeSubDomains; preload`, `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`, `X-Robots-Tag: noindex, nofollow`, `Permissions-Policy: camera=(), microphone=(), geolocation=()`. `GET /v1/callback` 302 and all `/v1/session` responses also include `Cache-Control: no-store` triple.
- **CORS deny by default:** no `Access-Control-Allow-Origin` is ever set; `OPTIONS` preflight returns `204` with `Allow-Methods/Headers` only. Broker is server-to-server (SRAT Go backend) + browser 302, not a browser `fetch` API.
- **Token exchange errors are generic:** provider `error`/`error_description` is logged server-side (`[broker] token exchange failed for provider …`) but the browser/client receives only `502 token exchange failed`, preventing secret detail leakage.
- `BROKER_PROVIDERS_JSON` / `BROKER_PROVIDERS_FILE` malformed JSON now warns `malformed … ignored` instead of silent swallow; `SESSION_TTL` unparseable warns and falls back to `600s`.
- Provider `client_secret` never committed; supply via `wrangler secret put` / Render env / GitHub secrets per environment.
- Session store precedence: `OAUTH_SESSIONS_DB` (D1, atomic `DELETE ... RETURNING`) > `OAUTH_SESSIONS` (KV, eventually consistent, non-atomic consume) > `MemorySessionStore`. Workers logs `warn` when D1 is missing / KV fallback. Use D1 in production for single-use guarantee.
