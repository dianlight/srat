package tempio

import (
	"fmt"
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
			wantLine: "vfs objects = acl_xattr catia fruit streams_xattr recycle",
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
			wantLine: "vfs objects = acl_xattr catia fruit streams_xattr",
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

// smbSection isolates directives so a global default cannot mask a share override.
func smbSection(t *testing.T, rendered []byte, name string) string {
	t.Helper()
	re := regexp.MustCompile(`(?ms)^\[` + regexp.QuoteMeta(name) + `\]\n(.*?)(?:^\[|\z)`)
	match := re.FindStringSubmatch(string(rendered))
	require.Len(t, match, 2, "missing section %s", name)
	return match[1]
}

func TestSmbTemplateVFSStack(t *testing.T) {
	for _, fs := range []string{"native", "ext4", "btrfs", "f2fs", "fuseblk", "exfat", "vfat", "msdos"} {
		for _, timeMachine := range []bool{false, true} {
			for _, recycle := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/tm=%t/recycle=%t", fs, timeMachine, recycle), func(t *testing.T) {
					data := smbConfigForShare(map[string]any{
						"name": "TEST_SHARE", "path": "/test", "fs": fs,
						"users": []any{"admin"}, "timemachine": timeMachine,
						"timemachine_max_size": "500G", "recycle_bin_enabled": recycle,
					})
					rendered, err := RenderTemplateBuffer(data, loadSmbTemplate(t))
					require.NoError(t, err)
					section := smbSection(t, rendered, "TEST_SHARE")
					noXattrs := fs == "exfat" || fs == "vfat" || fs == "msdos"
					assert.Equal(t, noXattrs, strings.Contains(section, "# Known FAT filesystems cannot store streams_xattr data"))
					assert.Equal(t, noXattrs, strings.Contains(section, "# This is not full Apple metadata support on these filesystems."))
					want := "vfs objects = acl_xattr catia fruit streams_xattr"
					if noXattrs {
						want = "vfs objects = acl_xattr catia fruit"
					}
					if recycle {
						want += " recycle"
					}
					line := regexp.MustCompile(`(?m)^\s*vfs objects =[^\n]*`).FindString(section)
					assert.Equal(t, want, strings.TrimSpace(line))
					tmSupported := !noXattrs && fs != "f2fs" && fs != "fuseblk"
					assert.Equal(t, timeMachine && tmSupported, strings.Contains(section, "fruit:time machine = yes"))
					assert.Equal(t, timeMachine && tmSupported, strings.Contains(section, "fruit:time machine max size = 500G"))
					global := smbSection(t, rendered, "global")
					assert.Contains(t, global, "vfs objects = acl_xattr catia fruit streams_xattr")
					assert.Contains(t, global, "fruit:aapl = yes")
				})
			}
		}
	}
}

func TestSmbTemplateMixedSharesAAPL(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("fat_disabled=%t", disabled), func(t *testing.T) {
			data := smbConfigForShare(map[string]any{
				"name": "TEST_SHARE", "path": "/test", "fs": "ext4", "timemachine": true,
			})
			(*data)["shares"].(map[string]any)["FAT_SHARE"] = map[string]any{
				"name": "FAT_SHARE", "path": "/fat", "fs": "exfat", "disabled": disabled,
			}
			rendered, err := RenderTemplateBuffer(data, loadSmbTemplate(t))
			require.NoError(t, err)
			global := smbSection(t, rendered, "global")
			assert.Contains(t, global, "fruit:aapl = yes")
			assert.Contains(t, smbSection(t, rendered, "TEST_SHARE"), "fruit:time machine = yes")
			if disabled {
				assert.NotContains(t, string(rendered), "[FAT_SHARE]")
			} else {
				assert.Contains(t, smbSection(t, rendered, "FAT_SHARE"), "vfs objects = acl_xattr catia fruit")
			}
		})
	}
}
