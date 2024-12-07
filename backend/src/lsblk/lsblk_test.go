package lsblk

import (
	"testing"
)

func TestListDevices(t *testing.T) {
	devices, err := ListDevices()
	if err != nil {
		t.Errorf("list devices failed: %v", err)
	}
	if len(devices) == 0 {
		t.Errorf("Empty return from lsbk")
	}
	t.Logf("devices: %+v", devices)
}

func TestPrintDevices(t *testing.T) {
	devices, err := ListDevices()
	if err != nil {
		t.Errorf("list devices failed: %v", err)
	}
	PrintDevices(devices)
}

func TestPrintPartitions(t *testing.T) {
	devices, err := ListDevices()
	if err != nil {
		t.Errorf("list devices failed: %v", err)
	}
	PrintPartitions(devices)
}
