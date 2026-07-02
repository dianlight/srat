<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [TypeScript 7.0 Migration Guide](#typescript-70-migration-guide)
  - [Current Status: ✅ TypeScript 7.0 RC](#current-status--typescript-70-rc)
    - [✅ Completed Changes](#-completed-changes)
      - [1. **TypeScript 7.0 RC Peer Dependency**](#1-typescript-70-rc-peer-dependency)
      - [2. **Updated tsgo (Native Preview)**](#2-updated-tsgo-native-preview)
      - [3. **Configuration Cleanup**](#3-configuration-cleanup)
      - [4. **Documentation**](#4-documentation)
    - [🚧 Still TODO](#-still-todo)
      - [Enable `noUncheckedIndexedAccess: true`](#enable-nouncheckedindexedaccess-true)
  - [TypeScript 7.0 Key Changes](#typescript-70-key-changes)
    - [Go-Based Compiler (tsgo)](#go-based-compiler-tsgo)
    - [Performance Improvements](#performance-improvements)
    - [Configuration & Defaults](#configuration--defaults)
    - [Removed Features (No Longer Supported)](#removed-features-no-longer-supported)
  - [Migration Checklist](#migration-checklist)
    - [Phase 1: TS 7.0 RC Upgrade (✅ Complete)](#phase-1-ts-70-rc-upgrade--complete)
    - [Phase 2: Code Refactoring (📋 Planned)](#phase-2-code-refactoring--planned)
    - [Phase 3: Testing & Validation](#phase-3-testing--validation)
  - [Testing the Changes](#testing-the-changes)
  - [References](#references)
  - [Notes](#notes)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# TypeScript 7.0 Migration Guide

This document tracks the migration from TypeScript 6.0 final to TypeScript 7.0 RC (Go-based).

## Current Status: ✅ TypeScript 7.0 RC

> **Update July 2026**: TypeScript 7.0 RC (`7.0.1-rc`) released with the Go-based `tsgo` compiler (`@typescript/native-preview`). The project has been upgraded to use the RC version.

> **MUI v9 Migration (2026)**: All MUI packages have been upgraded to v9 (`@mui/material`, `@mui/icons-material`, `@mui/x-charts`, `@mui/x-tree-view`, `react-hook-form-mui`). All deprecated APIs migrated: `InputProps`/`inputProps`/`InputLabelProps` → `slotProps`, `renderTags` → `renderValue`, deprecated icon `*Outline` → `*Outlined` suffixes, `disableEscapeKeyDown` → `onClose` reason check, `paragraph` prop removed from `Typography`. See [issue #589](https://github.com/dianlight/srat/issues/589).

### ✅ Completed Changes

#### 1. **TypeScript 7.0 RC Peer Dependency**

- ✅ Updated `peerDependencies` from `"typescript": "^6.0.2"` to `"typescript": "^7.0.1-rc"`
- ✅ All type-checking passes with `tsgo --noEmit`

#### 2. **Updated tsgo (Native Preview)**

- ✅ Updated `@typescript/native-preview` (tsgo) to `7.0.0-dev.20260701.1` (latest dev build)
- ✅ All `.mise.toml` build/test tasks run `tsgo --noEmit` before executing

#### 3. **Configuration Cleanup**

- ✅ Updated `tsconfig.json` header and comments to reflect TS 7.0
- ✅ `esModuleInterop` comment now reflects TS 7.0 default
- ✅ Removed stale TS 6.0 references from config comments

#### 4. **Documentation**

- ✅ Updated this migration guide to reflect TS 7.0 RC status
- ✅ Updated `.opencode/instructions/` to reference TypeScript 7.0

### 🚧 Still TODO

#### Enable `noUncheckedIndexedAccess: true`

This strict flag requires refactoring indexed access patterns in several files before it can be safely enabled:

**Files requiring changes:**

1. **Dashboard Metrics** (~6 locations)
   - `src/pages/DashBoard/DiskHealthMetrics.tsx` - Object property access without guards
   - `src/pages/DashBoard/NetworkHealthMetrics.tsx` - Similar pattern

   **Pattern to fix:**

   ```typescript
   // Before
   newHistory[deviceName].read_iops = updateHistory(...);

   // After
   if (newHistory[deviceName]) {
     newHistory[deviceName].read_iops = updateHistory(...);
   }
   ```

2. **Tree View Components** (~4 locations)
   - `src/components/SharesTreeView.tsx` - Array indexing in sort callbacks
   - `src/components/VolumesTreeView.tsx` - Similar pattern

   **Pattern to fix:**

   ```typescript
   // Before
   Object.entries(groups).sort((a, b) => a[0].localeCompare(b[0]));

   // After
   Object.entries(groups).sort((a, b) =>
     (a[0] ?? "").localeCompare(b[0] ?? ""),
   );
   ```

3. **Store Utilities** (~3 locations)
   - `src/store/sseApi.ts` - Dynamic object key assignment
   - `src/store/mdcSlice.ts` - Uint8Array indexing

   **Pattern to fix:**

   ```typescript
   // Before
   const versionByte = bytes[6];
   if (versionByte !== undefined) {...}

   // After
   const versionByte = bytes[6];
   if (versionByte != null) {...}
   // Or use optional chaining: const versionByte = bytes[6] ?? defaultValue;
   ```

**Estimated effort:** 2-3 hours of refactoring

## TypeScript 7.0 Key Changes

### Go-Based Compiler (tsgo)

- **7-10x faster type-checking** using the Go-based native compiler
- Full API compatibility with TypeScript 6.0+ codebase
- `bun tsgo --noEmit` used for type-checking (replaces `tsc`)
- **Stricter `import type` enforcement**: any symbol used only in type positions must use `import type`

### Performance Improvements

- **7-10x faster cold builds** with native (Go) compiler
- **Multi-threaded type-checking** by default
- **Smarter incremental builds** with improved caching
- **Reduced memory usage** compared to JavaScript-based compiler

### Configuration & Defaults

- `esModuleInterop: true` is now the **default** in TS 7.0
- `noUncheckedSideEffectImports: true` (enabled by default in TS 6.0, carried forward)
- `types: []` continues to provide 20-50% build speed improvement
- All TS 6.0 deprecated flags have been **removed** (not just deprecated)

### Removed Features (No Longer Supported)

- ❌ `experimentalDecorators` - Use native decorators instead
- ❌ `useDefineForClassFields: false` - ES2022+ requires default `true`
- ❌ `target: es5` - ES2015+ is the minimum
- ❌ Classic module resolution - Use `bundler` or `node`
- ❌ AMD/UMD module emit - ESM and CommonJS only
- ❌ `baseUrl` - No longer required for path mappings
- ❌ `outFile` - No longer supported
- ❌ Import assertion syntax (`import ... assert { ... }`) - Use import attributes instead
- ❌ `--downlevelIteration` - No longer needed with ES2015+

## Migration Checklist

### Phase 1: TS 7.0 RC Upgrade (✅ Complete)

- [x] Update peer dependency to `typescript: ^7.0.1-rc`
- [x] Update `@typescript/native-preview` (tsgo) to latest dev build
- [x] Update `tsconfig.json` comments and documentation
- [x] Verify type-checking passes with `tsgo --noEmit`
- [x] Update instruction files

### Phase 2: Code Refactoring (📋 Planned)

- [ ] Enable `noUncheckedIndexedAccess`
- [ ] Refactor dashboard metrics components
- [ ] Refactor tree view components
- [ ] Refactor store utilities
- [ ] Run full test suite
- [ ] Verify build performance improvements

### Phase 3: Testing & Validation

- [ ] Run `bun tsgo --noEmit` to verify type checking
- [ ] Run frontend tests
- [ ] Run production build
- [ ] Measure build time improvements

## Testing the Changes

```bash
# Type check
cd frontend
bun tsgo --noEmit

# Run tests
bun test

# Production build
bun run build

# Development server
bun run dev
```

## References

- [TypeScript 7.0 RC Release](https://devblogs.microsoft.com/typescript/announcing-typescript-7-0-rc/)
- [TypeScript 7.0 (Go-based) Discussion](https://github.com/microsoft/typescript-go/discussions/825)
- [TypeScript 6.0 Final Release](https://devblogs.microsoft.com/typescript/announcing-typescript-6-0/)
- [TypeScript 6.0 Documentation](https://www.typescriptlang.org/docs/handbook/release-notes/typescript-6-0.html)

## Notes

- This project uses `@typescript/native-preview` (tsgo) as the TypeScript 7.0 Go-based compiler
- The `bun tsgo` command is used for type checking (not regular `tsc`)
- All build scripts use `tsgo --noEmit`
- TypeScript 7.0 is the first Go-native TypeScript compiler, offering 7-10x faster type-checking
