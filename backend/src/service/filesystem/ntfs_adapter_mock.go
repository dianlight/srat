package filesystem

import "github.com/dianlight/srat/dto"

// SetIsDeviceMountedForTesting allows overriding mounted-state checks for tests.
func (a *NtfsAdapter) SetIsDeviceMountedForTesting(isDeviceMounted func(device string) bool) (reset func()) {
	return a.baseAdapter.SetIsDeviceMountedForTesting(isDeviceMounted)
}

// SetLastUnmountedStateForTesting pre-populates the cache used by GetState
// when the device is mounted. Allows tests to inject a known unmounted state
// without running real filesystem commands (which may require root).
func (a *NtfsAdapter) SetLastUnmountedStateForTesting(device string, state dto.FilesystemState) {
	a.setLastUnmountedState(device, state)
}
