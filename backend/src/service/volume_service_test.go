// Package service_test contains tests for the service layer.
package service_test

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"

	// Needed for MockBroadcaster

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/suite"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/loop"
	"github.com/xorcare/pointer"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

type VolumeServiceTestSuite struct {
	suite.Suite
	//mockMountRepo      repository.MountPointPathRepositoryInterface
	mockHardwareClient service.HardwareServiceInterface
	volumeService      service.VolumeServiceInterface
	ctrl               *matchers.MockController
	ctx                context.Context
	cancel             context.CancelFunc
	app                *fxtest.App
	db                 *gorm.DB
}

// helper to access concrete methods not exposed on the interface
func (suite *VolumeServiceTestSuite) mockMountOps(
	tryMount func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
	mountFn func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
	unmountFn func(target string, force, lazy bool) error,
) {
	if v, ok := suite.volumeService.(*service.VolumeService); ok {
		v.MockSetMountOps(tryMount, mountFn, unmountFn)
	}
}

func TestVolumeServiceTestSuite(t *testing.T) {
	suite.Run(t, new(VolumeServiceTestSuite))
}

func (suite *VolumeServiceTestSuite) SetupTest() {
	data, err := os.ReadFile("../../test/data/mount_info.txt")
	if err != nil {
		log.Fatal(err)
	}
	osutil.MockMountInfo(string(data))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			func() *dto.ContextState {
				return &dto.ContextState{
					DatabasePath: "file::memory:?cache=shared&_pragma=foreign_keys(1)",
				}
			},
			func() repository.IssueRepositoryInterface {
				// Provide a nil repository since it's only used in error cases in tests
				return nil
			},
			dbom.NewDB,
			service.NewVolumeService,
			service.NewFilesystemService,
			events.NewEventBus,
			//mock.Mock[service.BroadcasterServiceInterface],
			//mock.Mock[repository.MountPointPathRepositoryInterface],
			mock.Mock[service.HardwareServiceInterface],
			mock.Mock[service.ShareServiceInterface],
			mock.Mock[service.IssueServiceInterface],
			//mock.Mock[events.EventBusInterface],
		),
		fx.Populate(&suite.volumeService),
		//fx.Populate(&suite.mockMountRepo),
		fx.Populate(&suite.mockHardwareClient),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
		fx.Populate(&suite.db),
	)
	suite.volumeService.MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error) {
		return []*procfs.MountInfo{
			{MountID: 1217, ParentID: 819, MajorMinorVer: "0:52", Root: "/", Source: "/dev/sda1", MountPoint: "/mnt/test1", FSType: "ext4", Options: map[string]string{"noatime": ""}, SuperOptions: map[string]string{}},
			{MountID: 1218, ParentID: 820, MajorMinorVer: "0:53", Root: "/", Source: "/dev/sdb1", MountPoint: "/mnt/test2", FSType: "xfs", Options: map[string]string{"nodiratime": ""}, SuperOptions: map[string]string{}},
		}, nil
	})
	suite.app.RequireStart()
}

func (suite *VolumeServiceTestSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
	}
	if suite.ctx != nil {
		if wg := suite.ctx.Value("wg"); wg != nil {
			wg.(*sync.WaitGroup).Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

// --- MountVolume Tests ---

func (suite *VolumeServiceTestSuite) TestMountUnmountVolume_Success() {
	device, err := loop.FindDevice()
	if err != nil {
		suite.T().Skip("No loop device available, skipping test")
		return
	}
	suite.Require().NoError(err, "Error finding loop device")
	err = suite.volumeService.CreateBlockDevice(device)
	suite.Require().NoError(err, "Error creating block device")
	err = loop.SetFile(device, "../../test/data/image.dmg")
	suite.Require().NoError(err, "Error setting loop device file")
	mountPath := "/mnt/test1"
	fsType := "ext4"
	mountData := dto.MountPointData{
		Path:     mountPath,
		DeviceId: device,
		FSType:   &fsType,
		Flags: &dto.MountFlags{
			dto.MountFlag{Name: "noatime", NeedsValue: false},
		},
	}
	dbomMountData := &dbom.MountPointPath{
		Path:     mountPath,
		DeviceId: device,
		FSType:   fsType,
		Flags:    &dbom.MounDataFlags{dbom.MounDataFlag{Name: "noatime", NeedsValue: false}},
	}

	// Mock FindByPath
	// FindByDevice is called multiple times:
	// - Initial GetVolumesData for new disks (2 partitions)
	// - Subsequent GetVolumesData calls during mount/unmount refresh existing disks (2 partitions per call)
	//mock.When(suite.mockMountRepo.FindByDevice(device)).ThenReturn([]*dbom.MountPointPath{dbomMountData}, nil).Verify(matchers.AtLeastOnce())
	suite.db.Create(dbomMountData)

	/*
		mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenAnswer(matchers.Answer(func(args []any) []any {
			mp, ok := args[0].(*dbom.MountPointPath)
			if !ok {
				suite.T().Errorf("Expected argument to be of type *dbom.MountPointPath, got %T", args[0])
			}
			suite.T().Logf("MountPointPath saved: %+v", mp)
			suite.Equal(mountPath, mp.Path)
			//suite.Equal(device, mp.Device)
			suite.Equal(fsType, mp.FSType)
			suite.Require().NotNil(mp.Flags)
			suite.Contains(*mp.Flags, dbom.MounDataFlag{Name: "noatime", NeedsValue: false})
			dbomMountData.DeviceId = mp.DeviceId
			return []any{nil}
		})).Verify(matchers.AtLeastOnce())
	*/
	mock.When(suite.mockHardwareClient.GetHardwareInfo()).ThenReturn(
		map[string]dto.Disk{
			"SSD": {LegacyDeviceName: pointer.String("sda1"), Size: pointer.Int(100), Id: pointer.String("SSD"),
				Partitions: &map[string]dto.Partition{
					"SSD": {
						DevicePath:       &device,
						LegacyDeviceName: pointer.String("sda1"), Size: pointer.Int(100), Id: pointer.String("SSD"),
						DiskId: pointer.String("SSD"),
					},
				},
			},
			"HDD": {LegacyDeviceName: pointer.String("sda2"), Size: pointer.Int(200), Id: pointer.String("HDD"),
				Partitions: &map[string]dto.Partition{
					device: {
						LegacyDeviceName: pointer.String("sda2"), Size: pointer.Int(200), Id: &device, DevicePath: &device,
						DiskId: pointer.String("HDD"),
					},
				},
			},
		},
		nil)

	disks := suite.volumeService.GetVolumesData()
	suite.Require().NotNil(disks, "Expected GetVolumesData to return disks")
	suite.Require().NotEmpty(disks, "Expected GetVolumesData to return non-empty disks")
	suite.Require().Len(disks, 2, "Expected GetVolumesData to return 2 disks")

	defer func() {
		err := suite.volumeService.UnmountVolume(mountPath, true, false) // Cleanup
		suite.Require().Nil(err, "Expected no error on unmount")
		loop.ClearFile(device)
	}()
	// --- Execute ---
	errE := suite.volumeService.MountVolume(&mountData)

	// --- Assert ---
	suite.Require().NoError(errE, "Expected no error on successful mount", errE)
	suite.NotEmpty(*mountData.Flags)
	suite.Contains(*mountData.Flags, dto.MountFlag{Name: "noatime", Description: "", NeedsValue: false, FlagValue: "", ValueDescription: "", ValueValidationRegex: ""})

}

/*
func (suite *VolumeServiceTestSuite) TestMountVolume_RepoFindByPathError() {
	mountPath := "/mnt/test1"
	mountData := dto.MountPointData{Path: mountPath, DeviceId: "sda1"}
	expectedErr := errors.New("Invalid parameter")

	mock.When(suite.mockMountRepo.FindByPath(mountPath)).ThenReturn(nil, expectedErr).Verify(matchers.Times(1))

	err := suite.volumeService.MountVolume(&mountData)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, expectedErr)
}
*/

func (suite *VolumeServiceTestSuite) TestMountVolume_DeviceEmpty() {
	mountPath := "/mnt/test1"
	mountData := dto.MountPointData{Path: mountPath, DeviceId: ""} // Empty device
	err := suite.volumeService.MountVolume(&mountData)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, dto.ErrorInvalidParameter)
	details := err.Details()
	suite.Contains(details, "Message")
	suite.Equal("Source device name is empty in request", details["Message"])
}

func (suite *VolumeServiceTestSuite) TestMountVolume_DeviceInvalid() {
	mountPath := "/mnt/test1"
	mountData := dto.MountPointData{Path: mountPath, DeviceId: "/dev/pippo"} // Invalid device
	//dbomMountData := &dbom.MountPointPath{Path: mountPath, DeviceId: ""}

	//mock.When(suite.mockMountRepo.FindByPath(mountPath)).ThenReturn(dbomMountData, nil).Verify(matchers.Times(1))
	//suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Once()

	err := suite.volumeService.MountVolume(&mountData)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, dto.ErrorDeviceNotFound)
	details := err.Details()
	suite.Contains(details, "Message")
	suite.Equal("Source device does not exist on the system", details["Message"])
}

func (suite *VolumeServiceTestSuite) TestMountVolume_PathEmpty() {
	// Note: The converter step might prevent this if Path is required,
	// but testing the service logic defensively.
	mountPath := ""
	device := "sda1"
	mountData := dto.MountPointData{Path: mountPath, DeviceId: device}
	// FindByPath won't be called if path is empty early on
	// dbomMountData := &dbom.MountPointPath{Path: mountPath, Device: device}
	// suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Once()

	err := suite.volumeService.MountVolume(&mountData)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, dto.ErrorInvalidParameter)
	details := err.Details()
	suite.Contains(details, "Message")
	suite.Equal("Mount point path is empty", details["Message"])
}

func (suite *VolumeServiceTestSuite) TestUnmountVolume_NotInCache() {
	mountPath := "/mnt/notfound"

	// Path not in cache (GetVolumesData was never called or path doesn't exist)
	// UnmountVolume should attempt unmount even if path is not in cache
	// For this test, we expect unmount to be called but the path doesn't actually exist
	// So os.Remove will fail, but that's not an error case we return
	suite.mockMountOps(nil, nil, func(target string, force, lazy bool) error { return nil })

	// FindByPath should NOT be called with new implementation
	// UnmountVolume uses cache first, then falls back to path-only unmount
	// No database calls should be made unless the partition info exists in cache

	err := suite.volumeService.UnmountVolume(mountPath, false, false)
	suite.Require().Nil(err, "Expected no error when unmounting path not in cache")
}

// --- GetVolumesData Tests ---

func (suite *VolumeServiceTestSuite) TestGetVolumesData_Success() {

	mountPath1 := "/mnt/test1"
	mountPath2 := "/mnt/test2"

	device1 := pointer.String("/dev/disk/by-id/virtio-disk1-part1")
	device2 := pointer.String("/dev/disk/by-id/virtio-disk2-part1")
	device2Legacy := pointer.String("/dev/sdb1")
	device2LegacyName := pointer.String("sdb1")
	mockHWResponse := map[string]dto.Disk{
		"disk-1": {
			Id:               pointer.String("disk-1"),
			LegacyDevicePath: pointer.String("/dev/sda"),
			Size:             pointer.Int(100),
			Vendor:           pointer.String("ATA"),
			Model:            pointer.String("Model-1"),
			Partitions: &map[string]dto.Partition{
				"part-1": {
					Id:               pointer.String("part-1"),
					Name:             pointer.String("RootFS"),
					LegacyDevicePath: pointer.String("/dev/sda1"),
					LegacyDeviceName: pointer.String("sda1"),
					DevicePath:       device1,
					Size:             pointer.Int(50),
					DiskId:           pointer.String("disk-1"),
					HostMountPointData: &map[string]dto.MountPointData{
						mountPath1: {
							DeviceId: *device1,
							Path:     mountPath1,
						},
					},
				},
			},
		},
		"disk-2": {
			Id:               pointer.String("disk-2"),
			LegacyDevicePath: pointer.String("/dev/sdb"),
			Vendor:           pointer.String("SATA"),
			Model:            pointer.String("Model-2"),
			Size:             pointer.Int(100),
			Partitions: &map[string]dto.Partition{
				"part-2": {
					Id:               pointer.String("part-2"),
					Name:             pointer.String("DataFs"),
					LegacyDevicePath: device2Legacy,
					LegacyDeviceName: device2LegacyName,
					DevicePath:       device2,
					Size:             pointer.Int(50),
					DiskId:           pointer.String("disk-2"),
					HostMountPointData: &map[string]dto.MountPointData{
						mountPath2: {
							DeviceId: *device2,
							Path:     mountPath2,
						},
					},
				},
			},
		},
	} // Prepare mock repo responses
	//dbomMountData1 := &dbom.MountPointPath{Path: mountPath1, DeviceId: "sda1", Type: "ADDON"} // Initial state in DB
	//dbomMountData2 := &dbom.MountPointPath{Path: mountPath2, DeviceId: "sdb1", Type: "ADDON"} // Initial state in DB

	//mock.When(suite.mockMountRepo.FindByDevice("part-1")).ThenReturn([]*dbom.MountPointPath{{Path: mountPath1, DeviceId: *device1, Type: "ADDON"}}, nil).Verify(matchers.Times(1))
	suite.Require().NoError(suite.db.Create(&dbom.MountPointPath{Path: mountPath1, DeviceId: *device1, Type: "ADDON"}).Error)
	//mock.When(suite.mockMountRepo.FindByDevice("part-2")).ThenReturn([]*dbom.MountPointPath{{Path: mountPath2, DeviceId: *device2, Type: "ADDON"}}, nil).Verify(matchers.Times(1))
	suite.Require().NoError(suite.db.Create(&dbom.MountPointPath{Path: mountPath2, DeviceId: *device2, Type: "ADDON"}).Error)
	mock.When(suite.mockHardwareClient.GetHardwareInfo()).ThenReturn(mockHWResponse, nil).Verify(matchers.AtLeastOnce())

	/*
		mock.When(suite.mockMountRepo.AllByDeviceId()).ThenReturn(map[string]dbom.MountPointPath{
			"sdb1": {Path: mountPath2, DeviceId: "sdb1", Type: "ADDON"},
		}, nil).Verify(matchers.Times(1))
	*/

	// Expect FindByPath and Save for each mount point found in hardware data
	//mock.When(suite.mockMountRepo.FindByPath(mountPath1)).ThenReturn(dbomMountData1, nil).Verify(matchers.Times(1))
	//mock.When(suite.mockMountRepo.FindByPath(mountPath2)).ThenReturn(dbomMountData2, nil).Verify(matchers.Times(1))

	// Call the function
	disks := suite.volumeService.GetVolumesData()

	// Assertions
	suite.Require().NotNil(disks)
	suite.Require().Len(disks, 2)

	// Build a lookup by disk ID to avoid order dependency
	byID := map[string]*dto.Disk{}
	for _, d := range disks {
		if d.Id != nil {
			byID[*d.Id] = d
		}
	}

	// Validate disk-1
	d1, ok1 := byID["disk-1"]
	suite.Require().True(ok1, "disk-1 should be present")
	suite.Equal(mockHWResponse["disk-1"].Vendor, d1.Vendor)
	suite.Equal(mockHWResponse["disk-1"].Model, d1.Model)
	suite.Require().NotNil(d1.Partitions)
	suite.Require().Len(*d1.Partitions, 1)
	p1 := (*d1.Partitions)["part-1"]
	suite.Require().NotNil(p1.LegacyDevicePath)
	suite.Require().NotNil(p1.Name)
	suite.Equal(*(*mockHWResponse["disk-1"].Partitions)["part-1"].Name, *p1.Name)
	suite.Require().NotNil(p1.MountPointData)
	suite.Require().Len(*p1.MountPointData, 1, "Expected 1 mount point for partition 1")
	mp1, ok := (*p1.MountPointData)[mountPath1]
	suite.Require().True(ok, "Expected mount path %s in partition 1", mountPath1)
	suite.Equal(mountPath1, mp1.Path)

	// Validate disk-2
	d2, ok2 := byID["disk-2"]
	suite.Require().True(ok2, "disk-2 should be present")
	suite.Equal(mockHWResponse["disk-2"].Vendor, d2.Vendor)
	suite.Equal(mockHWResponse["disk-2"].Model, d2.Model)
	suite.Require().NotNil(d2.Partitions)
	suite.Require().Len(*d2.Partitions, 1)
	p2 := (*d2.Partitions)["part-2"]
	suite.Require().NotNil(p2.LegacyDevicePath)
	suite.Require().NotNil(p2.Name)
	suite.Require().NotNil(p2.MountPointData)
	suite.Require().Len(*p2.MountPointData, 1, "Expected 1 mount point for partition 2")
	mp2, ok := (*p2.MountPointData)[mountPath2]
	suite.Require().True(ok, "Expected mount path %s in partition 2", mountPath2)
	suite.Equal(mountPath2, mp2.Path)
}

// --- Additional GetVolumesData focused tests ---

// Ensures GetVolumesData returns Partitions with MountPointData (addon-side) populated
// in addition to any HostMountPointData provided by the hardware client.
func (suite *VolumeServiceTestSuite) TestGetVolumesData_ReturnsMountPointData() {
	mountPathAddon := "/mnt/addon-mp"
	device := pointer.String("/dev/disk/by-id/testdisk1-part1")
	partID := pointer.String("test-part-1")

	// Mock hardware: one disk, one partition with only HostMountPointData set
	hostMount := dto.MountPointData{Path: "/host/mount", DeviceId: *partID, Type: "HOST"}
	hostMap := map[string]dto.MountPointData{hostMount.Path: hostMount}

	mockHW := map[string]dto.Disk{
		"disk-1": {
			Id:     pointer.String("disk-1"),
			Vendor: pointer.String("VEND"),
			Model:  pointer.String("MODEL"),
			Partitions: &map[string]dto.Partition{
				*partID: {
					Id:                 partID,
					DevicePath:         device,
					LegacyDevicePath:   pointer.String("/dev/sda1"),
					HostMountPointData: &hostMap,
					MountPointData:     &map[string]dto.MountPointData{},
					DiskId:             pointer.String("disk-1"),
				},
			},
		},
	}

	// Repo: no pre-existing mount configuration for this device
	//mock.When(suite.mockMountRepo.FindByDevice(*partID)).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound)).Verify(matchers.Times(1))
	mock.When(suite.mockHardwareClient.GetHardwareInfo()).ThenReturn(mockHW, nil).Verify(matchers.AtLeastOnce())

	// Procfs mounts contain an addon-side mount for our partition
	suite.volumeService.MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error) {
		return []*procfs.MountInfo{
			{MountID: 2001, ParentID: 1, MajorMinorVer: "0:99", Root: "/", Source: *device, MountPoint: mountPathAddon, FSType: "ext4", Options: map[string]string{"rw": ""}, SuperOptions: map[string]string{}},
		}, nil
	})

	disks := suite.volumeService.GetVolumesData()
	suite.Require().NotNil(disks)
	suite.Require().Len(disks, 1)

	d := (disks)[0]
	suite.Require().NotNil(d.Partitions)
	p, ok := (*d.Partitions)[*partID]
	suite.Require().True(ok)

	// HostMountPointData remained intact
	suite.Require().NotNil(p.HostMountPointData)
	suite.Require().Contains(*p.HostMountPointData, hostMount.Path)

	// MountPointData (addon-side) is present and contains the procfs mount
	suite.Require().NotNil(p.MountPointData)
	mp, ok := (*p.MountPointData)[mountPathAddon]
	suite.Require().True(ok, "expected addon mount path in MountPointData")
	suite.Equal(mountPathAddon, mp.Path)
	suite.True(mp.IsMounted)
	suite.Equal("ADDON", mp.Type)
}

// Ensures addon MountPointData and HostMountPointData are not mixed.
func (suite *VolumeServiceTestSuite) TestGetVolumesData_NoMixHostAndAddon() {
	hostPath := "/host/point"
	addonPath := "/addon/point"
	device := pointer.String("/dev/disk/by-id/testdisk2-part1")
	partID := pointer.String("test-part-2")

	hostMount := dto.MountPointData{Path: hostPath, DeviceId: *partID, Type: "HOST"}
	hostMap := map[string]dto.MountPointData{hostPath: hostMount}

	mockHW := map[string]dto.Disk{
		"disk-2": {
			Id:     pointer.String("disk-2"),
			Vendor: pointer.String("VEND"),
			Model:  pointer.String("MODEL"),
			Partitions: &map[string]dto.Partition{
				*partID: {
					Id:                 partID,
					DevicePath:         device,
					DiskId:             pointer.String("disk-2"),
					HostMountPointData: &hostMap,
					MountPointData:     &map[string]dto.MountPointData{},
				},
			},
		},
	}

	//mock.When(suite.mockMountRepo.FindByDevice(*partID)).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound)).Verify(matchers.Times(1))
	mock.When(suite.mockHardwareClient.GetHardwareInfo()).ThenReturn(mockHW, nil).Verify(matchers.AtLeastOnce())

	// Procfs: only addon mount present
	suite.volumeService.MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error) {
		return []*procfs.MountInfo{
			{MountID: 3001, ParentID: 1, MajorMinorVer: "0:98", Root: "/", Source: *device, MountPoint: addonPath, FSType: "xfs", Options: map[string]string{"rw": ""}, SuperOptions: map[string]string{}},
		}, nil
	})

	disks := suite.volumeService.GetVolumesData()
	suite.Require().NotNil(disks)
	suite.Require().Len(disks, 1)
	part := (*disks[0].Partitions)[*partID]

	// Host mount should not appear in addon MountPointData
	suite.Require().NotNil(part.MountPointData)
	suite.Require().NotContains(*part.MountPointData, hostPath)
	suite.Require().Contains(*part.MountPointData, addonPath)

	// Addon mount should not appear in HostMountPointData
	suite.Require().NotNil(part.HostMountPointData)
	suite.Require().NotContains(*part.HostMountPointData, addonPath)
	suite.Require().Contains(*part.HostMountPointData, hostPath)
}

// --- Mount/Unmount affecting GetVolumesData state ---
/*

func (suite *VolumeServiceTestSuite) TestMountVolume_UpdatesMountPointDataState() {
	devicePath := "/dev/disk/by-id/mockdisk3-part1"
	partID := "mock-part-3"
	mountPath := "/mnt/mock3"
	fstype := "ext4"

	// Hardware with 1 partition, no addon mounts initially
	mockHW := map[string]dto.Disk{
		"disk-3": {
			Id:     pointer.String("disk-3"),
			Vendor: pointer.String("VEND"),
			Model:  pointer.String("MODEL"),
			Partitions: &map[string]dto.Partition{
				partID: {
					Id:             pointer.String(partID),
					DevicePath:     pointer.String(devicePath),
					MountPointData: &map[string]dto.MountPointData{},
				},
			},
		},
	}

	mock.When(suite.mockHardwareClient.GetHardwareInfo()).ThenReturn(mockHW, nil).Verify(matchers.AtLeastOnce())
	// No existing repo record for device
	// FindByDevice is called multiple times:
	// - Initial GetVolumesData for new disks (1 partition)
	// - Subsequent GetVolumesData calls during mount operation refresh existing disks (1 partition per call)
	mock.When(suite.mockMountRepo.FindByDevice(devicePath)).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound)).Verify(matchers.AtLeastOnce())
	// Expect a save during mount event persistence
	mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenReturn(nil).Verify(matchers.AtLeastOnce())

	// Before mounting, procfs does not list our mount
	suite.volumeService.MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error) {
		return []*procfs.MountInfo{}, nil
	})

	// Stub mount operation to succeed and reflect our desired mountpoint
	suite.mockMountOps(
		func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error) {
			return &mount.MountPoint{Path: target, Device: devicePath, FSType: fstype, Flags: 0, Data: ""}, nil
		},
		func(source, target, fstypeP, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error) {
			return &mount.MountPoint{Path: target, Device: devicePath, FSType: fstypeP, Flags: 0, Data: data}, nil
		},
		nil,
	)

	// Now pretends the system shows the mount right after mounting
	suite.volumeService.MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error) {
		return []*procfs.MountInfo{
			{MountID: 4001, ParentID: 1, MajorMinorVer: "0:97", Root: "/", Source: devicePath, MountPoint: mountPath, FSType: fstype, Options: map[string]string{"rw": ""}, SuperOptions: map[string]string{}},
		}, nil
	})

	// Perform mount
	// Ensure disks cache is populated so MountVolume can resolve Partition from DeviceId
	_ = suite.volumeService.GetVolumesData()
	md := dto.MountPointData{
		Path:     mountPath,
		DeviceId: partID,
		FSType:   &fstype,
		Flags:    &dto.MountFlags{},
	}
	err := suite.volumeService.MountVolume(&md)
	suite.Require().NoError(err)

	// Validate state via GetVolumesData
	disks := suite.volumeService.GetVolumesData()
	suite.Require().NotNil(disks)
	part := (*(*disks)[0].Partitions)[partID]
	suite.Require().NotNil(part.MountPointData)
	mpd, ok := (*part.MountPointData)[mountPath]
	suite.Require().True(ok)
	suite.True(mpd.IsMounted)
	suite.Equal(mountPath, mpd.Path)
	suite.Require().NotNil(mpd.Partition)
	suite.Equal(partID, *mpd.Partition.Id)
}
*/
func (suite *VolumeServiceTestSuite) TestUnmountVolume_UpdatesMountPointDataState() {
	devicePath := "/dev/disk/by-id/mockdisk4-part1"
	partID := "mock-part-4"
	mountPath := "/mnt/mock4"
	fstype := "xfs"

	// Hardware
	mockHW := map[string]dto.Disk{
		"disk-4": {
			Id:     pointer.String("disk-4"),
			Vendor: pointer.String("VEND"),
			Model:  pointer.String("MODEL"),
			Partitions: &map[string]dto.Partition{
				partID: {
					Id:             pointer.String(partID),
					DevicePath:     pointer.String(devicePath),
					MountPointData: &map[string]dto.MountPointData{},
					DiskId:         pointer.String("disk-4"),
				},
			},
		},
	}
	mock.When(suite.mockHardwareClient.GetHardwareInfo()).ThenReturn(mockHW, nil).Verify(matchers.Times(1))

	// Repo returns an existing mount configuration by device
	// NOTE: Using AtLeastOnce() because GetVolumesData calls loadMountPointFromDB for existing disks
	//dbrec := &dbom.MountPointPath{Path: mountPath, DeviceId: partID, FSType: fstype, Type: "ADDON"}
	//mock.When(suite.mockMountRepo.FindByDevice(partID)).ThenReturn([]*dbom.MountPointPath{dbrec}, nil).Verify(matchers.AtLeastOnce())
	//suite.db.Create(dbrec)

	// FindByPath is only called if we have partition info to persist (i.e., if unmount updates DB)
	// With new implementation using cache-first approach, this may not be called
	//mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenReturn(nil).Verify(matchers.AtLeastOnce())

	// Initially, procfs indicates the mount is active
	suite.volumeService.MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error) {
		return []*procfs.MountInfo{
			{MountID: 5001, ParentID: 1, MajorMinorVer: "0:96", Root: "/", Source: devicePath, MountPoint: mountPath, FSType: fstype, Options: map[string]string{"rw": ""}, SuperOptions: map[string]string{}},
		}, nil
	})
	// Trigger initial load so MountPointData is populated as mounted
	disks := suite.volumeService.GetVolumesData()
	suite.Require().NotNil(disks)
	part := (*disks[0].Partitions)[partID]
	suite.Require().NotNil(part.MountPointData)
	mpd, ok := (*part.MountPointData)[mountPath]
	suite.Require().True(ok)
	suite.True(mpd.IsMounted)

	// After unmount, procfs should not contain the entry anymore
	suite.volumeService.MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error) { return []*procfs.MountInfo{}, nil })

	// Stub unmount to succeed
	suite.mockMountOps(nil, nil, func(target string, force, lazy bool) error { return nil })

	// Perform unmount
	err := suite.volumeService.UnmountVolume(mountPath, true, false)
	suite.Require().NoError(err)

	// Validate state reflects unmounted
	disks = suite.volumeService.GetVolumesData()
	suite.Require().NotNil(disks)
	part = (*disks[0].Partitions)[partID]
	suite.Require().NotNil(part.MountPointData)
	mpd, ok = (*part.MountPointData)[mountPath]
	suite.Require().True(ok)
	suite.False(mpd.IsMounted)
}

// --- PatchMountPointSettings Tests ---
func (suite *VolumeServiceTestSuite) TestPatchMountPointSettings_Success_OnlyStartup() {
	path := "/mnt/testpatch"
	originalStartup := pointer.Bool(true)
	patchedStartup := pointer.Bool(false)

	dbData := &dbom.MountPointPath{
		Path:               path,
		DeviceId:           "/dev/sdc1",
		FSType:             "ext4",
		Type:               "ADDON",
		IsToMountAtStartup: originalStartup,
	}

	patch := dto.MountPointData{
		IsToMountAtStartup: patchedStartup,
	}

	//mock.When(suite.mockMountRepo.FindByPath(path)).ThenReturn(dbData, nil).Verify(matchers.Times(1))
	suite.Require().NoError(suite.db.Create(dbData).Error)
	/*
		mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenAnswer(matchers.Answer(func(args []any) []any {
			savedDbData := args[0].(*dbom.MountPointPath)
			suite.Equal(path, savedDbData.Path)
			suite.Equal("ext4", savedDbData.FSType) // Should not change
			suite.Equal(patchedStartup, savedDbData.IsToMountAtStartup)
			return []any{nil}
		})).Verify(matchers.Times(2))
	*/
	resultDto, errE := suite.volumeService.PatchMountPointSettings(path, patch)
	suite.Require().Nil(errE)
	suite.Require().NotNil(resultDto)
	suite.Equal(path, resultDto.Path)
	suite.Equal("ext4", *resultDto.FSType)
	suite.Equal(patchedStartup, resultDto.IsToMountAtStartup)
}

func (suite *VolumeServiceTestSuite) TestPatchMountPointSettings_NoChanges() {
	path := "/mnt/testpatch_nochange"
	originalStartup := pointer.Bool(true)

	dbData := &dbom.MountPointPath{
		Path:               path,
		DeviceId:           "/dev/sdd1",
		FSType:             "btrfs",
		Type:               "ADDON",
		IsToMountAtStartup: originalStartup,
	}

	patch := dto.MountPointData{
		IsToMountAtStartup: originalStartup, // Same value
	}

	//mock.When(suite.mockMountRepo.FindByPath(path)).ThenReturn(dbData, nil).Verify(matchers.Times(1))
	suite.Require().NoError(suite.db.Create(dbData).Error)
	// Save should NOT be called if no changes
	//mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenReturn(nil).Verify(matchers.Times(1))

	resultDto, errE := suite.volumeService.PatchMountPointSettings(path, patch)
	suite.Require().Nil(errE)
	suite.Require().NotNil(resultDto)
	suite.Equal(path, resultDto.Path)
	suite.Equal("btrfs", *resultDto.FSType)
	suite.Equal(originalStartup, resultDto.IsToMountAtStartup)
}

// Ensures that after patching IsToMountAtStartup the subsequent GetVolumesData reflects the updated value.
func (suite *VolumeServiceTestSuite) TestPatchMountPointSettings_UpdatesStartupFlagInGetVolumesData() {
	mountPath := "/mnt/startup-test"
	devicePath := "/dev/disk/by-id/startdisk1-part1"
	partID := pointer.String("startup-part-1")
	diskID := pointer.String("startup-disk-1")

	// Initial DB state: IsToMountAtStartup = false
	originalStartup := pointer.Bool(false)
	dbData := &dbom.MountPointPath{
		Path:               mountPath,
		DeviceId:           *partID, // repository is keyed by device path
		FSType:             "ext4",
		IsToMountAtStartup: originalStartup,
		Type:               "ADDON",
	}

	// Hardware snapshot: one disk with one partition, no mounts (unmounted)
	mockHW := map[string]dto.Disk{
		*diskID: {
			Id:     diskID,
			Vendor: pointer.String("VEND"),
			Model:  pointer.String("MODEL"),
			Partitions: &map[string]dto.Partition{
				*partID: {
					Id:             partID,
					DiskId:         diskID,
					DevicePath:     pointer.String(devicePath),
					MountPointData: &map[string]dto.MountPointData{},
				},
			},
		},
	}

	// Mock repository and hardware client calls
	// Note: After refactoring, GetVolumesData reloads mount data from DB for existing disks,
	// causing multiple calls to FindByDevice (initial load + subsequent refreshes)
	mock.When(suite.mockHardwareClient.GetHardwareInfo()).ThenReturn(mockHW, nil).Verify(matchers.AtLeastOnce())
	//mock.When(suite.mockMountRepo.FindByDevice(*partID)).ThenReturn([]*dbom.MountPointPath{dbData}, nil).Verify(matchers.AtLeastOnce())
	suite.Require().NoError(suite.db.Create(dbData).Error)
	//mock.When(suite.mockMountRepo.FindByPath(mountPath)).ThenReturn(dbData, nil).Verify(matchers.AtLeastOnce())
	//mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenReturn(nil).Verify(matchers.AtLeastOnce())

	// Ensure no active mounts in procfs
	suite.volumeService.MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error) { return []*procfs.MountInfo{}, nil })

	// Initial load
	disks := suite.volumeService.GetVolumesData()
	suite.Require().NotNil(disks)
	suite.Require().Len(disks, 1)
	part := (*disks[0].Partitions)[*partID]
	suite.Require().NotNil(part.MountPointData)
	// Mount point should have been added from repository
	mpd, ok := (*part.MountPointData)[mountPath]
	suite.Require().True(ok, "expected mount point from repo to be present")
	suite.Require().NotNil(mpd.IsToMountAtStartup)
	suite.False(*mpd.IsToMountAtStartup, "expected initial IsToMountAtStartup to be false")
	suite.False(mpd.IsMounted, "expected mount point to be unmounted")

	// Patch: set IsToMountAtStartup = true
	patchedStartup := pointer.Bool(true)
	patch := dto.MountPointData{IsToMountAtStartup: patchedStartup}
	resultDto, errE := suite.volumeService.PatchMountPointSettings(mountPath, patch)
	suite.Require().Nil(errE)
	suite.Require().NotNil(resultDto)
	suite.Require().NotNil(resultDto.IsToMountAtStartup)
	suite.True(*resultDto.IsToMountAtStartup, "expected patched IsToMountAtStartup to be true")

	// Reload (should use cached data)
	disksAfter := suite.volumeService.GetVolumesData()
	suite.Require().NotNil(disksAfter)
	partAfter := (*disksAfter[0].Partitions)[*partID]
	mpdAfter, ok := (*partAfter.MountPointData)[mountPath]
	suite.Require().True(ok, "expected mount point to still be present after patch")
	suite.Require().NotNil(mpdAfter.IsToMountAtStartup)
	suite.True(*mpdAfter.IsToMountAtStartup, "expected IsToMountAtStartup to be true after patch and refresh")
}
