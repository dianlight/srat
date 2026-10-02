package tempio

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// extractVfsObjectsLine returns the first "vfs objects = ..." line rendered
// after the given section header (e.g. "[TEST_SHARE]"), trimmed of
// surrounding whitespace.
func extractVfsObjectsLine(t *testing.T, rendered []byte, section string) string {
	t.Helper()
	rest := string(rendered)
	idx := strings.Index(rest, section)
	require.GreaterOrEqual(t, idx, 0, "section %s not found in rendered config:\n%s", section, rest)
	rest = rest[idx:]
	re := regexp.MustCompile(`(?m)^[ \t]*vfs objects =.*$`)
	line := re.FindString(rest)
	require.NotEmpty(t, line, "vfs objects line not found after section %s:\n%s", section, rest)
	return strings.TrimSpace(line)
}

// TestSmbTemplateVfsObjects verifies issue #1312: whitespace trim markers on
// the optional recycle module must not concatenate with the preceding module
// name. The Time Machine branch previously rendered
// "streams_xattrrecycle" (an invalid VFS module) when recycle_bin_enabled
// was true.
func TestSmbTemplateVfsObjects(t *testing.T) {
	tests := []struct {
		name     string
		share    map[string]any
		wantLine string
	}{
		{
			name: "time machine share with recycle bin enabled",
			share: map[string]any{
				"name":                "TEST_SHARE",
				"path":                "/test",
				"fs":                  "ext4",
				"users":               []any{"admin"},
				"timemachine":         true,
				"recycle_bin_enabled": true,
			},
			wantLine: "vfs objects = acl_xattr catia fruit streams_xattr recycle",
		},
		{
			name: "time machine share without recycle bin",
			share: map[string]any{
				"name":        "TEST_SHARE",
				"path":        "/test",
				"fs":          "ext4",
				"users":       []any{"admin"},
				"timemachine": true,
			},
			wantLine: "vfs objects = acl_xattr catia fruit streams_xattr",
		},
		{
			name: "ordinary share with recycle bin enabled",
			share: map[string]any{
				"name":                "TEST_SHARE",
				"path":                "/test",
				"fs":                  "ext4",
				"users":               []any{"admin"},
				"timemachine":         false,
				"recycle_bin_enabled": true,
			},
			wantLine: "vfs objects = acl_xattr catia fruit recycle",
		},
		{
			name: "ordinary share without recycle bin",
			share: map[string]any{
				"name":        "TEST_SHARE",
				"path":        "/test",
				"fs":          "ext4",
				"users":       []any{"admin"},
				"timemachine": false,
			},
			wantLine: "vfs objects = acl_xattr catia fruit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := smbConfigForShare(tt.share)

			rendered, err := RenderTemplateBuffer(data, loadSmbTemplate(t))
			require.NoError(t, err)

			got := extractVfsObjectsLine(t, rendered, "[TEST_SHARE]")
			assert.Equal(t, tt.wantLine, got)
			assert.NotContains(t, got, "streams_xattrrecycle",
				"recycle must be a separate token, never concatenated with streams_xattr")
		})
	}
}
