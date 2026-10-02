# [FEATURE]: Allow Share Subfolders

**Target Repo:** `srat`  
**Status:** 🔄 In Progress  
**Issue Link:** [srat#184](https://github.com/dianlight/srat/issues/184)

## 🎯 Objective

Allow users to configure a Samba share that exposes a specific subdirectory of a mounted volume rather than its root. Currently every share maps directly to the mount-point root (`/mnt/<device>`). The goal is to add an optional `subfolder` (or `path`) field to the share configuration so that `smb.conf` uses `path = /mnt/<device>/<subfolder>` instead.

> _Context for Copilot: Each share is stored as a `dbom.ExportedShare` and rendered into Samba config via `backend/src/templates/smb.gtpl`. The `dto.SharedResource` struct is the API surface. Frontend share creation/edit form lives in `frontend/src/pages/shares/`._

## 🛠️ Technical Specifications

- **Inputs:**
  - Share create/update request body — new optional field `subfolder string` (relative path segment, e.g. `"movies"` or `"photos/2024"`)
  - Mounted volume root path (resolved from `dto.DiskMap`)

- **Outputs:**
  - Samba `path =` directive uses `<mount_root>/<subfolder>` when `subfolder` is non-empty
  - Subfolder is created on the filesystem if it does not exist (with appropriate permissions)
  - API rejects paths with `..` traversal attempts (security: directory traversal prevention)
  - Share listing returns the resolved `path` for display in the UI

- **Dependencies:**
  - `backend/src/dto/share.go` (or equivalent DTO file) — `SharedResource` struct
  - `backend/src/dbom/` — `ExportedShare` model and migration
  - `backend/src/service/share_service.go` — `CreateShare`, `UpdateShare`
  - `backend/src/templates/smb.gtpl` — Samba share template
  - `frontend/src/pages/shares/` — share creation/edit form

## 📝 Task List

- [x] Task 1: Add `Subfolder string` (optional) field to `dto.SharedResource`, `dbom.ExportedShare`, and `config.Share`; update generated converters
- [x] Task 2: Validate `Subfolder` in `share_service.go`: reject absolute paths and any path containing `..`
- [x] Task 3: In `share_service.go`, create subfolder directory on disk; resolve subfolder path in `jSONFromDatabase()` for template rendering
- [x] Task 4: Samba template unaffected — subfolder resolved into `config.Share.Path` before template runs, so `path = {{ .data.path }}` already emits the correct resolved path
- [x] Task 5: Add optional "Subfolder" `TextFieldElement` to `ShareEditForm.tsx` with helper text, shown for non-internal shares
- [ ] Task 6: Unit tests — `share_service.go`: path resolution, directory creation, traversal rejection
- [ ] Task 7: API handler tests — valid subfolder, missing subfolder (defaults to root), traversal attempt returns 422
- [ ] Task 8: Frontend component test — subfolder input renders, submits correct value
- [ ] Task 9: Update OpenAPI spec and regenerate frontend types (`cd frontend && bun run gen`)
- [ ] Task 10: Documentation — update share configuration docs with `subfolder` field description

## 🧠 Implementation Notes (Copilot Context)

### Path validation (security)

```go
// In API validation or share_service.go
func validateSubfolder(subfolder string) error {
    if subfolder == "" {
        return nil // empty means mount root — OK
    }
    if filepath.IsAbs(subfolder) {
        return fmt.Errorf("subfolder must be a relative path")
    }
    cleaned := filepath.Clean(subfolder)
    if strings.HasPrefix(cleaned, "..") {
        return fmt.Errorf("subfolder must not traverse above mount root")
    }
    return nil
}
```

### Samba template change

```ini
# smb.gtpl — before
path = {{ .MountPath }}

# smb.gtpl — after
path = {{ .ResolvedPath }}
```

Where `ResolvedPath = MountPath` when `Subfolder` is empty, else `filepath.Join(MountPath, Subfolder)`.

### DB migration

Add a new migration `000NN_add_share_subfolder.go` using `pressly/goose`:

```go
func Up(db *sql.DB) error {
    _, err := db.Exec(`ALTER TABLE exported_shares ADD COLUMN subfolder TEXT NOT NULL DEFAULT ''`)
    return err
}
```

## 🔗 Code References & TODOs

- [ ] [srat#184](https://github.com/dianlight/srat/issues/184) — "Allow share subfolders" — feature request
- [ ] `backend/src/dto/` — `SharedResource` struct (add `Subfolder` field)
- [ ] `backend/src/dbom/` — `ExportedShare` model (add `Subfolder` column + migration)
- [ ] `backend/src/service/share_service.go` — path resolution logic
- [ ] `backend/src/templates/smb.gtpl` — `path =` directive
- [ ] `frontend/src/pages/shares/` — share form component
