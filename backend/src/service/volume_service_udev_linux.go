//go:build linux

package service

import (
	"log/slog"
	"strings"

	"github.com/dianlight/tlog"
	"github.com/pilebones/go-udev/netlink"
)

func (self *VolumeService) udevEventHandler() {
	tlog.TraceContext(self.ctx, "Starting Udev event handler...")

	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		tlog.ErrorContext(self.ctx, "Unable to connect to Netlink Kobject UEvent socket", "err", err)
		return
	}
	defer conn.Close()

	queue := make(chan netlink.UEvent, 10)
	errorChan := make(chan error, 1)
	quit := conn.Monitor(queue, errorChan, nil)
	tlog.TraceContext(self.ctx, "Udev monitor started successfully.")

	for {
		select {
		case <-self.ctx.Done():
			slog.InfoContext(self.ctx, "Udev event handler stopping due to context cancellation.", "err", self.ctx.Err())
			if quit != nil {
				close(quit)
			}
			return
		case uevent := <-queue:
			if subsystem, ok := uevent.Env["SUBSYSTEM"]; ok && subsystem == "block" {
				action := uevent.Action
				devName := uevent.Env["DEVNAME"]
				devType := uevent.Env["DEVTYPE"]

				slog.DebugContext(self.ctx, "Received Udev block event", "action", action, "devname", devName, "devtype", devType, "env", uevent.Env)

				if devType != "disk" && devType != "partition" {
					slog.DebugContext(self.ctx, "Ignoring Udev event for non-disk/partition block device", "devname", devName, "devtype", devType)
					continue
				}
				if action == "remove" && devType == "disk" {
					bus := uevent.Env["ID_BUS"]
					suffix := uevent.Env[".PART_SUFFIX"]
					serial := uevent.Env["ID_SERIAL"]

					slog.InfoContext(self.ctx, "Processing block device removal event", "devname", devName, "bus", bus, "serial", serial, "suffix", suffix)
					self.disks.Remove(bus + "-" + serial + suffix)
				} else if devType == "disk" && action == "add" {
					slog.InfoContext(self.ctx, "Processing block device event", "action", action, "devname", devName)

					if self.hardwareClient != nil {
						self.hardwareClient.InvalidateHardwareInfo()
					}
					err := self.getVolumesData()
					if err != nil {
						slog.ErrorContext(self.ctx, "Failed to get volumes data after udev event", "err", err)
						continue
					}
				} else if devType == "disk" && action == "change" {
					slog.InfoContext(self.ctx, "Ignore: Processing block device change event", "action", action, "devname", devName)
					continue
				} else if devType == "partition" && action == "add" {
					slog.InfoContext(self.ctx, "Processing partition addition event", "action", action, "devname", devName)
					if self.handlePartitionUdevAddEvent(devName) {
						continue
					}
					if self.hardwareClient != nil {
						self.hardwareClient.InvalidateHardwareInfo()
					}
					err := self.getVolumesData()
					if err != nil {
						slog.ErrorContext(self.ctx, "Failed to refresh volume cache after partition add event", "devname", devName, "err", err)
					}
				} else if devType == "partition" && action == "remove" {
					slog.InfoContext(self.ctx, "Processing partition removal event", "action", action, "devname", devName)
					self.handlePartitionUdevRemoveEvent(devName)
				}
			}
		case err := <-errorChan:
			if err != nil && strings.Contains(err.Error(), "unable to parse uevent") {
				errMsg := err.Error()
				if strings.Contains(errMsg, "invalid env data") {
					slog.DebugContext(self.ctx, "Ignoring malformed uevent with invalid env data",
						"err", err,
						"detail", "This can occur when kernel sends events with non-standard formatting")
				} else {
					slog.DebugContext(self.ctx, "Failed to parse uevent, skipping",
						"err", err,
						"detail", "Event format not recognized or incompatible")
				}
			} else {
				slog.ErrorContext(self.ctx, "Error received from Udev monitor", "err", err)
			}
		}
	}
}
