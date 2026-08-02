package service_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type HardwareServiceSuite struct {
	suite.Suite
	hardwareService service.HardwareServiceInterface
	haClient        hardware.ClientWithResponsesInterface
	smartService    service.SmartServiceInterface
	hdidleService   service.HDIdleServiceInterface
	app             *fxtest.App
}

func TestHardwareServiceSuite(t *testing.T) {
	suite.Run(t, new(HardwareServiceSuite))
}

func (suite *HardwareServiceSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			// Provide state with HA Core ready to allow hardware info retrieval
			func() *dto.ContextState { return &dto.ContextState{HACoreReady: true} },
			// Provide an EventBus bound to the same context
			func(ctx context.Context) events.EventBusInterface { return events.NewEventBus(ctx) },
			service.NewHardwareService,
			mock.Mock[hardware.ClientWithResponsesInterface],
			mock.Mock[service.HDIdleServiceInterface],
			mock.Mock[service.SmartServiceInterface],
		),
		fx.Populate(&suite.hardwareService),
		fx.Populate(&suite.haClient),
		fx.Populate(&suite.smartService),
		fx.Populate(&suite.hdidleService),
	)
	suite.app.RequireStart()
}

func (suite *HardwareServiceSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_Success() {
	// Setup mock response
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives: &[]hardware.Drive{
					{
						Id: new("drive1"),
						Filesystems: &[]hardware.Filesystem{
							{
								Device: new("/dev/sda1"),
								Name:   new("filesystem1"),
							},
						},
					},
				},
				Devices: &[]hardware.Device{},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.NoError(err)
	suite.NotNil(disks)
	_, _ = mock.Verify(suite.haClient, matchers.Times(1)).GetHardwareInfoWithResponse(mock.Any[context.Context]())
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_EmptyDrives() {
	// Setup mock response with no drives
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives:  &[]hardware.Drive{},
				Devices: &[]hardware.Device{},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.NoError(err)
	suite.NotNil(disks)
	suite.Empty(disks)
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_ErrorResponse() {
	// Setup mock response with error
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 500},
		Body:         []byte(`{"error":"internal server error"}`),
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.Error(err)
	suite.Nil(disks)
}

func (suite *HardwareServiceSuite) TestInvalidateHardwareInfo() {
	// Setup: First call to populate cache
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives:  &[]hardware.Drive{},
				Devices: &[]hardware.Device{},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// First call - should hit the API
	_, err := suite.hardwareService.GetHardwareInfo()
	suite.NoError(err)

	// Invalidate cache
	suite.hardwareService.InvalidateHardwareInfo()

	// Second call - should hit the API again after cache invalidation
	_, err = suite.hardwareService.GetHardwareInfo()
	suite.NoError(err)

	// Verify API was called twice (not cached after invalidation)
	_, _ = mock.Verify(suite.haClient, matchers.Times(2)).GetHardwareInfoWithResponse(mock.Any[context.Context]())
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_ClientError() {
	// Setup mock with an error from the client
	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(nil, context.DeadlineExceeded)

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.Error(err)
	suite.Nil(disks)
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_KeepsDrivesWithoutFilesystems() {
	// Setup mock response with drives that have no filesystems.
	// These drives should be kept (not dropped) so the frontend can show them
	// and the user can take action (e.g. mount a whole-disk filesystem).
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives: &[]hardware.Drive{
					{
						Id:          new("drive1"),
						Filesystems: nil, // No filesystems
					},
					{
						Id:          new("drive2"),
						Filesystems: &[]hardware.Filesystem{}, // Empty filesystems
					},
				},
				Devices: &[]hardware.Device{},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.NoError(err)
	suite.NotNil(disks)
	suite.Len(disks, 2, "Drives without filesystems should be kept, not dropped")
	suite.Contains(disks, "drive1")
	suite.Contains(disks, "drive2")
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_FallbackProbeDetectsFilesystem() {
	// Setup mock response with a drive that has no filesystems but has a serial
	// matching a whole-disk device. The fallback probe should detect the filesystem
	// via mount.FSFromBlock and synthesize a filesystem entry.
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives: &[]hardware.Drive{
					{
						Id:          new("drive1"),
						Serial:      new("SERIAL123"),
						Filesystems: nil, // No filesystems from Supervisor
					},
				},
				Devices: &[]hardware.Device{
					{
						Name:    new("sda"),
						DevPath: new("/dev/sda"),
						ById:    new("/dev/disk/by-id/ata-DRIVE_SERIAL123"),
						Attributes: &hardware.Attributes{
							IDSERIALSHORT: new("SERIAL123"),
						},
					},
				},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)
	mock.When(suite.smartService.GetSmartInfo(mock.Any[context.Context](), mock.Any[string]())).
		ThenReturn(nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", "sda"))
	mock.When(suite.hdidleService.GetDeviceConfig(mock.Any[string]())).
		ThenReturn(nil, errors.WithDetails(dto.ErrorHDIdleNotSupported, "device", "sda"))

	// Inject a fake filesystem probe so the test doesn't touch real block devices
	suite.hardwareService.MockSetFSProbeFunc(func(devPath string) (string, uintptr, error) {
		suite.Equal("/dev/sda", devPath)
		return "vfat", 0, nil
	})

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.NoError(err)
	suite.NotNil(disks)
	suite.Len(disks, 1, "Drive should be kept after fallback probe")

	disk, ok := disks["ata-DRIVE_SERIAL123"]
	suite.True(ok, "Disk should be present by by-id name")
	suite.NotNil(disk.Partitions, "Partitions should be synthesized after probe")
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_FallbackProbeSynthesizesWithoutFilesystemMagic() {
	// Setup mock response with a drive that has no filesystems but has a serial
	// matching a whole-disk device. The fallback probe returns no filesystem
	// magic; the drive must still get a synthesized whole-disk partition so the
	// frontend can offer mount/unmount/check/format actions.
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives: &[]hardware.Drive{
					{
						Id:          new("drive1"),
						Serial:      new("SERIAL123"),
						Filesystems: nil, // No filesystems from Supervisor
					},
				},
				Devices: &[]hardware.Device{
					{
						Name:    new("sda"),
						DevPath: new("/dev/sda"),
						ById:    new("/dev/disk/by-id/ata-DRIVE_SERIAL123"),
						Attributes: &hardware.Attributes{
							IDSERIALSHORT: new("SERIAL123"),
						},
					},
				},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)
	mock.When(suite.smartService.GetSmartInfo(mock.Any[context.Context](), mock.Any[string]())).
		ThenReturn(nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", "sda"))
	mock.When(suite.hdidleService.GetDeviceConfig(mock.Any[string]())).
		ThenReturn(nil, errors.WithDetails(dto.ErrorHDIdleNotSupported, "device", "sda"))

	// Inject a fake filesystem probe that reports no readable filesystem magic
	suite.hardwareService.MockSetFSProbeFunc(func(devPath string) (string, uintptr, error) {
		suite.Equal("/dev/sda", devPath)
		return "", 0, nil
	})

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.NoError(err)
	suite.NotNil(disks)
	suite.Len(disks, 1, "Drive should be kept after fallback probe")

	disk, ok := disks["ata-DRIVE_SERIAL123"]
	suite.True(ok, "Disk should be present by by-id name")
	suite.NotNil(disk.Partitions, "Partitions should be synthesized after probe")
	suite.Len(*disk.Partitions, 1, "Exactly one whole-disk partition should be synthesized")
	for _, part := range *disk.Partitions {
		suite.Equal("sda", *part.LegacyDeviceName)
		suite.Equal("/dev/disk/by-id/ata-DRIVE_SERIAL123", *part.DevicePath)
	}
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_WholeDiskPartitionGetsFsType() {
	// Setup mock response with a drive that has a whole-disk filesystem whose
	// partition LegacyDeviceName equals the disk name. The device loop must not
	// stop at the disk match: the partition match must also run so the partition
	// gets DevicePath and FsType populated.
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives: &[]hardware.Drive{
					{
						Id:     new("drive1"),
						Serial: new("SERIAL123"),
						Filesystems: &[]hardware.Filesystem{
							{
								Id:          new("by-id-ata-DRIVE_SERIAL123"),
								Name:        new("UDISK"),
								Device:      new("/dev/sda"),
								MountPoints: &[]string{},
							},
						},
					},
				},
				Devices: &[]hardware.Device{
					{
						Name:    new("sda"),
						DevPath: new("/dev/sda"),
						ById:    new("/dev/disk/by-id/ata-DRIVE_SERIAL123"),
						Attributes: &hardware.Attributes{
							IDSERIALSHORT: new("SERIAL123"),
							IDFSTYPE:      new("vfat"),
						},
					},
				},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)
	mock.When(suite.smartService.GetSmartInfo(mock.Any[context.Context](), mock.Any[string]())).
		ThenReturn(nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", "sda"))
	mock.When(suite.hdidleService.GetDeviceConfig(mock.Any[string]())).
		ThenReturn(nil, errors.WithDetails(dto.ErrorHDIdleNotSupported, "device", "sda"))

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.NoError(err)
	suite.NotNil(disks)
	suite.Len(disks, 1, "Drive should be kept")

	disk, ok := disks["ata-DRIVE_SERIAL123"]
	suite.True(ok, "Disk should be present by by-id name")
	suite.NotNil(disk.Partitions, "Whole-disk filesystem should produce a partition")
	for _, part := range *disk.Partitions {
		suite.Equal("sda", *part.LegacyDeviceName)
		suite.Equal("/dev/disk/by-id/ata-DRIVE_SERIAL123", *part.DevicePath)
		suite.Equal("vfat", *part.FsType, "Partition FsType should be populated")
	}
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_DeviceWithChildrenSynthesizesPartitions() {
	// A device with child devices (e.g. a USB stick with a real partition table)
	// must synthesize one partition per child instead of a whole-disk filesystem.
	// The child partition keeps an unknown FsType (nil) so the format action
	// remains available.
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives: &[]hardware.Drive{
					{
						Id:          new("drive1"),
						Serial:      new("SERIAL123"),
						Filesystems: nil, // No filesystems reported by Supervisor
					},
				},
				Devices: &[]hardware.Device{
					{
						Name:    new("sdc"),
						DevPath: new("/dev/sdc"),
						ById:    new("/dev/disk/by-id/usb-TESTFLASH_123"),
						Attributes: &hardware.Attributes{
							IDSERIALSHORT: new("SERIAL123"),
						},
						Children: &[]string{
							"/sys/devices/pci0000:00/0000:00:14.0/usb1/1-2/1-2:1.0/host2/target2:0:0/2:0:0:0/block/sdc/sdc1",
						},
					},
					{
						Name:    new("sdc1"),
						DevPath: new("/dev/sdc1"),
						ById:    new("/dev/disk/by-id/usb-TESTFLASH_123-0:1"),
					},
				},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)
	mock.When(suite.smartService.GetSmartInfo(mock.Any[context.Context](), mock.Any[string]())).
		ThenReturn(nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", "sdc"))
	mock.When(suite.hdidleService.GetDeviceConfig(mock.Any[string]())).
		ThenReturn(nil, errors.WithDetails(dto.ErrorHDIdleNotSupported, "device", "sdc"))

	suite.hardwareService.MockSetFSProbeFunc(func(devPath string) (string, uintptr, error) {
		suite.Equal("/dev/sdc", devPath)
		return "", 0, nil
	})

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.NoError(err)
	suite.NotNil(disks)
	suite.Len(disks, 1, "Drive should be kept")

	disk, ok := disks["usb-TESTFLASH_123"]
	suite.True(ok, "Disk should be present by by-id name")
	suite.NotNil(disk.Partitions, "Child partition should be synthesized")
	suite.Len(*disk.Partitions, 1, "One real partition from the child device")
	for _, part := range *disk.Partitions {
		suite.Equal("sdc1", *part.LegacyDeviceName)
		suite.Equal("/dev/sdc1", *part.LegacyDevicePath)
		suite.Equal("/dev/disk/by-id/usb-TESTFLASH_123-0:1", *part.DevicePath)
		suite.Nil(part.FsType, "Partition with unknown filesystem keeps FsType nil")
	}
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_RawWholeDiskIgnoresStaleUdevFsType() {
	// A raw whole-disk device (no children) with a stale udev ID_FS_TYPE must
	// stay raw when the probe finds no filesystem magic. The stale udev value
	// must not leak into the synthesized partition FsType.
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives: &[]hardware.Drive{
					{
						Id:          new("drive1"),
						Serial:      new("SERIAL123"),
						Filesystems: nil, // No filesystems reported by Supervisor
					},
				},
				Devices: &[]hardware.Device{
					{
						Name:    new("sdd"),
						DevPath: new("/dev/sdd"),
						ById:    new("/dev/disk/by-id/usb-TESTRAW_456"),
						Attributes: &hardware.Attributes{
							IDSERIALSHORT: new("SERIAL123"),
							// Stale udev entry from a previous filesystem that has
							// since been removed from the raw disk.
							IDFSTYPE: new("vfat"),
						},
					},
				},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)
	mock.When(suite.smartService.GetSmartInfo(mock.Any[context.Context](), mock.Any[string]())).
		ThenReturn(nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", "sdd"))
	mock.When(suite.hdidleService.GetDeviceConfig(mock.Any[string]())).
		ThenReturn(nil, errors.WithDetails(dto.ErrorHDIdleNotSupported, "device", "sdd"))

	// Probe reports no filesystem magic: the disk is raw, so the stale udev
	// ID_FS_TYPE must not leak into the partition FsType.
	suite.hardwareService.MockSetFSProbeFunc(func(devPath string) (string, uintptr, error) {
		suite.Equal("/dev/sdd", devPath)
		return "", 0, nil
	})

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.NoError(err)
	suite.NotNil(disks)
	suite.Len(disks, 1, "Drive should be kept")

	disk, ok := disks["usb-TESTRAW_456"]
	suite.True(ok, "Disk should be present by by-id name")
	suite.NotNil(disk.Partitions, "Whole-disk filesystem should produce a partition")
	suite.Len(*disk.Partitions, 1, "Exactly one whole-disk partition")
	for _, part := range *disk.Partitions {
		suite.Equal("sdd", *part.LegacyDeviceName)
		suite.Equal("/dev/disk/by-id/usb-TESTRAW_456", *part.DevicePath)
		suite.Nil(part.FsType, "Raw whole-disk partition must not use stale udev FsType")
	}
}
