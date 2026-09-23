<!-- DOCTOC SKIP -->

# Back-end Future Improvements

This document lists identified refactoring opportunities and unfinished work in the back-end
that require interface changes or significant restructuring and were therefore deferred.

## HDIdle Service: Missing Global-Status and Delete-Config Endpoints

**Location:** `backend/src/api/hdidle_handler.go`, `backend/src/service/hdidle_service.go`

**Status:** Handler implementations exist but are not registered. The methods they rely on
(`GetStatus`, `GetEffectiveConfig`) are absent from `HDIdleServiceInterface`.

**Endpoints to restore:**

| Method | Path                            | Handler              |
| ------ | ------------------------------- | -------------------- |
| GET    | `/hdidle/status`                | `getServiceStatus`   |
| GET    | `/hdidle/effective-config`      | `getEffectiveConfig` |
| DELETE | `/disk/{disk_id}/hdidle/config` | `deleteConfig`       |

**Required interface changes:**

```go
// Add to HDIdleServiceInterface in service/hdidle_service.go
GetStatus() (*HDIdleStatus, errors.E)
GetEffectiveConfig() HDIdleEffectiveConfig
```

**Work to do:**

1. Add `GetStatus` and `GetEffectiveConfig` to `HDIdleServiceInterface`.
2. Implement both methods in `HDIdleService`.
3. Re-register the three route handlers in `RegisterHDIdleHandler`.
4. Add tests for the new endpoints.

---

## MountPointPath Query: Unimplemented FindByPath and FindByDevice

**Location:** `backend/src/dbom/query/mount_point_path_query.go`

**Status:** Deferred post-1.0. The `MountPointPathQuery` interface currently
exposes `All()` only; path/device lookups are performed in service code.

**Required interface changes:**

```go
// Add to MountPointPathQuery interface
FindByPath(path string) (dbom.MountPointPath, error)
FindByDevice(device string) ([]*dbom.MountPointPath, error)
```

**Work to do:**

1. Uncomment the methods in the interface and implement them in the generated query helper.
2. Add tests covering both methods.

---

## Converter: ExportedShareToSharedResource Missing from Interface

**Location:** `backend/src/converter/dto_to_dbom_conv.go`

**Status:** `ExportedShareToSharedResource` is commented out in `DtoToDbomConverterInterface`.
A generated implementation exists in `dto_to_dbom_conv_gen.go` (via goverter), but it is not
exposed through the interface.

**Required interface change:**

```go
// Add to DtoToDbomConverterInterface
ExportedShareToSharedResource(source dbom.ExportedShare, target *dto.SharedResource) errors.E
```

**Work to do:**

1. Uncomment the method in `DtoToDbomConverterInterface`.
2. Verify the goverter-generated implementation is correct and has tests.

---

## Migration 00009: Body Left as No-op

**Location:** `backend/src/dbom/migrations/00009_write_properties_from_default.go`

**Status:** `Up00009` was written to seed the `properties` table with default settings values,
but the implementation is wrapped in a `/* ... */` comment block, leaving the migration as a
no-op. The migration depends on packages (`dto`, `converter`, `dbom`) that would create an
import cycle at migration time, which is why it was disabled.

**Work to do:**

1. Resolve the import-cycle issue (e.g., move default-seeding logic to application startup
   rather than a SQL migration).
2. Re-enable or replace the migration body.

---

## errors.As → errors.AsType Migration

**Location:** `backend/src/unixsamba/unixsamba.go`

**Status:** Four call sites use the pre-Go-1.26 `errors.As(err, &e)` pattern with the
`gitlab.com/tozd/go/errors` package. Go 1.26 introduced `errors.AsType[T](err)` in the
standard library, which is more concise and avoids pre-declaring a target variable.

**Note:** The `unixsamba` package imports `gitlab.com/tozd/go/errors`, not the standard
`"errors"` package. To use `errors.AsType` from the standard library, a separate import alias
would be required (e.g., `stderrors "errors"`), or the call sites can keep the current pattern.

**Optional improvement:**

```go
// Current
var e errors.E
if errors.As(err, &e) { ... }

// Alternative (requires stderrors "errors" import alias)
if e, ok := stderrors.AsType[tozderrors.E](err); ok { ... }
```

---

## Large Service Files — Splitting Candidates

`service/volume_service.go` mount/unmount logic has been extracted to
`service/volume_mount_manager.go` (`volumeMountManager` type). The remaining
candidates are:

| File                            | Lines | Suggestion                                            |
| ------------------------------- | ----- | ----------------------------------------------------- |
| `service/filesystem_service.go` | ~944  | Extract async operation runner into a separate struct |
| `service/hdidle_service.go`     | ~823  | Extract disk-state tracking into a separate type      |

Splitting these files requires no interface changes if the public `*Interface` types are
preserved; internal helper types and functions can be moved freely.
