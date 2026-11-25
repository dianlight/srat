package service

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/osutil"
	"gitlab.com/tozd/go/errors"
)

// isMountPoint is mockable so tests can use temporary directories as volumes.
var isMountPoint = osutil.IsMounted

// MockIsMountPoint allows overriding the mount point check in tests.
func MockIsMountPoint(fn func(string) (bool, error)) {
	isMountPoint = fn
}

// resolveShareDirectory returns the directory a share exposes (issue #184):
// the mount point itself, or the subfolder inside it. subfolder must already
// be clean (dto.CleanShareSubfolder).
//
// Each existing segment is resolved through symlinks and must stay inside the
// real volume root before the walk goes deeper, so a share can never expose,
// or create folders in, anything outside its volume. With create, missing
// folders are made only when the volume is actually mounted (never in the
// empty mount point directory) and copy the volume root's mode and owner, so
// the share keeps the permission regime of the volume it lives on.
func resolveShareDirectory(mountPath, subfolder string, create bool) (string, errors.E) {
	if subfolder == "" {
		return mountPath, nil
	}
	if create {
		mounted, err := isMountPoint(mountPath)
		if err != nil || !mounted {
			return "", shareDirectoryError("volume is not mounted", mountPath)
		}
	}

	realRoot, err := filepath.EvalSymlinks(mountPath)
	if err != nil {
		return "", shareDirectoryError("volume is not accessible", mountPath)
	}
	rootInfo, err := os.Stat(realRoot)
	if err != nil {
		return "", shareDirectoryError("volume is not accessible", mountPath)
	}

	current := mountPath
	for segment := range strings.SplitSeq(subfolder, "/") {
		current = filepath.Join(current, segment)
		_, err := os.Lstat(current)
		switch {
		case err == nil:
			realDir, errE := filepath.EvalSymlinks(current)
			if errE != nil {
				return "", shareDirectoryError("subfolder is not accessible", current)
			}
			if !isWithin(realRoot, realDir) {
				return "", shareDirectoryError("subfolder points outside the volume", current)
			}
			if info, errS := os.Stat(realDir); errS != nil || !info.IsDir() {
				return "", shareDirectoryError("subfolder is not a directory", current)
			}
		case errors.Is(err, fs.ErrNotExist) && create:
			if errM := mkdirLike(current, rootInfo); errM != nil {
				return "", errM
			}
		case errors.Is(err, fs.ErrNotExist):
			return "", shareDirectoryError("subfolder does not exist", current)
		default:
			return "", shareDirectoryError("subfolder is not accessible", current)
		}
	}
	return current, nil
}

// mkdirLike creates dir with the mode and owner of the volume root.
func mkdirLike(dir string, rootInfo fs.FileInfo) errors.E {
	mode := rootInfo.Mode() & (fs.ModePerm | fs.ModeSetgid | fs.ModeSticky)
	if err := os.Mkdir(dir, mode.Perm()); err != nil {
		return errors.WithDetails(dto.ErrorShareValidation,
			"Message", "cannot create subfolder", "Path", dir, "Error", err.Error())
	}
	// Mkdir applies the umask and drops the special bits: set the exact mode.
	if err := os.Chmod(dir, mode); err != nil {
		slog.Warn("Cannot copy volume permissions to share subfolder", "path", dir, "err", err)
	}
	if stat, ok := rootInfo.Sys().(*syscall.Stat_t); ok {
		if err := os.Lchown(dir, int(stat.Uid), int(stat.Gid)); err != nil {
			slog.Warn("Cannot copy volume owner to share subfolder", "path", dir, "err", err)
		}
	}
	return nil
}

// maxListedFolders caps one folder listing; the UI filters within it.
var maxListedFolders = 1000

// ListVolumeFolders lists the folders inside `subfolder` of a mounted volume,
// for picking a share subfolder (issue #184). The volume must be a known,
// mounted mount point, and the listed folder resolves through the same
// containment checks as share directories, so nothing outside the volume is
// ever listed. Hidden folders and the volume root's lost+found are skipped;
// symlinks are not listed.
func (s *ShareService) ListVolumeFolders(mountPath, subfolder string) (*dto.VolumeFolders, errors.E) {
	if mp, ok := s.disks.GetMountPointByPath(mountPath); !ok || mp == nil || !mp.IsMounted || mp.IsInvalid {
		return nil, errors.WithDetails(dto.ErrorNotFound, "Message", "volume is not mounted", "Path", mountPath)
	}
	clean, err := dto.CleanShareSubfolder(subfolder)
	if err != nil {
		return nil, err
	}
	dir, err := resolveShareDirectory(mountPath, clean, false)
	if err != nil {
		return nil, err
	}
	entries, errR := os.ReadDir(dir)
	if errR != nil {
		return nil, shareDirectoryError("folder is not readable", dir)
	}
	result := &dto.VolumeFolders{MountPath: mountPath, Path: clean, Folders: []string{}}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || strings.HasPrefix(name, ".") || (clean == "" && name == "lost+found") {
			continue
		}
		if len(result.Folders) == maxListedFolders {
			result.Truncated = true
			break
		}
		result.Folders = append(result.Folders, name)
	}
	slices.SortFunc(result.Folders, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
	return result, nil
}

// isWithin reports whether path is root or inside it. Both must be clean.
func isWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, "../")
}

func shareDirectoryError(message, path string) errors.E {
	return errors.WithDetails(dto.ErrorShareValidation, "Message", message, "Path", path)
}
