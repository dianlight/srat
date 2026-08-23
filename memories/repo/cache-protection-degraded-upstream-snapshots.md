# Cache Protection Against Degraded Upstream Snapshots

**Verified**: 2026-08-23 · rc16/rc17 fix chain · `backend/src/service/volume_service.go`

## Rule

When a service caches enriched data fed by a flaky upstream source (Supervisor API, udev events),
never let a degraded/fallback snapshot overwrite richer cached state.

Before calling the cache-update method (e.g. `dbom.DiskMap.AddOrUpdate`), guard:

```go
if currentDisk != nil && updateDisk &&
    self.isWholeDiskSynthesized(&disk) && !self.isWholeDiskSynthesized(currentDisk) {
    // incoming snapshot is a synthesized fallback; cached entry holds real data
    currentDisk.RefreshVersion = refreshVersion // keep eviction loop from dropping it
    continue
}
```

## Why

The Supervisor can momentarily return a drive without filesystem data; the hardware-service
fallback probe then synthesizes a whole-disk entry (partition named after the disk itself).
Without the guard, real partitions (sdb1/sdb2) are silently replaced by a synthesized
"Raw disk (no partition table)" entry on every hardware flap.

## Reference implementation

- Guard: `getVolumesData()` (~lines 739–756) in `volume_service.go`
- Degraded-form detector: `isWholeDiskSynthesized` (`volume_service.go:809`) — true iff disk has
  exactly 1 partition whose `LegacyDeviceName` equals the disk's own name
- Regression test: `TestGetVolumesData_SynthesizedDowngradeProtected` in
  `volume_service_reconcile_test.go`

## Known limitation

The guard protects only "real → synthesized whole-disk". Other degraded forms
(real → empty partitions, real → partial partitions) still overwrite when upstream genuinely
reports less data — correct behavior for mirroring live state.
