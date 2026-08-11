package service

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetectBestServerVariantWithLinkerCheck covers the smartlib variant policy
// (Task 2/4): musl dynamic preferred on musl systems, glibc dynamic on glibc
// systems, static as the safe fallback. The linker probe is injected so the
// tests are independent of the test machine's libc layout.
func TestDetectBestServerVariantWithLinkerCheck(t *testing.T) {
	// Target dir must exist and contain the variants for them to be considered.
	targetDir := t.TempDir()
	muslLinker := "/lib/ld-musl-x86_64.so.1"
	if runtime.GOARCH == "arm64" {
		muslLinker = "/lib/ld-musl-aarch64.so.1"
	}
	allVariantPaths := []string{"srat-server-musl", "srat-server-glib", "srat-server-static"}
	for _, name := range allVariantPaths {
		require.NoError(t, os.WriteFile(filepath.Join(targetDir, name), []byte("bin"), 0o755))
	}

	packageWithVariants := func(names ...string) *UpdatePackage {
		var files []UpdateFile
		for _, name := range names {
			files = append(files, UpdateFile{Path: filepath.Join("/updates", name)})
		}
		return &UpdatePackage{FilesPaths: files}
	}

	tests := []struct {
		name         string
		updatePkg    *UpdatePackage
		linkerExists func(string) bool
		want         string
	}{
		{
			name:         "musl system prefers musl variant",
			updatePkg:    packageWithVariants(allVariantPaths...),
			linkerExists: func(p string) bool { return p == muslLinker },
			want:         "srat-server-musl",
		},
		{
			name:         "glibc system prefers glibc variant",
			updatePkg:    packageWithVariants(allVariantPaths...),
			linkerExists: func(p string) bool { return p == "/lib64/ld-linux-x86-64.so.2" || p == "/lib/ld-linux-aarch64.so.1" },
			want:         "srat-server-glib",
		},
		{
			name:         "no linker detected falls back to static",
			updatePkg:    packageWithVariants(allVariantPaths...),
			linkerExists: func(string) bool { return false },
			want:         "srat-server-static",
		},
		{
			name:         "static is chosen when dynamic variants absent from package",
			updatePkg:    packageWithVariants("srat-server-static"),
			linkerExists: func(p string) bool { return p == muslLinker },
			want:         "srat-server-static",
		},
		{
			name:         "musl variant not packaged falls back to glibc",
			updatePkg:    packageWithVariants("srat-server-glib", "srat-server-static"),
			linkerExists: func(p string) bool { return p == "/lib64/ld-linux-x86-64.so.2" },
			want:         "srat-server-glib",
		},
		{
			name:         "nil update package falls back to static",
			updatePkg:    nil,
			linkerExists: func(string) bool { return true },
			want:         "srat-server-static",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectBestServerVariantWithLinkerCheck(targetDir, tt.updatePkg, tt.linkerExists)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDetectBestServerVariantWrapper exercises the production wrapper that probes
// the real system linker layout via os.Stat. It must always return one of the
// three known variant names regardless of the host libc (musl on Alpine CI,
// glibc on Debian/Ubuntu CI, static elsewhere).
func TestDetectBestServerVariantWrapper(t *testing.T) {
	targetDir := t.TempDir()
	for _, name := range []string{"srat-server-musl", "srat-server-glib", "srat-server-static"} {
		require.NoError(t, os.WriteFile(filepath.Join(targetDir, name), []byte("bin"), 0o755))
	}

	got := detectBestServerVariant(targetDir, &UpdatePackage{
		FilesPaths: []UpdateFile{
			{Path: "/updates/srat-server-musl"},
			{Path: "/updates/srat-server-glib"},
			{Path: "/updates/srat-server-static"},
		},
	})
	assert.Contains(t, []string{"srat-server-musl", "srat-server-glib", "srat-server-static"}, got)
}
