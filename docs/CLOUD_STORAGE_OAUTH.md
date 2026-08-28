# Cloud Storage OAuth Requirements for SRAT

This document explains why OAuth app registration is required for each cloud storage provider and how to set it up for use with SRAT's OAuth broker.

## Why App Registration Is Required

OAuth providers (Dropbox, Google, Microsoft) require each integration to have its own **client ID** and **client secret** for several reasons:

| Reason | Explanation |
| -------- | ------------- |
| **Identity & Audit** | Providers track which app requests access for security auditing |
| **Rate Limiting** | Quotas are enforced per client ID (shared defaults get throttled) |
| **User Consent** | Users grant/revoke access per-app in their account settings |
| **Security Policy** | Providers can revoke a single compromised app without affecting others |
| **Branding** | Consent screen shows your app name/logo, not "rclone" or "SRAT" |

**Bottom line**: You cannot use a generic "rclone" or "SRAT" client ID in production—each deployment needs its own credentials.

---

## Provider Comparison

| Provider | Auth Protocol | App Registration Required? | Built-in Fallback? | Notes |
| ---------- | --------------- | --------------------------- | ------------------- | ------- |
| **Dropbox** | OAuth 2.0 | **Yes** (mandatory) | No | Must register at dropbox.com/developers/apps |
| **Google Drive** | OAuth 2.0 | **Yes** (recommended) | Yes (shared, rate-limited) | Google Cloud Console |
| **Google Photos** | OAuth 2.0 | **Yes** (recommended) | Yes (same as Drive) | Same project as Drive, extra scopes |
| **OneDrive** | OAuth 2.0 (Microsoft Graph) | **Yes** (recommended) | Yes (shared, rate-limited) | Azure AD App Registration |
| **iCloud** | CloudKit + App-Specific Passwords | **No (different model)** | N/A | Requires Apple Developer account ($99/yr) |

---

## Dropbox

### Why Required
Dropbox **does not provide** a shared client ID. Every integration must register its own app at <https://www.dropbox.com/developers/apps>.

### Setup
1. Go to <https://www.dropbox.com/developers/apps>
2. Select **Create app** → **Scoped access** → **Full Dropbox** (or App folder)
3. Name your app (e.g., "SRAT Staging", "SRAT Production")
4. **Redirect URI**: `{BROKER_PUBLIC_URL}/v1/callback`
   - Staging: `https://srat-oauth-broker-staging.lucio-tarantino.workers.dev/v1/callback`
   - Production: `https://srat-oauth-broker.workers.dev/v1/callback`
5. Copy **App key** → `DROPBOX_CLIENT_ID`
6. Copy **App secret** → `DROPBOX_CLIENT_SECRET`

### OAuth Broker Config
```bash
# Environment variables or providers JSON
DROPBOX_CLIENT_ID=your_app_key
DROPBOX_CLIENT_SECRET=your_app_secret
```

---

## Google Drive & Google Photos

### Why Required (Recommended)
Google provides a **shared default client ID** that rclone uses, but it's:
- Rate-limited (shared across all users)
- Not configurable (can't add custom scopes)
- Subject to Google's unpublished quotas

For production SRAT deployments, register your own Google Cloud project.

### Setup (Single Project for Both)
1. Go to <https://console.cloud.google.com>
2. Create or select a project
3. **Enable APIs**:
   - Google Drive API
   - Google Photos Library API
4. **OAuth Consent Screen**: Configure (External user type, add scopes)
5. **Credentials → Create Credentials → OAuth client ID**:
   - Application type: **Web application**
   - Name: "SRAT OAuth Broker"
   - **Authorized redirect URIs**:
     - `https://srat-oauth-broker-staging.lucio-tarantino.workers.dev/v1/callback`
     - `https://srat-oauth-broker.workers.dev/v1/callback`
6. Copy **Client ID** and **Client Secret**

### Scopes

| Service | Scope | Access Level |
| --------- | ------- | -------------- |
| Google Drive | `https://www.googleapis.com/auth/drive` | Full Drive access |
| Google Photos (read) | `https://www.googleapis.com/auth/photoslibrary.readonly` | List albums/media, no downloads |
| Google Photos (full) | `https://www.googleapis.com/auth/photoslibrary` | Full access including downloads |

**Note**: `photoslibrary.readonly` does **not** allow downloading original media files. Use the full `photoslibrary` scope for backup/sync use cases.

### OAuth Broker Config (providers JSON)
```json
{
  "gdrive": {
    "client_id": "your-client-id.apps.googleusercontent.com",
    "client_secret": "your-client-secret",
    "authorize_url": "https://accounts.google.com/o/oauth2/v2/auth",
    "token_url": "https://oauth2.googleapis.com/token",
    "scopes": [
      "https://www.googleapis.com/auth/drive",
      "https://www.googleapis.com/auth/photoslibrary.readonly"
    ],
    "auth_params": {
      "access_type": "offline",
      "prompt": "consent"
    }
  }
}
```

### Quotas
- **Google Drive**: Generous daily quotas (billions of requests)
- **Google Photos Library API**: 10,000 requests/day default — [request increase](https://developers.google.com/photos/library/guides/quota) if needed

---

## OneDrive (Microsoft Graph)

### Why Required (Recommended)
Microsoft provides a shared default client ID in rclone, but it's:
- Rate-limited
- Not configurable for tenant-specific permissions
- Intended for testing only

### Setup
1. Go to <https://portal.azure.com> → **App registrations** → **New registration**
2. Name: "SRAT OAuth Broker"
3. Supported account types: **Accounts in any organizational directory and personal Microsoft accounts**
4. **Redirect URI** (Web): `{BROKER_PUBLIC_URL}/v1/callback`
   - Staging: `https://srat-oauth-broker-staging.lucio-tarantino.workers.dev/v1/callback`
   - Production: `https://srat-oauth-broker.workers.dev/v1/callback`
5. **Certificates & secrets** → **New client secret** → copy value immediately
6. **API permissions** → **Add a permission** → **Microsoft Graph** → **Delegated permissions**:
   - `Files.ReadWrite.All`
   - `offline_access`
7. **Grant admin consent** (if required by your tenant)

### OAuth Broker Config (providers JSON)
```json
{
  "onedrive": {
    "client_id": "your-azure-ad-app-id",
    "client_secret": "your-client-secret",
    "authorize_url": "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
    "token_url": "https://login.microsoftonline.com/common/oauth2/v2.0/token",
    "scopes": ["https://graph.microsoft.com/Files.ReadWrite.All", "offline_access"]
  }
}
```

---

## iCloud (CloudKit)

### Different Auth Model
iCloud **does not use OAuth**. It uses Apple's **CloudKit** framework with **app-specific passwords**.

### Requirements

| Requirement | Details |
| ------------- | --------- |
| Apple Developer Account | $99/year (required for CloudKit container) |
| CloudKit Container | Created in Certificates, Identifiers & Profiles |
| App-Specific Password | Generated by user at appleid.apple.com |

### Flow
1. User provides their **Apple ID** (email)
2. User generates an **app-specific password** (one-time, labeled "SRAT")
3. rclone uses: `Apple ID` + `app-specific password` for auth
4. No client_id/client_secret, no redirect URI, no consent screen

### SRAT Integration Status
**Not currently supported** by oauth_broker. Would require a separate auth implementation:
- No OAuth flow → broker's `/v1/start` + `/v1/callback` pattern doesn't apply
- Could add a "direct credentials" endpoint in broker
- Requires Apple Developer account for SRAT project

---

## Quick Reference: Redirect URIs by Environment

| Environment | Redirect URI |
| ------------- | -------------- |
| Local Dev (Node) | `http://localhost:8080/v1/callback` |
| Local Dev (Workers) | `http://localhost:8787/v1/callback` (wrangler dev) |
| Staging (Workers) | `https://srat-oauth-broker-staging.lucio-tarantino.workers.dev/v1/callback` |
| Staging (Render) | `https://srat-oauth-broker-staging.onrender.com/v1/callback` |
| Production (Workers) | `https://srat-oauth-broker.workers.dev/v1/callback` |
| Production (Render) | `https://srat-oauth-broker.onrender.com/v1/callback` |

**Add all applicable URIs** to each provider's app registration.

---

## Testing Checklist

- [ ] Dropbox: App created, redirect URIs match, credentials in broker env/secrets
- [ ] Google: Project created, both APIs enabled, OAuth client configured, scopes added
- [ ] Google Photos: Photos Library API enabled, appropriate scope selected
- [ ] OneDrive: Azure AD app registered, Graph permissions granted, admin consent if needed
- [ ] All: `mise run //oauth_broker:test:ci` passes
- [ ] All: `GET /v1/healthz` returns provider list including configured providers
- [ ] All: End-to-end flow works via SRAT addon wizard ("Hosted SRAT OAuth" option)

---

## Related Files

- `oauth_broker/README.md`—Broker setup, deployment, configuration
- `backend/src/service/rclone/broker.go`—Go client that consumes broker API
- `.github/workflows/build.yaml`—CI/CD for broker (test + deploy jobs)
