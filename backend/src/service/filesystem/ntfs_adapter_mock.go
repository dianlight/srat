package filesystem

// SetIsDeviceMountedForTesting allows overriding mounted-state checks for tests.
func (a *NtfsAdapter) SetIsDeviceMountedForTesting(isDeviceMounted func(device string) bool) (reset func()) {
	return a.baseAdapter.SetIsDeviceMountedForTesting(isDeviceMounted)
}
