# Design

## Context

Fix three reliability issues that share the theme of defensive guards around infrastructure components: 1. **`replaceDatabase` infinite recursion** — `NewDB` and `replaceDatabase` can recurse indefinitely when the database path is persistently unwritable, causing a stack overflow.

## Goals / Non-Goals

**Goals:**

- Deliver: Database Recovery Safety, HTTP Request Size Limits, and Goroutine Leak

**Non-Goals:**

- Unrelated refactoring outside the task scope.

## Decisions

- Add a `recovered bool` parameter to `replaceDatabase` (or an internal `newDBWithDepth(depth int)` variant); if `depth > 1`, call `tlog.Fatal` instead of recursing
- Replace `return nil` in `replaceDatabase` (when `os.Remove` fails) with `tlog.Fatal("cannot remove corrupt database file: %v", removeErr)`
- Add an HTTP middleware in `NewMuxRouter` that wraps `r.Body` with `http.MaxBytesReader(w, r.Body, 1<<20)` (1 MiB); return HTTP 413 on overflow; document the constant as `maxRequestBodySize`
- Modify `broadcaster.ProcessWebSocketChannel` to accept a `context.Context` parameter; return when the context is done
- In `HandleWebSocket` (`api/ws.go`), derive a connection-scoped context from the request context; pass it to `ProcessWebSocketChannel`; ensure the goroutine returns on connection close
- Apply the safe `if wg, ok := apiCtx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup); ok && wg != nil` guard in `main-server.go:288`
- Add tests: (a) verify `NewDB` with a permanently unwritable path does not recurse more than once; (b) verify a 2 MiB request body returns 413; (c) verify `ProcessWebSocketChannel` goroutine exits when the connection context is cancelled
- Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-REL-02, B-REL-03, B-REL-04, B-REL-05, B-PERF-04 resolved

## Risks / Trade-offs

- Behaviour changes are covered by targeted tests before handoff.
- Legacy task content remains the reference until the change is archived.
