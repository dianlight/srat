# Tasks

## 1. Implementation

- [ ] 1.1 Uncomment `FindByPath` and `FindByDevice` in `MountPointPathQuery` interface and implement them using GORM scopes (`.Where("path = ?", path)` / `.Where("device = ?", device)`) — verified by targeted tests
- [ ] 1.2 Add tests for `FindByPath` (found, not found) and `FindByDevice` (empty, multiple results) — verified by targeted tests
- [ ] 1.3 Uncomment `ExportedShareToSharedResource` in `DtoToDbomConverterInterface` and verify the goverter-generated mapping is complete and correct — verified by targeted tests
- [ ] 1.4 Add or update converter tests for `ExportedShareToSharedResource` — verified by targeted tests
- [ ] 1.5 Resolve migration 00009 import cycle — move default-seeding logic to application startup (`appsetup.go` or a dedicated `SeedDefaults` service called after DB migration), instead of running it inside the migration itself — verified by targeted tests
- [ ] 1.6 Re-enable (or replace) the `Up00009` migration body once the import cycle is broken; ensure it is idempotent (only seeds rows that don't already exist) — verified by targeted tests
- [ ] 1.7 Integration test — run migrations against an in-memory SQLite DB and verify the `properties` table is seeded correctly after `00009` — verified by targeted tests
- [ ] 1.8 Documentation — update `docs/FUTURE_IMPROVEMENTS.md` to remove completed items — verified by targeted tests
- [ ] 1.9 Fix share soft-delete so that re-creating a share with the same name succeeds — ensure the `delete` operation physically removes or marks the DB record in a way that allows `CreateShare` to insert a new row without hitting the unique-key constraint (see [hassio-addons#573](https://github.com/dianlight/hassio-addons/issues/573)) — verified by targeted tests

## 2. Verification

- [ ] 2.1 Run the relevant test suites and confirm the change passes — verified by a green test run
