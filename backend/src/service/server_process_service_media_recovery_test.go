package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

type mediaRecoveryShareServiceStub struct {
	shares []dto.SharedResource
	err    errors.E
}

func (s *mediaRecoveryShareServiceStub) ListShares() ([]dto.SharedResource, errors.E) {
	if s.err != nil {
		return nil, s.err
	}
	return s.shares, nil
}

func (s *mediaRecoveryShareServiceStub) GetShare(string) (*dto.SharedResource, errors.E) {
	return nil, errors.New("not implemented")
}

func (s *mediaRecoveryShareServiceStub) CreateShare(dto.SharedResource) (*dto.SharedResource, errors.E) {
	return nil, errors.New("not implemented")
}

func (s *mediaRecoveryShareServiceStub) UpdateShare(string, dto.SharedResource) (*dto.SharedResource, errors.E) {
	return nil, errors.New("not implemented")
}

func (s *mediaRecoveryShareServiceStub) DeleteShare(string) errors.E {
	return errors.New("not implemented")
}

func (s *mediaRecoveryShareServiceStub) DisableShare(string) (*dto.SharedResource, errors.E) {
	return nil, errors.New("not implemented")
}

func (s *mediaRecoveryShareServiceStub) EnableShare(string) (*dto.SharedResource, errors.E) {
	return nil, errors.New("not implemented")
}

func (s *mediaRecoveryShareServiceStub) GetShareFromPath(string) (*dto.SharedResource, errors.E) {
	return nil, errors.New("not implemented")
}

func (s *mediaRecoveryShareServiceStub) SetShareFromPathEnabled(string, bool) (*dto.SharedResource, errors.E) {
	return nil, errors.New("not implemented")
}

func (s *mediaRecoveryShareServiceStub) VerifyShare(*dto.SharedResource) errors.E {
	return nil
}

func (s *mediaRecoveryShareServiceStub) SetSupervisorService(SupervisorServiceInterface) {
}

func TestRecoverMediaUsageSymlinks_CreatesMissingSymlink(t *testing.T) {
	targetRoot := t.TempDir()
	targetPath := filepath.Join(targetRoot, "mnt", "media-share")
	err := os.MkdirAll(targetPath, 0o755)
	if err != nil {
		t.Fatalf("failed to create target path: %v", err)
	}

	linkRoot := filepath.Join(targetRoot, "media")
	svc := &ServerService{
		share_service: &mediaRecoveryShareServiceStub{
			shares: []dto.SharedResource{
				{
					Name:     "SERVER",
					Usage:    dto.UsageAsMedia,
					Disabled: new(false),
					Status:   &dto.SharedResourceStatus{IsValid: true},
					MountPointData: &dto.MountPointData{
						Path: targetPath,
					},
				},
			},
		},
	}

	err = svc.recoverMediaUsageSymlinks(context.Background(), linkRoot)
	if err != nil {
		t.Fatalf("recoverMediaUsageSymlinks returned error: %v", err)
	}

	linkPath := filepath.Join(linkRoot, "SERVER")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("expected symlink to exist: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to be a symlink", linkPath)
	}

	resolved, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}
	if resolved != targetPath {
		t.Fatalf("expected symlink target %q, got %q", targetPath, resolved)
	}
}

func TestRecoverMediaUsageSymlinks_ReplacesStaleSymlink(t *testing.T) {
	targetRoot := t.TempDir()
	oldTarget := filepath.Join(targetRoot, "mnt", "old")
	newTarget := filepath.Join(targetRoot, "mnt", "new")
	err := os.MkdirAll(oldTarget, 0o755)
	if err != nil {
		t.Fatalf("failed to create old target: %v", err)
	}
	err = os.MkdirAll(newTarget, 0o755)
	if err != nil {
		t.Fatalf("failed to create new target: %v", err)
	}

	linkRoot := filepath.Join(targetRoot, "media")
	err = os.MkdirAll(linkRoot, 0o755)
	if err != nil {
		t.Fatalf("failed to create link root: %v", err)
	}

	linkPath := filepath.Join(linkRoot, "SERVER")
	err = os.Symlink(oldTarget, linkPath)
	if err != nil {
		t.Fatalf("failed to create initial symlink: %v", err)
	}

	svc := &ServerService{
		share_service: &mediaRecoveryShareServiceStub{
			shares: []dto.SharedResource{
				{
					Name:     "SERVER",
					Usage:    dto.UsageAsMedia,
					Disabled: new(false),
					Status:   &dto.SharedResourceStatus{IsValid: true},
					MountPointData: &dto.MountPointData{
						Path: newTarget,
					},
				},
			},
		},
	}

	err = svc.recoverMediaUsageSymlinks(context.Background(), linkRoot)
	if err != nil {
		t.Fatalf("recoverMediaUsageSymlinks returned error: %v", err)
	}

	resolved, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("failed to read updated symlink: %v", err)
	}
	if resolved != newTarget {
		t.Fatalf("expected updated symlink target %q, got %q", newTarget, resolved)
	}
}