//go:build darwin

package service

import (
	"log/slog"
)

func (self *VolumeService) udevEventHandler() {
	slog.WarnContext(self.ctx, "Udev event handler is not supported on this platform")
}
