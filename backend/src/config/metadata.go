//go:build !cgo

package config

import (
	"bytes"
	"debug/elf"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/Masterminds/semver/v3"
)

// NOTE: This is a fallback implementation for builds without CGO.
// The metadata is embedded as a string constant that can be extracted
// from the binary using 'strings' or similar tools.
// This string is placed in the .rodata section and is externally readable.

// metadataJSON is embedded at compile time by the build system.
// It contains the version and build information in JSON format.
// This string is stored in read-only data section of the binary.
// The version part is set via linker flags and can be extracted with tools like readelf.
// Extract with: grep -a "SRAT_METADATA_VERSION=" binary | strings

// GetCurrentBinaryVersion returns the version from the embedded metadata string.
// Unlike the CGO version, this reads from an embedded constant string, not an ELF section.
func GetCurrentBinaryVersion() *semver.Version {
	if Version == "" {
		slog.Warn("Version constant is empty in no-CGO build")
		return nil
	}

	slog.Debug("Extracting version from embedded metadata string", "metadata", MetadataJSON)

	parsed, err := semver.NewVersion(Version)
	if err != nil {
		slog.Error("Failed to parse build version as semantic version", "version", Version, "error", err)
		return nil
	}
	return parsed
}

// parseMetadataVersion extracts the "version" field from a JSON blob.
// This is kept for API compatibility with the CGO version.
func parseMetadataVersion(data []byte) *semver.Version {
	if len(data) == 0 {
		return nil
	}
	cleaned := bytes.TrimRight(data, "\x00")
	if len(cleaned) == 0 {
		return nil
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal(cleaned, &metadata); err != nil {
		return nil
	}
	versionValue, ok := metadata["version"].(string)
	if !ok {
		return nil
	}
	parsed, err := semver.NewVersion(versionValue)
	if err != nil {
		slog.Error("Failed to parse metadata version as semantic version", "version", versionValue, "error", err)
		return nil
	}
	return parsed
}

// GetBinaryVersion attempts to read metadata from an ELF binary.
// For no-CGO builds, it searches for the embedded metadata string marker.
// For CGO builds (when inspecting from outside), it reads the .note.metadata section.
func GetBinaryVersion(path string) *semver.Version {
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	f, err := elf.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	// Try to read .note.metadata section if it exists (from CGO builds)
	for _, section := range f.Sections {
		if section.Name == ".note.metadata" {
			data, err := section.Data()
			if err != nil {
				return nil
			}
			if version := parseMetadataVersion(data); version != nil {
				return version
			}
		}
	}

	// Try to read from .rodata section (from no-CGO builds)
	for _, section := range f.Sections {
		if section.Name == ".rodata" {
			data, err := section.Data()
			if err != nil {
				continue
			}
			// Search for SRAT_METADATA marker
			if idx := bytes.Index(data, []byte("SRAT_METADATA:")); idx >= 0 {
				// Extract JSON after the marker
				startIdx := idx + len("SRAT_METADATA:")
				endIdx := bytes.Index(data[startIdx:], []byte(`}`))
				if endIdx > 0 {
					jsonData := data[startIdx : startIdx+endIdx+1]
					if version := parseMetadataVersion(jsonData); version != nil {
						return version
					}
				}
			}
		}
	}

	return nil
}
