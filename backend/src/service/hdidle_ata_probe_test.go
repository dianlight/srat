package service

import (
	"reflect"
	"testing"

	"github.com/adelolmo/hd-idle/sgio"
)

// TestAtaProbeFn_IsReadOnlyCheckPowerMode verifies that the ATA support probe
// uses a read-only CHECK POWER MODE (0xE5) command, not STANDBY IMMEDIATE (0xE0).
//
// CheckATASupport is called on every Start() (via convertConfig ->
// CheckDeviceSupport) and for unsaved disks via GetDeviceConfig. Using
// StopAtaDevice (STANDBY IMMEDIATE) physically spins down every ATA disk on
// every probe — a destructive side-effect for what should be a read-only
// capability check. The spindownDisk path intentionally keeps using
// sgio.StopAtaDevice for actual spindown operations.
//
// Note: Go does not allow comparing non-nil func values with == or !=.
// We use reflect.ValueOf().Pointer() to compare the underlying code addresses.
func TestAtaProbeFn_IsReadOnlyCheckPowerMode(t *testing.T) {
	if ataProbeFn == nil {
		t.Fatal("ataProbeFn must be initialized")
	}
	probeAddr := reflect.ValueOf(ataProbeFn).Pointer()
	checkAddr := reflect.ValueOf(sgio.CheckAtaDevice).Pointer()
	stopAddr := reflect.ValueOf(sgio.StopAtaDevice).Pointer()

	if probeAddr != checkAddr {
		t.Errorf("ataProbeFn must point to CheckAtaDevice (CHECK POWER MODE 0xE5), "+
			"got addr %v, want CheckAtaDevice addr %v", probeAddr, checkAddr)
	}
	if probeAddr == stopAddr {
		t.Error("ataProbeFn must NOT point to StopAtaDevice — that spins down disks during probing")
	}
}
