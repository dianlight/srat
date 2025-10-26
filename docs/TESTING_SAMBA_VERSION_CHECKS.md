<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Samba Version Checks - Test & Verification Guide](#samba-version-checks---test--verification-guide)
  - [Quick Start](#quick-start)
    - [Verify Implementation](#verify-implementation)
    - [Compilation Check](#compilation-check)
  - [Unit Test Coverage](#unit-test-coverage)
    - [Test osutil Version Functions](#test-osutil-version-functions)
    - [Test Template Functions](#test-template-functions)
    - [Integration Test](#integration-test)
  - [Manual Testing](#manual-testing)
    - [1. Test Version Detection](#1-test-version-detection)
    - [2. Test Template Rendering](#2-test-template-rendering)
    - [3. Test Configuration Generation](#3-test-configuration-generation)
  - [Integration Testing Scenarios](#integration-testing-scenarios)
    - [Scenario 1: Samba 4.21.x](#scenario-1-samba-421x)
    - [Scenario 2: Samba 4.22.x](#scenario-2-samba-422x)
    - [Scenario 3: Samba 4.23.x](#scenario-3-samba-423x)
  - [Debugging](#debugging)
    - [Enable Debug Logging](#enable-debug-logging)
    - [Check Health Endpoint](#check-health-endpoint)
    - [Manual Version Parsing](#manual-version-parsing)
  - [Pre-Deployment Checklist](#pre-deployment-checklist)
  - [Troubleshooting](#troubleshooting)
    - [Issue: "smbd: command not found"](#issue-smbd-command-not-found)
    - [Issue: Version shows as empty](#issue-version-shows-as-empty)
    - [Issue: Template syntax error](#issue-template-syntax-error)
    - [Issue: Configuration option "unknown"](#issue-configuration-option-unknown)
  - [References](#references)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Samba Version Checks - Test & Verification Guide

## Quick Start

### Verify Implementation

```bash
cd /workspaces/srat

# Check modified files
git diff --name-only

# Show changes summary
git diff --stat
```

### Compilation Check

```bash
cd /workspaces/srat/backend/src

# Build all modified packages
go build ./internal/osutil ./tempio ./service

# Build full backend (if Makefile available)
cd /workspaces/srat/backend && make build
```

## Unit Test Coverage

### Test osutil Version Functions

Create `backend/src/internal/osutil/osutil_version_test.go`:

```go
package osutil

import (
	"testing"
)

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		majorRequired  int
		minorRequired  int
		expectedResult bool
	}{
		// Exact match
		{"4.23.0 >= 4.23", "4.23.0", 4, 23, true},
		{"4.21.0 >= 4.23", "4.21.0", 4, 23, false},

		// Higher version
		{"4.25.0 >= 4.23", "4.25.0", 4, 23, true},

		// Lower version
		{"4.20.0 >= 4.23", "4.20.0", 4, 23, false},

		// Same major, different minor
		{"4.22.5 >= 4.23", "4.22.5", 4, 23, false},
		{"4.23.5 >= 4.23", "4.23.5", 4, 23, true},

		// Invalid versions
		{"invalid >= 4.23", "invalid", 4, 23, false},
		{"empty >= 4.23", "", 4, 23, false},
		{"4 >= 4.23", "4", 4, 23, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This would need the versionAtLeast function
			// exported from tempio package for testing
			result := testVersionAtLeast(tt.version, tt.majorRequired, tt.minorRequired)
			if result != tt.expectedResult {
				t.Errorf("got %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

// Helper function for testing
func testVersionAtLeast(versionStr string, majorRequired, minorRequired int) bool {
	if versionStr == "" {
		return false
	}

	parts := strings.Split(versionStr, ".")
	if len(parts) < 2 {
		return false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	if major > majorRequired || (major == majorRequired && minor >= minorRequired) {
		return true
	}

	return false
}
```

### Test Template Functions

Create `backend/src/tempio/template_version_test.go`:

```go
package tempio

import (
	"testing"
)

func TestVersionAtLeastFunc(t *testing.T) {
	tests := []struct {
		versionStr       string
		majorRequired    int
		minorRequired    int
		expectedResult   bool
	}{
		{"4.23.0", 4, 23, true},
		{"4.23.0", 4, 22, true},
		{"4.23.0", 4, 24, false},
		{"4.21.0", 4, 23, false},
		{"invalid", 4, 23, false},
		{"", 4, 23, false},
	}

	for _, tt := range tests {
		result := versionAtLeast(tt.versionStr, tt.majorRequired, tt.minorRequired)
		if result != tt.expectedResult {
			t.Errorf("versionAtLeast(%q, %d, %d) = %v, want %v",
				tt.versionStr, tt.majorRequired, tt.minorRequired,
				result, tt.expectedResult)
		}
	}
}

func TestVersionBetweenFunc(t *testing.T) {
	tests := []struct {
		versionStr     string
		minMajor       int
		minMinor       int
		maxMajor       int
		maxMinor       int
		expectedResult bool
	}{
		// Within range
		{"4.22.0", 4, 21, 4, 23, true},
		{"4.23.0", 4, 21, 4, 23, true},
		{"4.21.0", 4, 21, 4, 23, true},

		// Outside range
		{"4.20.0", 4, 21, 4, 23, false},
		{"4.24.0", 4, 21, 4, 23, false},

		// Boundary
		{"4.21.0", 4, 21, 4, 21, true},
		{"4.22.0", 4, 21, 4, 21, false},
	}

	for _, tt := range tests {
		result := versionBetween(tt.versionStr, tt.minMajor, tt.minMinor, tt.maxMajor, tt.maxMinor)
		if result != tt.expectedResult {
			t.Errorf("versionBetween(%q, %d, %d, %d, %d) = %v, want %v",
				tt.versionStr, tt.minMajor, tt.minMinor, tt.maxMajor, tt.maxMinor,
				result, tt.expectedResult)
		}
	}
}
```

### Integration Test

Create `backend/src/service/samba_service_version_test.go`:

```go
package service

import (
	"strings"
	"testing"
)

func TestCreateConfigStreamWithVersionInfo(t *testing.T) {
	// This test would verify that:
	// 1. Version information is added to template context
	// 2. Template renders correctly with version checks
	// 3. No configuration errors occur

	// Note: This requires mocking the database and repository interfaces
	// Example structure:
	/*
	suite := &SambaServiceSuite{}
	suite.SetupTest()

	stream, err := suite.sambaService.CreateConfigStream()
	if err != nil {
		t.Fatalf("CreateConfigStream failed: %v", err)
	}

	config := string(*stream)

	// Verify version-appropriate options are present/absent
	if strings.Contains(config, "server smb transports") {
		// Should only be present in 4.23+
		if !strings.Contains(config, "# DEBUG: samba_version") {
			t.Error("QUIC option present but version info missing")
		}
	}
	*/
}
```

## Manual Testing

### 1. Test Version Detection

```bash
# Verify smbd --version output
smbd --version
# Expected output: Version 4.23.0

# Test the Go function (if available)
go test ./internal/osutil -v -run TestVersion
```

### 2. Test Template Rendering

```bash
# Create a test template file
cat > /tmp/test.gtpl << 'EOF'
{{if versionAtLeast .samba_version 4 23 -}}
QUIC_ENABLED=true
{{- else -}}
QUIC_ENABLED=false
{{- end }}
EOF

# Test with mock context (requires custom code)
```

### 3. Test Configuration Generation

```bash
# Start SRAT service
./srat start

# Request config stream
curl -X GET http://localhost:8000/samba/config

# Verify version-specific options
testparm -s /etc/samba/smb.conf | grep -i "smb transport"

# Check for any warnings
grep "WARNING\|ERROR" /etc/samba/smb.conf
```

## Integration Testing Scenarios

### Scenario 1: Samba 4.21.x

**Expected Behavior:**

- `server smb transports` NOT present
- `fruit:posix_rename = yes` IS present
- QUIC options NOT present

**Verification:**

```bash
testparm -s /etc/samba/smb.conf | grep -E "(smb transports|posix_rename|tls enable)"
# Should show only posix_rename
```

### Scenario 2: Samba 4.22.x

**Expected Behavior:**

- `server smb transports` NOT present
- `fruit:posix_rename` NOT present
- QUIC options NOT present (even if enabled in settings)

**Verification:**

```bash
testparm -s /etc/samba/smb.conf | grep -i "posix_rename"
# Should return nothing
```

### Scenario 3: Samba 4.23.x

**Expected Behavior:**

- `server smb transports = tcp` OR `server smb transports = tcp, quic` present
- `fruit:posix_rename` NOT present
- QUIC options (tls enable, etc.) present if enabled

**Verification:**

```bash
testparm -s /etc/samba/smb.conf | grep "server smb transports"
# Should show version-appropriate transports
```

## Debugging

### Enable Debug Logging

```bash
# Set log level to debug in config
log_level: "debug"

# Check logs for template rendering
journalctl -u srat -f | grep -i "template\|version"
```

### Check Health Endpoint

```bash
curl http://localhost:8000/health | jq '.samba_version, .samba_version_sufficient'
```

### Manual Version Parsing

```go
package main

import (
	"fmt"
	"strings"
	"strconv"
)

func main() {
	version := "4.23.0"
	parts := strings.Split(version, ".")

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])

	fmt.Printf("Version %s parsed as: Major=%d, Minor=%d\n", version, major, minor)
	fmt.Printf("Is >= 4.23? %v\n", major > 4 || (major == 4 && minor >= 23))
}
```

## Pre-Deployment Checklist

- [ ] All packages compile without errors
- [ ] Unit tests pass (if created)
- [ ] Template renders without syntax errors
- [ ] Version detection works on target Samba version
- [ ] Configuration options appropriate for Samba version
- [ ] No "unknown option" errors from testparm
- [ ] CHANGELOG.md updated with version information
- [ ] Documentation reviewed and complete
- [ ] Tested with at least one Samba version (preferably 4.21, 4.22, 4.23)

## Troubleshooting

### Issue: "smbd: command not found"

**Solution:** Samba must be installed. Install with:

```bash
apt-get install samba  # Debian/Ubuntu
apk add samba          # Alpine
```

### Issue: Version shows as empty

**Solution:** Check that smbd is in PATH and returns version:

```bash
which smbd
/usr/bin/smbd --version
```

### Issue: Template syntax error

**Solution:** Validate template syntax:

```bash
go test ./tempio -v
```

### Issue: Configuration option "unknown"

**Solution:** Check Samba version and update template checks as needed:

```bash
testparm -s /etc/samba/smb.conf 2>&1 | grep -i "unknown"
```

## References

- [Implementation Guide](./IMPLEMENTATION_SAMBA_VERSION_CHECKS.md)
- [Samba Version Checks Documentation](./SAMBA_VERSION_CHECKS.md)
- [Samba Release Notes](<https://wiki.samba.org/index.php/Samba_Features_added/changed_(by_release)>)
