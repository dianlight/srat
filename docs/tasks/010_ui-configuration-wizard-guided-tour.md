# [FEATURE]: Configuration Wizard and Guided Help Tour

**Target Repo:** `srat`  
**Status:** 🔄 In Progress  
**Issue Link:** [srat#116](https://github.com/dianlight/srat/issues/116) · [srat#82](https://github.com/dianlight/srat/issues/82)

## 🎯 Objective

Improve the onboarding experience for new users by adding two complementary UI features:

1. **Configuration Wizard** — a step-by-step modal that guides users through initial setup (network interfaces, first share, user account) on first launch or via an explicit "Setup Wizard" button.
2. **Guided Help / UI Tour** — an optional overlay tour (using `@reactour/tour`, already in `package.json`) that highlights key UI elements with tooltips and context.

> _Context for Copilot: The `@reactour/tour` package is already installed (`frontend/package.json`). The frontend uses React Router v7 (`react-router-dom`) and MUI v7. A "first run" flag can be stored in `localStorage` or as a backend setting. The wizard should integrate with existing RTK Query mutations for settings and share creation._

## 🛠️ Technical Specifications

- **Inputs:**
  - User explicit trigger: "Run Setup Wizard" button in Settings or an empty-state screen
  - Existing API: `GET /settings`, `PUT /settings`, `POST /shares`, `GET /volumes`

- **Outputs:**
  - Multi-step wizard dialog: Step 1 — Network (interface selection), Step 2 — First share (volume picker, share name), Step 3 — User account (username/password confirm)
  - Guided tour (optional, dismissible): tooltip overlays on Dashboard, Shares, Settings, Volumes pages
  - Tour progress persisted in `localStorage` so it doesn't repeat on page reload
  - "Skip" and "Finish" controls on every step; no required fields block wizard completion

- **Dependencies:**
  - `frontend/src/pages/` — page components for tour step targets
  - `frontend/src/store/sratApi.ts` — RTK Query hooks (do **not** edit; use generated hooks)
  - `@reactour/tour` — already installed
  - `react-hook-form` — for wizard form steps (already installed)

## 📝 Task List

- [x] Task 1: Create `frontend/src/components/wizard/SetupWizard.tsx` — 4-step dialog using MUI `Stepper` + `Dialog`; replaces both `BaseConfigModal.tsx` and `TelemetryModal.tsx`. Wired into App.tsx via `WizardOpenContext`.
- [x] Task 1.1: Security step (hostname, workgroup, admin password) — replaces `BaseConfigModal.tsx`
- [x] Task 1.2: Network step — fetch available network interfaces via `GET /nics`, bind_all_interfaces checkbox, interfaces autocomplete
- [x] Task 1.3: First Share step — enter share name (optional); if provided, calls `POST /share`
- [x] Task 1.4: Security step handles admin password change with validation (no "changeme!", min 6 chars, confirm match)
- [x] Task 1.5: Telemetry step — RadioGroup with `Telemetry_mode.All/Errors/Disabled`; handles no-internet state
- [x] Task 2: Implement first-launch detection logic —  auto-opens wizard when default password / missing hostname / telemetry=Ask+internet
- [x] Task 3: Wire wizard steps to existing RTK Query mutations: `usePutApiSettingsMutation`, `usePutApiUseradminMutation`, `usePostApiShareMutation`
- [x] Task 4: Add "Run Setup Wizard" button to the Settings page (`frontend/src/pages/settings/Settings.tsx`) — placed in search bar row
- [x] Task 5: Create `@reactour/tour` integration — tour steps already implemented per-page in `*TourStep.tsx` files and wired into `NavBar.tsx` via `useTour` + `setSteps`. `TourProvider` is set up in `index.tsx`. Tour utility events in `frontend/src/utils/TourEvents.ts`.
- [x] Task 6: Add a "?" / Help button — already implemented in `NavBar.tsx` (`HelpIcon`/`HelpOutlinedIcon` toggle button wired to `setTourOpen`).
- [x] Task 7: Unit tests — `frontend/src/components/wizard/__tests__/SetupWizard.test.tsx` — wizard renders all 4 step labels, "Skip" closes without mutations, open=false hides content, 5 tests all pass
- [x] Task 8: Unit tests — tour step definitions render target selectors, "Close Tour" sets `localStorage` flag (step-definition coverage exists for Dashboard/Shares/Volumes/Settings; `NavBar.test.tsx` asserts `srat_tour_seen` is written when tour closes)
- [x] Task 9: Accessibility — wizard dialog now has explicit ARIA labeling (`aria-labelledby`, `aria-describedby`) and stepper has `aria-label`; tour toggle button has explicit accessible label
- [x] Task 10: Documentation — added a "Getting Started" section to frontend `README.md` describing wizard and guided tour behavior

## 🧠 Implementation Notes (Copilot Context)

### Current State (as of review)

**Already implemented:**
- `@reactour/tour` `TourProvider` is set up in `frontend/src/index.tsx`
- Tour steps exist as per-page files: `DashboardTourStep.tsx`, `SharesTourStep.tsx`, `VolumesTourStep.tsx`, `UsersSteps.tsx`, `SettingsTourStep.tsx`
- The `NavBar.tsx` wires all tour steps together via `useTour` + `setSteps` when tab changes, and has a `?` help button (`HelpIcon`/`HelpOutlinedIcon`) using `setTourOpen`
- Tour target selectors use `data-tutor` attribute (NOT `data-tour` as originally documented)
- `frontend/src/utils/TourEvents.ts` provides tour event utilities
- `DashboardTourStep.test.tsx` provides a test for one set of tour steps
- `BaseConfigModal.tsx` — handles essential samba config (hostname, workgroup, admin password)
- `TelemetryModal.tsx` + `useTelemetryModal` hook — handles telemetry opt-in, already shown on first run

**Remaining work** is focused on the Setup Wizard (Tasks 1–4, 7, 9, 10).

### Wizard step structure

```tsx
const STEPS = [
  { label: 'Network', component: <NetworkStep /> },
  { label: 'First Share', component: <FirstShareStep /> },
  { label: 'User Account', component: <UserAccountStep /> },
];
```

Use MUI `Stepper` (horizontal for desktop, vertical for mobile) inside a `Dialog` with `fullWidth maxWidth="sm"`.

### Tour integration (IMPLEMENTED — do not re-implement)

Tour is already live. Target selectors use `data-tutor` (not `data-tour`):

```tsx
// Already implemented in NavBar.tsx
const { setIsOpen: setTourOpen, isOpen: isTourOpen } = useTour();
// Help button:
<IconButton onClick={() => setTourOpen(!isTourOpen)}>
  {isTourOpen ? <HelpIcon /> : <HelpOutlinedIcon />}
</IconButton>
```

Add `data-tutor="reactour__tab<TabID>__step<N>"` attributes to new UI elements.

### References

- [react-joyride](https://docs.react-joyride.com/) — alternative tour library (issue #82)
- [@reactour/tour](https://docs.react.tours/quickstart) — already installed (preferred)
- [issue #116](https://github.com/dianlight/srat/issues/116) — Gemini-generated wizard design

## 🔗 Code References & TODOs

- [ ] [srat#116](https://github.com/dianlight/srat/issues/116) — Configuration Wizards (RoadMap)
- [ ] [srat#82](https://github.com/dianlight/srat/issues/82) — Help screen or overlay help/tour (RoadMap)
- [x] `frontend/package.json` — `@reactour/tour` already installed
- [x] `frontend/src/pages/settings/Settings.tsx` — "Setup Wizard" button added to search bar row
- [x] `frontend/src/pages/` — `data-tutor` attributes already on key UI landmarks (Shares, Volumes, Dashboard, Settings, Users)
- [x] `frontend/src/components/wizard/SetupWizard.tsx` — created (4 steps: Security, Network, First Share, Telemetry)
- [x] `frontend/src/hooks/useSetupWizard.ts` — created (replaces useBaseConfigModal + useTelemetryModal)
- [x] `frontend/src/components/wizard/__tests__/SetupWizard.test.tsx` — created (5 tests)
- [x] `frontend/src/App.tsx` — `SetupWizard` fully replaces `BaseConfigModal` and `TelemetryModal`
- [x] `frontend/src/components/BaseConfigModal.tsx` — existing essential samba config modal (to be merged as wizard step 1.1)
- [x] `frontend/src/components/TelemetryModal.tsx` — existing telemetry modal (to be merged as wizard step 1.5)
- [x] `frontend/src/components/NavBar.tsx` — tour help button implemented with explicit aria-label; sets `srat_tour_seen` when tour is closed
- [x] `frontend/src/utils/TourEvents.ts` — tour event utilities
- [x] `frontend/src/index.tsx` — `TourProvider` already set up
