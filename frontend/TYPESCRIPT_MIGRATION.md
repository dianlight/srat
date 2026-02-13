# TypeScript 6.0/7.0 Migration Guide

This document tracks the migration from TypeScript 5.x to TypeScript 6.0 Beta (tsgo preview) and preparation for TypeScript 7.0 (Go-based).

## Current Status: ✅ TypeScript 6.0 Compatible (Partial)

### ✅ Completed Changes

#### 1. **Removed Deprecated Compiler Options**
- ✅ Removed `experimentalDecorators: true` - Deprecated in TS 6.0, native decorators should be used
- ✅ Removed `useDefineForClassFields: false` - Deprecated, now defaults to true for ES2022+ class fields
- ✅ Updated `target` from `ES2021` to `ES2022` - Better alignment with modern ECMAScript features
- ✅ Updated `lib` from `ES2021` to `ES2022`

#### 2. **Enabled New Strict Flags**
- ✅ `noImplicitOverride: true` - Code already uses `override` keyword correctly (see `ErrorBoundary.tsx`)
- ✅ `types: []` - Already configured, provides 20-50% build performance improvement

#### 3. **Code Cleanup**
- ✅ Removed legacy `/// <reference types="bun-types" />` directive from `src/macro/__tests__/Environment.test.ts`

#### 4. **Documentation**
- ✅ Added inline comments explaining TS 6.0/7.0 changes
- ✅ Created this migration guide

### 🚧 TODO for Full TypeScript 7.0 Readiness

#### Enable `noUncheckedIndexedAccess: true`

This strict flag requires refactoring indexed access patterns in several files:

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
   Object.entries(groups).sort((a, b) => a[0].localeCompare(b[0]))
   
   // After
   Object.entries(groups).sort((a, b) => (a[0] ?? "").localeCompare(b[0] ?? ""))
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

## TypeScript 6.0 Beta Key Changes

Based on the official release notes:

### Deprecated Features (Will be removed in TS 7.0)
- ❌ `target: es5` - ES2015 (ES6) is now the minimum
- ❌ `classic` module resolution - Use `node` or `bundler`
- ❌ `--downlevelIteration` - No longer needed with ES2015+
- ❌ `amd`, `umd`, `systemjs` module targets - Use ESM
- ❌ `experimentalDecorators` - Use native decorators
- ❌ `useDefineForClassFields: false` - Should use default true

### New Defaults
- ✅ `strict: true` - Already enabled
- ✅ `module: esnext` - Already set to ESNext
- ✅ `types: []` - Already configured
- ✅ `noUncheckedSideEffectImports: true` - Enabled by default in TS 6.0

### Performance Improvements
- **20-50% faster builds** with `types: []` (already enabled)
- Better type inference with reduced context sensitivity
- Improved consistency in type checking

## Migration Checklist

### Phase 1: TypeScript 6.0 Beta (✅ Complete)
- [x] Remove deprecated compiler options
- [x] Update target to ES2022+
- [x] Enable `noImplicitOverride`
- [x] Remove legacy reference directives
- [x] Document changes

### Phase 2: Full TS 7.0 Readiness (🚧 In Progress)
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

- [TypeScript 6.0 Beta Release Notes](https://devblogs.microsoft.com/typescript/announcing-typescript-6-0-beta/)
- [TypeScript 7.0 (Go-based) Discussion](https://github.com/microsoft/typescript-go/discussions/825)
- [Migration Guide](https://www.typescriptlang.org/docs/handbook/release-notes/typescript-6-0.html)

## Notes

- This project uses `@typescript/native-preview: 7.0.0-dev.20260212.1` (tsgo in preview mode)
- The `bun tsgo` command is used for type checking (not regular `tsc`)
- All build scripts have been updated to use `tsgo --noEmit`
