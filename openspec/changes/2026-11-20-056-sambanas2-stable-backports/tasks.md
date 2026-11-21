# Tasks

## 1. Upstream release backport (hassio-addons#815, hassio-addons#820)

- [x] 1.1 Merge release PR into `prerelease/sambanas2_2026.11.0` with the idempotent `/mnt` mount, `srat` → `init-nfsd` dependency, and single-dash flag fix (hassio-addons#815, #820)
- [ ] 1.2 Live-verify USB volume visibility and SRAT flags on a fresh 2026.11.0 stable install

## 2. Remaining fixes (hassio-addons#819, hassio-addons#818)

- [x] 2.1 Accept 4-part `YYYY.MM.DD.PATCH` in `is_semver_valid` and cover it in the upgrade unit/integration tests (hassio-addons#819)
- [x] 2.2 Remove `/tmp/.bashio` at image build time (hassio-addons#818)
- [x] 2.3 Flush stale bashio caches at startup in `srat/run`, `socat/run`, `modprobe/run`, `check-srat-update/run` (hassio-addons#818)
- [x] 2.4 Add a static regression check for the cache guards
- [x] 2.5 Update add-on README.md, DOCS.md, CHANGELOG.md and test README.md

## 3. Tracking and issue closure

- [x] 3.1 Record all four `hassio-addons#NNN` references in this change
- [x] 3.2 Close srat#1500 and srat#1605 with a comment pointing to the stable backport
- [x] 3.3 After the new stable image `2026.11.0.202` was released, close srat#1611, srat#1612 and srat#1613 with comments citing PRs #817/#822 (source-level verification: fix commits present on `prerelease/sambanas2_2026.11.0`)
