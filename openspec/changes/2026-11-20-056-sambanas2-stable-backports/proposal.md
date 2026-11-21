# Proposal

## Why

The 2026.11.0 stable add-on branch missed fixes that had already shipped in 2026.10.2 and were pending on `devrelease/sambanas2`. Four upstream issues are affected:

- [hassio-addons#815](https://github.com/dianlight/hassio-addons/issues/815) / [srat#1500](https://github.com/dianlight/srat/issues/1500) / [srat#1605](https://github.com/dianlight/srat/issues/1605): mounting `/root/nfs_root.img` at `/mnt` aborts startup when a USB volume already owns the mount point. The idempotent fix shipped in 2026.10.2 but was absent from the 2026.11.0 stable branch.
- [hassio-addons#820](https://github.com/dianlight/hassio-addons/issues/820) / [srat#1611](https://github.com/dianlight/srat/issues/1611): the `srat` startup script probes `--auto-update` / `--ipv4-only`, but Go's `flag` package prints single-dash forms, so the flags were silently dropped.
- [hassio-addons#819](https://github.com/dianlight/hassio-addons/issues/819) / [srat#1612](https://github.com/dianlight/srat/issues/1612): `is_semver_valid` rejects the 4-part `YYYY.MM.DD.PATCH` version scheme, so staged SRAT binary upgrades never apply.
- [hassio-addons#818](https://github.com/dianlight/hassio-addons/issues/818) / [srat#1613](https://github.com/dianlight/srat/issues/1613): stale `/tmp/.bashio/*.cache` files baked into the image override live Supervisor ingress port and options.

## What Changes

- Merge the existing release PR that ports the idempotent `/mnt` mount, the `init-nfsd` service dependency, and the single-dash flag fix into `prerelease/sambanas2_2026.11.0`.
- Accept an optional 4th numeric component in `is_semver_valid`, with unit coverage for 4-part versions.
- Remove `/tmp/.bashio` at image build time and flush stale bashio caches at startup in `srat/run`, `socat/run`, `modprobe/run`, and `check-srat-update/run`.
- Add a static regression check for the cache guards and document both fixes.
- Track all four upstream issues here so the mirrored SRAT issues can be closed per their close rule.

## Capabilities

### New Capabilities

- `sambanas2-stable-backports`: tracking of upstream add-on fixes backported to the 2026.11.0 stable branch.

### Modified Capabilities

(none)

## Impact

- `dianlight/hassio-addons` — `sambanas2/Dockerfile`, four s6 `run` scripts, upgrade/flag/cache tests, and documentation. No SRAT back-end, front-end, or custom-component code changes.
- SRAT issues #1500, #1605, #1611, #1612, #1613 are linked here; #1500 and #1605 are closed once the stable backport lands.
