package service

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildComponentArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	buffer := new(bytes.Buffer)
	writer := zip.NewWriter(buffer)

	for name, content := range files {
		entry, err := writer.Create(name)
		require.NoError(t, err)
		_, err = entry.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, writer.Close())

	return buffer.Bytes()
}

func TestReadManifestVersionFromCustomComponentArchive(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string
		expected    string
		expectError string
	}{
		{
			name:     "root layout",
			files:    map[string]string{"manifest.json": `{"version":"2026.05.1"}`},
			expected: "2026.05.1",
		},
		{
			name:     "legacy nested layout",
			files:    map[string]string{"srat/manifest.json": `{"version":"2026.05.1"}`},
			expected: "2026.05.1",
		},
		{
			name: "prefers root over nested",
			files: map[string]string{
				"manifest.json":      `{"version":"2026.06.1"}`,
				"srat/manifest.json": `{"version":"2026.05.1"}`,
			},
			expected: "2026.06.1",
		},
		{
			name:        "missing manifest",
			files:       map[string]string{"sensor.py": "# sensor"},
			expectError: "missing manifest.json",
		},
		{
			name:        "empty version",
			files:       map[string]string{"manifest.json": `{"version":""}`},
			expectError: "empty version",
		},
		{
			name:        "empty archive",
			files:       nil,
			expectError: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var archive []byte
			if tt.files != nil {
				archive = buildComponentArchive(t, tt.files)
			}

			version, err := readManifestVersionFromCustomComponentArchive(archive)
			if tt.expectError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, version)
		})
	}
}
