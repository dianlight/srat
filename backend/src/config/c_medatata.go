package config

import (
	"bytes"
	"debug/elf"
	"encoding/json"
	"os"

	"github.com/Masterminds/semver/v3"
)

/*
// We define a macro 'ELF_METADATA' that will be provided at build time
#ifndef ELF_METADATA
#define ELF_METADATA "{\"version\": \"no-metadata\"}"
#endif

// This attribute forces the string into the .note.metadata section to avoid assembler warnings
__attribute__((section(".note.metadata")))
const char metadata[] = ELF_METADATA;
*/
import "C"

var unknownBinaryVersion = semver.MustParse("0.0.0-unknown")

// GetCurrentBinaryVersion extracts the version from the C metadata constant and
// returns it as a semantic version. If metadata is missing or invalid, a
// sentinel unknown version is returned.
func GetCurrentBinaryVersion() semver.Version {
	// Convert C array to Go slice and then to string
	// C.metadata is a fixed array in cgo
	if len(C.metadata) == 0 {
		return *unknownBinaryVersion
	}

	// Create a nil-terminated Go string from the C array
	metadataBytes := make([]byte, 0, len(C.metadata))
	for i := 0; i < len(C.metadata); i++ {
		if C.metadata[i] == 0 {
			break
		}
		metadataBytes = append(metadataBytes, byte(C.metadata[i]))
	}

	if len(metadataBytes) == 0 {
		return *unknownBinaryVersion
	}

	if version := parseMetadataVersion(metadataBytes); version != nil {
		return *version
	}
	return *unknownBinaryVersion
}

// parseMetadataVersion extracts the "version" field from a JSON blob that may
// contain trailing NUL bytes (common when reading from C strings or ELF sections).
// It returns a parsed semantic version, or nil when parsing fails.
func parseMetadataVersion(data []byte) *semver.Version {
	if len(data) == 0 {
		return nil
	}
	// Trim trailing NULs often present in C strings/sections
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
		return nil
	}
	return parsed
}

// GetBinaryVersion opens the ELF binary at the provided path and attempts to read
// the JSON metadata stored in the .note.metadata section. It returns the value of
// the "version" field in that JSON as a semantic version. If the section is
// missing or cannot be parsed, a sentinel unknown version is returned.
func GetBinaryVersion(path string) semver.Version {
	if path == "" {
		return *unknownBinaryVersion
	}
	// Ensure file exists before opening with debug/elf for clearer early failures
	if _, err := os.Stat(path); err != nil {
		return *unknownBinaryVersion
	}
	f, err := elf.Open(path)
	if err != nil {
		return *unknownBinaryVersion
	}
	defer f.Close()

	// Prefer exact ".note.metadata" section, but also try a couple of sane fallbacks
	var sec *elf.Section
	if s := f.Section(".note.metadata"); s != nil {
		sec = s
	} else if s := f.Section("note.metadata"); s != nil { // rare, but try
		sec = s
	} else {
		// As a last resort, scan sections to find a close match
		for _, s := range f.Sections {
			if s != nil && (s.Name == ".note.metadata" || s.Name == "note.metadata") {
				sec = s
				break
			}
		}
	}

	if sec == nil {
		return *unknownBinaryVersion
	}
	contents, err := sec.Data()
	if err != nil || len(contents) == 0 {
		return *unknownBinaryVersion
	}
	if version := parseMetadataVersion(contents); version != nil {
		return *version
	}
	return *unknownBinaryVersion
}
