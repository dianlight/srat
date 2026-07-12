# Corporate Proxy SSL Fix for hk / PKL / bun

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents** _generated with [DocToc](https://github.com/thlorenz/doctoc)_

- [Problem](#problem)
  - [PKL / hk error](#pkl--hk-error)
  - [bun / Nexus error](#bun--nexus-error)
  - [Root causes](#root-causes)
- [Recommended: automated setup](#recommended-automated-setup)
  - [What the script does](#what-the-script-does)
  - [Run it](#run-it)
- [Manual fallback (if the script does not work)](#manual-fallback-if-the-script-does-not-work)
  - [1 - Export the Windows trusted root CAs to a PEM bundle](#1---export-the-windows-trusted-root-cas-to-a-pem-bundle)
  - [2 - Pre-download the hk PKL package](#2---pre-download-the-hk-pkl-package)
  - [3 - Verify PKL / hk](#3---verify-pkl--hk)
  - [4 - Fix bun / Nexus issues manually](#4---fix-bun--nexus-issues-manually)
- [Verify full install](#verify-full-install)
- [Re-running after a hk version upgrade](#re-running-after-a-hk-version-upgrade)
- [Why `JAVA_TOOL_OPTIONS` does not work for PKL](#why-java_tool_options-does-not-work-for-pkl)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Problem

On corporate networks with SSL inspection (MITM proxy), two separate issues appear when
running `mise install`:

1. **`hk install --mise` / PKL**: fails with a certificate error (see below).
2. **npm tool installs (bun)**: fails with `400 Bad Request` against the Nexus registry.

Both are fixed automatically by `scripts/setup-pkl-proxy.sh`.

---

### PKL / hk error

Running `hk install --mise` (or any `hk` command) fails with:

```text
–– Pkl Error ––
Exception when making request `GET https://github.com/jdx/hk/releases/download/...`:
Error during SSL handshake with host `github.com`:
unable to find valid certification path to requested target
```

### bun / Nexus error

`mise install` fails installing npm tools (`prettier`, `markdownlint-cli2`, etc.):

```text
error: GET https://reponexus.servizi.gr-u.it/repository/prettier - 400
mise ERROR bun failed
```

### Root causes

**PKL**: `hk` evaluates `hk.pkl` via **PKL** (Apple's Pkl language), which is a
GraalVM native image with a **bundled trust store**. It does not use the Windows
certificate store or respect `JAVA_TOOL_OPTIONS`. The corporate MITM proxy rewrites
TLS certificates using a CA not in PKL's bundled store, so all outbound HTTPS from PKL
fails. The fix is to pre-populate `~/.pkl/cache/` via `curl` (which uses system OpenSSL
and accepts `--cacert`) so PKL never needs to make a network call.

**bun / Nexus**: bun constructs npm registry URLs by appending the package name directly
to the last path segment of the registry URL, **stripping intermediate segments**. For a
Nexus registry like `https://nexus.host/repository/npm-all`, bun sends requests to
`https://nexus.host/repository/<package>` (400) instead of
`https://nexus.host/repository/npm-all/<package>`. Additionally, bun does not trust the
corporate CA. The fix is to switch mise npm tool installs to system npm and install a
Linux Node.js (so the Windows npm on the WSL PATH is not used).

---

## Recommended: automated setup

`scripts/setup-pkl-proxy.sh` handles everything in one shot. It is also wired into the
mise `postinstall` hook, so `mise install` will attempt it automatically on first run.

### What the script does

1. Reads the hk PKL package version from `hk.pkl` (no hardcoding — survives upgrades).
2. On Windows / WSL: exports trusted root CAs to `~/.pkl/windows-ca-bundle.pem` via
   `powershell.exe`. Skipped when the bundle is fresh (< 30 days old).
3. Downloads the PKL package metadata JSON + zip via `curl` (not `pkl download-package`,
   which hangs on the corporate CA) into `~/.pkl/cache/` using the correct PKL cache
   layout, so `hk install --mise` never needs network access.
4. Creates or updates `.mise.local.toml` (gitignored) with:
   - `npm.package_manager = "npm"` / `npm.bun = false` — bypasses bun
   - `node = "22"` — installs a Linux Node.js so the Linux `npm` binary shadows the
     Windows `npm` on the WSL PATH (Windows npm cannot install to Linux paths)
   - `NODE_EXTRA_CA_CERTS = "$HOME/.pkl/windows-ca-bundle.pem"` — corporate CA trust
   - `npm_config_registry = "https://registry.npmjs.org"` — bypasses broken Nexus
5. Writes a sentinel file per PKL package version so subsequent runs are instant.
6. Degrades gracefully on CI / Linux (skips CA export and Windows-specific steps).

### Run it

```bash
# Run once manually (or let `mise install` invoke it automatically via postinstall hook)
mise run setup-pkl-proxy

# Or call directly from the repo root
./scripts/setup-pkl-proxy.sh

# Force a re-download (e.g., after a corporate CA rotation)
./scripts/setup-pkl-proxy.sh --force

# Show which PKL package version will be downloaded
./scripts/setup-pkl-proxy.sh --version-only

# Help
./scripts/setup-pkl-proxy.sh --help
```

After the script runs, `mise install` should complete cleanly. The git pre-commit hook
is installed and `hk check` works from cache without any network access.

---

## Manual fallback (if the script does not work)

Use these steps only when `scripts/setup-pkl-proxy.sh` is not available or when you
need more control (e.g., a restricted environment where `powershell.exe` cannot be
called from WSL).

### 1 - Export the Windows trusted root CAs to a PEM bundle

The PEM file must be written to the **Linux filesystem** (not Windows) because `pkl`
and `curl` run as Linux processes inside WSL.

Open **PowerShell** and run:

```powershell
$pem = ""
Get-ChildItem Cert:\LocalMachine\Root | ForEach-Object {
    $b64 = [System.Convert]::ToBase64String($_.RawData, 'InsertLineBreaks')
    $pem += "-----BEGIN CERTIFICATE-----`n$b64`n-----END CERTIFICATE-----`n"
}
```

Then from a bash shell, redirect the output to the Linux filesystem:

```bash
# From a WSL bash shell (not PowerShell):
mkdir -p ~/.pkl
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "
  [Console]::OutputEncoding = [System.Text.Encoding]::ASCII
  Get-ChildItem Cert:\LocalMachine\Root | ForEach-Object {
    \$b64 = [System.Convert]::ToBase64String(\$_.RawData, 'InsertLineBreaks')
    Write-Output '-----BEGIN CERTIFICATE-----'
    Write-Output \$b64
    Write-Output '-----END CERTIFICATE-----'
  }
" | tr -d '\r' > ~/.pkl/windows-ca-bundle.pem
```

### 2 - Pre-download the hk PKL package

Download the metadata JSON and zip via `curl` into the exact PKL cache directory:

```bash
VERSION=$(grep -oE 'hk@[0-9]+\.[0-9]+\.[0-9]+' hk.pkl | head -1 | sed 's/hk@//')
PKG_DIR="$HOME/.pkl/cache/package-2/github.com/jdx/hk/releases/download/v${VERSION}/hk@${VERSION}"
mkdir -p "$PKG_DIR"
BASE="https://github.com/jdx/hk/releases/download/v${VERSION}"
curl -sL --cacert ~/.pkl/windows-ca-bundle.pem --proxy "${HTTPS_PROXY:-}" \
  "${BASE}/hk@${VERSION}" -o "${PKG_DIR}/hk@${VERSION}.json"
curl -sL --cacert ~/.pkl/windows-ca-bundle.pem --proxy "${HTTPS_PROXY:-}" \
  "${BASE}/hk@${VERSION}.zip" -o "${PKG_DIR}/hk@${VERSION}.zip"
```

### 3 - Verify PKL / hk

```bash
hk install --mise
hk check
```

Should complete without any network access (PKL reads from cache).

### 4 - Fix bun / Nexus issues manually

Create a **machine-local** mise config file (never committed — already in `.gitignore`):

```toml
# .mise.local.toml — machine-local, never commit
[settings]
npm.package_manager = "npm"
npm.bun = false

[env]
NODE_EXTRA_CA_CERTS = "$HOME/.pkl/windows-ca-bundle.pem"
npm_config_registry = "https://registry.npmjs.org"

[tools]
# Install a Linux Node.js so its npm shadows the Windows npm on the WSL PATH.
# Without this, mise runs the Windows npm which cannot install to Linux paths.
node = "22"
```

Then run `mise install` to install node and all npm tools.

---

## Verify full install

```bash
mise install
```

Should complete without errors. All npm tools (`prettier`, `markdownlint-cli2`, etc.)
install via Linux npm from `registry.npmjs.org`.

---

## Re-running after a hk version upgrade

No action needed when using `scripts/setup-pkl-proxy.sh` — it reads the PKL package
version from `hk.pkl` automatically and the `postinstall` hook reruns it on every
`mise install`.

To force a refresh after a corporate CA rotation:

```bash
./scripts/setup-pkl-proxy.sh --force
```

---

## Why `JAVA_TOOL_OPTIONS` does not work for PKL

PKL is a GraalVM native image. Standard JVM system properties
(`-Djavax.net.ssl.trustStoreType=Windows-ROOT`) and environment variables such as
`JAVA_TOOL_OPTIONS` are not effective because the GraalVM native image has its own
compiled-in SSL layer and does not delegate to a JVM at runtime.
