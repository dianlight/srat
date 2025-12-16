## Macro Import Validator

This directory contains files related to validating and working with macro imports in the SRAT frontend.

### What are Macros?

Macros are special TypeScript files that use Bun's macro feature to execute at compile time. They are stored in this directory and must be imported with a special assertion to work correctly with the bundler.

### Files

- **Validator script**: `../../scripts/validate-macro-imports.js` - Validates all macro imports

### Macro Files

All files in this directory should be macros that export functions used throughout the application:

- `CompileYear.ts` - Returns the current year as a string at compile time
- `Environment.ts` - Returns environment variables at compile time
- `GitCommitHash.ts` - Returns the Git commit hash at compile time

### Import Rules

All imports from the macro directory **MUST** use the proper import assertion:

```typescript
// ✅ CORRECT
import { getApiUrl } from "../macro/Environment" with { type: "macro" };

// ❌ INCORRECT
import { getApiUrl } from "../macro/Environment";
```

The `with { type: "macro" }` assertion tells the bundler to execute this import at compile time, not runtime.

### Validation

To validate that all macro imports are correct, run:

```bash
cd frontend
bun run validate-macros
```

This script will:
1. Scan all TypeScript files in the `src/` directory
2. Find all imports from the macro directory
3. Verify each import has the proper `with { type: "macro" }` assertion
4. Report any violations with file and line information

### Adding New Macros

1. Create a new `.ts` file in this directory
2. Export one or more functions that will be executed at compile time
3. Import it in your application with the proper assertion:
   ```typescript
   import { yourFunction } from "../macro/YourMacro" with { type: "macro" };
   ```
4. Verify the import by running `bun run validate-macros`
