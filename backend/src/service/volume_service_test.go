// Package service_test contains tests for the service layer.
package service_test

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	// Needed for MockBroadcaster

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/internal/darwinstubs/mount/loop"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

type VolumeServiceTestSuite struct {
	suite.Suite
	//mockMountRepo      repository.MountPointPathRepositoryInterface
	mockHardwareClient service.HardwareServiceInterface
	volumeService      service.VolumeServiceInterface
	filesystemService  service.FilesystemServiceInterface
	hardwareService    service.HardwareServiceInterface
	eventBus           events.EventBusInterface
	disks              *dto.DiskMap
	state              *dto.ContextState
	ctx                context.Context
	cancel             context.CancelFunc
	app                *fxtest.App
	db                 *gorm.DB
}

/*
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
*/
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
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			func() *dto.ContextState {
				return &dto.ContextState{
					DatabasePath: "file::memory:?cache=shared&_pragma=foreign_keys(1)",
				}
			},
			func() *dto.DiskMap { return dto.NewDiskMap() },
			dbom.NewDB,
			service.NewVolumeMountManager,
			service.NewVolumeService,
			service.NewFilesystemService,
			events.NewEventBus,
			//mock.Mock[service.BroadcasterServiceInterface],
			//mock.Mock[repository.MountPointPathRepositoryInterface],
			mock.Mock[service.HardwareServiceInterface],
			mock.Mock[service.ShareServiceInterface],
			//mock.Mock[events.EventBusInterface],
		),
		fx.Populate(&suite.volumeService),
		//fx.Populate(&suite.mockMountRepo),
		fx.Populate(&suite.mockHardwareClient),
		fx.Populate(&suite.filesystemService),
		fx.Populate(&suite.hardwareService),
		fx.Populate(&suite.eventBus),
		fx.Populate(&suite.disks),
		fx.Populate(&suite.state),
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
		if wg := suite.ctx.Value(ctxkeys.WaitGroup); wg != nil {
			wg.(*sync.WaitGroup).Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

func (suite *VolumeServiceTestSuite) TestEmitMountPointWithoutTypeDefaultsToAddon() {
	mountPath := "/mnt/test-type"
	root := "/mnt/test-type"
	deviceID := "dev-test-1"
	mountPoint := dto.MountPointData{
		Path:        mountPath,
		Root:        root,
		DeviceId:    deviceID,
		Flags:       &dto.MountFlags{},
		CustomFlags: &dto.MountFlags{},
		IsMounted:   true,
	}

	err := suite.eventBus.EmitMountPoint(events.MountPointEvent{
		Event:      events.Event{Type: events.EventTypes.UPDATE},
		MountPoint: &mountPoint,
	})
	suite.Require().NoError(err)

	var dbMount dbom.MountPointPath
	suite.Require().NoError(suite.db.Where("path = ? AND root = ?", mountPath, root).First(&dbMount).Error)
	suite.Equal("ADDON", dbMount.Type)
}

func (suite *VolumeServiceTestSuite) TestEmitMountPointEventWithSharePersistsAssociation() {
	mountPath := "/mnt/test-share-assoc"
	deviceID := "dev-share-assoc-1"
	shareName := "b2-share-assoc"
	mountPoint := dto.MountPointData{
		Path:        mountPath,
		Root:        "/",
		DeviceId:    deviceID,
		Flags:       &dto.MountFlags{},
		CustomFlags: &dto.MountFlags{},
		IsMounted:   true,
		Share: &dto.SharedResource{
			Name: shareName,
		},
	}

	err := suite.eventBus.EmitMountPoint(events.MountPointEvent{
		Event:      events.Event{Type: events.EventTypes.UPDATE},
		MountPoint: &mountPoint,
	})
	suite.Require().NoError(err)

	var dbMount dbom.MountPointPath
	suite.Require().NoError(suite.db.Preload("ExportedShare").Where("path = ?", mountPath).First(&dbMount).Error)
	suite.Require().NotNil(dbMount.ExportedShare)
	suite.Equal(shareName, dbMount.ExportedShare.Name)
	suite.Equal(mountPath, *dbMount.ExportedShare.MountPointDataPath)
}

func (suite *VolumeServiceTestSuite) TestFormatSuccessEventRefreshesPartitionCache() {
	diskID := "disk-format-refresh"
	partitionID := "disk-format-refresh-part1"
	devicePath := "/dev/sdz1"
	initialFsType := "ext4"
	updatedFsType := "xfs"
	initialName := "before-format"
	updatedName := "after-format"

	mock.When(suite.mockHardwareClient.GetHardwareInfo()).ThenReturn(
		map[string]dto.Disk{
			diskID: {
				Id:    &diskID,
				Model: new("Format Refresh Disk"),
				Partitions: &map[string]dto.Partition{
					partitionID: {
						Id:         &partitionID,
						DiskId:     &diskID,
						DevicePath: new(devicePath),
						FsType:     &initialFsType,
						Name:       &initialName,
					},
				},
			},
		},
		nil,
	).ThenReturn(
		map[string]dto.Disk{
			diskID: {
				Id:    &diskID,
				Model: new("Format Refresh Disk"),
				Partitions: &map[string]dto.Partition{
					partitionID: {
						Id:         &partitionID,
						DiskId:     &diskID,
						DevicePath: new(devicePath),
						FsType:     &updatedFsType,
						Name:       &updatedName,
					},
				},
			},
		},
		nil,
	).Verify(matchers.AtLeastOnce())

	suite.hardwareService.InvalidateHardwareInfo()
	disksBefore := suite.volumeService.GetVolumesData()
	suite.Require().Len(disksBefore, 1)
	beforePart, ok := (*disksBefore[0].Partitions)[partitionID]
	suite.Require().True(ok, "expected partition to be present before refresh")
	suite.Require().NotNil(beforePart.FsType)
	suite.Equal(initialFsType, *beforePart.FsType)
	suite.Require().NotNil(beforePart.Name)
	suite.Equal(initialName, *beforePart.Name)

	diskUpdate := make(chan struct{}, 1)
	unsubscribe := suite.eventBus.OnDisk(func(ctx context.Context, event events.DiskEvent) errors.E {
		if event.Type == events.EventTypes.UPDATE && event.Disk != nil && event.Disk.Id != nil && *event.Disk.Id == diskID {
			select {
			case diskUpdate <- struct{}{}:
			default:
			}
		}
		return nil
	})
	defer unsubscribe()

	suite.eventBus.EmitFilesystemTask(events.FilesystemTaskEvent{
		Event: events.Event{Type: events.EventTypes.STOP},
		Task: &dto.FilesystemTask{
			Device:         devicePath,
			Operation:      "format",
			FilesystemType: updatedFsType,
			Status:         "success",
			Message:        "Format operation completed successfully for " + devicePath,
			Progress:       100,
		},
	})

	select {
	case <-diskUpdate:
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for disk refresh event after format success")
	}

	disksAfter := suite.volumeService.GetVolumesData()
	suite.Require().Len(disksAfter, 1)
	afterPart, ok := (*disksAfter[0].Partitions)[partitionID]
	suite.Require().True(ok, "expected partition to be present after refresh")
	suite.Require().NotNil(afterPart.FsType)
	suite.Equal(updatedFsType, *afterPart.FsType)
	suite.Require().NotNil(afterPart.Name)
	suite.Equal(updatedName, *afterPart.Name)
}

// --- MountVolume Tests ---

func (suite *VolumeServiceTestSuite) TestMountUnmountVolume_Success() {
	device, err := loop.FindDevice()
	if err != nil {
		suite.T().Skip("No loop device available, skipping test")
		return
	}
	suite.Require().NoError(err, "Error finding loop device")
	err = suite.filesystemService.CreateBlockDevice(suite.ctx, device)
	suite.Require().NoError(err, "Error creating block device")
	err = loop.SetFile(device, "../../test/data/image.dmg")
	suite.Require().NoError(err, "Error setting loop device file")
	mountPath := "/mnt/test1"
	root := "/"
	fsType := "ext4"
	mountData := dto.MountPointData{
		Path:     mountPath,
		Root:     root,
		DeviceId: device,
		FSType:   &fsType,
		Flags: &dto.MountFlags{
			dto.MountFlag{Name: "noatime", NeedsValue: false},
		},
	}
	dbomMountData := &dbom.MountPointPath{
		Path:     mountPath,
		Root:     &root,
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
			"SSD": {LegacyDeviceName: new("sda1"), Size: new(100), Id: new("SSD"),
				Partitions: &map[string]dto.Partition{
					"SSD": {
						DevicePath:       &device,
						LegacyDeviceName: new("sda1"), Size: new(100), Id: new("SSD"),
						DiskId: new("SSD"),
					},
				},
			},
			"HDD": {LegacyDeviceName: new("sda2"), Size: new(200), Id: new("HDD"),
				Partitions: &map[string]dto.Partition{
					device: {
						LegacyDeviceName: new("sda2"), Size: new(200), Id: &device, DevicePath: &device,
						DiskId: new("HDD"),
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
		err := suite.volumeService.UnmountVolume(mountPath, true) // Cleanup
		suite.Require().Nil(err, "Expected no error on unmount")
		_ = loop.ClearFile(device)
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
	suite.Equal("Mount point root is empty", details["Message"])
}

func (suite *VolumeServiceTestSuite) TestMountVolume_DeviceInvalid() {
	mountPath := "/mnt/test1"
	root := "/"
	mountData := dto.MountPointData{Root: root, Path: mountPath, DeviceId: "/dev/pippo"} // Invalid device
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

// --- ProtectedMode Tests ---

func (suite *VolumeServiceTestSuite) TestVolumeMutations_ProtectedMode() {
	suite.state.ProtectedMode = true

	testCases := []struct {
		name      string
		operation string
		run       func() errors.E
	}{
		{
			name:      "MountVolume rejected",
			operation: "MountVolume",
			run: func() errors.E {
				return suite.volumeService.MountVolume(&dto.MountPointData{
					Path: "/mnt/protected", Root: "/mnt/protected", DeviceId: "sda1",
				})
			},
		},
		{
			name:      "UnmountVolume rejected",
			operation: "UnmountVolume",
			run: func() errors.E {
				return suite.volumeService.UnmountVolume("/mnt/protected", false)
			},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			err := tc.run()
			suite.Require().Error(err)
			suite.ErrorIs(err, dto.ErrorOperationNotPermittedInProtectedMode)
			details := err.Details()
			suite.Contains(details, "Operation")
			suite.Equal(tc.operation, details["Operation"])
		})
	}
}

// TestUnmountVolume_Paths covers the UnmountVolume branches: a cached mount
// point with an HA-mounted share (share removal event + unmount attempt) and a
// path that is not present in the cache (fallback unmount attempt).
func (suite *VolumeServiceTestSuite) TestUnmountVolume_Paths() {
	diskID := "disk-u"
	partID := "part-u"
	devicePath := "/dev/u1"

	disk := &dto.Disk{
		Id: &diskID,
		Partitions: &map[string]dto.Partition{
			partID: {Id: &partID, DiskId: &diskID, DevicePath: &devicePath},
		},
	}
	suite.Require().NoError(suite.disks.AddOrUpdate(disk))
	suite.Require().NoError(suite.disks.AddOrUpdateMountPoint(diskID, partID, dto.MountPointData{
		Path:      "/mnt/u-ha",
		DeviceId:  partID,
		IsMounted: true,
		Share: &dto.SharedResource{
			Status: &dto.SharedResourceStatus{IsHAMounted: true},
		},
	}))

	// Subscribe to share events to observe the HA-mounted share removal.
	shareRemoved := make(chan events.ShareEvent, 1)
	unsub := suite.eventBus.OnShare(func(ctx context.Context, e events.ShareEvent) errors.E {
		shareRemoved <- e
		return nil
	})
	defer unsub()

	// Cached mount point with HA-mounted share: emits a REMOVE share event and
	// proceeds to unmount (which fails for the non-existent path).
	err := suite.volumeService.UnmountVolume("/mnt/u-ha", false)
	suite.Require().Error(err)
	suite.ErrorIs(err, dto.ErrorUnmountFail)

	select {
	case ev := <-shareRemoved:
		suite.Equal(events.EventTypes.REMOVE, ev.Type)
		suite.NotNil(ev.Share)
	default:
		suite.Fail("expected a REMOVE share event for the HA-mounted share")
	}

	// Path not in cache: falls back to a synthesized mount point and attempts
	// the unmount (fails for the non-existent path).
	err = suite.volumeService.UnmountVolume("/mnt/u-missing", false)
	suite.Require().Error(err)
	suite.ErrorIs(err, dto.ErrorUnmountFail)
}

// --- GetVolumesData Tests ---

func (suite *VolumeServiceTestSuite) TestGetVolumesData_Success() {

	mountPath1 := "/mnt/test1"
	mountPath2 := "/mnt/test2"

	device1 := new("/dev/disk/by-id/virtio-disk1-part1")
	device2 := new("/dev/disk/by-id/virtio-disk2-part1")
	device2Legacy := new("/dev/sdb1")
	device2LegacyName := new("sdb1")
	mockHWResponse := map[string]dto.Disk{
		"disk-1": {
			Id:               new("disk-1"),
			LegacyDevicePath: new("/dev/sda"),
			Size:             new(100),
			Vendor:           new("ATA"),
			Model:            new("Model-1"),
			Partitions: &map[string]dto.Partition{
				"part-1": {
					Id:               new("part-1"),
					Name:             new("RootFS"),
					LegacyDevicePath: new("/dev/sda1"),
					LegacyDeviceName: new("sda1"),
					DevicePath:       device1,
					Size:             new(50),
					DiskId:           new("disk-1"),
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
			Id:               new("disk-2"),
			LegacyDevicePath: new("/dev/sdb"),
			Vendor:           new("SATA"),
			Model:            new("Model-2"),
			Size:             new(100),
			Partitions: &map[string]dto.Partition{
				"part-2": {
					Id:               new("part-2"),
					Name:             new("DataFs"),
					LegacyDevicePath: device2Legacy,
					LegacyDeviceName: device2LegacyName,
					DevicePath:       device2,
					Size:             new(50),
					DiskId:           new("disk-2"),
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
	device := new("/dev/disk/by-id/testdisk1-part1")
	partID := new("test-part-1")

	// Mock hardware: one disk, one partition with only HostMountPointData set
	hostMount := dto.MountPointData{Path: "/host/mount", DeviceId: *partID, Type: "HOST"}
	hostMap := map[string]dto.MountPointData{hostMount.Path: hostMount}

	mockHW := map[string]dto.Disk{
		"disk-1": {
			Id:     new("disk-1"),
			Vendor: new("VEND"),
			Model:  new("MODEL"),
			Partitions: &map[string]dto.Partition{
				*partID: {
					Id:                 partID,
					DevicePath:         device,
					LegacyDevicePath:   new("/dev/sda1"),
					HostMountPointData: &hostMap,
					MountPointData:     &map[string]dto.MountPointData{},
					DiskId:             new("disk-1"),
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
	device := new("/dev/disk/by-id/testdisk2-part1")
	partID := new("test-part-2")

	hostMount := dto.MountPointData{Path: hostPath, DeviceId: *partID, Type: "HOST"}
	hostMap := map[string]dto.MountPointData{hostPath: hostMount}

	mockHW := map[string]dto.Disk{
		"disk-2": {
			Id:     new("disk-2"),
			Vendor: new("VEND"),
			Model:  new("MODEL"),
			Partitions: &map[string]dto.Partition{
				*partID: {
					Id:                 partID,
					DevicePath:         device,
					DiskId:             new("disk-2"),
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

// --- PatchMountPointSettings Tests ---
func (suite *VolumeServiceTestSuite) TestPatchMountPointSettings_Success_OnlyStartup() {
	path := "/mnt/testpatch"
	root := "/"
	originalStartup := new(true)
	patchedStartup := new(false)

	dbData := &dbom.MountPointPath{
		Path:               path,
		Root:               &root,
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
	resultDto, errE := suite.volumeService.PatchMountPointSettings(root, path, patch)
	suite.Require().Nil(errE)
	suite.Require().NotNil(resultDto)
	suite.Equal(path, resultDto.Path)
	suite.Equal(root, resultDto.Root)
	suite.Equal("ext4", *resultDto.FSType)
	suite.Equal(patchedStartup, resultDto.IsToMountAtStartup)
}

func (suite *VolumeServiceTestSuite) TestPatchMountPointSettings_NoChanges() {
	root := "/"
	path := "/mnt/testpatch_nochange"
	originalStartup := new(true)

	dbData := &dbom.MountPointPath{
		Path:               path,
		Root:               &root,
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

	resultDto, errE := suite.volumeService.PatchMountPointSettings(root, path, patch)
	suite.Require().Nil(errE)
	suite.Require().NotNil(resultDto)
	suite.Equal(path, resultDto.Path)
	suite.Equal(root, resultDto.Root)
	suite.Equal("btrfs", *resultDto.FSType)
	suite.Equal(originalStartup, resultDto.IsToMountAtStartup)
}

// TestPatchMountPointSettings_EmptyPatch_NoOp is a regression test for the H4
// finding: a PATCH with an empty / all-nil body on an existing record must be
// a successful no-op (200), not a misleading 404 "not found", and must not
// change any persisted field.
func (suite *VolumeServiceTestSuite) TestPatchMountPointSettings_EmptyPatch_NoOp() {
	root := "/"
	path := "/mnt/testpatch_noop"
	originalStartup := new(true)
	flags := dbom.MounDataFlags{{Name: "noatime"}}

	dbData := &dbom.MountPointPath{
		Path:               path,
		Root:               &root,
		DeviceId:           "/dev/sde1",
		FSType:             "ext4",
		Type:               "ADDON",
		IsToMountAtStartup: originalStartup,
		Flags:              &flags,
	}

	suite.Require().NoError(suite.db.Create(dbData).Error)

	// Empty patch: every field nil/zero.
	patch := dto.MountPointData{}

	resultDto, errE := suite.volumeService.PatchMountPointSettings(root, path, patch)
	suite.Require().Nil(errE, "empty patch on existing record must be a no-op, not an error")
	suite.Require().NotNil(resultDto)
	suite.Equal(path, resultDto.Path)

	// Response DTO must keep the persisted flags: the converter nil-wipes the
	// in-memory Flags on an all-nil patch, but the service must reflect the
	// DB state (see H4 finding).
	suite.Require().NotNil(resultDto.Flags, "response DTO must not lose Flags on an empty patch")
	suite.Equal("noatime", (*resultDto.Flags)[0].Name)

	// DB row must be untouched.
	var persisted dbom.MountPointPath
	err := suite.db.Where("path = ? AND root = ?", path, root).First(&persisted).Error
	suite.Require().NoError(err)
	suite.Equal(originalStartup, persisted.IsToMountAtStartup)
	suite.Equal("ext4", persisted.FSType)
	suite.Equal("ADDON", persisted.Type)
	suite.Equal(flags, *persisted.Flags)
}

// TestPatchMountPointSettings_OnlyStartup_KeepsFlagsAndShareAtDBLevel extends
// the H4 verification: a PATCH that only flips IsToMountAtStartup must leave
// flags, fstype, and the share foreign key unchanged in the DB.
func (suite *VolumeServiceTestSuite) TestPatchMountPointSettings_OnlyStartup_KeepsFlagsAndShareAtDBLevel() {
	root := "/"
	path := "/mnt/testpatch_flags"
	originalStartup := new(true)
	patchedStartup := new(false)
	flags := dbom.MounDataFlags{{Name: "noatime"}, {Name: "nofail"}}
	data := dbom.MounDataFlags{{Name: "x-systemd.automount"}}

	dbData := &dbom.MountPointPath{
		Path:               path,
		Root:               &root,
		DeviceId:           "/dev/sdf1",
		FSType:             "xfs",
		Type:               "ADDON",
		IsToMountAtStartup: originalStartup,
		Flags:              &flags,
		Data:               &data,
	}
	suite.Require().NoError(suite.db.Create(dbData).Error)

	patch := dto.MountPointData{IsToMountAtStartup: patchedStartup}

	resultDto, errE := suite.volumeService.PatchMountPointSettings(root, path, patch)
	suite.Require().Nil(errE)
	suite.Require().NotNil(resultDto)
	suite.Equal(patchedStartup, resultDto.IsToMountAtStartup)

	// Response DTO must carry the untouched flags/data: the partial PATCH must
	// not nil them out in the response (H4 finding — the converter nil-wipes
	// the in-memory record, the service must re-read the DB state).
	suite.Require().NotNil(resultDto.Flags, "response DTO must keep Flags on a partial PATCH")
	suite.Equal("noatime", (*resultDto.Flags)[0].Name)
	suite.Require().NotNil(resultDto.CustomFlags, "response DTO must keep CustomFlags on a partial PATCH")
	suite.Equal("x-systemd.automount", (*resultDto.CustomFlags)[0].Name)

	// DB level: flags/data/fstype must be untouched by a partial PATCH.
	var persisted dbom.MountPointPath
	err := suite.db.Where("path = ? AND root = ?", path, root).First(&persisted).Error
	suite.Require().NoError(err)
	suite.Equal(patchedStartup, persisted.IsToMountAtStartup)
	suite.Equal("xfs", persisted.FSType)
	suite.Equal(flags, *persisted.Flags)
	suite.Equal(data, *persisted.Data)
}

// TestPatchMountPointSettings_RecordNotFound covers the error branch: PATCHing
// a path that has no DB record must return ErrorNotFound.
func (suite *VolumeServiceTestSuite) TestPatchMountPointSettings_RecordNotFound() {
	root := "/"
	path := "/mnt/does_not_exist"

	patch := dto.MountPointData{IsToMountAtStartup: new(true)}
	resultDto, errE := suite.volumeService.PatchMountPointSettings(root, path, patch)
	suite.Require().Error(errE)
	suite.Nil(resultDto)
	suite.True(errors.Is(errE, dto.ErrorNotFound), "expected ErrorNotFound, got %v", errE)
}

// TestPatchMountPointSettings_FallbackCacheUpdate_CachedMountPoint covers the
// fallback cache path (GetMountPointByPath branch): when the DB record's
// DeviceId cannot be resolved to a partition of the disk map, the service
// must still refresh the cached mount point looked up by path.
func (suite *VolumeServiceTestSuite) TestPatchMountPointSettings_FallbackCacheUpdate_CachedMountPoint() {
	root := "/"
	path := "/mnt/fallback_test"
	diskID := "fallback-disk-1"
	partID := "fallback-part-1"

	// Seed the disk map with a partition whose MountPointData contains the path.
	suite.Require().NoError(suite.disks.AddOrUpdate(&dto.Disk{Id: &diskID}))
	suite.Require().NoError(suite.disks.AddPartition(diskID, dto.Partition{
		Id:         &partID,
		DiskId:     &diskID,
		DevicePath: new("/dev/fallback-part-1"),
	}))
	suite.Require().NoError(suite.disks.AddOrUpdateMountPoint(diskID, partID, dto.MountPointData{
		Path:               path,
		IsToMountAtStartup: new(true),
	}))

	// DB record with a DeviceId that does NOT match the seeded partition, so
	// partition resolution fails and the fallback path is exercised.
	originalStartup := new(false)
	dbData := &dbom.MountPointPath{
		Path:               path,
		Root:               &root,
		DeviceId:           "/dev/unresolvable-device",
		FSType:             "ext4",
		Type:               "ADDON",
		IsToMountAtStartup: originalStartup,
	}
	suite.Require().NoError(suite.db.Create(dbData).Error)

	patchedStartup := new(true)
	patch := dto.MountPointData{IsToMountAtStartup: patchedStartup}
	resultDto, errE := suite.volumeService.PatchMountPointSettings(root, path, patch)
	suite.Require().Nil(errE)
	suite.Require().NotNil(resultDto)
	suite.Equal(patchedStartup, resultDto.IsToMountAtStartup)

	// The cached mount point must have been refreshed through the fallback.
	cached, ok := suite.disks.GetMountPointByPath(path)
	suite.Require().True(ok, "expected cached mount point to still exist")
	suite.Equal(patchedStartup, cached.IsToMountAtStartup)
}

// TestPatchMountPointSettings_FallbackSnapshotLoop covers the second fallback
// branch (Snapshot loop): when neither partition resolution nor
// GetMountPointByPath yields an updateable entry, the service must scan the
// whole disk snapshot for the path and refresh it there.
func (suite *VolumeServiceTestSuite) TestPatchMountPointSettings_FallbackSnapshotLoop() {
	root := "/"
	path := "/mnt/fallback_snapshot_test"
	diskID := "fallback-snapshot-disk"
	partID := "fallback-snapshot-part"

	// Seed a disk + partition whose MountPointData already holds the path, but
	// with a Partition reference that is nil so GetMountPointByPath branch is
	// skipped (updated == false) and the Snapshot loop must take over.
	suite.Require().NoError(suite.disks.AddOrUpdate(&dto.Disk{Id: &diskID}))
	suite.Require().NoError(suite.disks.AddPartition(diskID, dto.Partition{
		Id:         &partID,
		DiskId:     &diskID,
		DevicePath: new("/dev/fallback-snapshot-part"),
	}))
	// Directly inject a mount point WITHOUT a Partition reference.
	d, ok := suite.disks.Get(diskID)
	suite.Require().True(ok)
	part := (*d.Partitions)[partID]
	mp := make(map[string]dto.MountPointData)
	mp[path] = dto.MountPointData{Path: path, IsToMountAtStartup: new(false)}
	part.MountPointData = &mp
	(*d.Partitions)[partID] = part

	dbData := &dbom.MountPointPath{
		Path:               path,
		Root:               &root,
		DeviceId:           "/dev/unresolvable-snapshot",
		FSType:             "ext4",
		Type:               "ADDON",
		IsToMountAtStartup: new(false),
	}
	suite.Require().NoError(suite.db.Create(dbData).Error)

	patchedStartup := new(true)
	patch := dto.MountPointData{IsToMountAtStartup: patchedStartup}
	resultDto, errE := suite.volumeService.PatchMountPointSettings(root, path, patch)
	suite.Require().Nil(errE)
	suite.Require().NotNil(resultDto)
	suite.Equal(patchedStartup, resultDto.IsToMountAtStartup)

	// The mount point inside the snapshot partition must have been refreshed.
	cached, ok := suite.disks.GetMountPointByPath(path)
	suite.Require().True(ok, "expected cached mount point to still exist")
	suite.Equal(patchedStartup, cached.IsToMountAtStartup)
}

// Ensures that after patching IsToMountAtStartup the subsequent GetVolumesData reflects the updated value.
func (suite *VolumeServiceTestSuite) TestPatchMountPointSettings_UpdatesStartupFlagInGetVolumesData() {
	mountPath := "/mnt/startup-test"
	root := "/"
	devicePath := "/dev/disk/by-id/startdisk1-part1"
	partID := new("startup-part-1")
	diskID := new("startup-disk-1")

	// Initial DB state: IsToMountAtStartup = false
	originalStartup := new(false)
	dbData := &dbom.MountPointPath{
		Path:               mountPath,
		Root:               &root,
		DeviceId:           *partID, // repository is keyed by device path
		FSType:             "ext4",
		IsToMountAtStartup: originalStartup,
		Type:               "ADDON",
	}

	// Hardware snapshot: one disk with one partition, no mounts (unmounted)
	mockHW := map[string]dto.Disk{
		*diskID: {
			Id:     diskID,
			Vendor: new("VEND"),
			Model:  new("MODEL"),
			Partitions: &map[string]dto.Partition{
				*partID: {
					Id:             partID,
					DiskId:         diskID,
					DevicePath:     new(devicePath),
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
	suite.hardwareService.InvalidateHardwareInfo()
	disks := suite.volumeService.GetVolumesData()
	suite.Require().NotNil(disks)
	suite.Require().Len(disks, 1)
	part := (*disks[0].Partitions)[*partID]
	suite.Require().NotNil(part.MountPointData)
	// Mount point should have been added from repository
	mpd, ok := (*part.MountPointData)[mountPath]
	suite.Require().True(ok, "expected mount point from repo to be present", "mountPath", mountPath, "MountPointData", part.MountPointData)
	suite.Require().NotNil(mpd.IsToMountAtStartup)
	suite.False(*mpd.IsToMountAtStartup, "expected initial IsToMountAtStartup to be false")
	suite.False(mpd.IsMounted, "expected mount point to be unmounted")

	// Patch: set IsToMountAtStartup = true
	patchedStartup := new(true)
	patch := dto.MountPointData{IsToMountAtStartup: patchedStartup}
	resultDto, errE := suite.volumeService.PatchMountPointSettings(root, mountPath, patch)
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

// TestHandlePartitionEvent_DiscoveryPreservesPersistedMountPointConfig is a
// regression test for the B2 finding: when a partition event rediscovers a
// procfs mount whose DB record is not keyed by the current partition device
// id, the discovery ADD path built a fresh MountPointData (only procfs
// fields) and persistMountPoint's ON CONFLICT UPDATE ALL wiped the persisted
// configuration (automount flag, flags, custom flags). The persisted values
// must survive the discovery persist.
func (suite *VolumeServiceTestSuite) TestHandlePartitionEvent_DiscoveryPreservesPersistedMountPointConfig() {
	mountPath := "/mnt/b2-discovery"
	root := "/"
	devicePath := "/dev/b2disk1-part1"
	partID := new("b2-part-1")
	diskID := new("b2-disk-1")

	// DB record exists but under a different device id, so it is invisible to
	// loadMountPointFromDB (which queries by DeviceId) and absent from the
	// in-memory cache -> the discovery ADD path is exercised.
	persistedFlags := dbom.MounDataFlags{{Name: "user_custom_flag"}}
	persistedData := dbom.MounDataFlags{{Name: "custom_super_opt", NeedsValue: true, FlagValue: "1"}}
	dbData := &dbom.MountPointPath{
		Path:               mountPath,
		Root:               &root,
		DeviceId:           "b2-other-partition-id",
		FSType:             "ext4",
		Type:               "ADDON",
		IsToMountAtStartup: new(true),
		Flags:              &persistedFlags,
		Data:               &persistedData,
	}
	suite.Require().NoError(suite.db.Create(dbData).Error)

	mockHW := map[string]dto.Disk{
		*diskID: {
			Id:     diskID,
			Vendor: new("VEND"),
			Model:  new("MODEL"),
			Partitions: &map[string]dto.Partition{
				*partID: {
					Id:         partID,
					DiskId:     diskID,
					DevicePath: new(devicePath),
				},
			},
		},
	}
	mock.When(suite.mockHardwareClient.GetHardwareInfo()).ThenReturn(mockHW, nil).Verify(matchers.AtLeastOnce())

	// Procfs reports the mount as active for the partition's device path.
	suite.volumeService.MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error) {
		return []*procfs.MountInfo{
			{MountID: 1300, ParentID: 900, MajorMinorVer: "0:99", Root: "/", Source: devicePath, MountPoint: mountPath, FSType: "ext4", Options: map[string]string{"rw": ""}, SuperOptions: map[string]string{}},
		}, nil
	})

	suite.hardwareService.InvalidateHardwareInfo()
	disks := suite.volumeService.GetVolumesData()
	suite.Require().NotNil(disks)
	suite.Require().Len(disks, 1)

	// The DB record must keep its persisted configuration after the discovery
	// ADD event is persisted.
	var dbMount dbom.MountPointPath
	suite.Require().NoError(suite.db.Where("path = ? AND root = ?", mountPath, root).First(&dbMount).Error)
	suite.Require().NotNil(dbMount.IsToMountAtStartup, "automount flag must survive discovery persist")
	suite.True(*dbMount.IsToMountAtStartup, "expected IsToMountAtStartup to stay true after discovery persist")
	suite.Require().NotNil(dbMount.Flags, "persisted flags must survive discovery persist")
	suite.Require().Len(*dbMount.Flags, 1)
	suite.Equal("user_custom_flag", (*dbMount.Flags)[0].Name)
	suite.Require().NotNil(dbMount.Data, "persisted custom flags must survive discovery persist")
	suite.Require().Len(*dbMount.Data, 1)
	suite.Equal("custom_super_opt", (*dbMount.Data)[0].Name)
}

// TestHandlePartitionEvent_StaleMarkingScopedToPartition is a regression test
// for the B3 finding: the stale-marking loop in handlePartitionEvent used to
// iterate GetAllMountPoints() — every disk and partition — and mark unmounted
// anything whose RefreshVersion trailed the current one. Because the loop runs
// for a single partition event, mount points of sibling partitions that were
// not part of the latest snapshot's emit path were falsely marked unmounted.
// The loop must only consider mount points of the partition being processed.
func (suite *VolumeServiceTestSuite) TestHandlePartitionEvent_StaleMarkingScopedToPartition() {
	diskID := "b3-disk-1"
	partA := "b3-part-a"
	partB := "b3-part-b"
	deviceA := "/dev/b3disk1-part1"
	deviceB := "/dev/b3disk1-part2"

	// Seed the disk cache with two partitions, each carrying a mounted mount
	// point stamped with the initial refresh version (0).
	disk := &dto.Disk{
		Id: &diskID,
		Partitions: &map[string]dto.Partition{
			partA: {Id: &partA, DiskId: &diskID, DevicePath: &deviceA},
			partB: {Id: &partB, DiskId: &diskID, DevicePath: &deviceB},
		},
	}
	suite.Require().NoError(suite.disks.AddOrUpdate(disk))
	suite.Require().NoError(suite.disks.AddOrUpdateMountPoint(diskID, partA, dto.MountPointData{
		Path:           "/mnt/b3-a",
		DeviceId:       partA,
		IsMounted:      true,
		RefreshVersion: 0,
	}))
	suite.Require().NoError(suite.disks.AddOrUpdateMountPoint(diskID, partB, dto.MountPointData{
		Path:           "/mnt/b3-b",
		DeviceId:       partB,
		IsMounted:      true,
		RefreshVersion: 0,
	}))

	// The service warms the cache at startup (getVolumesData), which already
	// advanced the refresh version once. Bump it again so both seeded mount
	// points (stamped 0) trail the current version and become stale
	// candidates for the stale-marking loop.
	baseVersion := suite.disks.CurrentRefreshVersion()
	suite.disks.NextRefreshVersion()
	currentVersion := suite.disks.CurrentRefreshVersion()
	suite.Equal(baseVersion+1, currentVersion)

	// Procfs reports no mounts: the sync loop stamps nothing, so any mounted
	// mount point still carrying version 0 is a stale-marking candidate.
	suite.volumeService.MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error) {
		return []*procfs.MountInfo{}, nil
	})

	// Emit a partition event for partition A only.
	partAEvent := (*disk.Partitions)[partA]
	suite.eventBus.EmitPartition(events.PartitionEvent{
		Event:     events.Event{Type: events.EventTypes.ADD},
		Partition: &partAEvent,
		Disk:      disk,
	})

	// Partition A's own mount point must have been marked unmounted ...
	mpA, ok := suite.disks.GetMountPoint(diskID, partA, "/mnt/b3-a")
	suite.Require().True(ok)
	suite.False(mpA.IsMounted, "stale-marking must still apply to the partition being processed")
	suite.Equal(currentVersion, mpA.RefreshVersion)

	// ... and partition A must NOT have gained B's mount point as a phantom
	// entry. The unscoped loop wrote every stale mount point (including B's)
	// into the partition being processed, polluting its map and persisting
	// cross-partition state.
	_, phantom := suite.disks.GetMountPoint(diskID, partA, "/mnt/b3-b")
	suite.False(phantom, "partition A must not contain partition B's mount point as a phantom entry")

	// ... but partition B's mount point must be untouched: still mounted and
	// still carrying the older refresh version.
	mpB, ok := suite.disks.GetMountPoint(diskID, partB, "/mnt/b3-b")
	suite.Require().True(ok)
	suite.True(mpB.IsMounted, "sibling partition mount point must not be marked unmounted")
	suite.Equal(uint32(0), mpB.RefreshVersion, "sibling partition refresh version must be unchanged")
}

// TestOnSmartEvent_EmptyDiskId_DoesNotUpdateDiskCache verifies that when a SmartEvent
// carrying an empty DiskId (e.g. a self-test progress event) is emitted, the volume
// service's OnSmart handler does NOT call AddSmartInfo on the disk map. This prevents
// spurious WARN logs every 5 s while a self-test is in progress.
func (suite *VolumeServiceTestSuite) TestOnSmartEvent_EmptyDiskId_DoesNotUpdateDiskCache() {
	diskID := "ata-DISK-SMART-GUARD-TEST"
	devicePath := "/dev/sda"
	suite.disks.AddOrUpdate(&dto.Disk{
		Id:         &diskID,
		DevicePath: &devicePath,
	})

	// Capture SmartInfo state before the event
	diskBefore, _ := suite.disks.Get(diskID)
	suite.Nil(diskBefore.SmartInfo, "SmartInfo should be nil before any event")

	// Emit a SmartEvent with empty DiskId (self-test progress event)
	suite.eventBus.EmitSmart(events.SmartEvent{
		SmartInfo:       dto.SmartInfo{DiskId: ""},
		SmartTestStatus: dto.SmartTestStatus{DiskId: diskID, Running: true},
	})

	// Disk cache should be unchanged
	diskAfter, _ := suite.disks.Get(diskID)
	suite.Nil(diskAfter.SmartInfo,
		"OnSmart with empty DiskId must not call AddSmartInfo on the disk cache")
}

// TestOnSmartEvent_ValidDiskId_UpdatesDiskCache verifies that when a SmartEvent with a
// non-empty DiskId is emitted, the volume service's OnSmart handler calls AddSmartInfo
// and the disk cache is updated with the new SMART data.
func (suite *VolumeServiceTestSuite) TestOnSmartEvent_ValidDiskId_UpdatesDiskCache() {
	diskID := "ata-DISK-SMART-UPDATE-TEST"
	devicePath := "/dev/sda"
	suite.disks.AddOrUpdate(&dto.Disk{
		Id:         &diskID,
		DevicePath: &devicePath,
	})

	smartInfo := dto.SmartInfo{
		DiskId:    diskID,
		Supported: true,
		Enabled:   true,
	}

	// Emit a SmartEvent with a valid DiskId
	suite.eventBus.EmitSmart(events.SmartEvent{
		SmartInfo: smartInfo,
	})

	// Disk cache should be updated
	diskAfter, _ := suite.disks.Get(diskID)
	suite.Require().NotNil(diskAfter.SmartInfo,
		"OnSmart with valid DiskId should call AddSmartInfo and update the disk cache")
	suite.Equal(diskID, diskAfter.SmartInfo.DiskId)
}

// TestGetDevicePathByDeviceID covers the B5 contract: DeviceId identifies a
// partition (not a disk), disk IDs must not match, and the device path lookup
// must fall back to legacy paths and never panic on nil DevicePath.
func (suite *VolumeServiceTestSuite) TestGetDevicePathByDeviceID() {
	diskID := "ata-B5-DISK"
	partFull := "part-B5-full"
	partLegacy := "part-B5-legacy"
	partEmpty := "part-B5-empty"
	fullPath := "/dev/disk/by-id/ata-B5-DISK-part1"
	legacyPath := "/dev/sdb1"

	suite.Require().NoError(suite.disks.AddOrUpdate(&dto.Disk{
		Id:         &diskID,
		DevicePath: &fullPath, // disk-level path must not be returned for a partition lookup
		Partitions: &map[string]dto.Partition{
			partFull: {
				Id:               &partFull,
				DiskId:           &diskID,
				DevicePath:       &fullPath,
				LegacyDevicePath: &legacyPath,
			},
			partLegacy: {
				Id:               &partLegacy,
				DiskId:           &diskID,
				LegacyDevicePath: &legacyPath,
			},
			partEmpty: {
				Id:     &partEmpty,
				DiskId: &diskID,
			},
		},
	}))

	testCases := []struct {
		name          string
		deviceID      string
		expectedPath  string
		expectedError error
	}{
		{
			name:         "partition id hit returns device path",
			deviceID:     partFull,
			expectedPath: fullPath,
		},
		{
			name:          "disk id passed must not match",
			deviceID:      diskID,
			expectedError: dto.ErrorNotFound,
		},
		{
			name:         "missing DevicePath falls back to legacy",
			deviceID:     partLegacy,
			expectedPath: legacyPath,
		},
		{
			name:          "all empty returns device not found without panic",
			deviceID:      partEmpty,
			expectedError: dto.ErrorDeviceNotFound,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			path, err := suite.volumeService.GetDevicePathByDeviceID(tc.deviceID)
			if tc.expectedError != nil {
				suite.Require().Error(err)
				suite.True(errors.Is(err, tc.expectedError),
					"error %v should wrap %v", err, tc.expectedError)
				return
			}
			suite.Require().NoError(err)
			suite.Equal(tc.expectedPath, path)
		})
	}
}
