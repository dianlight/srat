# Design

## Context

Share creation currently resolves the Samba share path directly from the mounted volume root. The share model carries only the mount binding; the Samba template renders the share path from the mount data.

## Goals / Non-Goals

**Goals:**

- Allow a share to point at a subdirectory of its mount.
- Keep the empty-subfolder case backward compatible (mount root).
- Prevent path traversal in user-supplied subfolder values.

**Non-Goals:**

- Multi-subfolder (nested) shares per volume.
- Permission/ACL mapping strategies beyond default creation.

## Decisions

- Store the subfolder as an optional relative path; empty means mount root (backward compatible).
- Validate the subfolder: reject absolute paths, parent traversal, and empty segments.
- Resolve via `filepath.Join(mountRoot, subfolder)` and create the directory when missing.
- Render the resolved path through the Samba template.

## Risks / Trade-offs

- Migration risk is low because the new column defaults to empty (mount root).
- Filesystem creation failures surface without corrupting share state.
- The goose migration tolerates the column already existing (created by GORM AutoMigrate).
