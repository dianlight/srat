<!-- DOCTOC SKIP -->

# Backend Future Improvements

This document lists identified refactoring opportunities and unfinished work in the backend
that require interface changes or significant restructuring and were therefore deferred.

## HDIdle Service: Missing Global-Status and Delete-Config Endpoints

**Location:** `backend/src/api/hdidle_handler.go`, `backend/src/service/hdidle_service.go`

**Status:** Handler implementations exist but are not registered. The methods they rely on
(`GetStatus`, `GetEffectiveConfig`) are absent from `HDIdleServiceInterface`.

**Endpoints to restore:**

| Method | Path | Handler |
| ------ | ---- | ------- |
| GET | `/hdidle/status` | `getServiceStatus` |
| GET | `/hdidle/effective-config` | `getEffectiveConfig` |
| DELETE | `/disk/{disk_id}/hdidle/config` | `deleteConfig` |

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

**Status:** The `MountPointPathQuery` interface declares `FindByPath` and `FindByDevice` as
commented-out stubs.

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

## Volume Service: EjectDisk Feature Not Implemented

**Location:** `backend/src/api/volumes.go`, `backend/src/service/volume_service.go`

**Status:** The `EjectDiskHandler` method exists in `VolumeHandler` but is not registered as a
route. `EjectDisk` was also omitted from `VolumeServiceInterface`.

**Required interface change:**

```go
// Add to VolumeServiceInterface in service/volume_service.go
EjectDisk(diskID string) error
```

**Work to do:**
1. Implement `EjectDisk` in `VolumeService`.
2. Add `EjectDisk` to `VolumeServiceInterface`.
3. Register the route in `RegisterVolumeHandlers`.
4. Add tests for the new endpoint.

---

## DirtyDataService Removal from Share and Volume Handlers

**Location:** `backend/src/api/shares.go`, `backend/src/api/volumes.go`

**Status:** `DirtyDataServiceInterface` was removed from the `ShareHandler` and
`VolumeHandler` constructors. The dirty-state tracking for shares and volumes is now handled
internally by the respective services (`ShareService`, `VolumeService`) via event bus
subscriptions rather than direct handler injection.

**Verification needed:** Confirm that all dirty-state updates that were previously triggered
from the handlers are now correctly emitted by the services.

---

## Large Service Files — Splitting Candidates

The following files exceed 800 lines and are candidates for decomposition:

| File | Lines | Suggestion |
| ---- | ----- | ---------- |
| `service/volume_service.go` | ~1 241 | Extract mount/unmount logic into a `MountManager` helper |
| `service/filesystem_service.go` | ~944 | Extract async operation runner into a separate struct |
| `service/hdidle_service.go` | ~823 | Extract disk-state tracking into a separate type |

Splitting these files requires no interface changes if the public `*Interface` types are
preserved; internal helper types and functions can be moved freely.
