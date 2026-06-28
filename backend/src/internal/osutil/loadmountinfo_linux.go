//go:build linux

package osutil

import (
	"github.com/prometheus/procfs"
)

// LoadMountInfo loads the current mount information for the running process.
// On Linux, it reads from /proc/self/mountinfo via procfs.
func LoadMountInfo() ([]*MountInfoEntry, error) {
	overrideMu.RLock()
	override := mountsOverride
	overrideMu.RUnlock()

	if override != nil {
		mounts, err := override()
		if err != nil {
			return nil, err
		}
		return convertFromProcs(mounts), nil
	}

	mounts, err := procfs.GetMounts()
	if err != nil {
		return nil, err
	}
	return convertFromProcs(mounts), nil
}
