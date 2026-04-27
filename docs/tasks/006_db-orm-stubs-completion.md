# [REFACTOR]: Database and ORM Stubs Completion

**Target Repo:** `srat`  
**Status:** 📅 Planned  
**Issue Link:** [hassio-addons#573](https://github.com/dianlight/hassio-addons/issues/573)

## 🎯 Objective

Complete three deferred database/ORM items that were left as stubs or no-ops: implement `FindByPath` and `FindByDevice` on `MountPointPathQuery`; expose `ExportedShareToSharedResource` through `DtoToDbomConverterInterface`; and resolve the import-cycle that left migration `00009` as a commented-out no-op so default property values are seeded correctly at startup.

> _Context for Copilot: All three items are catalogued in `docs/FUTURE_IMPROVEMENTS.md`. The GORM query helpers are generated (`dbom/query/`) and the converter uses `goverter`. Migration 00009 has a body that was wrapped in `/* */` to break an import cycle between `dto`, `converter`, and `dbom` packages._

## 🛠️ Technical Specifications

- **Inputs:**
  - Mount point path string or device path string (for query methods)
  - `dbom.ExportedShare` source and `*dto.SharedResource` target (for converter)
  - Default property values from `dto` / `AppConfig` (for migration seeding)

- **Outputs:**
  - `FindByPath(path string) (dbom.MountPointPath, error)` — single result or `ErrNotFound`
  - `FindByDevice(device string) ([]*dbom.MountPointPath, error)` — slice, empty if none
  - `ExportedShareToSharedResource` available via `DtoToDbomConverterInterface`
  - Migration 00009 seeds the `properties` table with default settings on first run after upgrade

- **Dependencies:**
  - `backend/src/dbom/query/mount_point_path_query.go` — query interface and implementation
  - `backend/src/converter/dto_to_dbom_conv.go` — `DtoToDbomConverterInterface`
  - `backend/src/converter/dto_to_dbom_conv_gen.go` — goverter-generated implementation
  - `backend/src/dbom/migrations/00009_write_properties_from_default.go` — no-op migration
  - `backend/src/dto/` and `backend/src/dbom/` packages

## 📝 Task List

- [ ] Task 1: Uncomment `FindByPath` and `FindByDevice` in `MountPointPathQuery` interface and implement them using GORM scopes (`.Where("path = ?", path)` / `.Where("device = ?", device)`)
- [ ] Task 2: Add tests for `FindByPath` (found, not found) and `FindByDevice` (empty, multiple results)
- [ ] Task 3: Uncomment `ExportedShareToSharedResource` in `DtoToDbomConverterInterface` and verify the goverter-generated mapping is complete and correct
- [ ] Task 4: Add or update converter tests for `ExportedShareToSharedResource`
- [ ] Task 5: Resolve migration 00009 import cycle — move default-seeding logic to application startup (`appsetup.go` or a dedicated `SeedDefaults` service called after DB migration), instead of running it inside the migration itself
- [ ] Task 6: Re-enable (or replace) the `Up00009` migration body once the import cycle is broken; ensure it is idempotent (only seeds rows that don't already exist)
- [ ] Task 7: Integration test — run migrations against an in-memory SQLite DB and verify the `properties` table is seeded correctly after `00009`
- [ ] Task 8: Documentation — update `docs/FUTURE_IMPROVEMENTS.md` to remove completed items
- [ ] Task 9: Fix share soft-delete so that re-creating a share with the same name succeeds — ensure the `delete` operation physically removes or marks the DB record in a way that allows `CreateShare` to insert a new row without hitting the unique-key constraint (see [hassio-addons#573](https://github.com/dianlight/hassio-addons/issues/573))

## 🧠 Implementation Notes (Copilot Context)

### FindByPath / FindByDevice

These follow the pattern of other generated query helpers in `dbom/query/`:

```go
func (q *mountPointPathQuery) FindByPath(path string) (dbom.MountPointPath, error) {
    var result dbom.MountPointPath
    err := q.db.Where("path = ?", path).First(&result).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return result, dto.ErrorNotFound
    }
    return result, err
}

func (q *mountPointPathQuery) FindByDevice(device string) ([]*dbom.MountPointPath, error) {
    var results []*dbom.MountPointPath
    err := q.db.Where("device = ?", device).Find(&results).Error
    return results, err
}
```

### Import cycle fix for migration 00009

Option A (recommended): Move seeding to an `fx.Hook` in `appsetup.go` that runs after `AutoMigrate`:
```go
fx.Invoke(func(db *gorm.DB, converter ConverterInterface, ...) {
    SeedDefaultProperties(db, converter, ...)
})
```
Option B: Duplicate the minimal default values as raw SQL constants inside the migration, avoiding the `dto`/`converter` import.
Option A is cleaner and keeps defaults close to the application startup logic.

### ExportedShareToSharedResource

- The goverter-generated implementation in `dto_to_dbom_conv_gen.go` should already exist.
- Verify field mappings are complete; add `//goverter:map` annotations if any field requires custom transformation.
- Add the method signature to `DtoToDbomConverterInterface` and regenerate if needed: `mise run //backend:gen`.

## 🔗 Code References & TODOs

- [ ] `backend/src/dbom/query/mount_point_path_query.go` — `FindByPath`, `FindByDevice` commented-out stubs
- [ ] `backend/src/converter/dto_to_dbom_conv.go` — `ExportedShareToSharedResource` commented out of interface
- [ ] `backend/src/converter/dto_to_dbom_conv_gen.go` — generated implementation to verify
- [ ] `backend/src/dbom/migrations/00009_write_properties_from_default.go` — `Up00009` body wrapped in `/* */`
- [ ] `docs/FUTURE_IMPROVEMENTS.md` — sections "MountPointPath Query", "Converter", "Migration 00009" (remove once done)
- [ ] [hassio-addons#573](https://github.com/dianlight/hassio-addons/issues/573) — `failed to save share 'TIMEMACHINE' to repository: duplicated key not allowed` after deleting and recreating a share with the same name
