# Tasks

## 1. Implementation

- [ ] 1.1 Check whether `gitlab.com/tozd/go/errors` exposes `AsType[T]`; if yes, migrate the four `errors.As` call sites in `unixsamba/unixsamba.go`; if no, import stdlib `errors` with an alias and use `stderrors.AsType[T]` — verified by targeted tests
- [ ] 1.2 Extract `filesystemOperationRunner` struct from `filesystem_service.go` — move `startOperation`, `finishOperation`, `wg`, `operationMu`, and operation-context tracking to `service/filesystem_operation_runner.go` — verified by targeted tests
- [ ] 1.3 Update `FilesystemService` to embed or hold a `*filesystemOperationRunner`; ensure all callers compile without change — verified by targeted tests
- [ ] 1.4 Extract `hdIdleDiskState` struct from `hdidle_service.go` — move per-disk state tracking fields and their methods to `service/hdidle_disk_state.go` — verified by targeted tests
- [ ] 1.5 Update `HDIdleService` to embed or hold a `*hdIdleDiskState`; ensure all callers compile without change — verified by targeted tests
- [ ] 1.6 Run `go build ./...` and `go vet ./...` to confirm no regressions after each split — verified by targeted tests
- [ ] 1.7 Run existing tests: `cd backend/src && go test ./service ./unixsamba` — all must pass — verified by targeted tests
- [ ] 1.8 Run `go_diagnostics` (gopls) on modified files and fix any reported issues — verified by targeted tests
- [ ] 1.9 Remove stale "errors.As → errors.AsType Migration" section from `docs/FUTURE_IMPROVEMENTS.md` — verified by targeted tests

## 2. Verification

- [ ] 2.1 Run the relevant test suites and confirm the change passes — verified by a green test run
