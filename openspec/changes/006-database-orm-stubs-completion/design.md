# Design

## Context

Complete three deferred database/ORM items that were left as stubs or no-ops: implement `FindByPath` and `FindByDevice` on `MountPointPathQuery`; expose `ExportedShareToSharedResource` through `DtoToDbomConverterInterface`; and resolve the import-cycle that left migration `00009` as a commented-out no-op so default property values are seeded correctly at startup.

## Goals / Non-Goals

**Goals:**

- Deliver: Database and ORM Stubs Completion

**Non-Goals:**

- Unrelated refactoring outside the task scope.

## Decisions

- Uncomment `FindByPath` and `FindByDevice` in `MountPointPathQuery` interface and implement them using GORM scopes (`.Where("path = ?", path)` / `.Where("device = ?", device)`)
- Add tests for `FindByPath` (found, not found) and `FindByDevice` (empty, multiple results)
- Uncomment `ExportedShareToSharedResource` in `DtoToDbomConverterInterface` and verify the goverter-generated mapping is complete and correct
- Add or update converter tests for `ExportedShareToSharedResource`
- Resolve migration 00009 import cycle — move default-seeding logic to application startup (`appsetup.go` or a dedicated `SeedDefaults` service called after DB migration), instead of running it inside the migration itself
- Re-enable (or replace) the `Up00009` migration body once the import cycle is broken; ensure it is idempotent (only seeds rows that don't already exist)
- Integration test — run migrations against an in-memory SQLite DB and verify the `properties` table is seeded correctly after `00009`
- Documentation — update `docs/FUTURE_IMPROVEMENTS.md` to remove completed items

## Risks / Trade-offs

- Behaviour changes are covered by targeted tests before handoff.
- Legacy task content remains the reference until the change is archived.
