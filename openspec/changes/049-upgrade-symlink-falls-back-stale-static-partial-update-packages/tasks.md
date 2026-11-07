# Tasks

## 1. Implementation

- [ ] 1.1 Reproduce — unit test where `updatePkg` contains only `srat-cli`, target dir has newer musl + older static on a musl-linker system; assert current code returns `srat-server-static` (bug) — verified by targeted tests
- [ ] 1.2 Fix `detectBestServerVariantWithLinkerCheck` to fall back to the newest on-disk variant (compare embedded versions) instead of unconditionally returning static when the package has no server variants — verified by targeted tests
- [ ] 1.3 Run `mise run //backend:test` (full suite green) + per-function coverage ≥70% for touched logic — verified by targeted tests
- [ ] 1.4 Update `docs/SMART_SERVICE.md` remote-deploy notes if behavior changes user-visible flow — verified by targeted tests

## 2. Verification

- [ ] 2.1 Run the relevant test suites and confirm the change passes — verified by a green test run
