# Tasks

## 1. Restore canary install method

- [ ] 1.1 Restore `.mise.toml` GitHub-backend canary settings and verify `mise install` installs `bun@canary` with working `bun lint --help` — verified by local `mise install` and `bun lint --help`
- [ ] 1.2 Restore canary comments in `bunfig.toml`, `hk.pkl`, `frontend/package.json`, `oauth_broker/package.json`, and CI workflow env comments — verified by `git diff` review

## 2. Detect stable lint availability

- [ ] 2.1 Define the stable-version check as `bun lint --help` succeeding on Linux amd64, Linux arm64 (Docker/HA OS target), and macOS — verified by a documented check in `design.md`
- [ ] 2.2 Record the first stable Bun version that passes the check in the change notes — verified by a version reference in `design.md` or change metadata

## 3. Migrate to pinned stable Bun

- [ ] 3.1 Pin `bun` in `.mise.toml` to the verified stable version and remove `[tool_alias]`, `use_versions_host = false`, and `fetch_remote_versions_cache = "5m"` — verified by `mise install` on macOS and Linux
- [ ] 3.2 Set `frontend/package.json` and `oauth_broker/package.json` `packageManager` to `bun@<stable-version>` — verified by `mise run //frontend:prepare` and `bun --version`
- [ ] 3.3 Update `bunfig.toml` and `hk.pkl` comments to state `bun lint` is stable — verified by documentation diff review
- [ ] 3.4 Restore generic GitHub-token comments in `.github/workflows/build.yaml` and `.github/workflows/documentation.yml` — verified by workflow diff review

## 4. Verification

- [ ] 4.1 Run `mise install`, `mise run //backend:test`, `mise run //frontend:test:ci`, and `hk check` — verified by green test runs
- [ ] 4.2 Verify `bun lint --help` and `mise run //frontend:prepare` inside the Docker/HA OS build environment for amd64 and arm64 — verified by successful platform builds
- [ ] 4.3 Run `openspec validate pin-bun-stable-when-lint-ships --strict` — verified by clean validation output

## Workflow follow-up

- Archive the change after stable-bun migration is merged and CI is green on all target platforms.
