<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents** *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Corporate Proxy SSL Fix for hk / PKL](#corporate-proxy-ssl-fix-for-hk--pkl)
  - [Problem](#problem)
    - [Root cause](#root-cause)
  - [Fix (one-time setup per machine)](#fix-one-time-setup-per-machine)
    - [1 - Export the Windows trusted root CAs to a PEM bundle](#1---export-the-windows-trusted-root-cas-to-a-pem-bundle)
    - [2 - Pre-download the hk PKL package](#2---pre-download-the-hk-pkl-package)
    - [3 - Verify](#3---verify)
  - [Re-running after a hk version upgrade](#re-running-after-a-hk-version-upgrade)
  - [Why `JAVA_TOOL_OPTIONS` does not work](#why-java_tool_options-does-not-work)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Corporate Proxy SSL Fix for hk / PKL

## Problem

Running `hk check` (or any `hk` command) fails on corporate networks that perform SSL
inspection (MITM proxy), with the following error:

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

### 3 - Verify

```bash
hk check
```

Should now load the PKL config from cache without any network access.

## Re-running after a hk version upgrade

When `hk.pkl` is updated to a new version (e.g. `v1.44.0`), repeat step 2 with the new
package URL:

```bash
pkl download-package \
  --ca-certificates "$USERPROFILE/.pkl/windows-ca-bundle.pem" \
  "package://github.com/jdx/hk/releases/download/v1.44.0/hk@1.44.0"
```

The `windows-ca-bundle.pem` from step 1 does not need to be regenerated unless the
corporate root CA changes.

## Why `JAVA_TOOL_OPTIONS` does not work

PKL is a GraalVM native image. Standard JVM system properties
(`-Djavax.net.ssl.trustStoreType=Windows-ROOT`) and environment variables such as
`JAVA_TOOL_OPTIONS` are not effective because the GraalVM native image has its own
compiled-in SSL layer and does not delegate to a JVM at runtime.
