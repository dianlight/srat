<!-- DOCTOC SKIP -->

# [FEATURE]: Rclone Cloud Sync — Volume ↔ Cloud Links (Lab, Dropbox First)

**Target Repo:** `srat`
**Status:** ✅ Phase 1 complete (Tasks 1–12); ✅ Task 17 complete (rclonelib build variant); Tasks 13–16 tracked as follow-ups
**Issue Link:** https://github.com/dianlight/srat/issues/954

## 🎯 Objective

Implement a **Lab feature** (gated behind `experimental_lab_mode`) that lets the user **link/unlink a mounted volume to a cloud provider via rclone**, inspect the **link status and diff between local and remote**, and trigger **sync actions: Push (local→remote), Pull (remote→local), Bidirectional sync** — each optionally run as a **dry run** — with live progress over WebSocket.

- **First provider: Dropbox**, implemented through a **modular provider-driver architecture** so future providers (GDrive, S3, …) are additive (one driver file + registration), no core changes.
- The same "link/unlink" affordance will later extend to the **hassos-data partition** and to mounting clouds as filesystems / serving FTP-S3 (later phases of #954).
- Integration approach: **embed rclone as a Go library** (`github.com/rclone/rclone/librclone`), per maintainer decision — not as an external binary.

## 🛠️ Technical Specifications

- **Inputs:** Lab-mode flag, mounted volume (MountPointPath), provider credentials (Dropbox OAuth token), user-selected remote path, sync direction.
- **Outputs:** Persisted `RcloneLink` rows; REST endpoints for CRUD/link/auth/diff/sync/abort; `rclone_task` WebSocket progress events; Problems raised on sync failures; UI panels described below.
- **Dependencies:**
  - `github.com/rclone/rclone/librclone` (cgo; adds CGO requirement to musl/glibc addon builds — build workflow must be updated)
  - Existing: `internal/commandexec` (NOT used for rclone itself; kept for other tools), `events.EventBusInterface` + `BroadcasterService`, `ProblemService`, goose migrations, generated GORM helpers, goverter converters, huma v2 handlers, OpenAPI→RTK-Q codegen pipeline
  - Frontend: MUI, RTK Query (`sratApi.ts`), WS event map, existing `FilesystemCheckDialog` progress/terminal pattern, lab-gating pattern from `HomeAssistantPanel.tsx`

### Architecture (as shipped)

```
backend/src/service/rclone_service.go   // RcloneService: link CRUD, OAuth flows,
                                        // diff, async sync jobs (fx-wired in appsetup.go)
backend/src/service/rclone/
  driver.go                             // Driver interface + registry (RegisterDriver/GetDriver/ListDrivers)
  driver_dropbox.go                     // first provider: Dropbox OAuth2 (offline refresh tokens)
  rpc.go                                // RcloneRPC seam + CallRaw/Call helpers (test seam)
  rpc_librclone.go                      // librclone-backed RPC (build tag `rclonelib`, requires CGO)
  rpc_stub.go                           // no-op fallback for CGO_ENABLED=0 builds (reports unavailable)
```

The shipped layout intentionally deviates from the original draft: the service lives in the parent `service` package and providers are flat files in `service/rclone/` (no `provider/` subtree). Adding GDrive remains a one-file addition (`driver_gdrive.go` + `init()` registration).

`Driver` interface (each provider = ~1 small file):

```go
type Driver interface {
    Name() string        // rclone backend identifier, e.g. "dropbox"
    DisplayName() string // human-facing provider name
    ConfigFields() []ConfigField // what the wizard collects (name/label/secret/required)
    AuthStart(ctx context.Context, req AuthRequest) (string, error) // authorize URL
    ExchangeCode(ctx context.Context, req AuthRequest, code string) (*TokenResult, error)
}
```

There is deliberately no `AuthComplete`/`AuthStatus`/`Deauthorize` on the driver: status lives on the link row (`Status` field) and deauthorization is performed by `DeleteLink` via the `config/delete` RPC.

### Data model

- New table `rclone_links` (`dbom/rclone_link.go` + migration `00019_create_rclone_links.sql`):
  - Composite **primary key** `TargetKind+TargetID` (`volume|<mount path>` today, `hassos_data|hassos-data` reserved for Task 15) — no FK to `mount_point_path`
  - Plain-text `Provider` (registered driver name; deliberately **not** a goenums enum), `RemotePath`, `AutoSync bool`, `ScheduleMinutes int`
  - `Status` (unlinked|authorized|error), transient `OAuthState`, `LastSyncAt`, `LastSyncResult`, `LastSyncMessage`, soft delete (`deleted_at` index)
- `Property` rows for OAuth tokens are NOT used — tokens live in a managed `rclone.conf` under the addon data dir (written via the `config/create` RPC), path exported via settings.
- DTOs in `dto/rclone.go`: `RcloneLink`, `RcloneTask` (mirrors `FilesystemTask`), `RcloneDiffEntry`, `RcloneDiffResult`; DBOM↔DTO mapped exclusively by the generated goverter converter.

### Backend API (huma; every route lab-gated except the OAuth callback — 403 with `dto.ErrorLabModeRequired`)

- `GET /rclone/providers` — registered drivers (+ `library_available` flag from the RPC seam)
- `GET /rclone/links` — all configured links
- `GET|PUT|DELETE /rclone/link/{target_kind}/{target_id}` — link CRUD (GET answers 404 when absent, including the service's `(nil, nil)` not-found contract)
- `POST /rclone/link/{target_kind}/{target_id}/auth/start` — begin provider OAuth flow
- `POST /rclone/link/{target_kind}/{target_id}/diff` — compare local vs remote (`warning` set when the remote listing failed but a partial result was produced)
- `POST /rclone/link/{target_kind}/{target_id}/sync` `{direction: push|pull|bidi, dry_run}` + `POST .../abort` — mirrors CheckPartition/AbortCheckPartition semantics
- `GET /rclone/oauth/callback` — provider redirect target rendered as an auto-closing HTML page; **NOT lab-gated** because the provider redirect cannot carry settings headers; protected instead by the single-use, TTL-bounded OAuth state token
- OpenAPI regen: `mise run //backend:gen && mise run //frontend:gen` (never hand-edit `sratApi.ts`)

### Events & problems

- WS event `rclone_task` (`events.RcloneTaskEvent` wrapping `dto.RcloneTask`, enum `EVENTRCLONETASK` in `WebEventMap`): payload `{target_kind, target_id, operation, direction, status: start|running|success|failure, message, error, progress, notes[], result}`. Progress is polled from `core/stats` every 500ms and aggregated across bidi passes; **`999` means progress unsupported/indeterminate** (frontend falls back to an indeterminate bar). The schema is anchored into OpenAPI by the doc-stub handler `HandleRcloneEvents` (`api/system.go`, tags `system`,`internal`).
- Sync failures upserted into `ProblemService` (`rclone_sync_failed_<kind>_<sanitized-id>_<date>` problem key, severity warning); success dismisses; upsert failures are best-effort/non-fatal. Dry runs skip bookkeeping and never raise problems.

### Security/constraints

- Whole feature no-op unless `experimental_lab_mode=true` (403 on API; hidden in UI). The OAuth callback is exempt by design (see Backend API) and is protected by the single-use `oauthStateTTL`-bounded state token instead.
- Tokens never leave backend; never logged; command output sanitizer not applicable (no shell argv).
- CGO cross-build: librclone sits behind the `rclonelib` build tag so default static (CGO_ENABLED=0) builds keep working. The addon build workflow has been updated to include `rclonelib` in GOTAGS for both musl and glibc variants (Task 17).

## 📝 Task List

### Phase 1 — Core (Dropbox, volumes, manual sync)
- [x] Task 1: Add librclone dependency (vendored v1.75.0); introduce `rpc.go` seam interface with `rclonelib` build-tag implementations (lib + CGO-free stub)
- [x] Task 2: goose migration + `dbom.RcloneLink` + generated query helpers + goverter converters
- [x] Task 3: Provider Driver interface + registry + Dropbox driver (`AuthStart`/`ExchangeCode`, offline refresh tokens against managed rclone.conf)
- [x] Task 4: `RcloneService` wired in `appsetup.go`; shared lab-mode gate helper on every endpoint (callback exempt, state-token protected)
- [x] Task 5: Link CRUD endpoints + auth endpoints (+ openapi regen, frontend types)
- [x] Task 6: `Diff()` implementation returning typed entries (+ remote-failure warning)
- [x] Task 7: Sync engine (push/pull/bidi, dry-run, abort) with async job tracking + `rclone_task` WS events + ProblemService integration
- [x] Task 8: Backend tests (fake RPC seam): CRUD, driver registry, sync state machine, event emission — ≥70% coverage gate on touched functions
- [x] Task 9: Frontend — lab-gated "Cloud Sync" section in volume details (link/unlink wizard incl. OAuth popup + remote-path picker)
- [x] Task 10: Frontend — Cloud Sync panel: status card, diff table, Push/Pull/Sync confirm dialogs (with dry-run checkbox) with progress bar + terminal log + abort (FilesystemCheckDialog pattern)
- [x] Task 11: MSW handlers + RTL tests (user-event only) for panel, progress dialog and link wizard; `--rerun-each 10`
- [x] Task 12: CHANGELOG.md entry + task-doc finalization

### Phase 2 — Follow-ups (tracked, out of scope here unless pulled forward)
- [x] Task 13: Scheduled auto-sync worker (hdidle-style ticker) + conflict policy
- [ ] Task 14: Second provider driver (GDrive) proving modularity end-to-end
- [ ] Task 15: Extend linking to hassos-data partition target
- [ ] Task 16: `rclone mount` of clouds as filesystems + serve FTP/S3 (issue #954 remaining bullets)
- [x] Task 17: Update addon build workflow for the CGO/`rclonelib` variant (musl + glibc images) and verify images build

## 🖼️ UI Wireframe Proposal

### A. Volume details — unlinked (Lab mode ON)

```
┌─ Volumes › MyUSB-Disk ───────────────────────────────────────────┐
│  Tabs: [Overview] [Shares] [☁ Cloud Link]                        │
│ ┌──────────────────────────────────────────────────────────────┐ │
│ │ 🔬 CLOUD LINK                                    [ⓘ Lab]     │ │
│ │ This volume is not linked to a cloud provider.               │ │
│ │                                                              │ │
│ │        [ Link to cloud… ]                                    │ │
│ └──────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

### B. Link wizard (modal, 3 steps)

```
┌─ Link “MyUSB-Disk” to cloud ──────────────────── ✕ ─┐
│ ① Provider                                          │
│    ┌──────────────┐  ┌──────────────┐                │
│    │ ▣ Dropbox    │  │   GDrive     │  (soon)        │
│    └──────────────┘  └──────────────┘                │
│ ② Account                                           │
│    Status: ● Not connected     [ Connect Dropbox ]  │
│    → opens provider auth page, returns automatically│
│    ✓ Connected as l.tarantino@… (expires in 3h)     │
│ ③ Remote folder                                     │
│    /srat/MyUSB-Disk            [ Browse…]           │
│    ☑ Create if missing                              │
│                          [ Cancel ]  [ Create link ]│
└─────────────────────────────────────────────────────┘
```

### C. Linked volume — Cloud Sync tab (the new panel)

```
┌─ Volumes › MyUSB-Disk ───────────────────────────────────────────┐
│  Tabs: [Overview] [Shares] [☁ Cloud Sync]                        │
│ ┌─────────────────────┐ ┌──────────────────────────────────────┐ │
│ │ ☁ Dropbox           │ │ Last sync: 2h ago · success          │ │
│ │ /srat/MyUSB-Disk    │ │ State: ● Up to date                  │ │
│ │ ✓ l.tarantino@…     │ │ Auto-sync: Off   (Phase 2)           │ │
│ │        [ Unlink ]   │ │                                      │ │
│ └─────────────────────┘ └──────────────────────────────────────┘ │
│ ┌─ Differences (3) ────────────── [ Refresh diff ] [ Dry run ☐ ]┐│
│ │ LOCAL ONLY                    REMOTE ONLY       CHANGED       ││
│ │ photos/2026/ (1.2 GB)         backup.zip (88MB) notes.txt ±4KB ││
│ ├──────────────────────────────────────────────────────────────┤ │
│ │ [▲ Push Local→Remote] [▼ Pull Remote→Local] [⇄ Sync Both]    │ │
│ └──────────────────────────────────────────────────────────────┘ │
│   ↓ action opens confirm dialog → progress dialog:               │
│ ┌─ Pushing to Dropbox ──────────────────────────── [Abort] ─────┐│
│ │ ████████████░░░░░░░░░░  62% · 745MB/1.2GB · 12 files left     ││
│ │ ┌ terminal log (ReadonlyCommandTerminal-style) ──────────────┐││
│ │ │ INFO copied photos/2026/img_0042.jpg                       │││
│ └───────────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────────┘
```

Rules: every control hidden when `experimental_lab_mode=false` (`watch()` pattern + `ScienceOutlinedIcon`). Unlink asks confirmation ("remote files are NOT deleted"). Bidirectional warns about conflicts; Phase 2 adds policy picker (newer-wins default).

**Shipped deviations from these wireframes:** the Cloud Sync section renders **inline inside the volume details page** (`VolumeDetailsPanel.tsx`) rather than as a dedicated tab, and the wizard opens as a modal from it. The dry run ships as a checkbox inside each sync confirmation dialog (Phase 1) instead of a persistent toggle in the differences header.

## 🧠 Implementation Notes (Copilot Context)

- **Library usage**: all operations go through the `RcloneRPC` seam via JSON-RPC (`sync/sync`, `sync/copy` for bidi, `operations/lsjson`, `config/create`, `config/delete`, `core/stats`, `job/status`, `job/stop`). `librclone.Initialize()` is idempotent and invoked per transport call; `Finalize` runs on shutdown. Tests fake the seam, so coverage stays honest without network/cgo.
- **Dry runs**: `dryRun` adds rclone's `dryRun:true` to every pass and skips link-row bookkeeping + problem raising entirely (side-effect free by contract, enforced in tests).
- **No direct os/exec for rclone** — it is a library; `.opencode/instructions/backend-command-execution.instructions.md` applies to any residual binary invocations elsewhere only.
- **Tokens**: managed `rclone.conf` lives in addon data dir; `AuthComplete` writes token via config RPC; oauthutil auto-refreshes on subsequent calls. Never persist raw tokens in SQLite.
- **Async jobs**: mirror `FilesystemService.CheckPartition`: goroutine + atomic job handle for abort (`job/stop`), progress polled every 500ms → throttled `rclone_task` events on EventBus → BroadcasterService → wsApi (frontend reads via `useGetServerEventsQuery().data?.rclone_task`).
- **Frontend types**: all DTOs flow from openapi codegen; WS payload anchored with doc-stub handler in `backend/src/api/system.go` (tagged `"system","internal"`) only if no natural REST response anchors it.
- **Modularity acceptance test for Phase 2 Task 14**: adding GDrive must touch exactly one new file + one registry line + i18n/icon strings; nothing else.
- Build prerequisite: always mise tasks (`mise run //backend:test`, `mise run //frontend:test`), never raw go/vitest commands.

## 🔗 Code References

- RPC seam: `backend/src/service/rclone/rpc.go` (+ `rpc_librclone.go` / `rpc_stub.go` build-tag pair)
- Dropbox driver: `backend/src/service/rclone/driver_dropbox.go` (registry in `driver.go`)
- Service / handler / model: `backend/src/service/rclone_service.go`, `backend/src/api/rclone_handler.go`, `backend/src/dbom/rclone_link.go`
- Frontend: `frontend/src/pages/volumes/components/rclone/` (panel, wizard, progress dialog, guards)
- Pattern sources: `service/hdidle_service.go` (lab gate), `filesystem_service.go:CheckPartition` (async+abort+events), `pages/volumes/components/FilesystemCheckDialog.tsx` (progress UI), `043_zeroconf-mdns-addon-lab-feature.md` (lab checklist)

---

## Appendix — Draft comment for issue #954 (posted after plan approval)

> Detailed implementation plan created (docs/tasks/049_rclone-cloud-sync-lab-feature.md). Scope refined:
>
> - **Lab feature** gated by `experimental_lab_mode`; hidden/disabled otherwise.
> - **Link/Unlink mounted volumes to cloud providers**, Dropbox first, via modular `Driver` interface (`service/rclone/provider/*`) so new providers are one-file additions. Planned extensions: same linking for the **hassos-data partition**, plus mount-as-fs and FTP/S3 serving (kept as follow-up phases of this issue).
> - **Per-volume "Cloud Sync" panel**: link status (account, expiry), **local↔remote diff**, and actions **Push (L→R) / Pull (R→L) / Bidirectional**, each with dry-run, live progress over new `rclone_task` WebSocket events, and abort.
> - **Integration**: rclone embedded as Go library (librclone RPC: sync/copy·move·sync, lsjson, core/stats, job/stop) behind a testable seam; OAuth tokens held in managed rclone.conf (never in DB); failures surfaced as dashboard Problems.
> - Wireframes + phased checklist live in the task doc; Phase 1 = manual sync on volumes, Phase 2 = scheduler, GDrive, hassos-data target.
