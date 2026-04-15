# [REFACTOR]: Upgrade MUI to v9 and Optimize Frontend Dependencies

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** _TBD_

## 🎯 Objective

Upgrade `@mui/material`, `@mui/icons-material`, `@mui/x-charts`, and `@mui/x-tree-view` from v7/v8 to v9,
along with all peer libraries that declare MUI v9 compatibility requirements. Apply the v9 codemods,
fix all breaking changes, and leverage v9 quality-of-life improvements (improved `slot`/`slotProps` APIs,
accessibility improvements, and removed deprecated APIs). Also audit and update all other frontend
dependencies for compatibility and optimization opportunities.

## 🛠️ Technical Specifications

- **Inputs:** Current package versions (`@mui/material ^7.3.9`, `@mui/x-charts ^8.28.2`, `@mui/x-tree-view ^8.27.2`)
- **Outputs:** All MUI packages at v9, passing TypeScript check (`bun tsgo --noEmit`), passing tests (`mise run //frontend:test`), clean lint (`mise run //frontend:lint`)
- **Dependencies:**
  - `@mui/material` → `^9.0.0`
  - `@mui/icons-material` → `^9.0.0`
  - `@mui/x-charts` → `^9.0.0`
  - `@mui/x-tree-view` → `^9.0.0`
  - `@emotion/react`, `@emotion/styled` (peer dependency, check compatibility)
  - `mui-chips-input` (v8 currently, check if v9 MUI peer is supported)
  - `material-ui-confirm` (v4 currently, check if v9 MUI peer is supported)
  - `react-hook-form-mui` (v8 currently, check if v9 MUI peer is supported)

## 📝 Task List

- [ ] Task 1: Research all package compatibility for MUI v9 peers (`mui-chips-input`, `material-ui-confirm`, `react-hook-form-mui`, `@uiw/react-md-editor`, `openapi-explorer`)
- [ ] Task 2: Run MUI v9 codemod for Material UI: `npx @mui/codemod@latest v9.0.0/preset-safe ./src`
- [ ] Task 3: Run MUI X Charts v9 codemod: `npx @mui/x-codemod@latest v9.0.0/charts/rename-classes ./src`
- [ ] Task 4: Bump `@mui/material`, `@mui/icons-material` to `^9.0.0` and install
- [ ] Task 5: Bump `@mui/x-charts` to `^9.0.0` and `@mui/x-tree-view` to `^9.0.0` and install
- [ ] Task 6: Fix breaking change – `disableEscapeKeyDown` removed from Dialog/Modal (found in `TelemetryModal.tsx`, `BaseConfigModal.tsx`) → use `onClose` reason check instead
- [ ] Task 7: Fix breaking change – `GridLegacy` removed, migrate all usages to `Grid` with `size` prop
- [ ] Task 8: Fix breaking change – deprecated `InfoOutline`/`OutlineXxx` icon exports removed → use `Outlined` suffix equivalents
- [ ] Task 9: Fix breaking change – `TreeViewBaseItem` removed → use `TreeViewDefaultItemModelProperties`
- [ ] Task 10: Fix breaking change – `useTreeViewApiRef` removed → use component-specific hooks
- [ ] Task 11: Fix breaking change – Charts `ChartContainer` → `ChartsContainer` rename (if used)
- [ ] Task 12: Fix breaking change – Charts CSS highlighted/faded classes → `data-highlighted`/`data-faded` attribute selectors
- [ ] Task 13: Bump and verify compatibility of `mui-chips-input`, `material-ui-confirm`, `react-hook-form-mui` (upgrade or replace if no v9 support)
- [ ] Task 14: Audit and update remaining frontend dependencies for latest stable versions (`react-toastify`, `@reduxjs/toolkit`, `react-router-dom`, etc.)
- [ ] Task 15: Apply any v9 optimization opportunities (new `slot`/`slotProps` API patterns, `nativeButton` prop on button-like components)
- [ ] Task 16: Run TypeScript check: `bun tsgo --noEmit` — fix all type errors
- [ ] Task 17: Run full frontend test suite: `mise run //frontend:test` — fix all regressions
- [ ] Task 18: Run lint: `mise run //frontend:lint` — fix all warnings/errors
- [ ] Task 19: Run frontend build: `mise run //frontend:build` — verify no build errors
- [ ] Task 20: Update `frontend/TYPESCRIPT_MIGRATION.md` and any docs that reference MUI version
- [ ] Task 21: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)

### Migration guides

- Material UI v7 → v9: <https://mui.com/material-ui/migration/upgrade-to-v9/>
- MUI X Charts v8 → v9: <https://mui.com/x/migration/migration-charts-v8/>
- MUI X Tree View v8 → v9: <https://mui.com/x/migration/migration-tree-view-v8/>

### Key breaking changes in Material UI v7 → v9

1. **Dialog/Modal `disableEscapeKeyDown` removed**: Replace with `onClose` reason check:
   ```tsx
   // Before
   <Dialog open={open} disableEscapeKeyDown onClose={handleClose}>
   // After
   <Dialog open={open} onClose={(_e, reason) => { if (reason !== 'escapeKeyDown') setOpen(false); }}>
   ```
   Affected files: `src/components/TelemetryModal.tsx`, `src/components/BaseConfigModal.tsx`

2. **`GridLegacy` removed**: Use `Grid` with `size` prop instead:
   ```tsx
   // Before
   <Grid item xs={12} sm={6}>
   // After
   <Grid size={{ xs: 12, sm: 6 }}>
   ```
   Check all `Grid` usages for remaining `xs/sm/md/lg/xl` props or `item` prop.

3. **Deprecated icon exports removed**: 23 icons with `Outline` (no "d") suffix are removed.
   Rename to `Outlined`:
   ```tsx
   // Before: InfoOutlineIcon → After: InfoOutlinedIcon
   ```

4. **`Autocomplete` freeSolo type changes**: `getOptionLabel` and `isOptionEqualToValue` accept
   updated generic `AutocompleteValueOrFreeSoloValueMapping<Value, FreeSolo>` types.

5. **ButtonBase click propagation from keyboard**: Enter/Spacebar events now bubble; disabled
   non-native buttons no longer fire event handlers.

6. **`ListItemIcon` default min-width** changed from `56px` to `36px`.

7. **`nativeButton` prop** available on all button-like components for SSR/hydration correctness.

### Key breaking changes in MUI X Charts v8 → v9

1. **`Chart` → `Charts` rename**: All `ChartContainer*` types renamed to `ChartsContainer*`.
   Run codemod: `npx @mui/x-codemod@latest v9.0.0/charts/rename-classes ./src`

2. **CSS classes reorganized**: e.g. `barElementClasses.root` → `barClasses.element`,
   `pieArcClasses.root` → `pieClasses.arc`. Codemod handles most cases.

3. **CSS highlighted/faded state classes removed**: Replace with attribute selectors:
   ```css
   /* Before */ .MuiBarElement-root.MuiBarElement-highlighted
   /* After */  .MuiBarElement-root[data-highlighted]
   ```

### Key breaking changes in MUI X Tree View v8 → v9

1. **`useTreeViewApiRef` removed**: Use `useRichTreeViewApiRef` or `useSimpleTreeViewApiRef`.
2. **`TreeViewBaseItem` removed**: Use `TreeViewDefaultItemModelProperties` or define own type.
3. **Item virtualization** added to `RichTreeViewPro` (requires container with fixed height).
4. **New `domStructure` and `itemHeight` props** for `RichTreeViewPro`.

### Codemods available

```bash
# Material UI v9 preset (handles most breaking changes automatically)
npx @mui/codemod@latest v9.0.0/preset-safe ./src

# Charts CSS class renames
npx @mui/x-codemod@latest v9.0.0/charts/rename-classes ./src
```

### Compatibility notes

- **`mui-chips-input`**: Currently `^8.0.0` — check if v9 peer is declared; may need to wait for author
  update or patch/replace with MUI Autocomplete + chips pattern.
- **`material-ui-confirm`**: Currently `^4.0.0` — verify MUI v9 peer compat on npm/GitHub.
- **`react-hook-form-mui`**: Currently `^8.2.0` — verify MUI v9 peer compat on npm/GitHub.
- **`@emotion/react` / `@emotion/styled`**: Both at `^11.14.x` — MUI v9 requires `^11.0.0`, no change needed.
- **`openapi-explorer`**: Not a direct MUI consumer; verify no style conflicts with MUI v9 theme.

### MUI v9 Grid usage note

The project already uses MUI Grid2 (`size` prop) per existing copilot instructions:
> MUI Grid: use the `size` prop (Grid2 default).
Check for any remaining `GridLegacy` imports and `xs/sm/md/lg/xl` shorthand props.

### Build commands

```bash
cd frontend
bun install                     # after bumping versions
bun tsgo --noEmit               # TypeScript check
mise run //frontend:test        # run tests
mise run //frontend:lint        # lint
mise run //frontend:build       # production build
```

## 🔗 Code References & TODOs

- `src/components/TelemetryModal.tsx` — uses `disableEscapeKeyDown` (must be removed)
- `src/components/BaseConfigModal.tsx` — uses `disableEscapeKeyDown` (must be removed)
- `frontend/TYPESCRIPT_MIGRATION.md` — update MUI version references
- `frontend/package.json` — all `@mui/*` packages to bump
- Check entire `src/` for `import.*GridLegacy`, `InfoOutline` (non-d suffix), `TreeViewBaseItem`,
  `useTreeViewApiRef`, `ChartContainer` (singular) usages via grep before/after codemods
