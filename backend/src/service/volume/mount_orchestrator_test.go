// Package volume_test: orchestrator suites (validation, unmount fallback,
// settings patch, automount guard) with hand-written fakes.
package volume_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/service/volume"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
)

type fakeOrchestratorFS struct {
	flagsErr errors.E
	label    string
	labelErr errors.E
}

func (f *fakeOrchestratorFS) MountFlagsToSyscallFlagAndData(input []dto.MountFlag) (uintptr, string, errors.E) {
	if f.flagsErr != nil {
		return 0, "", f.flagsErr
	}
	return 0, "", nil
}

func (f *fakeOrchestratorFS) GetPartitionLabel(_ context.Context, _, _ string) (string, errors.E) {
	return f.label, f.labelErr
}

type fakeOrchestratorMounter struct {
	mountCalls       int
	unmountCalls     int
	mountErr         errors.E
	lastUnmountPath  string
	lastUnmountForce bool
}

func (f *fakeOrchestratorMounter) Mount(md *dto.MountPointData, _ uintptr, _, _ string) errors.E {
	f.mountCalls++
	if f.mountErr != nil {
		return f.mountErr
	}
	md.IsMounted = true
	return nil
}

func (f *fakeOrchestratorMounter) Unmount(md *dto.MountPointData, force bool) errors.E {
	f.unmountCalls++
	f.lastUnmountPath = md.Path
	f.lastUnmountForce = force
	md.IsMounted = false
	return nil
}

// MountOrchestratorTestSuite exercises the orchestrator through its public
// API: validation errors, protected mode, already-mounted, unmount fallback,
// settings patch and the automount guard.
type MountOrchestratorTestSuite struct {
	volumeTestFixture
	orchestrator *volume.MountOrchestrator
	mounter      *fakeOrchestratorMounter
	protected    bool
}

func TestMountOrchestratorTestSuite(t *testing.T) {
	suite.Run(t, new(MountOrchestratorTestSuite))
}

func (s *MountOrchestratorTestSuite) SetupTest() {
	s.volumeTestFixture.SetupTest()
	s.mounter = &fakeOrchestratorMounter{}
	s.protected = false
	s.orchestrator = volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx:        s.ctx,
		Disks:      s.disks,
		Filesystem: &fakeOrchestratorFS{},
		Mounter:    s.mounter,
		EventBus:   s.eventBus,
		Repo:       s.mountRepoIf,
		ProtectedMode: func() bool {
			return s.protected
		},
		Volumes: func() ([]*dto.Disk, errors.E) {
			return s.disks.All(), nil
		},
	})
}

func (s *MountOrchestratorTestSuite) TearDownTest() {
	s.volumeTestFixture.TearDownTest()
}

func (s *MountOrchestratorTestSuite) TestMountVolume_Nil() {
	err := s.orchestrator.MountVolume(nil)
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorInvalidParameter)
}

func (s *MountOrchestratorTestSuite) TestMountVolume_EmptyPath() {
	err := s.orchestrator.MountVolume(&dto.MountPointData{DeviceId: "part-x"})
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorInvalidParameter)
	s.Zero(s.mounter.mountCalls)
}

func (s *MountOrchestratorTestSuite) TestMountVolume_ProtectedMode() {
	s.protected = true
	err := s.orchestrator.MountVolume(&dto.MountPointData{Path: "/mnt/x", Root: "/", DeviceId: "part-x"})
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorOperationNotPermittedInProtectedMode)
	s.Zero(s.mounter.mountCalls)
}

func (s *MountOrchestratorTestSuite) TestMountVolume_AlreadyMounted() {
	mountPath := "/mnt/orch-already-mounted"
	restore := osutil.MockMountInfo("1217 819 0:52 / " + mountPath + " rw,relatime - ext4 /dev/sda1 rw\n")
	s.T().Cleanup(restore)

	partID := "part-orch-1"
	devFile := filepath.Join(s.T().TempDir(), "device.img")
	s.Require().NoError(os.WriteFile(devFile, []byte("test"), 0o600))
	partition := dto.Partition{Id: &partID, DevicePath: &devFile}
	md := &dto.MountPointData{
		Path: mountPath, Root: "/", DeviceId: partID,
		Flags: &dto.MountFlags{}, Partition: &partition,
	}

	err := s.orchestrator.MountVolume(md)
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorAlreadyMounted)
	s.Zero(s.mounter.mountCalls, "OS-level mount must not run when already mounted")
}

func (s *MountOrchestratorTestSuite) TestUnmountVolume_NotFound_DelegatesToMounter() {
	err := s.orchestrator.UnmountVolume("/mnt/orch-missing", false)
	s.Require().NoError(err)
	s.Equal(1, s.mounter.unmountCalls)
	s.Equal("/mnt/orch-missing", s.mounter.lastUnmountPath)
	s.False(s.mounter.lastUnmountForce)
}

func (s *MountOrchestratorTestSuite) TestUnmountVolume_ProtectedMode() {
	s.protected = true
	err := s.orchestrator.UnmountVolume("/mnt/x", false)
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorOperationNotPermittedInProtectedMode)
	s.Zero(s.mounter.unmountCalls)
}

func (s *MountOrchestratorTestSuite) TestPatchMountPointSettings_NotFound() {
	got, err := s.orchestrator.PatchMountPointSettings("/", "/mnt/orch-missing", dto.MountPointData{})
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorNotFound)
	s.Nil(got)
}

func (s *MountOrchestratorTestSuite) TestGetDevicePathByDeviceID_NotFound() {
	_, err := s.orchestrator.GetDevicePathByDeviceID("no-such-partition")
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorNotFound)
}

func (s *MountOrchestratorTestSuite) TestAutomountGuard_Exhaustion() {
	s.orchestrator.SetAutomountTuning(0, 2)

	allowed, exhausted := s.orchestrator.AllowAutomountAttempt("/mnt/guard")
	s.True(allowed)
	s.False(exhausted)

	s.orchestrator.RecordAutomountFailure("/mnt/guard")
	s.orchestrator.RecordAutomountFailure("/mnt/guard")

	allowed, exhausted = s.orchestrator.AllowAutomountAttempt("/mnt/guard")
	s.False(allowed)
	s.True(exhausted)

	s.orchestrator.ClearAutomountRetry("/mnt/guard")
	allowed, exhausted = s.orchestrator.AllowAutomountAttempt("/mnt/guard")
	s.True(allowed)
	s.False(exhausted)
}

func TestInferMountPointType(t *testing.T) {
	cases := []struct {
		name string
		mp   *dto.MountPointData
		want string
	}{
		{"nil", nil, "ADDON"},
		{"empty", &dto.MountPointData{}, "ADDON"},
		{"mnt path", &dto.MountPointData{Path: "/mnt/share"}, "ADDON"},
		{"mnt root", &dto.MountPointData{Root: "/mnt"}, "ADDON"},
		{"host path", &dto.MountPointData{Path: "/data/share"}, "HOST"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := volume.InferMountPointType(tc.mp); got != tc.want {
				t.Errorf("InferMountPointType = %q, want %q", got, tc.want)
			}
		})
	}
}

type fakeNotifier struct {
	created   []string
	dismissed []string
}

func (f *fakeNotifier) CreatePersistentNotification(id, _ string, _ string) error {
	f.created = append(f.created, id)
	return nil
}

func (f *fakeNotifier) DismissPersistentNotification(id string) error {
	f.dismissed = append(f.dismissed, id)
	return nil
}

func (s *MountOrchestratorTestSuite) seedDiskWithPartition(diskID, partID, devicePath string) {
	diskIDCopy, partIDCopy, devCopy := diskID, partID, devicePath
	parts := map[string]dto.Partition{partID: {Id: &partIDCopy, DiskId: &diskIDCopy, DevicePath: &devCopy}}
	disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
	s.Require().NoError(s.disks.AddOrUpdate(&disk))
}

func (s *MountOrchestratorTestSuite) TestMountVolume_Success_DismissesNotifications() {
	ha := &fakeNotifier{}
	s.orchestrator = volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks, Filesystem: &fakeOrchestratorFS{},
		Mounter: s.mounter, HA: ha, EventBus: s.eventBus, Repo: s.mountRepoIf,
		ProtectedMode: func() bool { return false },
		Volumes: func() ([]*dto.Disk, errors.E) {
			return s.disks.All(), nil
		},
	})

	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))
	s.seedDiskWithPartition("disk-orch-ok", "part-orch-ok", deviceFile)

	md := &dto.MountPointData{
		Path: filepath.Join(tmpDir, "mnt", "share"), Root: "/", DeviceId: "part-orch-ok",
		Flags: &dto.MountFlags{},
	}
	s.Require().NoError(s.orchestrator.MountVolume(md))
	s.Equal(1, s.mounter.mountCalls)
	s.True(md.IsMounted)
	s.Len(ha.dismissed, 2, "successful mount dismisses failure notifications")
}

func (s *MountOrchestratorTestSuite) TestMountVolume_MounterError() {
	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))
	s.seedDiskWithPartition("disk-orch-err", "part-orch-err", deviceFile)
	s.mounter.mountErr = errors.WithDetails(dto.ErrorMountFail, "Detail", "boom")

	md := &dto.MountPointData{
		Path: filepath.Join(tmpDir, "mnt", "share"), Root: "/", DeviceId: "part-orch-err",
		Flags: &dto.MountFlags{},
		Partition: &dto.Partition{
			Id: new("part-orch-err"), DevicePath: &deviceFile,
		},
	}
	err := s.orchestrator.MountVolume(md)
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorMountFail)
}

func (s *MountOrchestratorTestSuite) TestPatchMountPointSettings_Success() {
	s.seedDiskWithPartition("disk-orch-patch", "part-orch-patch", "/dev/sdz9")
	md := dto.MountPointData{
		Path: "/mnt/orch-patch", Root: "/", DeviceId: "part-orch-patch",
		Type: "ADDON", Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}
	s.Require().NoError(s.repo.Persist(&md))

	var emitted int
	unsub := s.eventBus.OnMountPoint(func(_ context.Context, _ events.MountPointEvent) errors.E {
		emitted++
		return nil
	})
	defer unsub()

	startup := true
	got, err := s.orchestrator.PatchMountPointSettings("/", "/mnt/orch-patch", dto.MountPointData{IsToMountAtStartup: &startup})
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Require().NotNil(got.IsToMountAtStartup)
	s.True(*got.IsToMountAtStartup)
	s.Equal(1, emitted, "patch must broadcast after the cache update")
}

func (s *MountOrchestratorTestSuite) TestGetDevicePathByDeviceID_Success() {
	s.seedDiskWithPartition("disk-orch-dev", "part-orch-dev", "/dev/sdz9")
	got, err := s.orchestrator.GetDevicePathByDeviceID("part-orch-dev")
	s.Require().NoError(err)
	s.Equal("/dev/sdz9", got)
}

func (s *MountOrchestratorTestSuite) TestMountVolume_ValidationBranches() {
	// Empty root.
	err := s.orchestrator.MountVolume(&dto.MountPointData{Path: "/mnt/x", DeviceId: "p"})
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorInvalidParameter)

	// Empty device id.
	err = s.orchestrator.MountVolume(&dto.MountPointData{Path: "/mnt/x", Root: "/"})
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorInvalidParameter)

	// Unknown device: no partition in the map.
	err = s.orchestrator.MountVolume(&dto.MountPointData{
		Path: "/mnt/x", Root: "/", DeviceId: "part-unknown", Flags: &dto.MountFlags{},
	})
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorDeviceNotFound)
	s.Zero(s.mounter.mountCalls)
}

func (s *MountOrchestratorTestSuite) TestMountVolume_InvalidFlags() {
	s.orchestrator = volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks,
		Filesystem: &fakeOrchestratorFS{flagsErr: errors.New("bad flags")},
		Mounter:    s.mounter, EventBus: s.eventBus, Repo: s.mountRepoIf,
		ProtectedMode: func() bool { return false },
		Volumes: func() ([]*dto.Disk, errors.E) {
			return s.disks.All(), nil
		},
	})

	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))
	s.seedDiskWithPartition("disk-orch-flags", "part-orch-flags", deviceFile)
	md := &dto.MountPointData{
		Path: filepath.Join(tmpDir, "mnt"), Root: "/", DeviceId: "part-orch-flags",
		Flags: &dto.MountFlags{},
		Partition: &dto.Partition{
			Id: new("part-orch-flags"), DevicePath: &deviceFile,
		},
	}
	err := s.orchestrator.MountVolume(md)
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorInvalidParameter)
	s.Zero(s.mounter.mountCalls)
}

func (s *MountOrchestratorTestSuite) TestPatchMountPointSettings_FallbackCache() {
	// Row references an unknown device, so the converter cannot resolve a
	// partition; the cache fallback via GetMountPointByPath must apply.
	s.seedDiskWithPartition("disk-orch-fb", "part-orch-fb", "/dev/sdzfb")
	s.Require().NoError(s.disks.AddOrUpdateMountPoint("disk-orch-fb", "part-orch-fb", dto.MountPointData{
		Path: "/mnt/orch-fb", DeviceId: "part-orch-fb",
		Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
		Partition: &dto.Partition{Id: new("part-orch-fb"), DiskId: new("disk-orch-fb")},
	}))
	s.Require().NoError(s.repo.Persist(&dto.MountPointData{
		Path: "/mnt/orch-fb", Root: "/", DeviceId: "part-orch-unknown",
		Type: "ADDON", Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}))

	startup := true
	got, err := s.orchestrator.PatchMountPointSettings("/", "/mnt/orch-fb", dto.MountPointData{IsToMountAtStartup: &startup})
	s.Require().NoError(err)
	s.Require().NotNil(got)

	cached, ok := s.disks.GetMountPointByPath("/mnt/orch-fb")
	s.Require().True(ok, "fallback must update the cached entry")
	s.Require().NotNil(cached.IsToMountAtStartup)
	s.True(*cached.IsToMountAtStartup)
}

func (s *MountOrchestratorTestSuite) TestPatchMountPointSettings_VolumesError() {
	s.seedDiskWithPartition("disk-orch-ve", "part-orch-ve", "/dev/sdzve")
	s.Require().NoError(s.repo.Persist(&dto.MountPointData{
		Path: "/mnt/orch-ve", Root: "/", DeviceId: "part-orch-ve",
		Type: "ADDON", Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}))
	s.orchestrator = volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks, Filesystem: &fakeOrchestratorFS{},
		Mounter: s.mounter, EventBus: s.eventBus, Repo: s.mountRepoIf,
		ProtectedMode: func() bool { return false },
		Volumes: func() ([]*dto.Disk, errors.E) {
			return nil, errors.New("volumes unavailable")
		},
	})

	startup := true
	got, err := s.orchestrator.PatchMountPointSettings("/", "/mnt/orch-ve", dto.MountPointData{IsToMountAtStartup: &startup})
	s.Require().NoError(err, "volumes failure must degrade to the fallback path, not fail")
	s.Require().NotNil(got)
}
