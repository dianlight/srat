// hd-idle ATA probe, vendored in-tree to avoid patching dependencies.
//
// Portions derived from github.com/adelolmo/hd-idle/sgio (GPL-3.0-only,
// Copyright (C) 2018-2023 Andoni del Olmo). This file is therefore made
// available under the GNU General Public License v3.0 only.
//
// Rationale: SRAT needs a read-only ATA PASS-THROUGH probe (CHECK POWER
// MODE, 0xE5) for device support detection. Upstream hd-idle only exposes
// StopAtaDevice (STANDBY IMMEDIATE, 0xE0), which physically spins disks
// down — a destructive side-effect for a capability check. Instead of
// maintaining a vendor patch (backend/patches/hd-idle-*.patch re-applied
// after every `go mod vendor`), the probe lives here and is built only on
// the public github.com/benmcclelland/sgio API (MIT).
//
// The error semantics match upstream StopAtaDevice: devices without ATA
// PASS-THROUGH support fail with an error containing
// "INVALID COMMAND OPERATION CODE", which callers rely on.
package ataprobe

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	sgio "github.com/benmcclelland/sgio"
)

const (
	sgAta16 = 0x85 // ATA PASS-THROUGH(16)
	sgAta12 = 0xa1 // ATA PASS-THROUGH (12)

	sgAtaProtoNonData = 3 << 1
	ataUsingLba       = 1 << 6

	// ataOpCheckPowerMode is a read-only probe: unlike STANDBY IMMEDIATE it
	// never spins the disk down.
	// https://wiki.osdev.org/ATA/ATAPI_Power_Management
	ataOpCheckPowerMode = 0xe5

	sgDxferNone = -1
)

// sysBlockRoot is the sysfs block-device root used for USB-bridge
// identification. It is a variable (not a const) so tests can point it at
// a fixture tree.
var sysBlockRoot = "/sys/block"

// openDeviceFn opens an SG device. It is a variable so tests can stub
// hardware access; production code always uses openDevice.
var openDeviceFn = openDevice

// CheckAtaDevice probes whether device supports ATA PASS-THROUGH by issuing
// a read-only CHECK POWER MODE (0xE5) command. It never spins the disk down,
// so it is safe to call during device support detection.
func CheckAtaDevice(device string, debug bool) error {
	f, err := openDeviceFn(device)
	if err != nil {
		return err
	}

	if err := probe(f, device, debug); err != nil {
		_ = f.Close()
		return err
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("cannot close file %s. Error: %s", device, err)
	}
	return nil
}

// probe issues the bridge-appropriate read-only command for an open device.
func probe(f *os.File, device string, debug bool) error {
	if isJmicronDevice(device, debug) {
		// JMicron USB bridges expose no read-only ATA power-mode probe via
		// their vendor command set. A register read confirms the bridge
		// responds without issuing any standby.
		if debug {
			fmt.Println(" issuing check power mode (jmicron read registers)")
		}
		return sendSgio(f, jmicronGetRegisters(), debug)
	}
	if debug {
		fmt.Println(" issuing check power mode")
	}
	return sendAtaCommand(f, ataOpCheckPowerMode, debug)
}

// openDevice opens an SG device read-only and verifies the SG driver version.
func openDevice(fname string) (*os.File, error) {
	f, err := os.OpenFile(fname, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	var version uint32
	if ioctl(f.Fd(), sgio.SG_GET_VERSION_NUM, uintptr(unsafe.Pointer(&version))) != nil || version < 30000 {
		_ = f.Close()
		return nil, fmt.Errorf("device does not appear to be an sg device")
	}
	return f, nil
}

func ioctl(fd, cmd, ptr uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, ptr)
	if errno != 0 {
		return errno
	}
	return nil
}

func sendAtaCommand(f *os.File, command uint8, debug bool) error {
	cbd := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0} // len 16
	cbd[0] = sgAta16
	cbd[1] = sgAtaProtoNonData
	cbd[13] = ataUsingLba
	cbd[14] = command
	return sendSgio(f, cbd, debug)
}

func sendSgio(f *os.File, inqCmdBlk []uint8, debug bool) error {
	senseBuf := make([]byte, sgio.SENSE_BUF_LEN)
	ioHdr := &sgio.SgIoHdr{
		InterfaceID:    'S',                   //  0	4
		DxferDirection: sgDxferNone,           //  4 	4
		CmdLen:         uint8(len(inqCmdBlk)), //  8	1
		MxSbLen:        sgio.SENSE_BUF_LEN,    //  9	1
		Cmdp:           &inqCmdBlk[0],         // 24	8
		Sbp:            &senseBuf[0],          // 32	8
		Timeout:        0,                     // 40	4
	}

	if debug {
		dumpBytes(inqCmdBlk)
	}

	if err := sgio.SgioSyscall(f, ioHdr); err != nil {
		return err
	}

	return sgio.CheckSense(ioHdr, &senseBuf)
}

func dumpBytes(p []uint8) {
	fmt.Print("outgoing cdb:  ")
	for i := range p {
		fmt.Printf("%02x ", p[i])
	}
	fmt.Print("\n")
}

func jmicronGetRegisters() []uint8 {
	cbd := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0} // len 12
	cbd[0] = 0xdf
	cbd[1] = 0x10 // read
	cbd[4] = 0x01
	cbd[6] = 0x72
	cbd[7] = 0x0f
	cbd[11] = 0xfd
	return cbd
}

const jmicronVendorID = "152d"

var jmicronProductIDs = map[string]bool{
	"2329": true,
	"2336": true,
	"2338": true,
	"2339": true,
}

// isJmicronDevice reports whether device sits behind a JMicron USB bridge,
// identified via sysfs USB IDs. Unknown or unreadable devices return false
// so the caller falls back to the standard ATA probe.
func isJmicronDevice(device string, debug bool) bool {
	parts := strings.Split(device, "/")
	if len(parts) < 3 {
		return false
	}
	diskname := parts[2]
	idVendor, err := findSystemFile(filepath.Join(sysBlockRoot, diskname), "idVendor")
	if err != nil {
		if debug {
			fmt.Println("APT: Unsupported device")
		}
		return false
	}
	idProduct, err := findSystemFile(filepath.Join(sysBlockRoot, diskname), "idProduct")
	if err != nil {
		return false
	}
	if debug {
		fmt.Printf("APT: USB ID = 0x%s:0x%s\n", idVendor, idProduct)
	}
	if idVendor != jmicronVendorID {
		if debug {
			fmt.Println("APT: Unsupported device")
		}
		return false
	}
	if jmicronProductIDs[idProduct] {
		if debug {
			fmt.Println("APT: Found supported device jmicron")
		}
		return true
	}
	return false
}

// findSystemFile reads filename from systemRoot or, for partitioned devices,
// from ancestor directories (up to 20 levels up).
func findSystemFile(systemRoot, filename string) (string, error) {
	content, err := os.ReadFile(filepath.Join(systemRoot, filename))
	relativeDir := ""
	depth := 0
	for depth < 20 {
		if err == nil {
			return strings.TrimSpace(string(content)), nil
		}
		relativeDir += "/.."
		content, err = os.ReadFile(systemRoot + relativeDir + "/" + filename)
		depth++
	}

	return "", fmt.Errorf("device not found")
}
