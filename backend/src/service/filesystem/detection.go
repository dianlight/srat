package filesystem

import (
	"bytes"
	"io"
	"os"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

const maxDeviceReadLength = 65608 // Max offset (btrfs: 65600) + max magic length (8)

// Custom error types for device detection
var (
	ErrorDeviceNotFound    = errors.New("device not found")
	ErrorDeviceAccess      = errors.New("failed to access device")
	ErrorUnknownFilesystem = errors.New("unknown filesystem type")
)

// DetectFilesystemType attempts to determine the filesystem type of a block device by reading its magic numbers.
// Returns the filesystem type string if detected, empty string otherwise.
func DetectFilesystemType(devicePath string, adapters []FilesystemAdapter) (string, errors.E) {
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
		return "", errors.WithDetails(ErrorDeviceAccess, "Path", devicePath, "Operation", "ReadAt", "Error", err)
	}

	if n == 0 {
		return "", errors.WithDetails(ErrorUnknownFilesystem, "Path", devicePath, "Reason", "Device is empty or too small")
	}

	validBuffer := buffer[:n]

	// Check against signatures from all adapters
	for _, adapter := range adapters {
		signatures := adapter.GetFsSignatureMagic()
		for _, sig := range signatures {
			sigEndOffset := sig.Offset + int64(len(sig.Magic))
			if sig.Offset < 0 || sigEndOffset > int64(len(validBuffer)) {
				continue
			}

			if bytes.Equal(validBuffer[sig.Offset:sigEndOffset], sig.Magic) {
				return adapter.GetName(), nil
			}
		}
	}

	return "", errors.WithDetails(ErrorUnknownFilesystem, "Path", devicePath)
}

// checkDeviceMatchesSignatures checks if a device matches any of the provided signatures
func checkDeviceMatchesSignatures(devicePath string, signatures []dto.FsMagicSignature) (bool, errors.E) {
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
		sigEndOffset := sig.Offset + int64(len(sig.Magic))
		if sig.Offset < 0 || sigEndOffset > int64(len(validBuffer)) {
			continue
		}

		if bytes.Equal(validBuffer[sig.Offset:sigEndOffset], sig.Magic) {
			return true, nil
		}
	}

	return false, nil
}
