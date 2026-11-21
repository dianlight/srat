# Design

## Approach

The fixes live entirely in the `dianlight/hassio-addons` repository; SRAT code is unchanged. Work is split into two deliveries against `prerelease/sambanas2_2026.11.0`:

1. Merge the already-open release PR #817, which carries the idempotent `/mnt` mount, the `srat` → `init-nfsd` service dependency, and the single-dash flag detection from `mergerelease/sambanas2`.
2. Add the remaining two fixes in PR #822:
   - Extend the `is_semver_valid` regex with an optional fourth numeric component so `YYYY.MM.DD.PATCH` is accepted; `sort -V` already compares such versions correctly.
   - Remove `/tmp/.bashio` in the Dockerfile after `COPY rootfs /`, and call `bashio::cache.flush_all` (falling back to deleting `*.cache` files when the helper is unavailable) at the top of each s6 script that reads ingress port or options before its first `bashio::` read.

## Alternatives considered

- Patching only `devrelease/sambanas2` and waiting for the next full release: rejected because affected users are on the current stable channel.
- Directly committing to the stable prerelease branch: rejected in favour of reviewable PRs so the release automation and CI remain authoritative.

## Risks

- Cache flushing forces fresh Supervisor API calls at startup, adding a small, bounded delay.
- The 4-part regex must not accept malformed strings; negative cases remain covered by the existing tests.
- The integration upgrade test cannot create mock ELF binaries on macOS; this failure is pre-existing and reproduced on the unmodified base branch.

## Verification

- `test-binary-upgrade.sh`: 22 passing, including new 4-part validation/comparison cases.
- `test-bashio-cache.sh`: 5 passing static checks.
- `test-srat-args-flags.sh`: 10 passing. `test-zfs-support-output.sh`: 3 passing.
- `bash -n` on all modified scripts; shellcheck shows no new findings.
