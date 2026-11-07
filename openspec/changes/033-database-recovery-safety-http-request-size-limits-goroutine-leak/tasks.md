# Tasks

## 1. Implementation

- [ ] 1.1 Add a `recovered bool` parameter to `replaceDatabase` (or an internal `newDBWithDepth(depth int)` variant); if `depth > 1`, call `tlog.Fatal` instead of recursing — verified by targeted tests
- [ ] 1.2 Replace `return nil` in `replaceDatabase` (when `os.Remove` fails) with `tlog.Fatal("cannot remove corrupt database file: %v", removeErr)` — verified by targeted tests
- [ ] 1.3 Add an HTTP middleware in `NewMuxRouter` that wraps `r.Body` with `http.MaxBytesReader(w, r.Body, 1<<20)` (1 MiB); return HTTP 413 on overflow; document the constant as `maxRequestBodySize` — verified by targeted tests
- [ ] 1.4 Modify `broadcaster.ProcessWebSocketChannel` to accept a `context.Context` parameter; return when the context is done — verified by targeted tests
- [ ] 1.5 In `HandleWebSocket` (`api/ws.go`), derive a connection-scoped context from the request context; pass it to `ProcessWebSocketChannel`; ensure the goroutine returns on connection close — verified by targeted tests
- [ ] 1.6 Apply the safe `if wg, ok := apiCtx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup); ok && wg != nil` guard in `main-server.go:288` — verified by targeted tests
- [ ] 1.7 Add tests: (a) verify `NewDB` with a permanently unwritable path does not recurse more than once; (b) verify a 2 MiB request body returns 413; (c) verify `ProcessWebSocketChannel` goroutine exits when the connection context is cancelled — verified by targeted tests
- [ ] 1.8 Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-REL-02, B-REL-03, B-REL-04, B-REL-05, B-PERF-04 resolved — verified by targeted tests

## 2. Verification

- [ ] 2.1 Run the relevant test suites and confirm the change passes — verified by a green test run
