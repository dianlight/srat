// Package volume_test: orchestrator suites (validation, unmount fallback,
// settings patch, automount guard) with hand-written fakes.
package volume_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/service/volume"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
)

type fakeOrchestratorFS struct {
	flagsErr     errors.E
	label        string
	labelErr     errors.E
	defaultFlags []dto.MountFlag
	defaultsErr  errors.E
	lastInput    []dto.MountFlag
}

func (f *fakeOrchestratorFS) MountFlagsToSyscallFlagAndData(input []dto.MountFlag) (uintptr, string, errors.E) {
	f.lastInput = append([]dto.MountFlag(nil), input...)
	if f.flagsErr != nil {
		return 0, "", f.flagsErr
	}
	return 0, "", nil
}

func (f *fakeOrchestratorFS) GetDefaultMountFlags(_ string) ([]dto.MountFlag, errors.E) {
	return f.defaultFlags, f.defaultsErr
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

func (s *MountOrchestratorTestSuite) TestMountVolume_DeviceAlreadyMountedElsewhere() {
	tmpDir := s.T().TempDir()
	devFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(devFile, []byte("test"), 0o600))
	restore := osutil.MockMountInfo("1217 819 0:52 / /mnt/other rw,relatime - ext4 " + devFile + " rw\n")
	s.T().Cleanup(restore)

	partID := "part-orch-devmounted"
	partition := dto.Partition{Id: &partID, DevicePath: &devFile}
	md := &dto.MountPointData{
		Path: filepath.Join(tmpDir, "mnt", "fresh"), Root: "/", DeviceId: partID,
		Flags: &dto.MountFlags{}, Partition: &partition,
	}

	err := s.orchestrator.MountVolume(md)
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorAlreadyMounted)
	s.Zero(s.mounter.mountCalls, "OS-level mount must not run when the device is mounted elsewhere")
}

func (s *MountOrchestratorTestSuite) TestMountVolume_MounterBusyRacedToAlreadyMounted() {
	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))
	s.seedDiskWithPartition("disk-orch-busy", "part-orch-busy", deviceFile)
	s.mounter.mountErr = errors.WithDetails(errors.New("mount: device or resource busy"), "Source", deviceFile)

	md := &dto.MountPointData{
		Path: filepath.Join(tmpDir, "mnt", "share"), Root: "/", DeviceId: "part-orch-busy",
		Flags: &dto.MountFlags{},
		Partition: &dto.Partition{
			Id: new("part-orch-busy"), DevicePath: &deviceFile,
		},
	}
	err := s.orchestrator.MountVolume(md)
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorAlreadyMounted)
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

func TestShouldSkipAutomount(t *testing.T) {
	native := "native"
	nativeUpper := "NATIVE"
	ext4 := "ext4"
	cases := []struct {
		name string
		mp   *dto.MountPointData
		want bool
	}{
		{"nil", nil, true},
		{"system path with ext4", &dto.MountPointData{Path: "/addon_configs", FSType: &ext4}, true},
		{"system path without fstype", &dto.MountPointData{Path: "/addon_configs"}, true},
		{"native fstype on data path", &dto.MountPointData{Path: "/mnt/data", FSType: &native}, true},
		{"native uppercase", &dto.MountPointData{Path: "/mnt/data", FSType: &nativeUpper}, true},
		{"native via partition fstype", &dto.MountPointData{
			Path:      "/mnt/data",
			Partition: &dto.Partition{FsType: &native},
		}, true},
		{"ext4 data path mounts", &dto.MountPointData{Path: "/mnt/data", FSType: &ext4}, false},
		{"empty fstype on data path mounts (auto-detect)", &dto.MountPointData{Path: "/mnt/data"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := volume.ShouldSkipAutomount(tc.mp); got != tc.want {
				t.Errorf("ShouldSkipAutomount = %v, want %v", got, tc.want)
			}
		})
	}
}

func (s *MountOrchestratorTestSuite) TestMountVolume_SkipsNativeAndSystemPaths() {
	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))

	native := "native"
	ext4 := "ext4"
	cases := []struct {
		name string
		path string
		fs   *string
	}{
		{"native on system path", "/addon_configs", &native},
		{"ext4 on system path", "/addon_configs", &ext4},
		{"native on data path", filepath.Join(tmpDir, "mnt", "data"), &native},
	}
	for _, tc := range cases {
		callsBefore := s.mounter.mountCalls
		safe := "skip-case"
		switch tc.name {
		case "native on system path":
			safe = "skip-native-system"
		case "ext4 on system path":
			safe = "skip-ext4-system"
		case "native on data path":
			safe = "skip-native-data"
		}
		diskID, partID := "disk-"+safe, "part-"+safe
		diskIDCopy, partIDCopy, devCopy := diskID, partID, deviceFile
		fsCopy := tc.fs
		parts := map[string]dto.Partition{partID: {
			Id: &partIDCopy, DiskId: &diskIDCopy, DevicePath: &devCopy, FsType: fsCopy,
		}}
		disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
		s.Require().NoError(s.disks.AddOrUpdate(&disk))
		md := &dto.MountPointData{
			Path: tc.path, Root: "/", DeviceId: partID,
			Flags: &dto.MountFlags{}, FSType: tc.fs,
			Partition: &dto.Partition{Id: &partIDCopy, DiskId: &diskIDCopy, DevicePath: &devCopy, FsType: fsCopy},
		}
		err := s.orchestrator.MountVolume(md)
		s.Require().Error(err, tc.name)
		s.ErrorIs(err, dto.ErrorInvalidParameter, tc.name)
		s.Equal(callsBefore, s.mounter.mountCalls, "skipped mount must not reach the mounter (%s)", tc.name)
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

func (s *MountOrchestratorTestSuite) TestMountVolume_NilFlagsInitializes() {
	restore := osutil.MockMountInfo("")
	s.T().Cleanup(restore)

	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))
	s.seedDiskWithPartition("disk-orch-nilflags", "part-orch-nilflags", deviceFile)

	md := &dto.MountPointData{
		Path: filepath.Join(tmpDir, "mnt", "nilflags"), Root: "/", DeviceId: "part-orch-nilflags",
		Flags: nil,
	}
	s.Require().NoError(s.orchestrator.MountVolume(md))
	s.Equal(1, s.mounter.mountCalls)
	s.NotNil(md.Flags, "nil Flags must be initialized before the OS mount")
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

func (s *MountOrchestratorTestSuite) TestMountVolume_AppliesDefaultMountFlags() {
	fs := &fakeOrchestratorFS{defaultFlags: []dto.MountFlag{
		{Name: "dmask", NeedsValue: true, FlagValue: "000"},
		{Name: "fmask", NeedsValue: true, FlagValue: "000"},
	}}
	s.orchestrator = volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks, Filesystem: fs,
		Mounter: s.mounter, EventBus: s.eventBus, Repo: s.mountRepoIf,
		ProtectedMode: func() bool { return false },
		Volumes: func() ([]*dto.Disk, errors.E) {
			return s.disks.All(), nil
		},
	})

	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))
	s.seedDiskWithPartition("disk-orch-def", "part-orch-def", deviceFile)
	fstype := "ntfs"
	md := &dto.MountPointData{
		Path: filepath.Join(tmpDir, "mnt"), Root: "/", DeviceId: "part-orch-def",
		FSType: &fstype,
		Flags:  &dto.MountFlags{{Name: "ro"}},
		Partition: &dto.Partition{
			Id: new("part-orch-def"), DevicePath: &deviceFile,
		},
	}
	s.Require().NoError(s.orchestrator.MountVolume(md))
	s.Equal(1, s.mounter.mountCalls)

	byName := make(map[string]string, len(fs.lastInput))
	for _, flag := range fs.lastInput {
		byName[flag.Name] = flag.FlagValue
	}
	s.Contains(byName, "ro", "user flag must be preserved")
	s.Equal("000", byName["dmask"], "adapter default must be merged")
	s.Equal("000", byName["fmask"], "adapter default must be merged")
}

func (s *MountOrchestratorTestSuite) TestMountVolume_UserFlagWinsOnDefaultCollision() {
	fs := &fakeOrchestratorFS{defaultFlags: []dto.MountFlag{
		{Name: "dmask", NeedsValue: true, FlagValue: "000"},
		{Name: "fmask", NeedsValue: true, FlagValue: "000"},
	}}
	s.orchestrator = volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks, Filesystem: fs,
		Mounter: s.mounter, EventBus: s.eventBus, Repo: s.mountRepoIf,
		ProtectedMode: func() bool { return false },
		Volumes: func() ([]*dto.Disk, errors.E) {
			return s.disks.All(), nil
		},
	})

	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))
	s.seedDiskWithPartition("disk-orch-coll", "part-orch-coll", deviceFile)
	fstype := "ntfs"
	md := &dto.MountPointData{
		Path: filepath.Join(tmpDir, "mnt"), Root: "/", DeviceId: "part-orch-coll",
		FSType: &fstype,
		Flags:  &dto.MountFlags{{Name: "dmask", NeedsValue: true, FlagValue: "0022"}},
		Partition: &dto.Partition{
			Id: new("part-orch-coll"), DevicePath: &deviceFile,
		},
	}
	s.Require().NoError(s.orchestrator.MountVolume(md))

	var dmaskValues []string
	var fmaskSeen bool
	for _, flag := range fs.lastInput {
		if flag.Name == "dmask" {
			dmaskValues = append(dmaskValues, flag.FlagValue)
		}
		if flag.Name == "fmask" {
			fmaskSeen = true
		}
	}
	s.Equal([]string{"0022"}, dmaskValues, "explicit user flag must win, exactly once")
	s.True(fmaskSeen, "non-colliding default must still be merged")
}

func (s *MountOrchestratorTestSuite) TestMountVolume_DefaultFlagsErrorStillMounts() {
	fs := &fakeOrchestratorFS{defaultsErr: errors.New("registry boom")}
	s.orchestrator = volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks, Filesystem: fs,
		Mounter: s.mounter, EventBus: s.eventBus, Repo: s.mountRepoIf,
		ProtectedMode: func() bool { return false },
		Volumes: func() ([]*dto.Disk, errors.E) {
			return s.disks.All(), nil
		},
	})

	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))
	s.seedDiskWithPartition("disk-orch-deferr", "part-orch-deferr", deviceFile)
	fstype := "ntfs"
	md := &dto.MountPointData{
		Path: filepath.Join(tmpDir, "mnt"), Root: "/", DeviceId: "part-orch-deferr",
		FSType: &fstype,
		Flags:  &dto.MountFlags{{Name: "ro"}},
		Partition: &dto.Partition{
			Id: new("part-orch-deferr"), DevicePath: &deviceFile,
		},
	}
	s.Require().NoError(s.orchestrator.MountVolume(md), "defaults resolution failure must not block the mount")
	s.Equal(1, s.mounter.mountCalls)
	s.Len(fs.lastInput, 1, "only user flags reach conversion when defaults fail")
	s.Equal("ro", fs.lastInput[0].Name)
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

type blockingCoalesceMounter struct {
	calls atomic.Int32
	delay time.Duration
}

func (f *blockingCoalesceMounter) Mount(md *dto.MountPointData, _ uintptr, _, _ string) errors.E {
	f.calls.Add(1)
	time.Sleep(f.delay)
	md.IsMounted = true
	return nil
}

func (f *blockingCoalesceMounter) Unmount(md *dto.MountPointData, _ bool) errors.E {
	md.IsMounted = false
	return nil
}

func (s *MountOrchestratorTestSuite) TestMountVolume_ConcurrentCoalesce() {
	restore := osutil.MockMountInfo("")
	s.T().Cleanup(restore)

	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))
	s.seedDiskWithPartition("disk-orch-coal", "part-orch-coal", deviceFile)

	blocking := &blockingCoalesceMounter{delay: 200 * time.Millisecond}
	orchestrator := volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks, Filesystem: &fakeOrchestratorFS{},
		Mounter: blocking, EventBus: s.eventBus, Repo: s.mountRepoIf,
		Volumes: func() ([]*dto.Disk, errors.E) { return s.disks.All(), nil },
	})

	const callers = 8
	mountPath := filepath.Join(tmpDir, "mnt", "coalesce")
	var wg sync.WaitGroup
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			partID := "part-orch-coal"
			md := &dto.MountPointData{
				Path: mountPath, Root: "/", DeviceId: partID,
				Flags:     &dto.MountFlags{},
				Partition: &dto.Partition{Id: &partID, DevicePath: &deviceFile},
			}
			if err := orchestrator.MountVolume(md); err != nil {
				errs[idx] = err
			}
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		s.NoError(err)
	}
	s.Equal(int32(1), blocking.calls.Load(), "concurrent mounts for the same device+path must coalesce to one OS attempt")
}
