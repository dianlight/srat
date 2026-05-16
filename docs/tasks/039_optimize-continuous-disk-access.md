# [REFACTOR]: Optimize Continuous Disk Access

**Target Repo:** `srat`  
**Status:** ✅ Complete  
**Issue Link:** https://github.com/dianlight/srat/issues/636

## 🎯 Objective
Optimize backend services and handlers to reduce continuous and redundant disk access patterns. We have identified several issues causing high I/O, such as polling ticks checking file hashes, ticker-based databases queries, continuous file writings, and inefficient caching flushes. The goal is to offload these to memory, use file watchers reliably, adjust polling intervals to lower impact, and batch DB operations.

## 🛠️ Technical Specifications
- **Inputs:** Current backend service components (AddonConfigWatcher, DiskStatsService, HDIdleService, NetworkStatsService, HealthHandler)
- **Outputs:** Optimized backend code that reduces disk read/writes per minute.
- **Dependencies:** `fsnotify` for reliable file watching, in-memory caching mechanisms.

## 📝 Task List
- [x] Task 1: Refactor `AddonConfigWatcherService` to rely primarily on `fsnotify` and avoid the 60-second polling ticker that computes SHA256 hashes of the file, or change the logic to use file modification timestamps.
- [x] Task 2: Optimize `HDIdleService` to avoid continuous database saves on each state update loop iteration by caching state in memory and saving only on changes.
- [x] **Task 3:** Adjust `HealthHandler` intervals and remove expensive I/O operations (e.g. `sambaService.GetSambaStatus()`) from the default high-frequency loop if not strictly required, caching the results instead.
- [x] **Task 4:** Improve `DiskStatsService` and `NetworkStatsService` to cache results effectively and not stress disk or db reads on 10-second intervals.
- [x] Task 5: Investigate excessive context logging calls (`slog.*Context()`) and ensure logging is properly level-gated and not excessively flushing.
- [x] Task 6: Analyze and optimize the logic for managed drives and partitions scanning, ensuring disk/partition metadata isn't unnecessarily re-read.
- [x] Task 7: Unit testing to ensure logic remains intact with the new optimizations
- [x] Task 8: Integration and documentation
- [x] Task 9: Code review and cleanup
- [x] Task 10: Final testing and validation
- [x] Task 11: Capture the lessons learned and update documentation
- [x] Task 12: Ask to create a PR with the task implementation and link it here for tracking → [PR #637](https://github.com/dianlight/srat/pull/637)

## 🧠 Implementation Notes (Copilot Context)
**Agreed Pre-Implementation Plan:**
- **Addon Config Watcher:** Replace 60s SHA256 polling with `mtime` check or `fsnotify`. (Completed)
- **HDIdle Service:** Add in-memory cache for device states, avoiding DB writes unless state changes. (Completed)
- **Health Handler & Samba:** Cache shell invokes (`smbstatus`) instead of hitting them every 5s tick.
- **Disk/Network Stats:** Fast-path cache for setting queries, and throttle heavy `smartctl` / `Statfs` calls.
- **Logging:** Reduce cyclic context logging noise/flush rate.

- **AddonConfigWatcherService**: `watchViaTicker()` reads `/data/options.json` every 60s and calculates a hash. Remove this or cache the `mtime` to avoid full reads.
- **HDIdleService**: Loop checks `hdidle` status and immediately writes `s.db.Save(&dbDevice).Error` to SQLite. Add an in-memory cache and only save if state actually changes.
- **HealthHandler**: Currently fetches disk stats, network stats, server processes, and samba status repeatedly. The default polling limit `OutputEventsInterleave` is just 5 seconds.
  - `SambaStatus` invokes `smbstatus -j` executing to shell. It uses a 1-minute `cache.DefaultExpiration` block, but once expired it hits disk immediately via shell subprocess on the next 5s health loop.
- **Disk Stats / Network / Managed Drives**:
  - `updateDiskStats()` continuously runs every 10 seconds.
  - Repeatedly fetching DB config via `settingService.Load()` in a ticker. Cache these settings.
  - **Partitions Scan (`syscall.Statfs`)**: Iterates over all discovered partitions and executes `syscall.Statfs` inside `disk_stats_service.go`, which probes VFS mount points.
  - **Filesystem States (`getFilesystemState`)**: Invokes filesystem adapter checks using shell commands (tune2fs, btrfs, etc.). Although cached in `base_adapter.go` with a 30m TTL, ensure any system-level probes or clearing (`invalidateCommandResultCache()`) via `Flush()` aren't forcing disk wake-ups unnecessarily.
  - **SMART Health Probing (`GetSmartStatus`)**: Executes `smartctl` periodically in the disk scan loop (`updateDiskStats()`). This is a very heavy operation and likely spinning up sleeping disks. We must heavily cache this check (e.g., limit execution unless explicitly requested or significantly throttle polling) since the health loop hits it every 10s.

## 🔗 Code References & TODOs
- `backend/src/service/addon_config_watcher_service.go`
- `backend/src/service/disk_stats_service.go`
- `backend/src/service/network_stats_service.go`
- `backend/src/service/hdidle_service.go`
- `backend/src/api/health.go`
