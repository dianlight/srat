package dto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
)

func TestCleanShareSubfolder(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected string
		wantErr  bool
	}{
		{name: "empty means volume root", raw: "", expected: ""},
		{name: "single folder", raw: "movies", expected: "movies"},
		{name: "nested folder", raw: "photos/2024", expected: "photos/2024"},
		{name: "inner spaces kept", raw: "Italie 2025/day 1", expected: "Italie 2025/day 1"},
		{name: "trailing slash dropped", raw: "photos/", expected: "photos"},
		{name: "blank is not the volume root", raw: "   ", wantErr: true},
		{name: "leading space", raw: " photos", wantErr: true},
		{name: "trailing space in a segment", raw: "photos /2024", wantErr: true},
		{name: "root slash", raw: "/", wantErr: true},
		{name: "dots inside a name are allowed", raw: "..hidden/v1.2", expected: "..hidden/v1.2"},
		{name: "unicode name", raw: "Rodina/Dovolená", expected: "Rodina/Dovolená"},
		{name: "absolute path", raw: "/etc", wantErr: true},
		{name: "parent traversal", raw: "..", wantErr: true},
		{name: "nested parent traversal", raw: "photos/../../etc", wantErr: true},
		{name: "current dir segment", raw: "./photos", wantErr: true},
		{name: "empty segment", raw: "photos//2024", wantErr: true},
		{name: "backslash separator", raw: `photos\2024`, wantErr: true},
		{name: "control character", raw: "pho\x00tos", wantErr: true},
		{name: "segment too long", raw: strings.Repeat("a", 256), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CleanShareSubfolder(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, ErrorShareValidation), "expected ErrorShareValidation, got %v", err)
				assert.Empty(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestShareDirectory(t *testing.T) {
	assert.Equal(t, "/mnt/DATA1", ShareDirectory("/mnt/DATA1", ""))
	assert.Equal(t, "/mnt/DATA1/emperor", ShareDirectory("/mnt/DATA1", "emperor"))
	assert.Equal(t, "/mnt/DATA1/photos/2024", ShareDirectory("/mnt/DATA1", "photos/2024"))
}

func TestSubfolderOrRoot(t *testing.T) {
	var nilShare *SharedResource
	assert.Empty(t, nilShare.SubfolderOrRoot())
	assert.Empty(t, (&SharedResource{}).SubfolderOrRoot())
	assert.Equal(t, "emperor", (&SharedResource{Subfolder: new("emperor")}).SubfolderOrRoot())
}
