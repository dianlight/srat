# [FEATURE]: Configuration Wizard and Guided Help Tour

**Target Repo:** `srat`  **Status:** 📅 Planned  **Issue Link:** [srat#116](https://github.com/dianlight/srat/issues/116) · [srat#82](https://github.com/dianlight/srat/issues/82)

## 🎯 Objective

Improve the onboarding experience for new users by adding two complementary UI features:

1. **Configuration Wizard** — a step-by-step modal that guides users through initial setup (network interfaces, first share, user account) on first launch or via an explicit "Setup Wizard" button.
2. **Guided Help / UI Tour** — an optional overlay tour (using `@reactour/tour`, already in `package.json`) that highlights key UI elements with tooltips and context.

> _Context for Copilot: The `@reactour/tour` package is already installed (`frontend/package.json`). The frontend uses React Router v7 (`react-router-dom`) and MUI v7. A "first run" flag can be stored in `localStorage` or as a backend setting. The wizard should integrate with existing RTK Query mutations for settings and share creation._

## 🛠️ Technical Specifications

- **Inputs:**
  - First-launch detection: `localStorage` flag `srat_wizard_seen` or backend setting
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

- [ ] Task 1: Create `frontend/src/components/wizard/SetupWizard.tsx` — multi-step dialog using MUI `Stepper` + `Dialog`; steps: Network → First Share → User Account
- [ ] Task 2: Implement first-launch detection logic — check `localStorage.getItem('srat_wizard_seen')`; if absent, auto-open wizard after initial data load
- [ ] Task 3: Wire wizard steps to existing RTK Query mutations: `useUpdateSettingsMutation`, `useCreateShareMutation`
- [ ] Task 4: Add "Run Setup Wizard" button to the Settings page (`frontend/src/pages/settings/Settings.tsx`)
- [ ] Task 5: Create `frontend/src/components/tour/AppTour.tsx` — `@reactour/tour` integration with step definitions for Dashboard, Shares list, Settings sections
- [ ] Task 6: Add a "?" / Help button (or floating action) to trigger the tour from any page; persist tour-seen state in `localStorage`
- [ ] Task 7: Unit tests — wizard renders correct steps, "Skip" dismisses without mutation, "Finish" submits mutations in order
- [ ] Task 8: Unit tests — tour step definitions render target selectors, "Close Tour" sets `localStorage` flag
- [ ] Task 9: Accessibility — ensure wizard and tour are keyboard-navigable and screen-reader-friendly (ARIA labels)
- [ ] Task 10: Documentation — add a "Getting Started" section to frontend `README.md` describing the wizard and tour

## 🧠 Implementation Notes (Copilot Context)

### Wizard step structure

```tsx
const STEPS = [
  { label: 'Network', component: <NetworkStep /> },
  { label: 'First Share', component: <FirstShareStep /> },
  { label: 'User Account', component: <UserAccountStep /> },
];
```

Use MUI `Stepper` (horizontal for desktop, vertical for mobile) inside a `Dialog` with `fullWidth maxWidth="sm"`.

### First-launch detection

```ts
const hasSeenWizard = localStorage.getItem('srat_wizard_seen') === 'true';
if (!hasSeenWizard) {
  // dispatch open wizard action
}
```

Set `localStorage.setItem('srat_wizard_seen', 'true')` on wizard close (skip or finish).

### Tour integration

```tsx
import { TourProvider, useTour } from '@reactour/tour';

const steps = [
  { selector: '[data-tour="dashboard"]', content: 'This is the Dashboard...' },
  { selector: '[data-tour="shares"]', content: 'Manage your Samba shares here.' },
  // ...
];
```

Add `data-tour="..."` attributes to target elements in existing page components.

### References

- [react-joyride](https://docs.react-joyride.com/) — alternative tour library (issue #82)
- [@reactour/tour](https://docs.react.tours/quickstart) — already installed (preferred)
- [issue #116](https://github.com/dianlight/srat/issues/116) — Gemini-generated wizard design

## 🔗 Code References & TODOs

- [ ] [srat#116](https://github.com/dianlight/srat/issues/116) — Configuration Wizards (RoadMap)
- [ ] [srat#82](https://github.com/dianlight/srat/issues/82) — Help screen or overlay help/tour (RoadMap)
- [ ] `frontend/package.json` — `@reactour/tour` already installed
- [ ] `frontend/src/pages/settings/Settings.tsx` — add "Run Setup Wizard" entry point
- [ ] `frontend/src/pages/` — add `data-tour` attributes to key UI landmarks
