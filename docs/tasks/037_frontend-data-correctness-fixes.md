<!-- DOCTOC SKIP -->

# [FIX]: Frontend Data Correctness — isLoading Bug, Hook Rules, Password Exposure

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** _None — discovered in reliability/security review 2026-04-28_

## 🎯 Objective

Fix three data-correctness and security defects in the frontend that cause silent incorrect behavior:

1. **`isLoading` uses `&&` instead of `||`** — the three data hooks (`healthHook`, `volumeHook`, `shareHook`) combine REST and WebSocket loading flags with `&&`, which means a REST API error makes `isLoading: false` while `evloading` is still `true`. Dashboard metrics, volumes, and shares render silently empty instead of showing an error state.
2. ~~**Rules of Hooks violation in `useRollbarTelemetry`**~~ (Resolved by Sentry migration, task 040: `useRollbarTelemetry.ts` rewritten as `useSentryTelemetry.ts` with no Rollbar-style hooks)
3. **Password comparison in frontend** — `useBaseConfigModal` receives the user's `password` field from the API and compares it to `"changeme!"`. Credential material should never be returned in GET responses or processed in the frontend.

## 🛠️ Technical Specifications

- **Inputs:** `frontend/src/hooks/healthHook.ts`, `volumeHook.ts`, `shareHook.ts`, `useBaseConfigModal.ts`; `backend/src/api/users.go`, `backend/src/dto/user.go`
- **Outputs:** Correct loading/error states; no credential material in frontend
- **Dependencies:** RTK Query, `@sentry/react`, `dto.User`

## 📝 Task List

- [ ] Task 1: Change `isLoading: isLoading && evloading` to `isLoading: isLoading || evloading` in `healthHook.ts:48`, `volumeHook.ts:29`, and `shareHook.ts:30`
- [ ] Task 2: Add tests for each hook: verify that a REST 500 error results in `isLoading: false` AND `error: defined` AND `data: undefined/empty` (not silently empty data with no error surfaced)
- [x] Task 3: ~~Move `useRollbar()` and `useRollbarConfiguration()` in `useRollbarTelemetry.ts` to unconditional top-level calls~~ (Resolved by Sentry migration, task 040; `useSentryTelemetry.ts` has no hook-ordering issues)
- [ ] Task 4: Add a `has_default_password bool` field to the backend `dto.User` struct, set to `true` when the stored password matches the default bootstrap hash
- [ ] Task 5: In `UserService.GetAdmin()` and `UserService.ListUsers()`, populate `has_default_password` by comparing the stored NT hash to the known default
- [ ] Task 6: Remove the `password` field from the `GET /users` and `GET /useradmin` response DTO (set `json:"-"` on the response path, or use a separate `UserResponse` DTO that omits the field)
- [ ] Task 7: Update `useBaseConfigModal.ts` to use `adminUser.has_default_password === true` instead of comparing the password string
- [ ] Task 8: Regenerate `sratApi.ts` (`mise run //frontend:gen`) after the backend DTO change
- [ ] Task 9: Update all existing tests that rely on `user.password` being present in GET responses
- [ ] Task 10: Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark F-REL-01, F-SEC-01, F-SEC-02 resolved

## 🧠 Implementation Notes

```typescript
// healthHook.ts — fix AND → OR
export function useHealth(): HealthHookResult {
    const { data: evdata, isLoading: evloading } = useGetServerEventsQuery();
    const { data: health, isLoading, error } = useGetApiHealthQuery(...);
    return {
        health: evdata?.heartbeat ?? health,
        isLoading: isLoading || evloading,  // WAS: isLoading && evloading
        error,
    };
}
```

```go
// dto/user.go — add has_default_password to response
type User struct {
    Username           string          `json:"username"`
    Password           Secret[string]  `json:"password,omitempty" write-only:"true" format:"password"`
    HasDefaultPassword bool            `json:"has_default_password,omitempty" read-only:"true"`
}
```

The `write-only:"true"` tag on `Password` should prevent it from appearing in GET responses via Huma's serialization rules. Verify this is enforced, or use a response-specific DTO.

## 🔗 Code References & TODOs

- [ ] `TODO: frontend/src/hooks/healthHook.ts:48` — change && to ||
- [ ] `TODO: frontend/src/hooks/volumeHook.ts:29` — change && to ||
- [ ] `TODO: frontend/src/hooks/shareHook.ts:30` — change && to ||
- [ ] `TODO: frontend/src/hooks/useBaseConfigModal.ts:50-52` — use has_default_password
- [ ] `TODO: backend/src/dto/user.go` — add has_default_password field
- [ ] `TODO: backend/src/service/user_service.go` — populate has_default_password in GetAdmin/ListUsers
- [ ] Related: F-REL-01, F-SEC-01, F-SEC-02 in `docs/SECURITY_OPTIMIZATION_REVIEW.md`
