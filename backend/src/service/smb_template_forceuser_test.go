package service

import (
	"os"
	"strings"
	"testing"

	"github.com/dianlight/srat/tempio"
	"github.com/stretchr/testify/require"
)

// baseSmbTemplateConfig returns the minimal template context needed to
// render smb.gtpl outside the full ServerService flow.
func baseSmbTemplateConfig() map[string]any {
	return map[string]any{
		"local_master":        true,
		"samba_version":       "4.23.8",
		"smb_over_quic":       false,
		"compatibility_mode":  false,
		"multi_channel":       false,
		"hostname":            "test-host",
		"workgroup":           "WORKGROUP",
		"allow_guest":         false,
		"log_level":           "warning",
		"bind_all_interfaces": true,
		"interfaces":          []string{"wlan0"},
		"docker_interface":    "hassio",
		"docker_net":          "172.30.32.0/23",
		"allow_hosts":         []string{},
		"username":            "admin",
		"medialibrary":        map[string]any{"enable": false},
	}
}

func shareEntry(name, path, fs, forceUser, forceGroup string) map[string]any {
	return map[string]any{
		"name":                name,
		"path":                path,
		"fs":                  fs,
		"users":               []string{"admin"},
		"force_user":          forceUser,
		"force_group":         forceGroup,
		"veto_files":          []string{},
		"disabled":            false,
		"guest_ok":            false,
		"timemachine":         false,
		"recycle_bin_enabled": false,
	}
}

// TestSmbTemplateForceUserOmission verifies the SHT block omits the
// "force user"/"force group" lines when the filesystem adapter resolves
// empty values (NTFS, srat#1264) and keeps them for legacy root shares.
func TestSmbTemplateForceUserOmission(t *testing.T) {
	templateData, err := os.ReadFile("../templates/smb.gtpl")
	require.NoError(t, err)

	config := baseSmbTemplateConfig()
	config["shares"] = []any{
		shareEntry("ntfstest", "/mnt/ntfsvol", "ntfs", "", ""),
		shareEntry("ext4test", "/mnt/ext4vol", "ext4", "root", "root"),
	}

	rendered, renderErr := tempio.RenderTemplateBuffer(&config, templateData)
	require.NoError(t, renderErr)
	out := string(rendered)

	ntfsSection := sectionOf(t, out, "NTFSTEST")
	require.NotContains(t, ntfsSection, "force user", "NTFS section must omit force user")
	require.NotContains(t, ntfsSection, "force group", "NTFS section must omit force group")

	ext4Section := sectionOf(t, out, "EXT4TEST")
	require.Contains(t, ext4Section, "force user = root", "ext4 section keeps legacy force user")
	require.Contains(t, ext4Section, "force group = root", "ext4 section keeps legacy force group")
}

func sectionOf(t *testing.T, rendered, name string) string {
	t.Helper()
	start := strings.Index(rendered, "["+name+"]")
	require.NotEqual(t, -1, start, "section [%s] missing", name)
	rest := rendered[start:]
	next := strings.Index(rest[1:], "\n[")
	if next == -1 {
		return rest
	}
	return rest[:next+1]
}
