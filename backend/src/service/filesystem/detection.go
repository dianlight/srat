package filesystem

import (
	"bytes"
	"io"
	"os"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

const maxDeviceReadLength = 65608 // Max offset (btrfs: 65600) + max magic length (8)

// checkDeviceMatchesSignatures checks if a device matches any of the provided signatures
func (b *baseAdapter) checkDeviceMatchesSignatures(devicePath string) (bool, errors.E) {
	signatures := b.GetFsSignatureMagic()
	if len(signatures) == 0 {
		// No signatures to check, return false
		return false, nil
	}

	file, err := os.Open(devicePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, errors.WithDetails(dto.ErrorDeviceNotFound, "Path", devicePath, "Error", err)
		}
		return false, errors.WithDetails(dto.ErrorDeviceAccess, "Path", devicePath, "Operation", "Open", "Error", err)
	}
	defer file.Close()

	buffer := make([]byte, maxDeviceReadLength)
	n, err := file.ReadAt(buffer, 0)
	if err != nil && err != io.EOF {
		return false, errors.WithDetails(dto.ErrorDeviceAccess, "Path", devicePath, "Operation", "ReadAt", "Error", err)
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
