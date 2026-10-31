package mount

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// bootSectorSize is the size of the volume boot record that carries the OEM
// ID used to identify NTFS and exFAT volumes.
const bootSectorSize = 512

var (
	// ntfsOEMID is the OEM ID at offset 3 of an NTFS boot sector.
	ntfsOEMID = []byte("NTFS    ")
	// exfatOEMID is the file system name at offset 3 of an exFAT boot sector.
	exfatOEMID = []byte("EXFAT   ")
	// bootSignature is the 0x55AA end-of-sector marker at offset 510.
	bootSignature = []byte{0x55, 0xAA}
)

// fsFromBootSector identifies file systems that u-root's FSFromBlock does not
// probe: it only knows ext2/3/4, vfat, squashfs, iso9660, xfs and btrfs, so
// NTFS and exFAT volumes come back as "no suitable filesystem". Both carry
// their name as the OEM ID at offset 3 of the boot sector, followed by the
// 0x55AA signature at offset 510.
//
// The returned names match udev's ID_FS_TYPE values ("ntfs", "exfat"), which
// the filesystem registry already resolves to the right adapter.
func fsFromBootSector(sector []byte) string {
	if len(sector) < bootSectorSize {
		return ""
	}
	if !bytes.Equal(sector[510:512], bootSignature) {
		return ""
	}
	switch {
	case bytes.Equal(sector[3:11], ntfsOEMID):
		return "ntfs"
	case bytes.Equal(sector[3:11], exfatOEMID):
		return "exfat"
	default:
		return ""
	}
}

// probeBootSectorFS reads the first sector of a block device (or image file)
// and returns the file system identified by fsFromBootSector.
func probeBootSectorFS(n string) (string, error) {
	f, err := os.Open(n)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sector := make([]byte, bootSectorSize)
	if _, err := io.ReadFull(f, sector); err != nil {
		return "", fmt.Errorf("read boot sector of %q: %w", n, err)
	}
	if fs := fsFromBootSector(sector); fs != "" {
		return fs, nil
	}
	return "", fmt.Errorf("no NTFS or exFAT boot sector on %q", n)
}
