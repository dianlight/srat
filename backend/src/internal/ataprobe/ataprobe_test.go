package ataprobe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubOpenDevice replaces openDeviceFn with a temp regular file so probe
// logic is exercised without SG hardware. The SG_IO ioctl on a regular
// file fails, which is exactly the error propagation under test.
func stubOpenDevice(t *testing.T) {
	t.Helper()
	prev := openDeviceFn
	openDeviceFn = func(_ string) (*os.File, error) {
		return os.CreateTemp(t.TempDir(), "sg-stub")
	}
	t.Cleanup(func() { openDeviceFn = prev })
}

func pointSysBlockRoot(t *testing.T, root string) {
	t.Helper()
	prev := sysBlockRoot
	sysBlockRoot = root
	t.Cleanup(func() { sysBlockRoot = prev })
}

func TestCheckAtaDevice_NonexistentDevice(t *testing.T) {
	err := CheckAtaDevice(filepath.Join(t.TempDir(), "does-not-exist"), false)
	require.Error(t, err)
}

func TestCheckAtaDevice_EmptyDevice(t *testing.T) {
	require.Error(t, CheckAtaDevice("", false))
}

func TestCheckAtaDevice_NotSgDevice(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "not-sg")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	err = CheckAtaDevice(f.Name(), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not appear to be an sg device")
}

func TestCheckAtaDevice_ProbeErrorPropagates(t *testing.T) {
	stubOpenDevice(t)
	pointSysBlockRoot(t, t.TempDir())
	require.Error(t, CheckAtaDevice("/dev/sda", false))
}

func TestCheckAtaDevice_ProbeErrorPropagatesDebug(t *testing.T) {
	stubOpenDevice(t)
	pointSysBlockRoot(t, t.TempDir())
	require.Error(t, CheckAtaDevice("/dev/sda", true))
}

func TestCheckAtaDevice_JmicronBranch(t *testing.T) {
	stubOpenDevice(t)
	root := t.TempDir()
	diskDir := filepath.Join(root, "sda")
	require.NoError(t, os.MkdirAll(diskDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(diskDir, "idVendor"), []byte("152d\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(diskDir, "idProduct"), []byte("2339\n"), 0o644))
	pointSysBlockRoot(t, root)
	require.Error(t, CheckAtaDevice("/dev/sda", false))
}

func TestCheckAtaDevice_JmicronBranchDebug(t *testing.T) {
	stubOpenDevice(t)
	root := t.TempDir()
	diskDir := filepath.Join(root, "sda")
	require.NoError(t, os.MkdirAll(diskDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(diskDir, "idVendor"), []byte("152d\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(diskDir, "idProduct"), []byte("2336\n"), 0o644))
	pointSysBlockRoot(t, root)
	require.Error(t, CheckAtaDevice("/dev/sda", true))
}

func TestIsJmicronDevice_Table(t *testing.T) {
	tests := []struct {
		name      string
		vendor    string
		product   string
		want      bool
		writeIDs  bool
		shortPath bool
	}{
		{name: "jmicron 2329", vendor: "152d", product: "2329", want: true, writeIDs: true},
		{name: "jmicron 2336", vendor: "152d", product: "2336", want: true, writeIDs: true},
		{name: "jmicron 2338", vendor: "152d", product: "2338", want: true, writeIDs: true},
		{name: "jmicron 2339", vendor: "152d", product: "2339", want: true, writeIDs: true},
		{name: "jmicron unknown product", vendor: "152d", product: "0000", want: false, writeIDs: true},
		{name: "other vendor", vendor: "0bc2", product: "2339", want: false, writeIDs: true},
		{name: "missing sysfs", want: false},
		{name: "short device path", want: false, shortPath: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := "/dev/sda"
			if tt.shortPath {
				device = "sda"
				assert.False(t, isJmicronDevice(device, false))
				return
			}
			root := t.TempDir()
			if tt.writeIDs {
				diskDir := filepath.Join(root, "sda")
				require.NoError(t, os.MkdirAll(diskDir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(diskDir, "idVendor"), []byte(tt.vendor+"\n"), 0o644))
				require.NoError(t, os.WriteFile(filepath.Join(diskDir, "idProduct"), []byte(tt.product+"\n"), 0o644))
			}
			pointSysBlockRoot(t, root)
			assert.Equal(t, tt.want, isJmicronDevice(device, false))
		})
	}
}

func TestIsJmicronDevice_MissingProductFile(t *testing.T) {
	root := t.TempDir()
	diskDir := filepath.Join(root, "sda")
	require.NoError(t, os.MkdirAll(diskDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(diskDir, "idVendor"), []byte("152d\n"), 0o644))
	pointSysBlockRoot(t, root)
	assert.False(t, isJmicronDevice("/dev/sda", true))
}

func TestFindSystemFile_DirectAndAncestor(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "idVendor"), []byte(" 152d \n"), 0o644))

	got, err := findSystemFile(nested, "idVendor")
	require.NoError(t, err)
	assert.Equal(t, "152d", got)
}

func TestFindSystemFile_NotFound(t *testing.T) {
	_, err := findSystemFile(t.TempDir(), "no-such-file")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device not found")
}

func TestJmicronGetRegisters_Layout(t *testing.T) {
	cbd := jmicronGetRegisters()
	require.Len(t, cbd, 12)
	assert.Equal(t, uint8(0xdf), cbd[0])
	assert.Equal(t, uint8(0x10), cbd[1])
	assert.Equal(t, uint8(0xfd), cbd[11])
}

func TestSendSgio_PropagatesIoctlError(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "sendSgio")
	require.NoError(t, err)
	defer f.Close()
	require.Error(t, sendSgio(f, jmicronGetRegisters(), true))
	require.Error(t, sendAtaCommand(f, ataOpCheckPowerMode, false))
}

func TestDumpBytes_DoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() { dumpBytes([]uint8{0x85, 0xe5}) })
}

func TestProbe_JmicronSendError(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "probe")
	require.NoError(t, err)
	defer f.Close()
	require.Error(t, probe(f, "sda", false))
}

func TestSgAtaConstants(t *testing.T) {
	assert.Equal(t, uint8(0x85), uint8(sgAta16))
	assert.Equal(t, uint8(0xe5), uint8(ataOpCheckPowerMode))
	assert.Equal(t, uint8(0xa1), uint8(sgAta12))
}
