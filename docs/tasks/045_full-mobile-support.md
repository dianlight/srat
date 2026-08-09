<!-- DOCTOC SKIP -->

# [FEATURE]: Full Mobile Support

**Target Repo:** `srat`
**Status:** 📅 Planned

## 🎯 Objective

Make the SRAT frontend fully usable on mobile and small-screen devices (phones 320–430px, tablets 600–1024px). Verified via `dev:remote` against a live backend (`http://192.168.0.68:3000`) with Playwright Chromium at 360px/375px/390px/600px/768px/850px/1024px viewports. No document-level horizontal overflow exists on any page; the issues below are element/container-level layout problems that degrade usability on small screens.

> _Context for Copilot: The app currently targets desktop. Mobile works at the document level (NavBar hamburger, no body overflow) but several components break inside their containers: Dashboard metric tables overflow their cards, the SMART self-test status row overflows the viewport, the setup wizard stepper is unusable, and the Volumes left panel is too narrow. The Settings page already implements the correct mobile pattern (hamburger → left Drawer at xs, static 300px tree at md+) and should be used as the reference._

## 🛠️ Technical Specifications

- **Inputs:** Frontend components only (`frontend/src/**`). No API changes.
- **Breakpoint convention:** `theme.breakpoints.down("sm")` for true mobile (xs), NOT `between("sm", "md")` which excludes xs.
- **Reference pattern:** `frontend/src/pages/settings/Settings.tsx` — hamburger IconButton `display: { xs: "flex", md: "none" }` (line 244), Drawer `display: { xs: "block", md: "none" }` (line 286), static left tree `display: { xs: "none", md: "block" }` (line 325).
- **Dependencies:**
  - `frontend/src/pages/dashboard/metrics/NetworkHealthMetrics.tsx` (TableCells `minWidth: 150` at lines 94, 124)
  - `frontend/src/pages/dashboard/metrics/ProcessMetrics.tsx` (TableCells `minWidth: 150` at lines 127, 154, 181)
  - `frontend/src/pages/volumes/components/SmartStatusPanel.tsx` (Self-Test Status row Stack, lines 460–492)
  - `frontend/src/components/wizard/SetupWizard.tsx` (horizontal `Stepper` line 481, Dialog `maxWidth="sm"`)
  - `frontend/src/pages/volumes/Volumes.tsx` (resizable left panel constants lines 50–52, MIN 15% / MAX 60% / DEFAULT 30%)
  - `frontend/src/components/NotificationCenter.tsx` (Card `minWidth: "22em"` line 168)
  - `frontend/src/components/NavBar.tsx` (right icon cluster Box `flexGrow: 0` line 556, `up("sm")` line 294)
  - `frontend/src/pages/shares/components/ShareActions.tsx` (`between("sm", "md")` line 40)
  - `frontend/src/pages/volumes/components/PartitionActions.tsx` (`between("sm", "lg")` line 57)
  - `frontend/src/index.html` (viewport meta line 5)

## 🔍 Verified Findings (dev:remote + Playwright, live backend)

| # | Component | Viewport(s) | Measured issue |
|---|-----------|-------------|----------------|
| 1 | Dashboard metric tables | 360–390px | Tables render at 726px / 1332px / 646px wide inside 264–294px `TableContainer`s → horizontal scroll, effectively unusable |
| 2 | SmartStatusPanel Self-Test row | 390px | Status Stack (`direction="row"`, no `flexWrap`) with Chip "short test completed without error" + `(unknown)` caption overflows to x=407 (viewport 390) |
| 3 | SetupWizard Stepper | 390px | Horizontal Stepper content 601px inside 326px Dialog (`maxWidth="sm"`, no `fullScreen`) → 6 steps scroll horizontally, cramped |
| 4 | NotificationCenter Popover | 360px / 320px | Card `minWidth: "22em"` = 352px; at 360px viewport card right edge = 368px → viewport overflow |
| 5 | Volumes left panel | 390px / 360px | Resizable panel renders 103px / 94px wide (15–30% of phone width); tree rows 87px / 78px → volume names + action icons squeezed |
| 6 | NavBar right cluster | 600–768px | Fixed 224px `flexGrow: 0` icon cluster; tab bar squeezed to 288px at 600px, 456px at 768px (7 tabs, scrollable) |
| 7 | ShareActions / PartitionActions breakpoints | xs | `useMediaQuery(between("sm","md"))` / `between("sm","lg")` exclude xs → phones get desktop layout (code-level finding) |
| 8 | index.html viewport meta | all | `width=device-width, initial-scale=1` present; missing `viewport-fit=cover` and `theme-color` |

**Passing (no changes needed):** Shares/Users/Settings pages (no overflow at 360–390px), Settings Drawer pattern, UserDetailsPanel/UserEditForm/PartitionInformationCard/HDIdleDiskSettings/VolumeMountDialog/VolumeDetailsPanel responsive Grids, Footer `down("sm")` usage, NavBar hamburger menu at xs, document-level scroll on all pages.

## 📝 Task List

- [x] Task 1: Dashboard metric tables — make `NetworkHealthMetrics` (Device/IP/Netmask/Inbound/Outbound) and `ProcessMetrics` (name/pid/cpu/connections/memory + child rows) responsive at xs: horizontal-scroll `TableContainer` is acceptable, but replace `minWidth: 150` with smaller min-widths and compact `size="small"`; verify no cell exceeds container at 360px
- [x] Task 2: SmartStatusPanel — add `flexWrap: "wrap"` to the Self-Test Status row Stack (lines 461–467); consider truncating the Chip label (`maxWidth` + ellipsis) for long status strings
- [x] Task 3: SetupWizard — on xs: use `fullScreen` Dialog (or `maxWidth="sm"` + vertical `Stepper orientation={{ xs: "vertical", sm: "horizontal" }}`) so the 6 steps are usable on phones
- [x] Task 4: NotificationCenter — replace fixed `minWidth: "22em"` with `minWidth: { xs: "min(22em, 100vw - 2rem)", sm: "22em" }` so the Popover fits 320px viewports
- [x] Task 5: Volumes left panel — enforce a sensible mobile width: min clamp at xs (e.g. `min(45%, 180px)`) or collapse to a Drawer/overlay below `sm` following the Settings pattern
- [x] Task 6: NavBar — audit right icon cluster at 600–768px: hide non-essential icons below `md` (e.g. DevInspector bug, GitHub), or move into an overflow menu, freeing tab bar width
- [x] Task 7: Breakpoint fixes — `ShareActions.tsx:40` and `PartitionActions.tsx:57`: change `between("sm", "md")` / `between("sm", "lg")` to include xs (e.g. `down("md")` / `down("lg")`) so action buttons/compact mode apply on phones
- [x] Task 8: index.html — add `viewport-fit=cover` and `theme-color` to the viewport meta
- [x] Task 9: Tests (Vitest + RTL + `user-event`, MSW handlers in `frontend/src/mocks/customHandlers.ts`) for each changed component: table renders without container overflow, SmartStatusPanel wraps, wizard step navigation at xs, NotificationCenter fits 320px
- [x] Task 10: Browser verification — extend/run `mise run //frontend:test:browser` (Playwright via vitest browser config) with a 375px viewport smoke test over Dashboard/Volumes/Shares/Users/Settings asserting `scrollWidth <= clientWidth` and no `PARSING_ERROR`-style console errors
- [x] Task 11: Update `CHANGELOG.md` under `## [ 🚧 Unreleased ]`
- [ ] Task 12: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)

### How verification was performed

- Frontend dev server: `API_URL=http://192.168.0.68:3000 NODE_ENV=remote bun --hot run bun.build.ts -w -s ./out` (serves on port 3080). The `-a` flag alone does NOT reach the API layer: `getApiUrl()` is a Bun macro evaluated at import time (module import of `src/index.html` at `bun.build.ts:8`), before `build()` sets `process.env.API_URL` — the app falls back to same-origin "dynamic" and hits `PARSING_ERROR`. The `dev:remote` task in `frontend/.mise.toml` passes `-a`, so when using it, also export `API_URL` in the environment.
- Playwright: `bunx playwright screenshot --viewport-size=390,844 --wait-for-timeout=12000 http://localhost:3080/ out.png`; programmatic checks via a temp `.mjs` script using `document.documentElement.scrollWidth` vs `clientWidth`, `getBoundingClientRect()` for out-of-viewport elements, and `.MuiTable-root` scrollWidth vs container width.
- Tab switching for measurement: seed `localStorage.setItem("srat_tab", idx)` via `page.addInitScript` (0=Dashboard, 1=Volumes, 2=Shares, 3=Users, 4=Settings, 5=API Docs) — the app reads it on load (NavBar lines 376–418).

### Patterns to follow

- Settings.tsx mobile pattern (hamburger + Drawer + responsive left tree) is the in-repo reference for any master/detail mobile layout.
- MUI Grid `size={{ xs: 12, sm: 6 }}` already used correctly in VolumeDetailsPanel/UserEditForm — do not regress these.
- `flexWrap: "wrap"` on row Stacks is the established fix (see VolumeDetailsPanel mount flags, lines 514–535).
