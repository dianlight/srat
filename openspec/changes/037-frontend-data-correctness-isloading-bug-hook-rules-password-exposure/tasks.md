# Tasks

## 1. Implementation

- [ ] 1.1 Change `isLoading: isLoading && evloading` to `isLoading: isLoading || evloading` in `healthHook.ts:48`, `volumeHook.ts:29`, and `shareHook.ts:30` — verified by targeted tests
- [ ] 1.2 Add tests for each hook: verify that a REST 500 error results in `isLoading: false` AND `error: defined` AND `data: undefined/empty` (not silently empty data with no error surfaced) — verified by targeted tests
- [ ] 1.3 ~~Move `useRollbar()` and `useRollbarConfiguration()` in `useRollbarTelemetry.ts` to unconditional top-level calls~~ (Resolved by Sentry migration, task 040; `useSentryTelemetry.ts` has no hook-ordering issues) — verified by targeted tests
- [ ] 1.4 Add a `has_default_password bool` field to the backend `dto.User` struct, set to `true` when the stored password matches the default bootstrap hash — verified by targeted tests
- [ ] 1.5 In `UserService.GetAdmin()` and `UserService.ListUsers()`, populate `has_default_password` by comparing the stored NT hash to the known default — verified by targeted tests
- [ ] 1.6 Remove the `password` field from the `GET /users` and `GET /useradmin` response DTO (set `json:"-"` on the response path, or use a separate `UserResponse` DTO that omits the field) — verified by targeted tests
- [ ] 1.7 Update `useBaseConfigModal.ts` to use `adminUser.has_default_password === true` instead of comparing the password string — verified by targeted tests
- [ ] 1.8 Regenerate `sratApi.ts` (`mise run //frontend:gen`) after the backend DTO change — verified by targeted tests
- [ ] 1.9 Update all existing tests that rely on `user.password` being present in GET responses — verified by targeted tests
- [ ] 1.10 Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark F-REL-01, F-SEC-01, F-SEC-02 resolved — verified by targeted tests

## 2. Verification

- [ ] 2.1 Run the relevant test suites and confirm the change passes — verified by a green test run
