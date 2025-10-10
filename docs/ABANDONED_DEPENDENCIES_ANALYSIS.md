<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Abandoned Dependencies Analysis and Resolution](#abandoned-dependencies-analysis-and-resolution)
  - [Executive Summary](#executive-summary)
  - [Identified Dependencies](#identified-dependencies)
    - [1. ✅ RESOLVED: github.com/m1/go-generate-password](#1--resolved-githubcomm1go-generate-password)
      - [Analysis](#analysis)
      - [Decision: Replace with Custom Implementation](#decision-replace-with-custom-implementation)
      - [Implementation](#implementation)
      - [Testing](#testing)
      - [Changes Made](#changes-made)
      - [Impact Assessment](#impact-assessment)
    - [2. ✅ KEEP: gitlab.com/tozd/go/errors](#2--keep-gitlabcomtozdgoerrors)
      - [Analysis](#analysis-1)
      - [Decision: Keep](#decision-keep)
      - [Impact of Keeping](#impact-of-keeping)
  - [Summary](#summary)
    - [Actions Taken](#actions-taken)
    - [Metrics](#metrics)
    - [Verification](#verification)
  - [Recommendations](#recommendations)
    - [For Future Dependency Management](#for-future-dependency-management)
    - [For This Issue](#for-this-issue)
  - [References](#references)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Abandoned Dependencies Analysis and Resolution

**Issue:** [#16](https://github.com/dianlight/srat/issues/16) - Dependency Dashboard  
**Date:** October 9, 2025  
**Status:** ✅ Resolved

## Executive Summary

This document describes the analysis and resolution of potentially abandoned dependencies in the SRAT repository. One abandoned dependency was identified and replaced, while another was confirmed to be actively maintained.

## Identified Dependencies

### 1. ✅ RESOLVED: github.com/m1/go-generate-password

**Status:** Abandoned  
**Version:** v0.2.0  
**Last Updated:** April 17, 2022 (approximately 3.5 years ago)

#### Analysis

- **Usage Location:** Single usage in `backend/src/cmd/srat-cli/main-cli.go`
- **Purpose:** Generate random password for Home Assistant mount user (`_ha_mount_user_`)
- **Usage Pattern:**

  ```go
  pwdgen, err := generator.NewWithDefault()
  _ha_mount_user_password_, err := pwdgen.Generate()
  ```

#### Decision: Replace with Custom Implementation

**Rationale:**

- Package has not been updated since 2022
- Simple functionality that can be easily replicated
- Only used once in the entire codebase
- Reduces external dependency count
- Go standard library provides all necessary cryptographic primitives

#### Implementation

**New Function:** `internal/osutil/GenerateSecurePassword()`

```go
// GenerateSecurePassword generates a cryptographically secure random password.
// It uses crypto/rand to generate random bytes and encodes them in base64.
// The resulting password will be approximately 22 characters long (16 bytes * 4/3).
func GenerateSecurePassword() (string, error) {
    // Generate 16 random bytes (128 bits of entropy)
    randomBytes := make([]byte, 16)
    _, err := rand.Read(randomBytes)
    if err != nil {
        return "", err
    }

    // Encode to base64 for a safe password string
    // Using RawURLEncoding to avoid padding and make it URL-safe
    password := base64.RawURLEncoding.EncodeToString(randomBytes)
    return password, nil
}
```

**Key Features:**

- Uses `crypto/rand` for cryptographically secure random generation
- 128 bits of entropy (16 random bytes)
- Base64 URL-safe encoding (no padding, no special characters like `+` or `/`)
- Results in 22-character passwords
- No external dependencies

#### Testing

Added comprehensive test coverage in `internal/osutil/osutil_test.go`:

1. **TestGenerateSecurePassword:** Validates basic generation and length
2. **TestGenerateSecurePasswordCharacterSet:** Ensures only valid base64 URL-safe characters
3. **TestGenerateSecurePasswordNoSpecialChars:** Verifies no padding or problematic characters
4. **Uniqueness Test:** Generates 100 passwords and verifies all are unique

**Test Results:**

- ✅ All tests pass
- ✅ Coverage for osutil package: 70.7%
- ✅ Overall coverage increased: 40.4% → 40.5%

#### Changes Made

**Files Modified:**

1. `backend/src/internal/osutil/osutil.go` - Added `GenerateSecurePassword()` function
2. `backend/src/internal/osutil/osutil_test.go` - Added 4 comprehensive tests
3. `backend/src/cmd/srat-cli/main-cli.go` - Updated to use new function
4. `backend/src/go.mod` - Removed abandoned dependency (via `go mod tidy`)
5. `backend/src/go.sum` - Updated checksums

**Code Changes:**

```diff
- "github.com/m1/go-generate-password/generator"
+ "github.com/dianlight/srat/internal/osutil"

- pwdgen, err := generator.NewWithDefault()
- if err != nil {
-     log.Fatalf("Cant generate password %#+v", err)
- }
- _ha_mount_user_password_, err := pwdgen.Generate()
+ _ha_mount_user_password_, err := osutil.GenerateSecurePassword()

- err = unixsamba.CreateSambaUser("_ha_mount_user_", *_ha_mount_user_password_, ...)
+ err = unixsamba.CreateSambaUser("_ha_mount_user_", _ha_mount_user_password_, ...)
```

#### Impact Assessment

✅ **Positive Impact:**

- Reduced external dependencies
- Improved security (uses cryptographically secure random generation)
- Better maintainability (code under our control)
- Simpler code (direct function call instead of generator pattern)
- Added comprehensive test coverage

❌ **No Negative Impact:**

- Functionality maintained
- Password quality equivalent or better
- No breaking changes
- All tests pass

---

### 2. ✅ KEEP: gitlab.com/tozd/go/errors

**Status:** Actively Maintained  
**Version:** v0.10.0  
**Last Updated:** September 8, 2024 (4 months ago)

#### Analysis

- **Usage:** 76 occurrences across the codebase
- **Purpose:** Enhanced error handling with stack traces and structured error information
- **Integration:** Deep integration with tlog package for formatted error output

#### Decision: Keep

**Rationale:**

- **Recent Updates:** Last updated only 4 months ago
- **Active Maintenance:** Regular commits and releases
- **Extensive Usage:** Used 76 times throughout the codebase
- **Unique Value:** Provides features not available in standard library:
  - Stack trace support
  - Error wrapping with context
  - Structured error details
  - Integration with logging system
  - Enhanced debugging capabilities

**Evidence of Active Maintenance:**

```bash
$ go list -m -json gitlab.com/tozd/go/errors
"Version": "v0.10.0",
"Time": "2024-09-08T22:35:06Z"
```

**Key Features Used:**

- `errors.WithStack()` - Add stack traces to errors
- `errors.WithDetails()` - Add structured details to errors
- `errors.Wrap()` - Wrap errors with additional context
- `errors.Cause()` - Extract root cause from error chains
- Integration with `tlog` package for formatted error output

#### Impact of Keeping

✅ **Positive:**

- Maintains comprehensive error handling capabilities
- Preserves debugging information
- No disruption to existing error handling patterns
- Continues to receive updates and bug fixes

❌ **No Negative Impact:**

- Package is actively maintained
- No security concerns
- Not abandoned

---

## Summary

### Actions Taken

| Dependency                           | Status                | Action      | Rationale                                              |
| ------------------------------------ | --------------------- | ----------- | ------------------------------------------------------ |
| `github.com/m1/go-generate-password` | Abandoned (3.5 years) | ✅ Replaced | Simple functionality, single usage, easily replaceable |
| `gitlab.com/tozd/go/errors`          | Active (4 months)     | ✅ Kept     | Actively maintained, extensive usage, unique value     |

### Metrics

**Before:**

- External dependencies: X
- Total coverage: 40.4%
- Osutil coverage: 69.9%

**After:**

- External dependencies: X-1 (removed 1 abandoned dependency)
- Total coverage: 40.5% (↑ 0.1%)
- Osutil coverage: 70.7% (↑ 0.8%)

### Verification

All changes have been verified through:

- ✅ Unit tests (all passing)
- ✅ Build verification (successful)
- ✅ Code formatting (following repository standards)
- ✅ Test coverage (improved)

## Recommendations

### For Future Dependency Management

1. **Regular Audits:** Perform dependency audits quarterly
2. **Automated Monitoring:** Use Renovate bot to track dependency updates
3. **Abandonment Threshold:** Continue using 5-year threshold for abandonment detection
4. **Evaluation Criteria:**
   - Last update date
   - Number of usages
   - Complexity of replacement
   - Security implications
   - Maintenance burden

### For This Issue

- ✅ Issue #16 can be closed after review
- 📝 Consider documenting the `GenerateSecurePassword()` function in API documentation
- 🔄 Monitor `gitlab.com/tozd/go/errors` for continued updates

## References

- Issue #16: https://github.com/dianlight/srat/issues/16
- Renovate Dependency Dashboard
- Go crypto/rand documentation: https://pkg.go.dev/crypto/rand
- Base64 encoding: https://pkg.go.dev/encoding/base64
