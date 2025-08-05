# Bun Compatibility for Documentation Validation

## Summary

Added comprehensive bun support to the documentation validation system, making it compatible with both bun and npm package managers. **Enhanced to support bun as a complete Node.js replacement**, allowing developers to use bun as both JavaScript runtime and package manager.

## Changes Made

### 1. Updated Validation Script (`scripts/validate-docs.sh`)

#### Dependency Detection

- Added automatic detection of available package managers
- Prefers bun over npm when both are available
- Sets `PACKAGE_MANAGER` environment variable for use throughout the script
- Updated error messages to mention both package managers

#### Package Installation

- Modified `install_packages()` function to support both bun and npm
- Uses `bun add -g` for global package installation with bun
- Uses `npm install -g` for npm fallback
- Improved package detection logic for both package managers

#### Auto-fix Feature

- Updated `--fix` option to detect package manager
- Automatically installs prettier if not available
- Uses appropriate package manager for installation

#### Help Documentation

- Added dependency information to help output
- Explains package manager auto-detection behavior
- Updated usage examples and tips

### 2. Updated Makefile (`Makefile`)

#### `docs-install` Target

- Added automatic package manager detection
- Uses bun by default if available
- Falls back to npm if bun is not found
- Provides clear error message if neither is available

#### Additional Documentation Targets

- **`docs-check`**: Validates all dependencies are installed
- **`docs-validate`**: Runs validation with package manager detection
- **`docs-fix`**: Auto-fixes formatting with appropriate package manager
- **`docs-help`**: Shows all available documentation commands with current package manager status

### 3. Updated GitHub Workflow (`.github/workflows/documentation.yml`)

#### Package Manager Setup

- Added bun setup action to all relevant jobs
- Updated all Node.js package installation steps
- Uses conditional logic to prefer bun over npm
- Maintains npm fallback for compatibility

### 4. Updated Documentation

#### Documentation Guidelines (`docs/DOCUMENTATION_GUIDELINES.md`)

- Added note about package manager support
- Updated installation instructions
- Mentioned automatic detection behavior

#### Setup Summary (`DOCUMENTATION_VALIDATION_SETUP.md`)

- Updated feature descriptions to mention bun support
- Added package manager compatibility information

## Benefits

### Performance

- **Faster installations**: Bun is significantly faster than npm for package installation
- **Better caching**: Bun's caching mechanism reduces redundant downloads
- **Parallel execution**: Bun installs packages in parallel by default
- **Runtime efficiency**: Bun as JavaScript runtime is faster than Node.js
- **Single tool**: Bun eliminates the need for separate Node.js + npm installation

### Developer Experience

- **Automatic detection**: No configuration needed - works out of the box
- **Consistent with project**: Aligns with frontend tooling already using bun
- **Fallback support**: Still works in environments without bun

### CI/CD Improvements

- **Faster workflows**: Reduced package installation time in GitHub Actions
- **Reliable execution**: Maintains npm fallback for maximum compatibility
- **Consistent tooling**: Uses same package manager across project

## Usage Examples

### Local Development

```bash
# Check dependencies first
make docs-check

# Works with both bun and npm automatically
make docs-install
make docs-validate
make docs-fix

# Get help on available commands
make docs-help

# Script automatically detects package manager
./scripts/validate-docs.sh
./scripts/validate-docs.sh --fix
```

### Manual Package Manager Selection

```bash
# Force bun usage (if available)
bun add -g markdownlint-cli2 markdown-link-check cspell prettier

# Force npm usage
npm install -g markdownlint-cli2 markdown-link-check cspell prettier
```

## Compatibility Matrix

| Environment    | Runtime | Package Manager | Status       | Notes                                |
| -------------- | ------- | --------------- | ------------ | ------------------------------------ |
| bun only       | bun     | bun             | ✅ Optimal   | Complete Node.js replacement         |
| Node.js + bun  | node    | bun             | ✅ Preferred | Fast package management              |
| Node.js + npm  | node    | npm             | ✅ Standard  | Traditional setup                    |
| bun + npm      | bun     | npm             | ✅ Hybrid    | Runtime modern, packages traditional |
| No runtime     | none    | any             | ❌ Error     | JavaScript runtime required          |
| Legacy systems | node    | npm             | ✅ Supported | Always available                     |

## Implementation Details

### Detection Logic

1. **JavaScript Runtime Detection**:
   - Check for `node` command availability
   - If not found, check for `bun` command availability
   - If found, set `NODE_RUNTIME=bun` (bun as Node.js alternative)
   - If neither found, report error with installation instructions

2. **Package Manager Detection**:
   - Check for `bun` command availability (preferred)
   - If found, set `PACKAGE_MANAGER=bun`
   - Otherwise, check for `npm` command availability
   - If found, set `PACKAGE_MANAGER=npm`
   - If neither found, report error with installation instructions

### Installation Strategy

- **Global packages**: Uses `-g` flag for both package managers
- **Verification**: Checks command availability after installation
- **Error handling**: Provides clear error messages for failures
- **Logging**: Reports which package manager is being used

### GitHub Actions Integration

- **Setup actions**: Both `setup-node` and `setup-bun` are configured
- **Conditional installation**: Uses shell logic to detect and use appropriate manager
- **Caching**: Leverages bun's built-in caching for performance
- **Fallback safety**: Ensures npm is always available as backup

## Future Considerations

### Potential Enhancements

- **Package-lock support**: Consider using bun.lockb or package-lock.json
- **Local installations**: Support for project-local tool installations
- **Version pinning**: Lock specific versions of documentation tools
- **Custom registries**: Support for private or custom package registries

### Monitoring Points

- **Performance metrics**: Track installation times in CI
- **Compatibility issues**: Monitor for package manager specific problems
- **User feedback**: Gather feedback on tool preference and issues

---

**Implementation Date**: August 5, 2025
**Status**: ✅ Complete and tested
**Compatibility**: bun ≥ 1.0, npm ≥ 6.0, Node.js ≥ 18.0
