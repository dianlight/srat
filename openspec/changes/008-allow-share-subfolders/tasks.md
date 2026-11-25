# Tasks

## 1. Data Model and Persistence

- [x] 1.1 Add optional `Subfolder` field to `dto.SharedResource` (pointer) and `dbom.ExportedShare`, plus goose migration 00020 defaulting to empty — verified via migration tests and model round-trip tests
- [x] 1.2 Regenerate converters (fixing goverter annotations stale since #1567) — verified via converter tests

## 2. Validation and Path Resolution

- [x] 2.1 Validate subfolder values: reject empty segments, absolute paths, parent traversal, backslashes and control characters — verified via `dto` table tests and service tests
- [x] 2.2 Resolve the share directory segment by segment, refuse symlinks leaving the volume, create missing folders only on mounted volumes with the volume root's mode and owner — verified via `share_directory_test.go`
- [x] 2.3 Reject two active shares on the same folder (409) and invalidate shares whose folder disappeared — verified via service and API tests
- [x] 2.4 Mount events update every share of a volume; the volume reports its root share — verified via service, repository and disk map tests

## 3. Samba, NFS, Lab Gating and Frontend

- [x] 3.1 Emit the resolved path in `smb.conf`, NFS exports and HA NFS mounts — verified via config stream, NFS exports and supervisor tests
- [x] 3.2 Register the `share_subfolders` beta lab feature and gate new or changed subfolders in the share API (403) — verified via API handler tests
- [x] 3.3 Add the lab-gated "Subfolder" input, allow shared volumes when subfolders are available, show the shared folder in share details — verified via component tests

## 4. Tests, API Surface, and Documentation

- [x] 4.1 Backend unit, service and API handler tests; every touched function at or above 70% coverage — verified via `mise run //backend:test`
- [x] 4.2 Frontend tests for the subfolder input and helpers — verified via vitest
- [x] 4.3 Update OpenAPI spec and frontend types — verified via type generation and type checking
- [x] 4.4 Document the feature (`docs/SHARE_SUBFOLDERS.md`, settings lab tiers, README, CHANGELOG)
