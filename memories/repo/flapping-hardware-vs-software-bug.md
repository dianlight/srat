# Flapping Hardware vs Software Bug Distinction

**Verified**: 2026-08-23 · sdb USB disk hardware fault investigation

## Rule

On physically unstable devices (USB disks that connect/disconnect every few seconds),
**partition-count variance across polls (0 → 1 → 2 partitions cycling) is correct mirroring
behavior**, not a bug. SRAT should reflect whatever state the hardware is in at query time.

## Bug criterion

A software bug is indicated only when a **synthesized/degraded entry REPLACES real cached data**
(e.g. "Raw disk (no partition table)" appearing where sdb1/sdb2 were previously shown) — see
`memories/repo/cache-protection-degraded-upstream-snapshots.md`.

## Practical guidance

- Do NOT file bugs for hardware-state variance; do not chase display "stability" when the
  underlying device is genuinely flapping.
- Confirm physical flapping via udev log bursts:
  `Processing partition addition/removal event action=add|remove devname=/dev/sdbN` cycling.
- Verification pass criteria for cache-protection fixes: across N polls,
  `synth=True` count == expected (only for genuinely partition-less disks like sdd), and real
  partitions reappear whenever the upstream snapshot contains them.
