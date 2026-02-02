package filesystem

import (
	"bytes"
	"io"
	"os"

	"gitlab.com/tozd/go/errors"
)

// fsMagicSignature defines a structure to hold filesystem signature information.
type fsMagicSignature struct {
	fsType string
	offset int64
	magic  []byte
}

// knownFsSignatures is a list of known filesystem signatures.
// The order can matter if signatures are subsets of others, though distinct offsets help.
var knownFsSignatures = []fsMagicSignature{
	// Filesystems with magic at/near offset 0
	{fsType: "xfs", offset: 0, magic: []byte{'X', 'F', 'S', 'B'}},
	{fsType: "squashfs", offset: 0, magic: []byte{0x68, 0x73, 0x71, 0x73}},              // "hsqs" little-endian
	{fsType: "ntfs", offset: 3, magic: []byte{'N', 'T', 'F', 'S', ' ', ' ', ' ', ' '}},  // "NTFS    "
	{fsType: "exfat", offset: 3, magic: []byte{'E', 'X', 'F', 'A', 'T', ' ', ' ', ' '}}, // "EXFAT   "

	// FAT types
	{fsType: "vfat", offset: 82, magic: []byte{'F', 'A', 'T', '3', '2', ' ', ' ', ' '}}, // FAT32 specific
	{fsType: "vfat", offset: 54, magic: []byte{'F', 'A', 'T', '1', '6', ' ', ' ', ' '}}, // FAT16 specific
	{fsType: "vfat", offset: 54, magic: []byte{'F', 'A', 'T', '1', '2', ' ', ' ', ' '}}, // FAT12 specific

	// Filesystems with magic at larger offsets
	{fsType: "f2fs", offset: 1024, magic: []byte{0x10, 0x20, 0xF5, 0xF2}}, // Little-endian 0xF2F52010
	{fsType: "ext4", offset: 1080, magic: []byte{0x53, 0xEF}},             // ext2/3/4, little-endian 0xEF53

	// ISO9660 - Primary Volume Descriptor
	{fsType: "iso9660", offset: 0x8001, magic: []byte{'C', 'D', '0', '0', '1'}}, // 32769

	// BTRFS
	{fsType: "btrfs", offset: 0x10040, magic: []byte{'_', 'B', 'H', 'R', 'f', 'S', '_', 'M'}}, // 65600
}

const maxDeviceReadLength = 65608 // Max offset (btrfs: 65600) + max magic length (8)

// Custom error types for device detection
var (
	ErrorDeviceNotFound    = errors.New("device not found")
	ErrorDeviceAccess      = errors.New("failed to access device")
	ErrorUnknownFilesystem = errors.New("unknown filesystem type")
)

// DetectFilesystemType attempts to determine the filesystem type of a block device by reading its magic numbers.
// Returns the filesystem type string if detected, empty string otherwise.
func DetectFilesystemType(devicePath string) (string, errors.E) {
	file, err := os.Open(devicePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.WithDetails(ErrorDeviceNotFound, "Path", devicePath, "Error", err)
		}
		return "", errors.WithDetails(ErrorDeviceAccess, "Path", devicePath, "Operation", "Open", "Error", err)
	}
	defer file.Close()

	buffer := make([]byte, maxDeviceReadLength)
	n, err := file.ReadAt(buffer, 0)
	if err != nil && err != io.EOF {
		// For ReadAt, io.EOF is reported only if no bytes were read.
		// If n > 0 and err == io.EOF, it means a partial read, which is fine.
		// If n == 0 and err == io.EOF, the file is empty or smaller than our read attempt from offset 0.
		return "", errors.WithDetails(ErrorDeviceAccess, "Path", devicePath, "Operation", "ReadAt", "Error", err)
	}

	if n == 0 {
		return "", errors.WithDetails(ErrorUnknownFilesystem, "Path", devicePath, "Reason", "Device is empty or too small")
	}

	// Use the actual number of bytes read for checks
	validBuffer := buffer[:n]

	for _, sig := range knownFsSignatures {
		// Ensure the signature's offset and length are within the bounds of what was read
		sigEndOffset := sig.offset + int64(len(sig.magic))
		if sig.offset < 0 || sigEndOffset > int64(len(validBuffer)) {
			continue // Signature is out of bounds for the data read
		}

		// Compare the magic bytes
		if bytes.Equal(validBuffer[sig.offset:sigEndOffset], sig.magic) {
			return sig.fsType, nil
		}
	}

	return "", errors.WithDetails(ErrorUnknownFilesystem, "Path", devicePath)
}

// getSignaturesForFilesystem returns all known magic signatures for a specific filesystem type
func getSignaturesForFilesystem(fsType string) []fsMagicSignature {
	var signatures []fsMagicSignature
	for _, sig := range knownFsSignatures {
		if sig.fsType == fsType {
			signatures = append(signatures, sig)
		}
	}
	return signatures
}

// checkDeviceMatchesSignatures checks if a device matches any of the provided signatures
func checkDeviceMatchesSignatures(devicePath string, signatures []fsMagicSignature) (bool, errors.E) {
	if len(signatures) == 0 {
		// No signatures to check, return false
		return false, nil
	}

	file, err := os.Open(devicePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, errors.WithDetails(ErrorDeviceNotFound, "Path", devicePath, "Error", err)
		}
		return false, errors.WithDetails(ErrorDeviceAccess, "Path", devicePath, "Operation", "Open", "Error", err)
	}
	defer file.Close()

	buffer := make([]byte, maxDeviceReadLength)
	n, err := file.ReadAt(buffer, 0)
	if err != nil && err != io.EOF {
		return false, errors.WithDetails(ErrorDeviceAccess, "Path", devicePath, "Operation", "ReadAt", "Error", err)
	}

	if n == 0 {
		return false, nil
	}

	validBuffer := buffer[:n]

	for _, sig := range signatures {
		sigEndOffset := sig.offset + int64(len(sig.magic))
		if sig.offset < 0 || sigEndOffset > int64(len(validBuffer)) {
			continue
		}

		if bytes.Equal(validBuffer[sig.offset:sigEndOffset], sig.magic) {
			return true, nil
		}
	}

	return false, nil
}
