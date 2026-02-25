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

## Large Service Files — Splitting Candidates

`service/volume_service.go` mount/unmount logic has been extracted to
`service/volume_mount_manager.go` (`volumeMountManager` type). The remaining
candidates are:

| File | Lines | Suggestion |
| ---- | ----- | ---------- |
| `service/filesystem_service.go` | ~944 | Extract async operation runner into a separate struct |
| `service/hdidle_service.go` | ~823 | Extract disk-state tracking into a separate type |

Splitting these files requires no interface changes if the public `*Interface` types are
preserved; internal helper types and functions can be moved freely.

---

## Security and Stability Findings

### Context Key Type Safety (Stability)

**Location:** `cmd/srat-cli/main-cli.go`, `cmd/srat-server/main-server.go`,
`events/events.go`, `server/ha_middleware.go`, and every test `SetupTest`.

**Issue:** All `context.WithValue` calls use untyped string literals as keys (`"wg"`,
`"user_id"`, `"event_uuid"`). The Go specification explicitly warns that using
built-in types as context keys can cause collisions between packages, and the
type assertion `ctx.Value("wg").(*sync.WaitGroup)` will panic if the key is
absent (returns `nil`).

**Fix:**

```go
// internal/ctxkeys/keys.go
package ctxkeys

type contextKey string

const (
    WaitGroup  contextKey = "wg"
    UserID     contextKey = "user_id"
    EventUUID  contextKey = "event_uuid"
)
```

Replace all `context.WithValue(ctx, "wg", ...)` with `context.WithValue(ctx, ctxkeys.WaitGroup, ...)`.
Add nil-guard before the type assertion:
```go
wg, _ := ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup)
if wg != nil {
    wg.Go(...)
}
```

---

### Hardcoded IP Allowlist (Security)

**Location:** `backend/src/server/ha_middleware.go:73`

**Issue:** The HA middleware only allows requests from `172.30.32.2` and `127.0.0.1`.
These addresses are specific to the default HA Supervisor network. Other Supervisor
network configurations (e.g., custom Docker networks) will be blocked, and the list
cannot be updated without recompiling.

**Fix:** Read the allowed IPs from the `ContextState` (populated from addon config) or
from an environment variable, falling back to the current defaults.

---

### CORS Wildcard with Credentials (Security)

**Location:** `backend/src/server/http_server.go:45-52`

**Issue:** The CORS configuration uses `AllowOriginFunc: func(origin string) bool { return true }`
combined with `AllowCredentials: true`. According to the CORS specification, a server
**must not** respond with a wildcard origin when the request includes credentials.
While this combination works in many browsers due to pre-flight checks, it is
semantically incorrect and may be exploitable in certain configurations.

**Fix:** When running in addon mode (`ContextState.AddonMode`), restrict `AllowedOrigins`
to the HA ingress origin. In development mode, the permissive policy is acceptable.

---

### Commented-out Ingress Session Validation (Security)

**Location:** `backend/src/server/ha_middleware.go:28-66`

**Issue:** The ingress session cookie validation against the Supervisor API is entirely
commented out. The middleware currently trusts any request from the allowed IP range
with a valid `X-Remote-User-Id` header, relying solely on network-level isolation.
If the network trust boundary is compromised (e.g., another container on the same
Docker network), unauthenticated access is possible.

**Fix:** Re-enable the ingress session validation using `ingressClient`. The commented-out
implementation already had caching via `gocache` to avoid per-request overhead.

---

### Unguarded `context.Value` Type Assertions (Stability)

**Location:** `api/health.go:79`, `api/upgrade.go:109`, `service/volume_service.go:169`,
`service/upgrade_service.go:94`, `service/upgrade_service.go:103`,
`service/filesystem_service.go:254`

**Issue:** Multiple call sites use the pattern:
```go
ctx.Value("wg").(*sync.WaitGroup).Go(...)
```
If the context does not contain the `"wg"` key the `Value` call returns `nil`, and the
subsequent type assertion panics at runtime.

**Fix:** Combine with the context key type safety fix above, and add nil-guards:
```go
if wg, ok := ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup); ok && wg != nil {
    wg.Go(...)
}
```
