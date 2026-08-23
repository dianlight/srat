# Hot-Swap Deploy Path for Fast Remote Addon Iteration

**Verified**: 2026-08-23 · remote HA at `192.168.0.68`, addon container `app_local_sambanas2`

## Rule

For single-binary verification cycles (faster than `mise run //backend:build:remote`):

```bash
# 1. Build (never raw go build in backend/src — mise generates metadata constants)
cd backend && mise run //backend:build --version rcN --arch x86_64
# binary lands in backend/dist/x86_64/srat-server-static

# 2. Hot-swap into the running addon container
scp backend/dist/x86_64/srat-server-static root@192.168.0.68:/tmp/
ssh root@192.168.0.68 "docker cp /tmp/srat-server-static app_local_sambanas2:/usr/local/bin/srat-server \
  && docker exec app_local_sambanas2 s6-svc -r /run/service/srat"
```

API is back up ~8s after restart.

## Gotchas

- **`srat-server --version` does NOT exist at runtime** — verify the embedded version from the
  mise build output only.
- musl/glib variant warnings from the mise build task are expected/benign.
- Verify with unauthenticated polling: `curl -s http://192.168.0.68:3000/api/volumes`
- Check logs post-restart: `ssh root@192.168.0.68 "ha addons logs local_sambanas2" | grep -iE 'error|panic'`
- Pre-existing benign WARN: hdidle_service.go "Error checking ATA device support"
  (SCSI INVALID FIELD IN CDB) on USB disks — not a regression.

## When to use build:remote instead

Use `mise run //backend:build:remote` (see `.agents/skills/test-remote-environment/SKILL.md`) when
the change also touches frontend assets or the custom component, since it rsyncs the full upgrade
package to `/addon_configs/local_sambanas2/upgrade/`.
