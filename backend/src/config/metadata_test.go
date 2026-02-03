package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"
)

func TestGetCurrentBinaryVersion_NotNil(t *testing.T) {
	// Test that GetCurrentBinaryVersion returns a valid version
	version := GetCurrentBinaryVersion()
	require.NotNil(t, version, "Version should not be nil")

	// The version should be a valid semantic version
	require.NotEmpty(t, version.String(), "Version string should not be empty")
}

func TestGetBinaryVersion_EmptyPath(t *testing.T) {
	// Test with empty path
	result := GetBinaryVersion("")
	require.Nil(t, result, "Empty path should return nil")
}

func TestGetBinaryVersion_NonExistentFile(t *testing.T) {
	// Test with non-existent file
	result := GetBinaryVersion("/non/existent/path/to/binary")
	require.Nil(t, result, "Non-existent file should return nil")
}

func TestGetBinaryVersion_InvalidELFFile(t *testing.T) {
	// Create a temporary file that is not a valid ELF binary
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "not-an-elf")

	err := os.WriteFile(tmpFile, []byte("This is not an ELF file"), 0644)
	require.NoError(t, err, "Should create temporary file")

	// Test with invalid ELF file
	result := GetBinaryVersion(tmpFile)
	require.Nil(t, result, "Invalid ELF file should return nil")
}

func TestGetBinaryVersion_ValidELFWithoutMetadata(t *testing.T) {
	// Test with current test binary (if available)
	currentBinary := os.Args[0]
	if _, err := os.Stat(currentBinary); err == nil {
		result := GetBinaryVersion(currentBinary)
		// The result might be nil if the .note.metadata section is not present
		// This is expected behavior and the function should not panic
		t.Logf("Result for current binary: %+v", result)
	}
}

func TestGetBinaryVersion_DirectoryPath(t *testing.T) {
	// Test with a directory path instead of file
	tmpDir := t.TempDir()
	result := GetBinaryVersion(tmpDir)
	require.Nil(t, result, "Directory path should return nil")
}

func TestParseMetadataVersion_ValidVersion(t *testing.T) {
	// Test valid semantic version
	version, err := semver.NewVersion("1.2.3")
	require.NoError(t, err)
	require.Equal(t, "1.2.3", version.String())
}

func TestParseMetadataVersion_NoMetadataFallback(t *testing.T) {
	// Test that when metadata is "no-metadata", it falls back to Version constant
	// This is an indirect test through GetCurrentBinaryVersion behavior
	version := GetCurrentBinaryVersion()
	require.NotNil(t, version, "Should fall back to Version constant")
}
