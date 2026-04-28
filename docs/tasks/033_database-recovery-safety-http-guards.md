<!-- DOCTOC SKIP -->

# [FIX]: Database Recovery Safety, HTTP Request Size Limits, and Goroutine Leak

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** _None — discovered in reliability review 2026-04-28_

## 🎯 Objective

Fix three reliability issues that share the theme of defensive guards around infrastructure components:

1. **`replaceDatabase` infinite recursion** — `NewDB` and `replaceDatabase` can recurse indefinitely when the database path is persistently unwritable, causing a stack overflow.
2. **Silent nil return from `replaceDatabase`** — returns `nil *gorm.DB` when `os.Remove` fails, leading to opaque nil-pointer panics in all database-dependent services.
3. **HTTP request body unbounded** — no `http.MaxBytesReader` allows arbitrarily large request bodies to be buffered in memory.
4. **Goroutine leak in `ProcessWebSocketChannel`** — one goroutine per WebSocket connection is spawned with no termination mechanism.
5. **Unchecked nil-pointer assertion in `main-server.go`** — unguarded type assertion panics if the WaitGroup key is absent.

## 🛠️ Technical Specifications

- **Inputs:** `dbom/db_config.go`, `server/http_server.go`, `api/ws.go`, `cmd/srat-server/main-server.go`
- **Outputs:** Predictable fatal errors instead of recursive panics; bounded request memory; goroutine lifecycle tied to connection lifetime
- **Dependencies:** Go standard `http`, `sync`, `context` packages

## 📝 Task List

- [ ] Task 1: Add a `recovered bool` parameter to `replaceDatabase` (or an internal `newDBWithDepth(depth int)` variant); if `depth > 1`, call `tlog.Fatal` instead of recursing
- [ ] Task 2: Replace `return nil` in `replaceDatabase` (when `os.Remove` fails) with `tlog.Fatal("cannot remove corrupt database file: %v", removeErr)`
- [ ] Task 3: Add an HTTP middleware in `NewMuxRouter` that wraps `r.Body` with `http.MaxBytesReader(w, r.Body, 1<<20)` (1 MiB); return HTTP 413 on overflow; document the constant as `maxRequestBodySize`
- [ ] Task 4: Modify `broadcaster.ProcessWebSocketChannel` to accept a `context.Context` parameter; return when the context is done
- [ ] Task 5: In `HandleWebSocket` (`api/ws.go`), derive a connection-scoped context from the request context; pass it to `ProcessWebSocketChannel`; ensure the goroutine returns on connection close
- [ ] Task 6: Apply the safe `if wg, ok := apiCtx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup); ok && wg != nil` guard in `main-server.go:288`
- [ ] Task 7: Add tests: (a) verify `NewDB` with a permanently unwritable path does not recurse more than once; (b) verify a 2 MiB request body returns 413; (c) verify `ProcessWebSocketChannel` goroutine exits when the connection context is cancelled
- [ ] Task 8: Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-REL-02, B-REL-03, B-REL-04, B-REL-05, B-PERF-04 resolved

## 🧠 Implementation Notes

```go
// db_config.go — recursion guard
func replaceDatabase(lc fx.Lifecycle, v ...) *gorm.DB {
    return replaceDatabaseDepth(lc, v, 0)
}

func replaceDatabaseDepth(lc fx.Lifecycle, v ..., depth int) *gorm.DB {
    if depth > 0 {
        tlog.Fatal("Cannot recover database after second attempt", "path", v.ApiCtx.DatabasePath)
        return nil // unreachable
    }
    filePath := strings.Split(v.ApiCtx.DatabasePath, "?")[0]
    if removeErr := os.Remove(filePath); removeErr != nil {
        tlog.Fatal("Cannot remove corrupt database file", "error", removeErr)
        return nil // unreachable
    }
    return newDBWithDepth(lc, v, depth+1)
}
```

```go
// http_server.go — request size middleware
const maxRequestBodySize = 1 << 20 // 1 MiB

func NewMaxBodyMiddleware(maxBytes int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
            next.ServeHTTP(w, r)
        })
    }
}
```

```go
// api/ws.go — goroutine lifecycle
connCtx, connCancel := context.WithCancel(r.Context())
defer connCancel()
go self.broadcaster.ProcessWebSocketChannel(connCtx, wsMessageSender.SendFunc)
```

## 🔗 Code References & TODOs

- [ ] `TODO: backend/src/dbom/db_config.go:220-232` — add recursion guard
- [ ] `TODO: backend/src/dbom/db_config.go:225-228` — replace return nil with Fatal
- [ ] `TODO: backend/src/server/http_server.go` — add MaxBytesReader middleware
- [ ] `TODO: backend/src/api/ws.go:318` — pass context to ProcessWebSocketChannel
- [ ] `TODO: backend/src/cmd/srat-server/main-server.go:288` — guard WaitGroup assertion
