# Corporate Proxy SSL Fix for hk / PKL / bun
<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents** *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Problem](#problem)
  - [PKL error](#pkl-error)
  - [Root cause](#root-cause)
- [Fix (one-time setup per machine)](#fix-one-time-setup-per-machine)
  - [1 - Export the Windows trusted root CAs to a PEM bundle](#1---export-the-windows-trusted-root-cas-to-a-pem-bundle)
  - [2 - Pre-download the hk PKL package](#2---pre-download-the-hk-pkl-package)
  - [3 - Verify PKL / hk](#3--verify-pkl--hk)
  - [4 - Fix `mise install` (npm tool downloads)](#4--fix-mise-install-npm-tool-downloads)
  - [5 - Verify full install](#5--verify-full-install)
- [Re-running after a hk version upgrade](#re-running-after-a-hk-version-upgrade)
- [Why `JAVA_TOOL_OPTIONS` does not work for PKL](#why-java_tool_options-does-not-work-for-pkl)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Problem

On corporate networks with SSL inspection (MITM proxy), two separate issues appear:

1. **`hk check` / PKL**: fails with a certificate error (see below).
2. **`mise install`**: fails with `SELF_SIGNED_CERT_IN_CHAIN` when bun downloads npm packages.

Both are fixed by the same CA bundle; only the configuration target differs.

---

### PKL error

Running `hk check` (or any `hk` command) fails with the following error:

```text
–– Pkl Error ––
Exception when making request `GET https://github.com/jdx/hk/releases/download/...`:
Error during SSL handshake with host `github.com`:
unable to find valid certification path to requested target
```

### Root cause

`hk` uses **PKL** (Apple's Pkl language) to evaluate `hk.pkl`. PKL is distributed as a
GraalVM native image with a **bundled trust store** - it does not use the Windows
certificate store or respect `JAVA_TOOL_OPTIONS`. When the corporate proxy rewrites
TLS certificates using a corporate root CA that is not in PKL's bundled store, all
outbound HTTPS requests from PKL fail.

---

## Fix (one-time setup per machine)

### 1 - Export the Windows trusted root CAs to a PEM bundle

Open **PowerShell** and run:

```powershell
$pem = ""
Get-ChildItem Cert:\LocalMachine\Root | ForEach-Object {
    $b64 = [System.Convert]::ToBase64String($_.RawData, 'InsertLineBreaks')
    $pem += "-----BEGIN CERTIFICATE-----`n$b64`n-----END CERTIFICATE-----`n"
}
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.pkl" | Out-Null
Set-Content -Path "$env:USERPROFILE\.pkl\windows-ca-bundle.pem" -Value $pem -Encoding utf8
```

This exports all trusted root CAs (including the corporate CA) into
`~/.pkl/windows-ca-bundle.pem`.

### 2 - Pre-download the hk PKL package

PKL caches downloaded packages locally. Once cached, it never re-downloads them
(no more SSL checks needed for that package version). Run this once from a bash shell:

```bash
pkl download-package \
  --ca-certificates "$USERPROFILE/.pkl/windows-ca-bundle.pem" \
  "package://github.com/jdx/hk/releases/download/v1.43.0/hk@1.43.0"
```

The package is stored at `%USERPROFILE%\.pkl\cache\`.

### 3 — Verify PKL / hk

```bash
hk check
```

Should now load the PKL config from cache without any network access.

### 4 — Fix `mise install` (npm tool downloads)

`mise install` uses bun to download npm tools, but bun has three issues on Windows corporate
networks:
- Does not trust the proxy CA → `SELF_SIGNED_CERT_IN_CHAIN`
- Misparses Nexus-style registry paths (drops the repo-name segment) → 400/404 on the
  corporate proxy
- Fails to enqueue lifecycle scripts with ENOENT on paths that contain spaces

Create a **machine-local** mise config file (never committed — already in `.gitignore`):

```bash
# create/edit .mise.local.toml at the repo root
cat > .mise.local.toml << 'EOF'
# Machine-local mise overrides — NOT committed to git.

# Switch mise npm-tool installs from bun to system npm.
# Bun has known ENOENT lifecycle-script failures on Windows paths with spaces.
[settings]
npm.package_manager = "npm"
npm.bun = false

[env]
# Trust the corporate CA bundle (created in step 1) for Node/npm HTTPS requests.
NODE_EXTRA_CA_CERTS = "C:/Users/<your-username>/.pkl/windows-ca-bundle.pem"

# Use the official npm registry (corporate Nexus misses preview/nightly packages).
npm_config_registry = "https://registry.npmjs.org"
EOF
```

Replace `<your-username>` with your Windows username.

### 5 — Verify full install

```bash
mise install
```

Should complete without errors.

## Re-running after a hk version upgrade

When `hk.pkl` is updated to a new version (e.g. `v1.44.0`), repeat the PKL download step
with the new package URL:

```bash
pkl download-package \
  --ca-certificates "$USERPROFILE/.pkl/windows-ca-bundle.pem" \
  "package://github.com/jdx/hk/releases/download/v1.44.0/hk@1.44.0"
```

The `windows-ca-bundle.pem` and `.mise.local.toml` do not need updating unless the
corporate root CA changes.

## Why `JAVA_TOOL_OPTIONS` does not work for PKL

PKL is a GraalVM native image. Standard JVM system properties
(`-Djavax.net.ssl.trustStoreType=Windows-ROOT`) and environment variables such as
`JAVA_TOOL_OPTIONS` are not effective because the GraalVM native image has its own
compiled-in SSL layer and does not delegate to a JVM at runtime.
