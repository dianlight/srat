// Package lib provides a Backend implementation that loads the smartmon wrapper
// library via purego (no CGO required). It is available on Linux and macOS.
//
// Build the wrapper library once:
//
//	scripts/setup-lib-backend.sh
//
// The script downloads the correct dianlight/smartmontools-sdk release for the
// current platform, installs the missing smartmon_config.h, and compiles the
// thin C++ wrapper into backends/lib/sdk/libsmartmon_go.{so,dylib}.
//
// # Library resolution order
//
//  1. [WithLibraryPath] option — always takes precedence.
//  2. SMARTMON_LIB_PATH environment variable.
//     • File exists → used directly.  A warning is logged if a library is also
//     found in a different standard system directory.
//     • File missing → warning logged; falls through to step 3.
//  3. Standard system paths — dynamic-linker names (LD_LIBRARY_PATH /
//     DYLD_LIBRARY_PATH / rpath) followed by well-known absolute paths such as
//     /usr/local/lib and /opt/homebrew/lib.
package lib

import smtypes "github.com/dianlight/smartmontools-go/internal/types"

// Shared interface aliases keep the lib backend decoupled from the root package.
type (
	LogAdapter = smtypes.LogAdapter
	Backend    = smtypes.Backend
)

// Shared type aliases reuse the module's SMART domain model in the lib backend.
type (
	Device          = smtypes.Device
	SMARTInfo       = smtypes.SMARTInfo
	SmartStatus     = smtypes.SmartStatus
	SmartSupport    = smtypes.SmartSupport
	AtaSmartData    = smtypes.AtaSmartData
	SelfTestInfo    = smtypes.SelfTestInfo
	DiscoveryResult = smtypes.DiscoveryResult
)
