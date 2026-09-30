<!-- DOCTOC SKIP -->

---

description: "addon image fresh-install verification procedure for the SRAT dev HA environment"
applyTo: "`docs/test/**/*.md`"

---

# Remote Addon Testing Instructions

**📖 See also**: `.opencode/skills/test-remote-environment/SKILL.md` (full remote test procedure) — this instruction covers the addon-image fresh-install variant.

## When to use

Use this procedure when a fix lives in the addon Dockerfile/rootfs (not the srat binary) and must be verified the way a real user receives it: **uninstall + reinstall**. A factory reset is NOT enough; dev-binary deploys (upgrade dir) do NOT exercise the addon image.

## Key facts (learned 2026-09-30, srat#1246)

- **Supervisor always pulls the image from the registry on install** — preloading with `docker save | ssh docker load` does NOT work (manifest fetch 404s, then pull fails). The tag must exist in GHCR.
- **Dev HA host is amd64 (x86_64)** — build with `docker buildx build --platform linux/amd64 --build-arg BUILD_ARCH=amd64` (SRAT downloads `srat_x86_64.zip`; arm64 builds fail with `no matching manifest for linux/amd64`).
- **GHCR auth**: `echo $(gh auth token) | docker login ghcr.io -u dianlight --password-stdin` (gh CLI has `write:packages` + `delete:packages`).
- **Install a locally built image**: build → `docker push ghcr.io/dianlight/addon-sambanas2:<unique-tag>` (e.g. `2026.9.1-iss1246`) → bump `version:` in host `/addons/hassio-addons/sambanas2/config.yaml` → `docker restart hassio_supervisor` (plain `ha supervisor reload` does NOT rescan the local repo) → uninstall → install.
- **Uninstall does NOT wipe `/addon_configs/<slug>`** — move it aside (`mv /addon_configs/local_sambanas2 /addon_configs/local_sambanas2.old-<tag>`) for a truly fresh state; supervisor only removes its own `/data/apps/data/<slug>`.
- **Fresh install runs the RELEASED srat binary** (SRAT_VERSION build arg in the image), not dev builds — binary-level fixes are invisible until a release is pinned.
- **`EnableHaDiscovery` defaults to `false`** (`dto/settings.go`) — default fresh installs never register supervisor discovery, so discovery symptoms (e.g. the unregister 404) cannot reproduce without enabling `enable_ha_discovery` first.
- **Dockerfile build gotcha**: base `ghcr.io/hassio-addons/base:21.0.5` pins `libcrypto3/libssl3=3.5.8` in world while Alpine serves `openssl-3.5.9` → `apk add openssl` fails; workaround `apk add -u libcrypto3 libssl3` before adding packages.
- **Cleanup**: delete the test tag after the run: `gh api -X DELETE /user/packages/container/addon-sambanas2/versions/<version-id>` (list first: `gh api /user/packages/container/addon-sambanas2/versions --jq '.[] | {id, name}'`).
