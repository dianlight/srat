package tempio

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
