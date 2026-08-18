# 📊 SRAT Task Status Report
_Generated: 2026-08-18_

## Summary
| Status | Count | Progress |
|--------|-------|----------|
| ✅ Complete | 3 | 3/22 tasks |
| 🔄 In Progress | 3 | 3/22 tasks |
| 📅 Planned | 16 | 16/22 tasks |
| **Total** | **22** | **27%** |

---

## ✅ - Done

### [043] Zeroconf mDNS Registration from Addon (Lab)
- **Type:** FEATURE | **Issues:** None
- **Progress:** 11 / 15 tasks ✓

### [045] Full Mobile Support
- **Type:** FEATURE | **Issues:** [srat#922](https://github.com/dianlight/srat/pull/922)
- **Progress:** 12 / 12 tasks ✓

### [048] Volume Stack Hardening — Code Review Findings
- **Type:** REFACTOR | **Issues:** None
- **Progress:** 25 / 25 tasks ✓

---

## 🔄 - In Progress

### [044] Support Disks Without Partitions
- **Type:** FIX | **Issues:** [dianlight/srat#849](https://github.com/dianlight/srat/issues/849), [dianlight/hassio-addons#716](https://github.com/dianlight/hassio-addons/issues/716), [PR #867](https://github.com/dianlight/srat/pull/867)
- **Progress:** 16 / 17 tasks (94%)
- **Next:** Task 16: Manual validation on HAOS with a physical USB prepared per `docs/replicate-partitionless-disk-macos.md` (Scenar

### [046] Enable SMART Lib Backend in Addon
- **Type:** FIX | **Issues:** [dianlight/hassio-addons#726](https://github.com/dianlight/hassio-addons/issues/726), [dianlight/smartmontools-sdk#14](https://github.com/dianlight/smartmontools-sdk/issues/14), [dianlight/smartmontools-go#38](https://github.com/dianlight/smartmontools-go/issues/38)
- **Progress:** 3 / 10 tasks (30%)
- **Next:** Task 1: SDK (smartmontools-sdk#14) — add `libsmartmon_go.{so,dylib}` build step to `.github/workflows/build.yml` per mat

### [047] Migrate SMART backend to smartmontools-sdk bindings/go/v8
- **Type:** REFACTOR | **Issues:** [dianlight/smartmontools-sdk#13](https://github.com/dianlight/smartmontools-sdk/issues/13), [dianlight/smartmontools-go#38](https://github.com/dianlight/smartmontools-go/issues/38)
- **Progress:** 15 / 17 tasks (88%)
- **Next:** Task 10: Capture lessons learned and ask to create a PR

---

## 📅 - Planned

### [004] Security Hardening — CORS, IP Allowlist, Ingress Session Validation
- **Type:** FIX | **Issues:** None
- **Progress:** 0 / 12 tasks

### [006] Database and ORM Stubs Completion
- **Type:** REFACTOR | **Issues:** [hassio-addons#573](https://github.com/dianlight/hassio-addons/issues/573)
- **Progress:** 0 / 15 tasks

### [007] Backend Code Quality — errors.AsType Migration and Service Splits
- **Type:** REFACTOR | **Issues:** None
- **Progress:** 2 / 13 tasks

### [008] Allow Share Subfolders
- **Type:** FEATURE | **Issues:** [srat#184](https://github.com/dianlight/srat/issues/184)
- **Progress:** 0 / 16 tasks

### [020] Missing SambaNAS2 Addon Detection
- **Type:** FEATURE | **Issues:** None
- **Progress:** 0 / 15 tasks

### [021] Samba Service Health Monitoring
- **Type:** FEATURE | **Issues:** None
- **Progress:** 0 / 15 tasks

### [022] Disk Health Degradation Alerts
- **Type:** FEATURE | **Issues:** None
- **Progress:** 0 / 15 tasks

### [023] HA SRAT Connectivity Loss Detection
- **Type:** FEATURE | **Issues:** None
- **Progress:** 0 / 15 tasks

### [029] WebSocket Origin Validation and pprof Route Isolation
- **Type:** FIX | **Issues:** None
- **Progress:** 0 / 9 tasks

### [030] commandexec Snapshot Memory Leak and Busy-Wait Elimination
- **Type:** FIX | **Issues:** None
- **Progress:** 0 / 10 tasks

### [031] Production Logging Safety — Body Logging and Secret Sanitization
- **Type:** FIX | **Issues:** None
- **Progress:** 0 / 8 tasks

### [032] WebSocket Reconnect Resilience and Frontend Safety Guards
- **Type:** FIX | **Issues:** None
- **Progress:** 0 / 15 tasks

### [033] Database Recovery Safety, HTTP Request Size Limits, and Goroutine Leak
- **Type:** FIX | **Issues:** None
- **Progress:** 0 / 13 tasks

### [035] Upgrade & HDIdle Path Traversal, Timer Race, and Data Race
- **Type:** FIX | **Issues:** None
- **Progress:** 0 / 13 tasks

### [036] Frontend Performance — NavBar Lazy Loading and Metrics Rendering
- **Type:** REFACTOR | **Issues:** None
- **Progress:** 0 / 15 tasks

### [037] Frontend Data Correctness — isLoading Bug, Hook Rules, Password Exposure
- **Type:** FIX | **Issues:** None
- **Progress:** 1 / 17 tasks
