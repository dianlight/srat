package dto

import (
	"path/filepath"
	"strings"
	"unicode"

	"gitlab.com/tozd/go/errors"
)

// maxSubfolderSegmentLength mirrors NAME_MAX on Linux filesystems.
const maxSubfolderSegmentLength = 255

// CleanShareSubfolder validates a share subfolder (issue #184) and returns it
// in canonical form: "/"-separated segments without leading or trailing "/".
// An empty value means the volume root.
//
// Absolute paths, "." and ".." segments, empty segments, backslashes and
// control characters are rejected, so a subfolder can never point outside
// its volume by its spelling. Symlinks are checked against the real
// filesystem by the share service.
func CleanShareSubfolder(raw string) (string, errors.E) {
	subfolder := strings.TrimSpace(raw)
	if subfolder == "" {
		return "", nil
	}
	if strings.HasPrefix(subfolder, "/") {
		return "", invalidSubfolder(raw, "subfolder must be relative to the volume")
	}
	if strings.ContainsRune(subfolder, '\\') {
		return "", invalidSubfolder(raw, `subfolder must use "/" as separator`)
	}
	segments := strings.Split(strings.TrimSuffix(subfolder, "/"), "/")
	for _, segment := range segments {
		switch {
		case segment == "":
			return "", invalidSubfolder(raw, "subfolder must not contain empty segments")
		case segment == "." || segment == "..":
			return "", invalidSubfolder(raw, `subfolder must not contain "." or ".." segments`)
		case len(segment) > maxSubfolderSegmentLength:
			return "", invalidSubfolder(raw, "subfolder segment is too long")
		case strings.ContainsFunc(segment, unicode.IsControl):
			return "", invalidSubfolder(raw, "subfolder must not contain control characters")
		}
	}
	return strings.Join(segments, "/"), nil
}

func invalidSubfolder(raw, message string) errors.E {
	return errors.WithDetails(ErrorShareValidation, "Message", message, "Subfolder", raw)
}

// ShareDirectory returns the directory a share exposes: the mount point
// itself, or the subfolder inside it.
func ShareDirectory(mountPath, subfolder string) string {
	if subfolder == "" {
		return mountPath
	}
	return filepath.Join(mountPath, subfolder)
}

// SubfolderOrRoot returns the share's subfolder, or "" (the volume root)
// when none is set.
func (s *SharedResource) SubfolderOrRoot() string {
	if s == nil || s.Subfolder == nil {
		return ""
	}
	return *s.Subfolder
}
