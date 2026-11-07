# Proposal

## Why

Complete three deferred database/ORM items that were left as stubs or no-ops: implement `FindByPath` and `FindByDevice` on `MountPointPathQuery`; expose `ExportedShareToSharedResource` through `DtoToDbomConverterInterface`; and resolve the import-cycle that left migration `00009` as a commented-out no-op so default property values are seeded correctly at startup.

## What Changes

- Uncomment `FindByPath` and `FindByDevice` in `MountPointPathQuery` interface and implement them using GORM scopes (`.Where("path = ?", path)` / `.Where("device = ?", device)`)
- Add tests for `FindByPath` (found, not found) and `FindByDevice` (empty, multiple results)
- Uncomment `ExportedShareToSharedResource` in `DtoToDbomConverterInterface` and verify the goverter-generated mapping is complete and correct
- Add or update converter tests for `ExportedShareToSharedResource`
- Resolve migration 00009 import cycle — move default-seeding logic to application startup (`appsetup.go` or a dedicated `SeedDefaults` service called after DB migration), instead of running it inside the migration itself
- Re-enable (or replace) the `Up00009` migration body once the import cycle is broken; ensure it is idempotent (only seeds rows that don't already exist)
- Integration test — run migrations against an in-memory SQLite DB and verify the `properties` table is seeded correctly after `00009`
- Documentation — update `docs/FUTURE_IMPROVEMENTS.md` to remove completed items
- Fix share soft-delete so that re-creating a share with the same name succeeds — ensure the `delete` operation physically removes or marks the DB record in a way that allows `CreateShare` to insert a new row without hitting the unique-key constraint (see [hassio-addons#573](https://github.com/dianlight/hassio-addons/issues/573))

## Capabilities

### New Capabilities

- `database-orm-stubs-completion`: Database and ORM Stubs Completion

### Modified Capabilities

(none)

## Impact

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
- Legacy task: `docs/tasks/006_db-orm-stubs-completion.md`
