// SPDX-License-Identifier: GPL-2.0-or-later

// Package lib provides a Backend implementation that loads the smartmon wrapper
// library via purego (no CGO required). It is available on Linux and macOS.
//
// Build the native core and the wrapper once, from the repository root:
//
//	cd native && ./autogen.sh --force && mkdir -p build && cd build && \
//	  ../configure --with-devel=yes --with-pic && make -C include && make -C lib libsmartmon.la
//	scripts/build-wrapper.sh
//
// This compiles the thin C++ wrapper (native/capi/smartmon_c_api.cpp) against
// libsmartmon.a and produces packaging/artifacts/lib/libsmartmon_go.{so,dylib}.
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

import smtypes "github.com/dianlight/smartmontools-sdk/bindings/go/v8/types"

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
