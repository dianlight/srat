//go:build linux

package osutil

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractMajorMinor(t *testing.T) {
	tests := []struct {
		name          string
		device        string
		expectedMajor int
		expectedMinor int
		expectError   bool
	}{
		{name: "loop zero", device: "/dev/loop0", expectedMajor: 7, expectedMinor: 0},
		{name: "typical value", device: "/dev/loop7", expectedMajor: 7, expectedMinor: 7},
		{name: "multi digit", device: "/dev/loop123", expectedMajor: 7, expectedMinor: 123},
		{name: "max int32 fits on all platforms", device: "/dev/loop2147483647", expectedMajor: 7, expectedMinor: 2147483647},
		{name: "overflow uint32", device: "/dev/loop4294967296", expectError: true},
		{name: "huge number truncates without bound check", device: "/dev/loop99999999999", expectError: true},
		{name: "non loop name", device: "/dev/sda1", expectError: true},
		{name: "empty device", device: "", expectError: true},
	}

	// Values that fit in uint32 but only fit in int on 64-bit platforms.
	// On 32-bit platforms math.MaxInt is 2147483647, so these must be rejected
	// by the explicit MaxInt bound check in extractMajorMinor.
	// wantMinor is int64 so the literals compile on both 32-bit and 64-bit.
	platformCases := []struct {
		name      string
		device    string
		wantMinor int64
	}{
		{name: "first value above 32-bit int", device: "/dev/loop2147483648", wantMinor: 2147483648},
		{name: "max uint32", device: "/dev/loop4294967295", wantMinor: 4294967295},
	}
	for _, tt := range platformCases {
		if int64(math.MaxInt) < tt.wantMinor {
			tests = append(tests, struct {
				name          string
				device        string
				expectedMajor int
				expectedMinor int
				expectError   bool
			}{name: tt.name + " overflows int", device: tt.device, expectError: true})
			continue
		}
		tests = append(tests, struct {
			name          string
			device        string
			expectedMajor int
			expectedMinor int
			expectError   bool
		}{name: tt.name, device: tt.device, expectedMajor: 7, expectedMinor: int(tt.wantMinor)})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, err := extractMajorMinor(tt.device)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMajor, major)
			assert.Equal(t, tt.expectedMinor, minor)
		})
	}
}
