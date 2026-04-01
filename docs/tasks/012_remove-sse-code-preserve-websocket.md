# [REFACTOR]: Remove SSE Code, Preserve WebSocket

**Target Repo:** `srat`  **Status:** ✅ Done  **Issue Link:** https://github.com/dianlight/srat/issues/501

## 🎯 Objective

Remove all Server-Sent Events (SSE) related code from the backend (Go) and frontend (TypeScript/React), keeping only the WebSocket (WS) transport layer. SSE was the original real-time event transport; it has already been superseded by the WebSocket implementation. The SSE backend endpoint (`/sse`), its handler, the `ProcessHttpChannel` method on `BroadcasterService`, the related vendor import of `huma/v2/sse`, and the commented-out SSE client in the frontend must all be cleanly removed. Particular care is needed where SSE and WS code share interfaces or fake/mock implementations.

## 🛠️ Technical Specifications

- **Inputs:** Existing codebase with both SSE dead code and active WebSocket code.
- **Outputs:** Clean codebase with only WebSocket transport; no SSE references outside of vendor/third-party code.
- **Dependencies:**
  - `backend/src/api/sse.go` — SSE broker handler (delete)
  - `backend/src/api/sse_test.go` — SSE broker tests (delete)
  - `backend/src/service/broadcaster_service.go` — `ProcessHttpChannel` method + interface entry (remove)
  - `backend/src/service/broadcaster_service_test.go` — `TestProcessHttpChannelAfterStop_DoesNotPanic` test (remove)
  - `backend/src/cmd/srat-server/main-server.go` — `server.AsHumaRoute(api.NewSSEBroker)` (remove)
  - `backend/src/cmd/srat-openapi/main-openapi.go` — `server.AsHumaRoute(api.NewSSEBroker)` (remove)
  - `backend/src/api/health_run_internal_test.go` — `fakeBroadcaster.ProcessHttpChannel` stub (remove method from fake)
  - `frontend/src/store/sseApi.ts` — large commented-out `sseApi` block + SSE comments (clean up; keep `wsApi` + `useGetServerEventsQuery`)
  - `frontend/src/store/__tests__/sseApi.test.tsx` — SSE-specific test blocks using `sseApi.endpoints` (remove; keep WS tests)
  - `frontend/src/store/store.ts` — currently imports `wsApi` from `sseApi`; no change needed unless file is renamed
  - All frontend files importing `useGetServerEventsQuery` from `../store/sseApi` — update import path if file is renamed

## 📝 Task List

- [x] Task 1: Delete `backend/src/api/sse.go` and `backend/src/api/sse_test.go`
- [x] Task 2: Remove `ProcessHttpChannel(send sse.Sender)` from `BroadcasterServiceInterface` in `broadcaster_service.go` and delete the corresponding method implementation
- [x] Task 3: Remove `TestProcessHttpChannelAfterStop_DoesNotPanic` from `broadcaster_service_test.go`
- [x] Task 4: Remove `ProcessHttpChannel` stub from `fakeBroadcaster` in `health_run_internal_test.go`
- [x] Task 5: Remove `server.AsHumaRoute(api.NewSSEBroker)` from `main-server.go` and `main-openapi.go`; remove any now-unused `api` package import if applicable
- [x] Task 6: Clean up `frontend/src/store/sseApi.ts` — delete the large commented-out `sseApi` `createApi` block (~lines 47–187), update/remove SSE-only JSDoc comments; rename file to `wsApi.ts` and update all imports across the frontend
- [x] Task 7: Remove SSE-specific test cases from `frontend/src/store/__tests__/sseApi.test.tsx` (those importing/using `sseApi`); rename file to `wsApi.test.tsx` if file is renamed in Task 6
- [x] Task 8: Run backend tests — `cd backend/src && go test ./api ./service` — and fix any compilation errors
- [x] Task 9: Run frontend tests — `mise run //frontend:test` — and fix any failures (SmartStatusPanel tests now all pass)
- [x] Task 10: Regenerate OpenAPI spec and verify `/sse` endpoint is gone — `cd backend && make gen` then inspect `backend/docs/openapi.json`
- [x] Task 11: Run `cd backend && make build` to confirm no build errors
- [x] Task 12: Documentation — update any doc files that mention SSE as the real-time transport (e.g. `docs/EVENT_DRIVEN_ARCHITECTURE.md`, `docs/HOME_ASSISTANT_INTEGRATION.md`, task files `001_*` and `002_*` that reference SSE events)

## 🧠 Implementation Notes (Copilot Context)

### Crossed concerns — what is shared and must be preserved

| Symbol | File | SSE-only? | Action |
|---|---|---|---|
| `BroadcasterServiceInterface.ProcessHttpChannel` | `service/broadcaster_service.go:27` | ✅ Yes | Remove from interface + impl |
| `BroadcasterServiceInterface.ProcessWebSocketChannel` | same file | ❌ WS-only | Keep |
| `BroadcasterServiceInterface.BroadcastMessage` | same file | ❌ Shared | Keep |
| `fakeBroadcaster.ProcessHttpChannel` (stub) | `api/health_run_internal_test.go:29` | ✅ Yes | Remove from fake struct |
| `fakeBroadcaster.ProcessWebSocketChannel` (stub) | same line :30 | ❌ WS | Keep |
| `dto.WebEventMap` | used by both `sse.go` and `ws.go` | ❌ Shared | Keep |
| `server.AsHumaRoute(api.NewSSEBroker)` | `main-server.go`, `main-openapi.go` | ✅ Yes | Remove |
| `server.AsHumaRoute(NewWebSocketBroker)` | same files | ❌ WS | Keep |

### Frontend file rename strategy

`sseApi.ts` is currently the file name, but it only exports WS code once SSE is removed. Two options:
1. **Rename to `wsApi.ts`** and update all `from "../store/sseApi"` imports — cleaner, preferred.
2. **Keep the filename** to avoid a large import churn — acceptable if rename adds too much noise.

All frontend hooks and pages import `useGetServerEventsQuery` from `sseApi` — there are ~15 import sites. A global find-and-replace is safe.

### Huma SSE vendor dependency

After removing `api/sse.go` the `"github.com/danielgtaylor/huma/v2/sse"` import disappears from production code. The vendor package itself (`backend/src/vendor/github.com/danielgtaylor/huma/v2/sse/`) can be left in vendor (removing vendor entries requires `go mod vendor` + re-applying patches per the patch workflow). If the `sse` sub-package is still listed in `go.mod` indirectly (it is part of the `huma/v2` module), no `go.mod` change is needed.

### Mock/interface impact

`mockio`-generated mocks of `BroadcasterServiceInterface` are regenerated automatically. Any hand-written mock or `fakeBroadcaster` struct must be manually updated to remove the `ProcessHttpChannel` method.

### Implementation execution notes

- 2026-03-17: Backend SSE route/handler and broadcaster HTTP channel path removed.
- 2026-03-17: Frontend `sseApi.ts` replaced by `wsApi.ts`; imports and tests migrated.
- 2026-03-17: Regenerated OpenAPI and frontend RTK client (`bun run gen`) to remove stale `/api/sse` contracts.
- Validation summary: targeted backend tests (`./api`, `BroadcasterService` suite), focused frontend tests, `make gen`, and `make build` pass; full frontend build succeeds with no compile errors.
- 2026-03-17 Polish Pass: Verified SmartStatusPanel tests now pass (18/18); verified all core SSE removal changes in place; frontend build completes successfully with zero compilation errors.

## 🔗 Code References & TODOs

- [x] `backend/src/api/sse.go` — delete entire file
- [x] `backend/src/api/sse_test.go` — delete entire file
- [x] `backend/src/service/broadcaster_service.go:27` — remove `ProcessHttpChannel(send sse.Sender)` from interface
- [x] `backend/src/service/broadcaster_service.go:222-265` (approx) — remove `ProcessHttpChannel` implementation
- [x] `backend/src/service/broadcaster_service_test.go:91` — remove `TestProcessHttpChannelAfterStop_DoesNotPanic`
- [x] `backend/src/api/health_run_internal_test.go:29` — remove `ProcessHttpChannel` stub from `fakeBroadcaster`
- [x] `backend/src/cmd/srat-server/main-server.go:181` — remove `server.AsHumaRoute(api.NewSSEBroker)`
- [x] `backend/src/cmd/srat-openapi/main-openapi.go:84` — remove `server.AsHumaRoute(api.NewSSEBroker)`
- [x] `frontend/src/store/sseApi.ts` — removed and replaced by `frontend/src/store/wsApi.ts`
- [x] `frontend/src/store/__tests__/sseApi.test.tsx` — removed and replaced by `frontend/src/store/__tests__/wsApi.test.tsx`
