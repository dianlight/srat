# Proposal

## Why

Fix three data-correctness and security defects in the frontend that cause silent incorrect behavior: 1. **`isLoading` uses `&&` instead of `||`** — the three data hooks (`healthHook`, `volumeHook`, `shareHook`) combine REST and WebSocket loading flags with `&&`, which means a REST API error makes `isLoading: false` while `evloading` is still `true`. Dashboard metrics, volumes, and shares render silently empty instead of showing an error state.

## What Changes

- Change `isLoading: isLoading && evloading` to `isLoading: isLoading || evloading` in `healthHook.ts:48`, `volumeHook.ts:29`, and `shareHook.ts:30`
- Add tests for each hook: verify that a REST 500 error results in `isLoading: false` AND `error: defined` AND `data: undefined/empty` (not silently empty data with no error surfaced)
- ~~Move `useRollbar()` and `useRollbarConfiguration()` in `useRollbarTelemetry.ts` to unconditional top-level calls~~ (Resolved by Sentry migration, task 040; `useSentryTelemetry.ts` has no hook-ordering issues)
- Add a `has_default_password bool` field to the backend `dto.User` struct, set to `true` when the stored password matches the default bootstrap hash
- In `UserService.GetAdmin()` and `UserService.ListUsers()`, populate `has_default_password` by comparing the stored NT hash to the known default
- Remove the `password` field from the `GET /users` and `GET /useradmin` response DTO (set `json:"-"` on the response path, or use a separate `UserResponse` DTO that omits the field)
- Update `useBaseConfigModal.ts` to use `adminUser.has_default_password === true` instead of comparing the password string
- Regenerate `sratApi.ts` (`mise run //frontend:gen`) after the backend DTO change
- Update all existing tests that rely on `user.password` being present in GET responses
- Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark F-REL-01, F-SEC-01, F-SEC-02 resolved

## Capabilities

### New Capabilities

- `frontend-data-correctness-isloading-bug-hook-rules-password-exposure`: Frontend Data Correctness — isLoading Bug, Hook Rules, Password Exposure

### Modified Capabilities

(none)

## Impact

- **Inputs:** `frontend/src/hooks/healthHook.ts`, `volumeHook.ts`, `shareHook.ts`, `useBaseConfigModal.ts`; `backend/src/api/users.go`, `backend/src/dto/user.go`
- **Outputs:** Correct loading/error states; no credential material in frontend
- **Dependencies:** RTK Query, `@sentry/react`, `dto.User`
- Legacy task: `docs/tasks/037_frontend-data-correctness-fixes.md`
