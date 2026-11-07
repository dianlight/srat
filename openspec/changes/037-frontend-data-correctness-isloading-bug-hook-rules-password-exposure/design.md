# Design

## Context

Fix three data-correctness and security defects in the frontend that cause silent incorrect behavior: 1. **`isLoading` uses `&&` instead of `||`** — the three data hooks (`healthHook`, `volumeHook`, `shareHook`) combine REST and WebSocket loading flags with `&&`, which means a REST API error makes `isLoading: false` while `evloading` is still `true`. Dashboard metrics, volumes, and shares render silently empty instead of showing an error state.

## Goals / Non-Goals

**Goals:**

- Deliver: Frontend Data Correctness — isLoading Bug, Hook Rules, Password Exposure

**Non-Goals:**

- Unrelated refactoring outside the task scope.

## Decisions

- Change `isLoading: isLoading && evloading` to `isLoading: isLoading || evloading` in `healthHook.ts:48`, `volumeHook.ts:29`, and `shareHook.ts:30`
- Add tests for each hook: verify that a REST 500 error results in `isLoading: false` AND `error: defined` AND `data: undefined/empty` (not silently empty data with no error surfaced)
- ~~Move `useRollbar()` and `useRollbarConfiguration()` in `useRollbarTelemetry.ts` to unconditional top-level calls~~ (Resolved by Sentry migration, task 040; `useSentryTelemetry.ts` has no hook-ordering issues)
- Add a `has_default_password bool` field to the backend `dto.User` struct, set to `true` when the stored password matches the default bootstrap hash
- In `UserService.GetAdmin()` and `UserService.ListUsers()`, populate `has_default_password` by comparing the stored NT hash to the known default
- Remove the `password` field from the `GET /users` and `GET /useradmin` response DTO (set `json:"-"` on the response path, or use a separate `UserResponse` DTO that omits the field)
- Update `useBaseConfigModal.ts` to use `adminUser.has_default_password === true` instead of comparing the password string
- Regenerate `sratApi.ts` (`mise run //frontend:gen`) after the backend DTO change

## Risks / Trade-offs

- Behaviour changes are covered by targeted tests before handoff.
- Legacy task content remains the reference until the change is archived.
