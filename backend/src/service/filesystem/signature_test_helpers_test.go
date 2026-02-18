package filesystem_test

import (
	"os"
	"testing"
)

func createTempDeviceWithMagic(t *testing.T, offset int64, magic []byte) string {
	t.Helper()

	file, err := os.CreateTemp("", "fs-magic-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Remove(file.Name())
	})

	if _, err := file.WriteAt(magic, offset); err != nil {
		_ = file.Close()
		t.Fatalf("failed to write magic bytes: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	return file.Name()
}
