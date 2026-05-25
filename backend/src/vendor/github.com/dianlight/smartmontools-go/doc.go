/*
Package smartmontools provides Go bindings for interacting with smartmontools
and collecting S.M.A.R.T. data from storage devices.

The root package is a thin facade over the shared domain model in internal/types
and the default exec backend in backends/exec. NewClient creates a Client that
delegates SMART operations to a pluggable Backend implementation. By default it
uses ExecBackend, which shells out to the smartctl binary. Alternative backends
can be supplied with WithBackend.

# Features

  - Device scanning and discovery
  - SMART health status checking
  - Detailed SMART attribute reading
  - Disk type detection (SSD, HDD, NVMe, Unknown)
  - Rotation rate (RPM) information for HDDs
  - Temperature monitoring
  - Power-on time tracking
  - Self-test execution and progress monitoring
  - Device information retrieval
  - SMART support detection and management
  - Self-test availability checking
  - Standby mode detection for ATA-family devices
  - Efficient SMART monitoring with minimal disk I/O

# Backend Layout

The default smartctl-backed implementation lives in
github.com/dianlight/smartmontools-go/backends/exec. Shared types and
interfaces are hosted in an internal package to avoid circular imports while
keeping the public API backward compatible through type aliases in the root
package.

An alternative purego FFI backend lives in
github.com/dianlight/smartmontools-go/backends/lib (LibBackend). It loads
a pre-built smartmon wrapper shared library at runtime using ebitengine/purego,
avoiding process-spawn overhead and the smartctl binary dependency.

# LibBackend (D1 — SDK wrapper via purego)

LibBackend loads libsmartmon_go.so (Linux) or libsmartmon_go.dylib (macOS) at
runtime. The shared library is a thin C++ wrapper that links against the
pre-built libsmartmon.a static library published in
github.com/dianlight/smartmontools-sdk releases.

Build the wrapper library once with the provided setup script:

	scripts/setup-lib-backend.sh

The script downloads the correct SDK archive for the current platform, installs
the missing smartmon_config.h, and compiles the wrapper into
backends/lib/sdk/libsmartmon_go.{so,dylib}.

# Library Resolution Order

New() resolves the library path in the following order:

 1. The path provided by [libbackend.WithLibraryPath].
 2. SMARTMON_LIB_PATH environment variable — if the file exists at that path it
    is used directly.  If SMARTMON_LIB_PATH is set but the file is absent a
    warning is logged and the search continues to step 3.  If the file exists
    but a library is also found in a different standard system directory a
    warning is logged (the configured path is still used).
 3. Standard system library paths: dynamic-linker names first
    (respects LD_LIBRARY_PATH / DYLD_LIBRARY_PATH / rpath), then a list of
    well-known absolute paths such as /usr/local/lib and /opt/homebrew/lib.

Use the LibBackend with WithBackend:

	lib, err := libbackend.New(
	    libbackend.WithLibraryPath("/usr/local/lib/libsmartmon_go.so"),
	)
	if err != nil {
	    log.Fatal(err)
	}
	defer lib.Close()
	client, err := smartmontools.NewClient(smartmontools.WithBackend(lib))

Or rely on automatic resolution via the environment variable:

	// export SMARTMON_LIB_PATH=/path/to/libsmartmon_go.dylib
	lib, err := libbackend.New()
*/
package smartmontools
