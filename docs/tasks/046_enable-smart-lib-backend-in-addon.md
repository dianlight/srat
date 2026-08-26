<!-- DOCTOC SKIP -->

# [FIX]: Enable SMART Lib Backend in Addon

> **Superseded (2026-08-11):** the `smartmontools-go` wrapper was retired. SRAT now
> vendors `github.com/dianlight/smartmontools-sdk/bindings/go/v8` and the CI ABI
> guard checks the SDK release tag + `libsmartmon_go.so` `smartmon_abi_version`
> symbol. See [047_smartmontools-sdk-bindings-migration.md](047_smartmontools-sdk-bindings-migration.md).
> Historical notes below document the original `smartmontools-go` design.

**Target Repo:** `srat` + `hassio-addons` (+ `smartmontools-sdk`, `smartmontools-go`)
**Status:** 🔄 In Progress
**Issue Link:** [dianlight/hassio-addons#726](https://github.com/dianlight/hassio-addons/issues/726) · [dianlight/smartmontools-sdk#14](https://github.com/dianlight/smartmontools-sdk/issues/14) · [dianlight/smartmontools-go#38](https://github.com/dianlight/smartmontools-go/issues/38)

## 🎯 Objective

Make smartmontools **library (direct) mode** available in the SambaNAS2 addon: today, in release 2026.8.0-rc12, SMART access only ever uses the **legacy exec backend** (spawning `smartctl` subprocesses) because (a) the addon runs the static `srat-server` variant, which is built **without** the `smartlib` tag, and (b) even the `smartlib`-capable musl/glib variants cannot load their backend in the addon image because the `libsmartmon_go.so` wrapper library is never built/installed. The fix spans four repos: **smartmontools-sdk** builds + ships the wrapper `.so`/`.dylib` in its release tarballs (Option A, decided), **smartmontools-go** documents the C-ABI stability contract, SRAT verifies/asserts the smartlib pieces and exposes the fallback reason, and the addon installs the wrapper and executes the smartlib-enabled server variant.

> _Context for Copilot: `smart_service_lib.go` is gated by `//go:build smartlib`. Release zips contain three server variants per arch (`srat-server-static`, `srat-server-musl`, `srat-server-glib`), but the `srat-server` symlink points to the static build (`CGO_ENABLED=0`, no `smartlib` tag). The lib backend loads `libsmartmon_go.so` at runtime via purego `dlopen` — the addon only installs the static SDK (`libsmartmon.a` + headers), never the wrapper shared object, so `resolveLibPath()` fails and the backend silently falls back to exec. The runtime upgrade service already prefers `srat-server-musl` on Alpine, but fresh installs run the static symlink and the `.so` is missing regardless._

## 🛠️ Technical Specifications

- **Inputs:**
  - SRAT release artifact `srat_${arch}.zip` (contains `srat-server-static`, `srat-server-musl`, `srat-server-glib`, `srat-cli`, `srat-server` symlink → static)
  - smartmontools-sdk release tarball `libsmartmon-${VERSION}-${target}.tar.gz` (already installed in addon image via `smartmontools-sdk` release) — **will now also contain** `libsmartmon_go.{so,dylib}` + `smartmontools-go-version` ref file (smartmontools-sdk#14)
  - `smartmontools-go/backends/lib/csrc/smartmon_c_api.cpp` thin C++ wrapper source (source of truth for the wrapper; SDK compiles it, SRAT does not vendor it)
- **Outputs:**
  - `libsmartmon_go.so` present in the addon rootfs at a path resolvable by `resolveLibPath()` (`/usr/local/lib/libsmartmon_go.so` — first `defaultLibPaths` entry)
  - Addon boots and runs a `smartlib`-enabled server variant (`srat-server-musl` on Alpine)
  - `GET /api/capabilities` → `lib_smart_available: true`; addon log shows `SMART lib backend loaded (direct mode available)`
  - Optionally, a human-readable fallback reason exposed to diagnose exec-mode fallbacks
- **Dependencies:**
  - `backend/src/service/smart_service_lib.go` (`//go:build smartlib`, purego `dlopen`)
  - `vendor/github.com/dianlight/smartmontools-go/backends/lib/lib.go` (`defaultLibPaths` lines 37–50, `resolveLibPath` lines 472–485)
  - `backend/src/service/upgrade_service.go:921-957` (musl/glib/static variant selection)
  - `backend/.mise.toml` build task (lines 72–114: `--zig`/`--cgo` add `smartlib` tag; musl needs `-dynamic`)
  - `.github/workflows/build.yaml` lines 561–643 (release zip contents)
  - Addon `sambanas2/Dockerfile` (SDK install, SRAT unzip) and `rootfs/etc/s6-overlay/s6-rc.d/srat/run` (executes `/usr/local/bin/srat-server`)
  - `docs/SMART_SERVICE.md` (build modes), `frontend/src/pages/settings/panels/GeneralPanel.tsx:231` (UI hint)

## 📝 Task List

- [ ] Task 1: SDK (smartmontools-sdk#14) — add `libsmartmon_go.{so,dylib}` build step to `.github/workflows/build.yml` per matrix target (using existing `cxx`, compile `smartmon_c_api.cpp` from pinned smartmontools-go ref, record `smartmontools-go-version` in tarball, skip windows). **Done in the SDK repo, not SRAT.** SRAT side: bump the SDK release consumed where relevant and verify the tarball now contains the wrapper. *(SDK side implemented in [smartmontools-sdk#15](https://github.com/dianlight/smartmontools-sdk/pull/15), commit `60ca1499` — not yet merged/released, so the SRAT CI guard is warn-only until a new SDK release ships the wrapper.)*
- [x] Task 2: SRAT — decide + document default variant policy for addon consumption (Alpine is musl → `srat-server-musl`), and add a CI assertion that (a) release zips always contain the musl smartlib variant and (b) the SDK tarball's `smartmontools-go-version` matches SRAT's vendored `smartmontools-go` version (guards the C-ABI contract, smartmontools-go#38). **Done:** `docs/SMART_SERVICE.md` → `## Default Variant Policy (Addon Consumption)`; `.github/workflows/build.yaml` → `verify-smartlib-wrapper-abi` job (warn-only exit 0 when the wrapper file is absent, hard fail on version mismatch once present) + zip-creation step asserts `unzip -l` greps `srat-server-musl` per arch.
- [x] Task 3: SRAT — expose exec-fallback reason in `SystemCapabilities` (e.g. `lib_smart_unavailable_reason` next to `lib_smart_available`) so addon/users can diagnose why direct mode is off. **Done:** `dto/system_capabilities.go` + `dto/context.go` gained `LibSmartUnavailableReason`; shared `recordLibSmartBackendOutcome()` helper in `service/smart_service.go`; wired in `smart_service_lib.go` (success/failure) and `smart_service_nolib.go` (static-build reason); surfaced by `api/system.go`. OpenAPI regenerated (`openapi.json`), `openapi.yaml` surgically updated (generator churn on `X-Trace-Id`/`X-Span-Id` ordering avoided), frontend types regenerated (`sratApi.ts`, tsc clean).
- [x] Task 4: SRAT — backend unit tests (testify/suite + fxtest + mockio per `backend_test.instructions.md`): lib-load failure path reports reason, capability flag truthiness, variant-selection helper prefers musl on musl-linker systems. **Done:** internal `package service` tests (`smart_service_internal_test.go`, `upgrade_service_internal_test.go`) cover `recordLibSmartBackendOutcome` (nil-ctx, clear-stale, record reason), nolib `initSmartClient` outcome recording, and variant selection via injectable linker probe (`detectBestServerVariantWithLinkerCheck`, 6 cases) + real-wrapper probe. Full `mise run //backend:test` green (exit 0); per-function coverage: helper 100%, nolib init 100%, variant wrapper 100%, linker-check core 95.5%.
- [ ] Task 5: Addon — install `libsmartmon_go.so` from the SDK tarball (already downloaded in the Dockerfile; no C++ toolchain needed) to `/usr/local/lib/libsmartmon_go.so`
- [ ] Task 6: Addon — execute the smartlib variant: update `srat/run` (or Dockerfile symlink) so Alpine boots `srat-server-musl` instead of the static symlink; keep static as documented fallback
- [ ] Task 7: Addon + SRAT — end-to-end verification on HAOS: boot, assert `lib_smart_available: true` via `/api/capabilities`, assert `SMART lib backend loaded` in addon log, and confirm disk list/health/self-test parity with exec mode
- [ ] Task 8: Documentation — update `docs/SMART_SERVICE.md` (addon build modes, `-dynamic` requirement, wrapper install), addon `DOCS.md`/`CHANGELOG.md`, and `CHANGELOG.md` under `## [ 🚧 Unreleased ]` (per `/update-changelog` skill)
- [ ] Task 9: Capture the lessons learned and update documentation
- [ ] Task 10: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)

### Root-cause chain (verified)

1. **Release binaries default to exec-only.** `backend/.mise.toml` builds the default variant with `CGO_ENABLED=0` and tags `p...,embedallowed` (no `smartlib`); only `--zig` (musl) and `--cgo` (glibc) add `smartlib` (lines 83, 96). The `srat-server` symlink in the release zip points to `srat-server-static` (line 114).
2. **The addon boots the static variant.** `sambanas2/Dockerfile` unzips `srat_${arch}.zip` to `/usr/local/bin/` and `rootfs/etc/s6-overlay/s6-rc.d/srat/run` executes `/usr/local/bin/srat-server` → static → `LibSmartAvailable` stays `false`.
3. **Even a smartlib variant cannot load in the current image.** `smart_service_lib.go:19` calls `libbackend.New(...)`; the lib backend `purego.Dlopen`s `libsmartmon_go.so` (`lib.go:172`) resolving via `defaultLibNames`/`defaultLibPaths` (`lib.go:37-50`). The addon installs only `libsmartmon.a` + headers (static SDK) — the C++ wrapper `smartmon_c_api.cpp` is never compiled, so no `.so` exists → `resolveLibPath()` error → `smart_service_lib.go:31` logs `falling back to exec backend` and returns the exec client. This fallback is **silent from the user's perspective** (only visible as `lib_smart_available: false` in capabilities and a low-key log line).
4. **The upgrade service already knows the right variant.** `upgrade_service.go:921-933` prefers `srat-server-musl` when `/lib/ld-musl-*.so.1` exists (Alpine case), falling back to glib/static — but this only applies on binary updates via the upgrade dir, not fresh installs, and the wrapper `.so` is missing anyway.

### Motivations (why it is gated this way)

- **Static portability is a hard release requirement**: default production builds are fully static (`CGO_ENABLED=0`) and must run on both glibc and musl; the build task enforces this with a static-linkage check (`.mise.toml` lines 117–135).
- The lib backend forces **dynamic loading** (`dlopen`, `libdl.so.2`), incompatible with the static default → gated behind the `smartlib` build tag.
- musl dynamic builds additionally require zig `-dynamic` (static musl cannot `dlopen` — "Dynamic loading not supported") and `-fno-sanitize=all` (UBSan link errors), noted in `.mise.toml` lines 74–81.

### Fix design

- **Wrapper library (Option A — decided)**: `smartmontools-sdk` builds `libsmartmon_go.{so,dylib}` in its own release pipeline (smartmontools-sdk#14). The SDK build matrix already has a `cxx` for every target (gcc/g++ for glibc, `zig c++` for musl, osxcross `clang++` for darwin), so the workflow compiles `smartmon_c_api.cpp` fetched from a **pinned** smartmontools-go ref (currently `v0.4.1`) against its own freshly-built `libsmartmon.a`, records the ref in a `smartmontools-go-version` file inside the tarball, and skips `windows-amd64` (lib.go is `//go:build linux || darwin`). The old `scripts/setup-lib-backend.sh` consumer-side script was removed from smartmontools-go (`18b27ff`); its compile contract is now reproduced in the SDK workflow. Using the SDK's autoconf-generated `smartmon_config.h` at build time also removes the old consumer-side substitute-header workaround. Tarball extracts to `/usr/local` → `/usr/local/lib/libsmartmon_go.so` = first `defaultLibPaths` entry (lib.go:44). SRAT's release zips cannot carry the unsigned `.so` (they only zip signed binaries), which is why the wrapper ships via the SDK tarball, not the SRAT zip.
- **Variant selection**: on Alpine (musl), boot `srat-server-musl`. Keep the static symlink as the universal fallback. Reuse/duplicate the `upgrade_service.go` selection logic in the addon run script or select the binary at Dockerfile install time.
- **Observability**: surface the fallback reason (`smart_service_lib.go:31` error string) via `SystemCapabilities` so the UI/general panel can show *why* direct mode is off (e.g. "libsmartmon_go.so not found").
- **ABI contract**: smartmontools-go#38 documents that the `csrc` C ABI is stable across tagged releases; SRAT CI (Task 2) asserts the SDK tarball's `smartmontools-go-version` matches SRAT's vendored version so a future symbol change can never silently fall back to exec.
- **Out of scope**: changing the static-build portability policy; building the wrapper inside the addon Dockerfile (needs a C++ toolchain; superseded by Option A).

### Test plan

- SRAT backend: mock lib-load failure → assert `LibSmartAvailable=false` + reason surfaced; variant-selection unit tests for musl/glib/static preference on different linkers. Follow `backend_test.instructions.md` (external `_test` package, testify/suite, fxtest, mockio v2, cancel context before WaitGroup).
- Addon: build the image with the new steps, boot, then assert `/api/capabilities.lib_smart_available == true` and the `SMART lib backend loaded` log line; run disk list + SMART health on the test host; confirm no addon `ERROR` logs.

## 🔗 Code References

- `backend/src/service/smart_service_lib.go` — smartlib-gated init + exec fallback (line 31)
- `backend/src/service/smart_service_nolib.go` — exec-only fallback compile path
- `backend/src/service/upgrade_service.go:921-957` — runtime variant selection (musl/glib/static)
- `backend/.mise.toml` lines 72–114 — `smartlib` tag via `--zig`/`--cgo`, musl `-dynamic` requirement
- `.github/workflows/build.yaml` lines 561–643 — release zip assembly (contains all three variants + static symlink)
- `vendor/github.com/dianlight/smartmontools-sdk/bindings/go/v8/backends/lib/lib.go` — purego `dlopen`, `defaultLibPaths` (37–50), `resolveLibPath` (472–485), wrapper build contract (5–15; wrapper now ships prebuilt in SDK release tarballs)
- `backend/src/dto/system_capabilities.go` + `backend/src/api/system.go` — `lib_smart_available` capability surface (candidate for fallback-reason field)
- `frontend/src/pages/settings/panels/GeneralPanel.tsx:231` — UI mention of `-tags smartlib` / `libsmartmon_go.so`
- `docs/SMART_SERVICE.md` — exec vs lib build modes documentation
- `hassio-addons/sambanas2/Dockerfile` — smartmontools-sdk install (static `libsmartmon.a` + headers), SRAT unzip + `/usr/local/bin/srat-server` check
- `hassio-addons/sambanas2/rootfs/etc/s6-overlay/s6-rc.d/srat/run` — starts `/usr/local/bin/srat-server` (static symlink)
- `CHANGELOG.md` — motivation for smartlib gating (static portability)
