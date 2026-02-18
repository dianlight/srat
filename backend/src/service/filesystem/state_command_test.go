package filesystem_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
)

func TestAdaptersStateCommandAvailability(t *testing.T) {
	cases := []struct {
		name        string
		newAdapter  func() filesystem.FilesystemAdapter
		commands    []string
		apfsSpecial bool
	}{
		{
			name:       "ext4",
			newAdapter: filesystem.NewExt4Adapter,
			commands:   []string{"mkfs.ext4", "fsck.ext4", "tune2fs"},
		},
		{
			name:       "xfs",
			newAdapter: filesystem.NewXfsAdapter,
			commands:   []string{"mkfs.xfs", "xfs_repair", "xfs_admin"},
		},
		{
			name:       "btrfs",
			newAdapter: filesystem.NewBtrfsAdapter,
			commands:   []string{"mkfs.btrfs", "btrfs"},
		},
		{
			name:       "f2fs",
			newAdapter: filesystem.NewF2fsAdapter,
			commands:   []string{"mkfs.f2fs", "fsck.f2fs"},
		},
		{
			name:       "gfs2",
			newAdapter: filesystem.NewGfs2Adapter,
			commands:   []string{"mkfs.gfs2", "fsck.gfs2"},
		},
		{
			name:       "ntfs",
			newAdapter: filesystem.NewNtfsAdapter,
			commands:   []string{"mkfs.ntfs", "ntfsfix", "ntfslabel"},
		},
		{
			name:       "vfat",
			newAdapter: filesystem.NewVfatAdapter,
			commands:   []string{"mkfs.vfat", "fsck.vfat", "fatlabel"},
		},
		{
			name:       "exfat",
			newAdapter: filesystem.NewExfatAdapter,
			commands:   []string{"mkfs.exfat", "fsck.exfat", "exfatlabel"},
		},
		{
			name:       "hfsplus",
			newAdapter: filesystem.NewHfsplusAdapter,
			commands:   []string{"mkfs.hfsplus", "fsck.hfsplus"},
		},
		{
			name:       "reiserfs",
			newAdapter: filesystem.NewReiserfsAdapter,
			commands:   []string{"mkfs.reiserfs", "fsck.reiserfs", "reiserfstune"},
		},
		{
			name:        "apfs",
			newAdapter:  filesystem.NewApfsAdapter,
			commands:    []string{"apfsutil"},
			apfsSpecial: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setFakePathWithCommands(t, tc.commands)

			adapter := tc.newAdapter()
			support, err := adapter.IsSupported(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !support.CanGetState {
				t.Fatalf("expected CanGetState to be true for %s", tc.name)
			}

			if len(support.MissingTools) != 0 {
				t.Fatalf("expected no missing tools for %s, got %v", tc.name, support.MissingTools)
			}

			if tc.apfsSpecial {
				if support.CanFormat || support.CanCheck || support.CanSetLabel {
					t.Fatalf("expected APFS format/check/label to be disabled")
				}
			}
		})
	}
}

func setFakePathWithCommands(t *testing.T, commands []string) {
	t.Helper()
	pathDir := t.TempDir()
	seen := make(map[string]struct{})
	for _, name := range commands {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		writeFakeCommand(t, pathDir, name)
	}
	t.Setenv("PATH", pathDir)
}

func writeFakeCommand(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to create fake command %s: %v", name, err)
	}
}
