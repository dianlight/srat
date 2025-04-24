package lsblk

import (
	"testing"

	"github.com/kr/pretty"
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

func TestGetInfoFromDevice(t *testing.T) {
	lsbkint := NewLSBKInterpreter()
	lsbkp, err := lsbkint.GetInfoFromDevice("loop1")
	if err != nil {
		t.Errorf("GetInfoFromDevice failed: %v", err)
		return
	}
	t.Logf("lsbk %v", pretty.Sprint(lsbkp))
}
