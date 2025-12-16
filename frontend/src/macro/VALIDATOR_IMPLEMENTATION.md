<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Macro Import Validator - Implementation Summary](#macro-import-validator---implementation-summary)
  - [Overview](#overview)
  - [Components Created](#components-created)
    - [1. **Validator Script** (`scripts/validate-macro-imports.js`)](#1-validator-script-scriptsvalidate-macro-importsjs)
    - [3. **Documentation** (`frontend/src/macro/README.md`)](#3-documentation-frontendsrcmacroreadmemd)
    - [4. **NPM Script** (updated `frontend/package.json`)](#4-npm-script-updated-frontendpackagejson)
  - [Files Modified](#files-modified)
  - [Files Created](#files-created)
  - [Usage](#usage)
    - [Manual Validation](#manual-validation)
    - [Direct Script Usage](#direct-script-usage)
    - [CI/CD Integration](#cicd-integration)
  - [Example Correct Imports](#example-correct-imports)
  - [Validation Results](#validation-results)
  - [Next Steps](#next-steps)
  - [Technical Details](#technical-details)
    - [Regex Pattern](#regex-pattern)
    - [Multi-line Support](#multi-line-support)
  - [Limitations & Future Improvements](#limitations--future-improvements)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Macro Import Validator - Implementation Summary

## Overview

A validation system for ensuring all imports from the `frontend/src/macro/` directory use the proper `with { type: "macro" }` import assertion required by Bun's macro feature.

## Components Created

### 1. **Validator Script** (`scripts/validate-macro-imports.js`)

A Bun script that:

- Recursively scans all TypeScript files in `src/`
- Identifies imports from the macro directory
- Validates each import has the proper assertion `with { type: "macro" }`
- Reports violations with file paths and line numbers
- Supports multi-line imports

**Features:**

- Excludes auto-generated files (files with `// @generated` comments)
- Excludes the `sratApi.ts` file (like Biome's configuration)
- Handles imports with or without trailing commas
- Supports assertions spanning multiple lines

### 3. **Documentation** (`frontend/src/macro/README.md`)

Comprehensive guide explaining:

- What macros are and why the assertion is needed
- Macro files in the directory
- Import rules and examples
- How to use the validation tool
- Instructions for adding new macros

### 4. **NPM Script** (updated `frontend/package.json`)

Added a convenient npm script:

```bash
bun run validate-macros
```

## Files Modified

1. `/workspaces/srat/frontend/package.json` - Added `validate-macros` script
2. `/workspaces/srat/frontend/src/pages/volumes/components/HDIdleDiskSettings.tsx` - Fixed macro import
3. `/workspaces/srat/frontend/src/pages/volumes/components/SmartStatusPanel.tsx` - Fixed macro import

## Files Created

1. `/workspaces/srat/frontend/src/macro/README.md` - Documentation
2. `/workspaces/srat/scripts/validate-macro-imports.js` - Validator script

## Usage

### Manual Validation

```bash
cd frontend
bun run validate-macros
```

Output on success:

```plaintext
✅ All macro imports are properly validated!
```

### Direct Script Usage

```bash
cd frontend
bun run ../scripts/validate-macro-imports.js
```

### CI/CD Integration

Add to GitHub Actions workflow or pre-commit hooks:

```bash
#!/bin/bash
cd frontend
bun run validate-macros || exit 1
```

## Example Correct Imports

```typescript
// Single-line import
import { getApiUrl } from "../macro/Environment" with { type: "macro" };

// Multi-line import (with proper assertion)
import {
  getApiUrl,
  getServerEventBackend,
} from "../macro/Environment" with { type: "macro" };

// Named imports
import { getCompileYear } from "../macro/CompileYear" with { type: "macro" };

// Multiple assertions (if needed)
import { getApiUrl } from "../macro/Environment" with {
  type: "macro",
  // other properties if needed
};
```

## Validation Results

All macro imports in the codebase have been verified and corrected:

- ✅ `src/components/Footer.tsx` - GitCommitHash, CompileYear, Environment
- ✅ `src/hooks/useRollbarTelemetry.ts` - Environment
- ✅ `src/pages/volumes/components/HDIdleDiskSettings.tsx` - Environment
- ✅ `src/pages/volumes/components/SmartStatusPanel.tsx` - Environment
- ✅ `src/store/emptyApi.ts` - Environment
- ✅ `src/store/sseApi.ts` - Environment

## Next Steps

1. **Pre-commit Integration**: Add `bun run validate-macros` to pre-commit hooks
2. **IDE Support**: Consider creating an ESLint or IDE-specific rule
3. **Automatic Fixing**: Could extend to automatically add/fix assertions
4. **Documentation**: Update project README with macro import guidelines

## Technical Details

### Regex Pattern

The validator uses this regex to detect macro assertions:

```javascript
/with\s*{\s*type\s*:\s*["']macro["']\s*,?\s*}/;
```

This pattern matches:

- `with { type: "macro" }`
- `with { type: 'macro' }`
- `with { type: "macro", }` (trailing comma)
- Variations with different whitespace/tabs/newlines

### Multi-line Support

The validator checks up to 5 lines after the import statement to account for assertions on the next line, handling Biome's formatting:

```typescript
import { getGitCommitHash } from "../macro/GitCommitHash.ts" with { type: "macro" };
```

## Limitations & Future Improvements

1. **ESLint Rule**: Could add an ESLint rule for IDE feedback during development
2. **Automatic Fixing**: Could extend to automatically add/fix assertions
3. **Configuration**: Could make excluded files configurable
