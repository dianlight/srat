package filesystem

import "github.com/prometheus/procfs"

// SetIsDeviceMountedForTesting allows overriding mounted-state checks for tests.
func (a *NtfsAdapter) SetIsDeviceMountedForTesting(isDeviceMounted func(device string) bool) (reset func()) {
	if isDeviceMounted != nil {
		a.isDeviceMounted = isDeviceMounted
	}

	return func() {
		a.isDeviceMounted = func(device string) bool {
			mounts, err := procfs.GetMounts()
			if err != nil {
				return false
			}

			for _, mount := range mounts {
				if mount.Source == device {
					return true
				}
			}

			return false
		}
	}
}
