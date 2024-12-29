package lsblk

import (
	"testing"
)

/*
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

*/

func TestGetLabelsFromDevice(t *testing.T) {
	_, _, _, err := GetLabelsFromDevice("loop1")
	if err != nil {
		t.Errorf("GetLabelsFromDevice failed: %v", err)
	}
	// t.Logf("label: %s, partlabel: %s mountpoint:%s", *label, *partlabel, *mountpoint)
	//
	//	if *label == "" && *partlabel == "" {
	//		t.Error("Empty labels returned")
	//	}
}
