# Phase-Scripted Fakes for Flaky-Upstream Reconciliation Tests

**Verified**: 2026-08-23 · `backend/src/service/volume_service_reconcile_test.go`

## Rule

To unit-test reconciliation logic that consumes a flaky upstream source, script the fake provider
with **sequential phases** instead of single-shot mocks:

- Phase 0: healthy data (e.g. drive returns real partitions sdb1 + sdb2)
- Phase 1: degraded snapshot (e.g. synthesized whole-disk entry)

Then assert the protected invariant survives each transition (real partitions never replaced by
synthesized ones).

## Reference implementation

`fakeReconcileHardware` in `volume_service_reconcile_test.go` returns phase-scripted responses;
helpers `reconcileDisk`, `wholeDiskPartition`, `realPartition`, `newReconcileVolumeService` build
fixtures. Pattern:

```go
fake := newFakeReconcileHardware(
    phases{realPartitions("sdb", "sdb1", "sdb2")},
    phases{synthesizedWholeDisk("sdb")},
)
// run getVolumesData twice; after phase 1 the cache must still hold sdb1+sdb2,
// and the synthesized "sdb" partition must never appear.
```

## Why

Single-shot mocks cannot express "upstream was good, then went bad" — exactly the scenario where
cache-protection guards must fire. Phase scripting makes the transition deterministic and the
assertion meaningful.
