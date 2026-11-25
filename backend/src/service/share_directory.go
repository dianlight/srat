package service

import (
	"io/fs"
	"log/slog"
	"os"
	"path"
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

// openVolume opens the volume root for confined access. Every name resolved
// through the returned os.Root stays inside the volume: symlinks are followed
// only while they stay inside it, and lookups and creations happen relative to
// the open directory, so a symlink swapped in meanwhile cannot redirect them
// outside the volume.
func openVolume(mountPath string) (*os.Root, errors.E) {
	root, err := os.OpenRoot(mountPath)
	if err != nil {
		return nil, shareDirectoryError("volume is not accessible", mountPath)
	}
	return root, nil
}

// rootName is the os.Root name of a clean subfolder ("" is the volume root).
func rootName(subfolder string) string {
	if subfolder == "" {
		return "."
	}
	return subfolder
}

// resolveShareDirectory returns the directory a share exposes (issue #184):
// the mount point itself, or the subfolder inside it. subfolder must already
// be clean (dto.CleanShareSubfolder).
//
// The subfolder is walked segment by segment through os.Root, so a share can
// never expose, or create folders in, anything outside its volume. With
// create, missing folders are made only when the volume is actually mounted
// (never in the empty mount point directory) and copy the volume root's mode
// and owner, so the share keeps the permission regime of the volume it lives on.
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

	root, errO := openVolume(mountPath)
	if errO != nil {
		return "", errO
	}
	defer root.Close()
	rootInfo, err := root.Stat(".")
	if err != nil {
		return "", shareDirectoryError("volume is not accessible", mountPath)
	}

	current := ""
	for segment := range strings.SplitSeq(subfolder, "/") {
		current = path.Join(current, segment)
		full := filepath.Join(mountPath, current)
		info, err := root.Stat(current)
		switch {
		case err == nil:
			if !info.IsDir() {
				return "", shareDirectoryError("subfolder is not a directory", full)
			}
		case errors.Is(err, fs.ErrNotExist) && create:
			if errM := mkdirLike(root, current, full, rootInfo); errM != nil {
				return "", errM
			}
		case errors.Is(err, fs.ErrNotExist):
			return "", shareDirectoryError("subfolder does not exist", full)
		default:
			// os.Root refuses names that resolve outside the volume.
			return "", shareDirectoryError("subfolder points outside the volume or is not accessible", full)
		}
	}
	return filepath.Join(mountPath, subfolder), nil
}

// mkdirLike creates name inside root with the mode and owner of the volume root.
func mkdirLike(root *os.Root, name, full string, rootInfo fs.FileInfo) errors.E {
	mode := rootInfo.Mode() & (fs.ModePerm | fs.ModeSetgid | fs.ModeSticky)
	if err := root.Mkdir(name, mode.Perm()); err != nil {
		return errors.WithDetails(dto.ErrorShareValidation,
			"Message", "cannot create subfolder", "Path", full, "Error", err.Error())
	}
	// Mkdir applies the umask and drops the special bits: set the exact mode.
	if err := root.Chmod(name, mode); err != nil {
		slog.Warn("Cannot copy volume permissions to share subfolder", "path", full, "err", err)
	}
	if stat, ok := rootInfo.Sys().(*syscall.Stat_t); ok {
		if err := root.Lchown(name, int(stat.Uid), int(stat.Gid)); err != nil {
			slog.Warn("Cannot copy volume owner to share subfolder", "path", full, "err", err)
		}
	}
	return nil
}

// sameDirectoryAsAny reports whether subfolder is the same directory as one of
// others on the volume, for example through a symlink inside the volume.
// Folders that cannot be resolved are not considered the same.
func sameDirectoryAsAny(mountPath, subfolder string, others []string) bool {
	if len(others) == 0 {
		return false
	}
	root, err := openVolume(mountPath)
	if err != nil {
		return false
	}
	defer root.Close()
	target, errS := root.Stat(rootName(subfolder))
	if errS != nil {
		return false
	}
	return slices.ContainsFunc(others, func(other string) bool {
		info, err := root.Stat(rootName(other))
		return err == nil && os.SameFile(target, info)
	})
}

// maxListedFolders caps one folder listing; beyond it the path can be typed.
var maxListedFolders = 1000

// ListVolumeFolders lists the folders inside `subfolder` of a mounted volume,
// for picking a share subfolder (issue #184). The volume must be a known,
// mounted mount point, and the folder is opened through os.Root, so nothing
// outside the volume is ever listed. Hidden folders and the volume root's
// lost+found are skipped; symlinks are not listed.
func (s *ShareService) ListVolumeFolders(mountPath, subfolder string) (*dto.VolumeFolders, errors.E) {
	if mp, ok := s.disks.GetMountPointByPath(mountPath); !ok || mp == nil || !mp.IsMounted || mp.IsInvalid {
		return nil, errors.WithDetails(dto.ErrorNotFound, "Message", "volume is not mounted", "Path", mountPath)
	}
	clean, err := dto.CleanShareSubfolder(subfolder)
	if err != nil {
		return nil, err
	}
	if _, err := resolveShareDirectory(mountPath, clean, false); err != nil {
		return nil, err
	}
	root, err := openVolume(mountPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	full := filepath.Join(mountPath, clean)
	dir, errO := root.Open(rootName(clean))
	if errO != nil {
		return nil, shareDirectoryError("folder is not readable", full)
	}
	defer dir.Close()
	entries, errR := dir.ReadDir(-1)
	if errR != nil {
		return nil, shareDirectoryError("folder is not readable", full)
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

func shareDirectoryError(message, path string) errors.E {
	return errors.WithDetails(dto.ErrorShareValidation, "Message", message, "Path", path)
}
