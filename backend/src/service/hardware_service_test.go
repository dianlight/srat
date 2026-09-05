package service_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
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
		// Child synthesis skips the whole-disk probe; the per-partition
		// fallback (#1072) probes /dev/sdc1. No magic here keeps FsType nil.
		suite.Equal("/dev/sdc1", devPath)
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

func (suite *HardwareServiceSuite) TestGetHardwareInfo_DriveWithFilesystemsSynthesizesMissingChildPartitions() {
	// A drive that already has filesystems reported by the Supervisor (e.g. the
	// KINGSTON system disk) must still synthesize missing partition children.
	// Partitions without filesystem magic (e.g. sda6, an 8M partition) are absent
	// from drive.Filesystems but present in device.Children; they must be added
	// so the volume listing matches the real partition table (issue #906).
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
								Id:          new("by-id-ata-KINGSTON_50026B77560145CF-part1"),
								Device:      new("/dev/sda1"),
								MountPoints: &[]string{},
							},
							{
								Id:          new("by-id-ata-KINGSTON_50026B77560145CF-part8"),
								Device:      new("/dev/sda8"),
								MountPoints: &[]string{},
							},
						},
					},
				},
				Devices: &[]hardware.Device{
					{
						Name:    new("sda"),
						DevPath: new("/dev/sda"),
						ById:    new("/dev/disk/by-id/ata-KINGSTON_50026B77560145CF"),
						Attributes: &hardware.Attributes{
							IDSERIALSHORT: new("SERIAL123"),
						},
						Children: &[]string{
							"/sys/devices/pci0000:00/0000:00:14.0/ata1/host0/target0:0:0/0:0:0:0/block/sda/sda1",
							"/sys/devices/pci0000:00/0000:00:14.0/ata1/host0/target0:0:0/0:0:0:0/block/sda/sda6",
							"/sys/devices/pci0000:00/0000:00:14.0/ata1/host0/target0:0:0/0:0:0:0/block/sda/sda8",
						},
					},
					{
						Name:    new("sda1"),
						DevPath: new("/dev/sda1"),
						ById:    new("/dev/disk/by-id/ata-KINGSTON_50026B77560145CF-part1"),
					},
					{
						Name:    new("sda6"),
						DevPath: new("/dev/sda6"),
						ById:    new("/dev/disk/by-id/ata-KINGSTON_50026B77560145CF-part6"),
					},
					{
						Name:    new("sda8"),
						DevPath: new("/dev/sda8"),
						ById:    new("/dev/disk/by-id/ata-KINGSTON_50026B77560145CF-part8"),
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

	suite.hardwareService.MockSetFSProbeFunc(func(devPath string) (string, uintptr, error) {
		return "", 0, nil
	})

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.NoError(err)
	suite.NotNil(disks)
	suite.Len(disks, 1, "Drive should be kept")

	disk, ok := disks["ata-KINGSTON_50026B77560145CF"]
	suite.True(ok, "Disk should be present by by-id name")
	suite.NotNil(disk.Partitions, "Missing child partition should be synthesized")
	suite.Len(*disk.Partitions, 3, "Two reported filesystems plus one synthesized missing child")

	var foundSda6 *dto.Partition
	for _, part := range *disk.Partitions {
		if part.LegacyDeviceName != nil && *part.LegacyDeviceName == "sda6" {
			partCopy := part
			foundSda6 = &partCopy
		}
	}
	suite.Require().NotNil(foundSda6, "sda6 partition should be synthesized from child devices")
	suite.Equal("/dev/sda6", *foundSda6.LegacyDevicePath)
	suite.Equal("/dev/disk/by-id/ata-KINGSTON_50026B77560145CF-part6", *foundSda6.DevicePath)
	suite.Nil(foundSda6.FsType, "Partition with unknown filesystem keeps FsType nil")
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_DeviceWithNilByIdDoesNotPanic() {
	// Regression test for hassio-addons#729: HA Supervisor started returning
	// devices where ById is nil but DevPath is non-nil. Before the fix the
	// device-matching loop dereferenced device.ById without a nil check,
	// causing a panic.
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
								Device:      new("/dev/sda1"),
								Name:        new("filesystem1"),
								MountPoints: &[]string{},
							},
						},
					},
				},
				Devices: &[]hardware.Device{
					{
						// DevPath is set but ById is nil — triggers the crash.
						Name:    new("sda"),
						DevPath: new("/dev/sda"),
						ById:    nil,
					},
				},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// Execute — must not panic
	disks, err := suite.hardwareService.GetHardwareInfo()

	suite.NoError(err)
	suite.NotNil(disks)
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_DeviceWithNilNameDoesNotPanic() {
	// Regression test for hassio-addons#729: device with nil Name must be
	// skipped gracefully instead of causing a nil pointer dereference.
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
								Device:      new("/dev/sda1"),
								Name:        new("filesystem1"),
								MountPoints: &[]string{},
							},
						},
					},
				},
				Devices: &[]hardware.Device{
					{
						// Name is nil — triggers the crash at line 274.
						Name:    nil,
						DevPath: new("/dev/sda"),
						ById:    new("/dev/disk/by-id/ata-some-disk"),
					},
				},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// Execute — must not panic
	disks, err := suite.hardwareService.GetHardwareInfo()

	suite.NoError(err)
	suite.NotNil(disks)
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

// TestGetHardwareInfo_EmptyIDFSTYPEFallsBackToProbe reproduces issue #1072:
// a healthy ext4 USB partition (sdc1) whose Supervisor udev entry carries no
// ID_FS_TYPE must still report FsType via the FSFromBlock magic probe so
// /api/volumes shows type ext4 and canMount true downstream.
func (suite *HardwareServiceSuite) TestGetHardwareInfo_EmptyIDFSTYPEFallsBackToProbe() {
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
						Id:     new("drive-usb"),
						Serial: new("USB123"),
						Filesystems: &[]hardware.Filesystem{
							{
								Id:          new("by-id-usb-TEST_123-part1"),
								Name:        new("sdc1"),
								Device:      new("/dev/sdc1"),
								MountPoints: &[]string{},
							},
						},
					},
				},
				Devices: &[]hardware.Device{
					{
						Name:    new("sdc"),
						DevPath: new("/dev/sdc"),
						ById:    new("/dev/disk/by-id/usb-TEST_123"),
						Attributes: &hardware.Attributes{
							IDSERIALSHORT: new("USB123"),
						},
						Children: &[]string{
							"/sys/devices/pci0000:00/0000:00:14.0/usb1/1-2/1-2:1.0/host2/target2:0:0/2:0:0:0/block/sdc/sdc1",
						},
					},
					{
						Name:       new("sdc1"),
						DevPath:    new("/dev/sdc1"),
						ById:       new("/dev/disk/by-id/usb-TEST_123-part1"),
						Attributes: &hardware.Attributes{
							// Intentionally no IDFSTYPE: Supervisor reports
							// empty udev fstype while blkid sees ext4.
						},
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
		if devPath == "/dev/sdc1" {
			return "ext4", 0, nil
		}
		return "", 0, nil
	})

	disks, err := suite.hardwareService.GetHardwareInfo()

	suite.NoError(err)
	suite.NotNil(disks)
	suite.Len(disks, 1)
	disk, ok := disks["usb-TEST_123"]
	suite.Require().True(ok, "disk should be present by by-id name")
	suite.Require().NotNil(disk.Partitions)
	suite.Require().Len(*disk.Partitions, 1)
	for _, part := range *disk.Partitions {
		suite.Equal("sdc1", *part.LegacyDeviceName)
		suite.Equal("/dev/sdc1", *part.LegacyDevicePath)
		suite.Equal("/dev/disk/by-id/usb-TEST_123-part1", *part.DevicePath)
		suite.Require().NotNil(part.FsType, "empty IDFSTYPE must fall back to FS probe")
		suite.Equal("ext4", *part.FsType)
	}
}

// TestGetHardwareInfo_MissingDeviceEntryFallsBackToLegacyProbe reproduces the
// observed #1072 remote state: sdc1 present in drive.Filesystems with
// LegacyDevicePath /dev/sdc1 but no matching Devices entry, leaving DevicePath
// nil and FsType nil. The fallback must probe LegacyDevicePath for ext4 and
// reconstruct DevicePath from the by-id partition Id.
func (suite *HardwareServiceSuite) TestGetHardwareInfo_MissingDeviceEntryFallsBackToLegacyProbe() {
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
						Id:     new("drive-usb"),
						Serial: new("USB123"),
						Filesystems: &[]hardware.Filesystem{
							{
								Id:          new("by-id-usb-TEST_123-part1"),
								Name:        new("sdc1"),
								Device:      new("/dev/sdc1"),
								MountPoints: &[]string{},
							},
						},
					},
				},
				Devices: &[]hardware.Device{
					{
						Name:    new("sdc"),
						DevPath: new("/dev/sdc"),
						ById:    new("/dev/disk/by-id/usb-TEST_123"),
						Attributes: &hardware.Attributes{
							IDSERIALSHORT: new("USB123"),
						},
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
		if devPath == "/dev/sdc1" {
			return "ext4", 0, nil
		}
		return "", 0, nil
	})

	disks, err := suite.hardwareService.GetHardwareInfo()

	suite.NoError(err)
	suite.NotNil(disks)
	suite.Len(disks, 1)
	disk, ok := disks["usb-TEST_123"]
	suite.Require().True(ok, "disk should be present by by-id name")
	suite.Require().NotNil(disk.Partitions)
	suite.Require().Len(*disk.Partitions, 1)
	for _, part := range *disk.Partitions {
		suite.Equal("sdc1", *part.LegacyDeviceName)
		suite.Equal("/dev/sdc1", *part.LegacyDevicePath)
		suite.Require().NotNil(part.DevicePath, "missing device entry must reconstruct DevicePath")
		suite.Equal("/dev/disk/by-id/usb-TEST_123-part1", *part.DevicePath)
		suite.Require().NotNil(part.FsType, "missing device entry must fall back to legacy probe")
		suite.Equal("ext4", *part.FsType)
	}
}

// sn740SupervisorResponse builds a Supervisor /hardware/info response modeling
// the WD PC SN740 layout from issue #990: one NVMe drive matched by serial to a
// whole-disk device exposing `childCount` partition children and no reported
// filesystems, so the fallback synthesis turns every child into a partition.
func sn740SupervisorResponse(childCount int) *hardware.GetHardwareInfoResponse {
	children := make([]string, 0, childCount)
	devices := []hardware.Device{
		{
			Name:     new("nvme0n1"),
			DevPath:  new("/dev/nvme0n1"),
			ById:     new("/dev/disk/by-id/nvme-SN740"),
			Children: &children,
			Attributes: &hardware.Attributes{
				IDSERIALSHORT: new("SN740TEST"),
			},
		},
	}
	for i := 1; i <= childCount; i++ {
		partName := fmt.Sprintf("nvme0n1p%d", i)
		children = append(children, "/sys/devices/virtual/block/"+partName)
		devices = append(devices, hardware.Device{
			Name:    new(partName),
			DevPath: new("/dev/" + partName),
			ById:    new("/dev/disk/by-id/" + partName),
		})
	}
	return &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives: &[]hardware.Drive{
					{
						Id:     new("drive-sn740"),
						Serial: new("SN740TEST"),
					},
				},
				Devices: &devices,
			},
		},
	}
}

// TestGetHardwareInfo_WarnsOnPhantomPartitionCountIncrease reproduces the
// issue #990 fingerprint: an unchanged NVMe drive whose partition count grows
// (2→3) between two enumerations without any local partitioning action. The
// first published snapshot is the silent baseline; the growth must surface as
// a slog warning carrying serial, previous/current counts and the appeared
// partition name so operators can correlate it with probe/udev activity.
func (suite *HardwareServiceSuite) TestGetHardwareInfo_WarnsOnPhantomPartitionCountIncrease() {
	var buf bytes.Buffer
	oldDefault := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(oldDefault)

	mock.When(suite.smartService.GetSmartInfo(mock.Any[context.Context](), mock.Any[string]())).
		ThenReturn(nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", "nvme0n1"))
	mock.When(suite.hdidleService.GetDeviceConfig(mock.Any[string]())).
		ThenReturn(nil, errors.WithDetails(dto.ErrorHDIdleNotSupported, "device", "nvme0n1"))
	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).
		ThenReturn(sn740SupervisorResponse(2), nil).
		ThenReturn(sn740SupervisorResponse(3), nil)

	// Baseline enumeration: two partitions, no anomaly expected.
	disks, err := suite.hardwareService.GetHardwareInfo()
	suite.Require().NoError(err)
	disk, ok := disks["nvme-SN740"]
	suite.Require().True(ok, "drive should be keyed by its by-id name")
	suite.Require().NotNil(disk.Partitions)
	suite.Len(*disk.Partitions, 2)
	suite.NotContains(buf.String(), "Anomalous partition count increase",
		"the first published snapshot is the diff baseline and must not warn")

	// Second enumeration observing the phantom third partition.
	suite.hardwareService.InvalidateHardwareInfo()
	buf.Reset()

	disks, err = suite.hardwareService.GetHardwareInfo()
	suite.Require().NoError(err)
	disk, ok = disks["nvme-SN740"]
	suite.Require().True(ok)
	suite.Require().NotNil(disk.Partitions)
	suite.Len(*disk.Partitions, 3)

	logs := buf.String()
	suite.Contains(logs, "Anomalous partition count increase without local partitioning action")
	suite.Contains(logs, "serial=SN740TEST")
	suite.Contains(logs, "disk_id=nvme-SN740")
	suite.Contains(logs, "previous_partition_count=2")
	suite.Contains(logs, "current_partition_count=3")
	suite.Contains(logs, "new_partitions=nvme0n1p3")
}
