---
name: sentry-triage
description: 'Triage Sentry issues for SRAT backend/frontend. Use when user says sentry triage, check sentry issues, inspect last N backend/frontend issues, identify cause, classify and report. Covers version-pinned code review against deployed tags, Sentry MCP lookup, and SRAT-specific noise patterns.'
argument-hint: '"last 5 backend and frontend" or "triage SRAT-BACKEND-X" — optional issue IDs or counts'
---

# Sentry Triage — SRAT Backend/Frontend

Triage unresolved Sentry issues for `srat-backend` and `srat-frontend` using Sentry issue data plus version-pinned code only. External installs are unreachable, so never assume live repro is possible.

## Golden Rules

- Read-only triage. Do not modify source to investigate. Use Sentry MCP + `git show <tag>:<path>` + `git grep -n <pattern> <tag> -- <path>`.
- Never use HEAD for cause analysis. Always map the Sentry event `release`/`version` tag to a git tag (e.g. `2026.10.0-rc17`, `2026.9.2-rc16`, `2026.10.0-rc18`) and quote `file:line` from that tag.
- If `sentry` CLI is missing, use Sentry MCP tools instead. Do not ask the user to paste tokens.
- Redact PII (emails, IPs, user ids). Summarize stacktraces, do not paste raw dumps.

## Step 1 — Locate Org and Projects

1. `find_organizations({})` → expect `lucio-tarantino`.
2. `find_projects({organizationSlug: "lucio-tarantino"})` → expect `srat-backend`, `srat-frontend`.
3. Known quirk: frontend JS events often land in the `srat-backend` project because `VITE_SENTRY_DSN` is shared. Check `Project` field in issue detail and call this out when it happens. Verify with `git show <tag>:frontend/src/macro/Environment.ts` (`getSentryDsn`) and `frontend/src/index.tsx` + `frontend/src/hooks/useSentryTelemetry.ts`.

## Step 2 — List Issues

Honor user-supplied scope first. If the user names a single issue (e.g. triage SRAT-BACKEND-X), work only on that issue plus a similarity search for the same root cause (same culprit frame, error string, device/path, or wrapper chain) and group the matches as one case. If the user gives a count, query string, sort, period, or project filter, use those values instead of the defaults below and state what was used.

Default when no scope is given, per project:

```
search_issues({organizationSlug: "lucio-tarantino", projectSlugOrId: "srat-backend", query: "is:unresolved", sort: "date", limit: 5, period: "90d"})
search_issues({organizationSlug: "lucio-tarantino", projectSlugOrId: "srat-frontend", query: "is:unresolved", sort: "date", limit: 5, period: "90d"})
```

Capture per issue: shortId, title, status/substatus, users, events/occurrences, first/last seen, culprit, URL.

## Step 3 — Issue Detail

For each shortId use:

```
get_sentry_resource({organizationSlug: "lucio-tarantino", resourceType: "issue", resourceId: "<SHORT-ID>"})
```

Capture: event `release`/`version`, `arch`, `os`, culprit frames with `path:line`, `extra` (device id, mount path, `detail` vs `message`), wrapped errors. Note when one event is a wrapper of another (e.g. `Mount Fail` wrapping `no such device` at the same timestamp).

## Step 4 — Version-Pinned Code Check

Map each issue to its tag, then inspect:

```bash
git tag --list | grep -E "2026\.(9|10)\.0-rc1[678]"
git show <tag>:<path> | sed -n '<start>,<end>p'
git grep -n "<error string>" <tag> -- backend/src frontend/src | head
```

SRAT hot spots learned from triage:

- Backend smbd restart: `backend/src/service/server_process_service.go` — `restartServerServices` (~:777), `runCommandWithRunner` (~:345), `DirtyDataService` 5s debounce (`backend/src/service/dirty_data_service.go:125`). Check timeout scope: one `WithTimeout(30s)` covering the whole service loop is a known failure mode (`context deadline exceeded` with empty output).
- Protected mode guard: `backend/src/service/volume/mount_orchestrator.go:133` (`MountVolume`) is intentional. If reached via `udev_handler.go:721 HandleMountPointEvent` on HA-connect/udev sync, it is noise unless the caller should have skipped earlier.
- Auth noise: `backend/src/server/ha_middleware.go:92` in rc16 logs `Unauthorized access from` at Error level. rc17 added `defaultTrustedPrefixes` + `clientIP`. Compare tags before blaming allowlist.
- Mount cascade: `backend/src/service/volume_mount_manager.go:60` wrapping `backend/src/service/filesystem/base_adapter.go:400 Mount` loses errno as `Mount Fail`. Same-timestamp `no such device` for `/addon_configs` fstype `native` means one root cause, double-reported. Check for missing native/system-path skip-list and device-existence stat before mount.
- Frontend empty messages: `frontend/src/components/wizard/SetupWizard.tsx:441` (`Error applying settings in wizard`), `frontend/src/pages/volumes/hooks/useMountVolume.ts:64` (`Mount Error`), `frontend/src/pages/volumes/utils.ts:279` (`Unmount Error`). rc16 vs rc18 moved files but kept the same strings. `useSentryTelemetry.ts` `reportError` uses `Sentry.withScope` + `captureMessage(string)` producing titles like `withScope` / `Object.current` with `console_args: [Object]`. Check whether handler reads `detail` (correct, as in `useMountVolume`) or only `message` (loses backend `detail`).

## Step 5 — Classify

Assign one per issue:

- real bug / reliability — affects users, needs code fix (e.g. smbd timeout, wizard failure).
- instrumentation noise — expected guard or info sent as error (e.g. Protected mode, `Unauthorized access` info).
- wrong level / routing — error level or project mis-attributed (e.g. info log in Sentry, JS in backend project).
- race / duplicate cascade — same root reported at two layers or on remove/add storms.
- observability debt — empty message, `[Object]` context, missing sourcemap, `detail` vs `message` mismatch.
- symptom — frontend toast surfacing a backend failure; link to the backend issue instead of fixing twice.

## Step 6 — Report and Proceed

Output is always a report plus a suggestion to proceed. Group issues by root cause first: when several shortIds share timestamp, device, path, or wrapper chain (e.g. `Mount Fail` wrapping `no such device`), report them as one case with all linked Sentry URLs.

Keep it short, plain sentences, answer first. A per-case entry needs: volume/impact, likely cause with tag-pinned `file:line`, classification, and one concrete next step. End with a prioritized order: silence noise and fix routing first so counts are trustworthy, then skip-list/debounce mounts, then smbd timeout, then frontend enrichment (sourcemaps, serialize extra, unify `detail`/`message`, log which wizard commit failed).

For every case (grouped by root cause), ask the user to choose exactly one: fix, report as GitHub issue, or ignore. Ask via the question tool with the case title plus recommended option first, e.g. fix for real bugs, report for needs-tracking, ignore for noise. Do not start fixing or filing until the user answers. If the user defers, leave the case undecided and move on.

When the user chooses report as GitHub issue, draft the issue with a detailed description and a proposed solution, and always link the Sentry issue URL(s). Required issue shape: title naming the symptom and scope, environment (release tags, arch/os from Sentry), Sentry link section, what happened vs expected, root cause with tag-pinned `file:line` evidence, proposed solution with concrete code pointers, and acceptance criteria. Search existing open issues first to avoid duplicates, then create in `dianlight/srat` with `bug` label for FIX-type cases and paste the new issue URL back into the triage report.

## Step 7 — GitHub dedup + Sentry sync (explicit sync only)

Run only when the user explicitly asks (`check github`, `update sentry`, `sync`) or right after creating a GitHub issue from Step 6. Never auto-sync during read-only triage.

1. Per grouped case, search GitHub: `search_issues({owner: "dianlight", repo: "srat", query: "<Sentry title + error string + culprit>", perPage: 5})` plus `search_pull_requests` for fix keywords. Then `issue_read({owner: "dianlight", repo: "srat", issue_number: <N>, method: "get"})` to capture `state`, `closed_by_pull_requests` (PR number/title/state/url), and `updated_at`.
2. Match rule: exact Sentry URL in the GitHub body, or same error string + same tag-pinned `file:line` + same path/device. Record the GitHub issue URL and the merged PR URL.
3. Link: call `execute_sentry_tool({name: "link_issue", arguments: {organizationSlug: "lucio-tarantino", issueId: "<SHORT-ID>", externalIssueUrl: "<github-issue-url>"}})` and repeat for the PR URL. Native link fails with `No active issue integration` when the GitHub integration is missing — fallback is mandatory: `execute_sentry_tool({name: "add_issue_note", arguments: {organizationSlug: "lucio-tarantino", issueId: "<SHORT-ID>", text: "GitHub done: <issue-url> closed by <pr-url> ..."}})` so the link lives in the activity feed.
4. Status (auto by dates): GitHub open → link/note only, leave Sentry `unresolved`. GitHub closed → compare GitHub `updated_at`/merge date against Sentry `last seen` and release tag. Fix released before last seen → `update_issue({organizationSlug: "lucio-tarantino", issueId: "<SHORT-ID>", status: "resolved", reason: "Fixed by #<PR>, GitHub #<N> closed <date>."})`. Fix merged after last seen → `status: "resolvedInNextRelease"`. Never resolve on open GitHub issues.
5. Also `session.link` every Sentry shortId plus the matched GitHub issue URL so the user sees them with the session.

Known SRAT fingerprints:

- `Error performing hard restart of service smbd: context deadline exceeded` → timeout scope, make restart async with per-service timeout.
- `Operation not permitted in Protected mode` at `MountVolume` → early-return before emit/persist, downgrade to info, ignore in Sentry.
- `Unauthorized access from` level info → drop to warn, stop sending info to Sentry, verify rc17 prefixes.
- `Mount Fail` + `mount /addon_configs ... no such device (fs type native)` → skip `native` fstype and HAOS system paths, stat device first, collapse fingerprint.
- `Error applying settings in wizard:` / `Unmount Error:` / `Mount Error:` with empty suffix → include backend response body, fix `detail` handling, upload sourcemaps for `-rc` builds.
