<!-- DOCTOC SKIP -->

# [REFACTOR]: Migrate SMART backend to smartmontools-sdk bindings/go/v8

**Target Repo:** `srat`
**Status:** 🔄 In Progress
**Issue Link:** [dianlight/smartmontools-sdk#13](https://github.com/dianlight/smartmontools-sdk/issues/13) · [dianlight/smartmontools-go#38](https://github.com/dianlight/smartmontools-go/issues/38)

## 🎯 Objective

Replace the retired `github.com/dianlight/smartmontools-go` v0.4.1 wrapper with the official Go bindings module `github.com/dianlight/smartmontools-sdk/bindings/go/v8` v8.0.0 (shipped inside the smartmontools-sdk repo, which also releases the prebuilt `libsmartmon_go.so` wrapper). Update all imports, re-vendor, regenerate goverter converters, and rewrite the `verify-smartlib-wrapper-abi` CI job to the new contract: release tag must equal the vendored bindings ref, tarball must contain `libsmartmon_go.so`, and `nm -D` must expose the `smartmon_abi_version` symbol.

## 🛠️ Technical Specifications

- **Inputs:** `backend/src/go.mod` (`github.com/dianlight/smartmontools-go` → `github.com/dianlight/smartmontools-sdk/bindings/go/v8 v8.0.0`); vendored module tree; `.github/workflows/build.yaml` `verify-smartlib-wrapper-abi` job.
- **Outputs:** All imports point at the new module; vendor tree re-vendored; goverter regenerated (`smartmontools_to_dto_conv_gen.go`); CI job asserts tag == vendored ref + `.so` presence + `smartmon_abi_version` symbol via `nm -D`; docs updated (SMART_SERVICE.md, go.instructions.md, task 046).
- **Dependencies:** `dianlight/smartmontools-sdk` release tarballs (musl variant `linux-amd64-musl.tar.gz` containing `lib/libsmartmon_go.so`); Go 1.26; `mise run //backend:patch|gen|test`.

## 📝 Task List

- [x] Task 1: Swap `go.mod` dependency to `smartmontools-sdk/bindings/go/v8 v8.0.0` and update all Go imports (6 files)
- [x] Task 2: Update code comments referencing `smartmontools-go` (smart_service.go, 8 sites)
- [x] Task 3: `go mod tidy` + `go mod vendor` (module tree re-vendored, old wrapper removed)
- [x] Task 4: Re-apply vendor patches (`mise run //backend:patch`) after re-vendor
- [x] Task 5: Regenerate goverter converters (`mise run //backend:gen`) — smartmontools regen OK; note pre-existing unrelated `SambaUserToUser`/`HasDefaultPassword` goverter failure at HEAD (dto_to_dbom.go:136), not caused by this migration
- [x] Task 6: Rewrite `verify-smartlib-wrapper-abi` CI job (tag == vendored ref, `.so` + `smartmon_abi_version` via `nm -D`, hard-fail)
- [x] Task 7: Validate — targeted gotest (converter + service, `embedallowed_no smartlib` tags) and full `mise run //backend:test` suite (all green, 66s)
- [x] Task 8: Update documentation (SMART_SERVICE.md, go.instructions.md Service Ownership Rules, task 046 notes)
- [x] Task 9: Coverage gate check for touched functions — change is import/comment-only (no new logic); `initSmartClient` 60% is a pre-existing exempt thin lib adapter, untouched
- [ ] Task 10: Capture lessons learned and ask to create a PR

## 🧠 Implementation Notes (Copilot Context)

- **Module layout**: new module is `github.com/dianlight/smartmontools-sdk/bindings/go/v8` with subpackages `backends/compare|exec|lib|types` — import path shape matches the old `smartmontools-go` so the swap is mechanical (old `types` package → `.../bindings/go/v8/types`).
- **Vendor + patch ordering**: `mise run //backend:patch` is cache-keyed on `src/vendor/**/*.pdone` outputs; after `go mod vendor` rewrites the tree the markers vanish, so the patch task re-runs and regenerates the Darwin stubs (`mount_darwin.go`) and applies `hd-idle-check-power-mode.patch` + `gorm-generics.patch`.
- **Workdir convention**: mise tasks assume `backend/` workdir (`go -C src ...`); run `go mod tidy`/`vendor` from `backend/`, never from repo root (fails with `chdir src: no such file or directory`).
- **CI contract (new)**: query `https://api.github.com/repos/dianlight/smartmontools-sdk/releases/tags/${VENDORED_REF}`; hard-fail if `tag_name != VENDORED_REF`; find `libsmartmon_go.so` in the musl tarball; `nm -D` must grep `smartmon_abi_version` (bindings verify abiMajor=1, abiMinor>=0 at load time). Old job compared a `smartmontools-go-version` ref file from `releases/latest` (warn-only).
- **Pre-existing unrelated failure**: `dto.User.HasDefaultPassword` (readOnly) has no goverter mapping at `dto_to_dbom.go:136` — `//backend:gen` fails at HEAD before this migration; the committed gen file is stale but compiles, so it does not block the build or tests. Fix (add `goverter:ignore`) belongs to a separate task.
- **GOMODCACHE** is project-local: `/Users/ltarantino/Documents/Sources/srat/backend/vendor/mod/sdk`; GOPROXY `https://goproxy.io.direct,https://proxy.golang.org,direct`.
- **Test command**: `go -C src tool gotest -p 1 -failfast -timeout 120s -tags embedallowed_no ./converter/... ./service/...`; full suite via `mise run //backend:test`.

## 🔗 Code References & TODOs

- [x] `backend/src/go.mod` line 24 — new dependency
- [x] `backend/src/service/smart_service*.go` — import + comment swaps
- [x] `backend/src/converter/smartmontools_to_dto*.go` — import swap + regen
- [x] `.github/workflows/build.yaml` — `verify-smartlib-wrapper-abi` job
- [x] `.opencode/instructions/go.instructions.md` — "SmartService is the sole smartmontools bindings contact point"
- [x] `docs/SMART_SERVICE.md` — runtime requirement + CI contract sections
- [ ] `dto_to_dbom.go:136` `SambaUserToUser` `HasDefaultPassword` goverter gap (pre-existing, separate task)
