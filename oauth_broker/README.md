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
browser GET auth_url  → provider sign-in / consent
provider GET {BROKER_PUBLIC_URL}/v1/callback?code&state  (state = session_id)
broker  exchanges code with ITS client secret, stores rclone-shaped token
        under session_id (single use, short TTL) and 302s browser → srat_callback_url
SRAT  GET /v1/session/{session_id}  → {token_json, account_label, client_id, client_secret}
```

- `POST /v1/start` + `GET /v1/session/{id}` require `Authorization: Bearer <BROKER_API_TOKEN>`
  (constant-time compare). `GET /v1/callback` is browser-facing, protected by
  the opaque single-use `session_id` as OAuth `state`.
- `GET /v1/healthz` → `{status:"ok", providers:[...]}` (public).

See the full spec in GitHub issue #1002.

## Configuration

| Variable | Required | Default | Notes |
| --- | --- | --- | --- |
| `BROKER_PUBLIC_URL` | yes | – | Externally reachable https base; used as `redirect_uri={BROKER_PUBLIC_URL}/v1/callback` |
| `BROKER_API_TOKEN` | yes (prod) | – | Shared bearer secret SRAT presents; empty allows dev without auth |
| `BROKER_PROVIDERS_FILE` | no | – | Path to JSON providers file (Node/Render) |
| `BROKER_PROVIDERS_JSON` | no | – | Inline JSON providers (Workers secret/KV). Merged with file if both set; env shorthand wins |
| `DROPBOX_CLIENT_ID` / `DROPBOX_CLIENT_SECRET` | no | – | Shorthand for built-in `dropbox`. Either these or an entry in providers JSON/file is required for `dropbox` |
| `PORT` | no | `8080` | Listen port for Node/Render (Workers ignores) |
| `SESSION_TTL` | no | `10m` | Completed-session fetch window. Workers KV `expirationTtl`, memory store TTL |

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

Workflow: `.github/workflows/build.yaml`

- `test-oauth-broker` runs on every PR/push (`mise run //oauth_broker:test:ci`).
- `test-e2e-broker-smoke` hits `BROKER_STAGING_URL/v1/healthz` + bearer smoke.
- `deploy-oauth-broker` (only on non-PR) deploys to Workers (wrangler) and
  triggers Render via deploy hooks, then smokes `healthz`.

### Cloudflare Workers

```sh
# One-time setup per env (staging / production)
wrangler kv namespace create OAUTH_SESSIONS --env staging
wrangler kv namespace create OAUTH_SESSIONS --env production
# Paste IDs into oauth_broker/wrangler.toml kv_namespaces

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

`wrangler.toml` vars: `BROKER_PUBLIC_URL`, `SESSION_TTL`. KV binding
`OAUTH_SESSIONS` uses `expirationTtl = SESSION_TTL` seconds; single-use
consumption does KV `delete` on first successful `GET /v1/session/{id}` (race
acceptable at this scale; eventual consistency tolerable for single-writer short sessions).

### Render

`oauth_broker/render.yaml` defines the free web service:

- `buildCommand: bun install && bun tsc --noEmit`
- `startCommand: bun run src/index.ts`
- `healthCheckPath: /v1/healthz`
- `envVars: PORT=10000`, `BROKER_PUBLIC_URL`, `BROKER_API_TOKEN`, `DROPBOX_*`, optionally `BROKER_PROVIDERS_JSON`

Connect the GitHub repo to Render, then create **two services** (staging + production) from `render.yaml` or set deploy hooks:

- `RENDER_DEPLOY_HOOK_STAGING` / `RENDER_DEPLOY_HOOK_PRODUCTION` in GitHub
  repo secrets → `deploy-oauth-broker` triggers them via `curl -X POST`.
- `BROKER_STAGING_URL` / `BROKER_PRODUCTION_URL` (e.g. `https://srat-oauth-broker-staging.onrender.com`)
  → healthz smoke.

The same Hono code runs on both: Workers uses KV (`OAUTH_SESSIONS` binding) when
present, otherwise falls back to `MemorySessionStore` (Render / local). No code
change is needed per platform.

## Dropbox app registration

The Dropbox app's `redirect_uri` **must** be `{BROKER_PUBLIC_URL}/v1/callback`:

- Staging: `https://srat-oauth-broker-staging.lucio-tarantino.workers.dev/v1/callback` (and/or staging Render URL)
- Production: `https://srat-oauth-broker.workers.dev/v1/callback` (and/or prod Render URL)

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
   - Production: `https://srat-oauth-broker.workers.dev/v1/callback`
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

- `POST /v1/start` happy path + early validation + bearer + public URL trimming
- `GET /v1/callback` 400/410/502 + token envelope wrapping (`expiry` RFC3339)
- `GET /v1/session/{id}` early polling 404 without consuming + single-use 200 + `Cache-Control: no-store` + path escaping
- Providers: `BROKER_PROVIDERS_FILE` + `BROKER_PROVIDERS_JSON` merging, built-in defaults, missing file ignored
- Session stores: `MemorySessionStore` TTL + `KVSessionStore` JSON roundtrip
- Coverage gate ≥70% lines/functions (see `vitest.config.ts`)

```sh
mise run //oauth_broker:test:ci   # must pass before PR merge
```

SRAT addon wizard: set `SRAT_OAUTH_BROKER_URL` to the deployed `BROKER_PUBLIC_URL`
and `SRAT_OAUTH_BROKER_TOKEN` to `BROKER_API_TOKEN`; the “Hosted SRAT OAuth”
wizard option becomes available when `broker_available` is true (`GET /rclone/providers`).

## Security notes

- `client_secret` only leaves the broker inside the single-use `GET /v1/session/{id}`
  handover (librclone needs it bound to refresh token); never shipped in binary.
- Sessions expire after `SESSION_TTL`, consumed on first fetch; early polling cannot
  destroy an in-flight flow.
- `srat_callback_url` validated as absolute `https` (loopback `http` allowed for dev).
- Bearer auth uses `crypto.timingSafeEqual` constant-time compare.
- Provider `client_secret` never committed; supply via `wrangler secret put` / Render env / GitHub secrets per environment.
