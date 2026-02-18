package service

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"fmt"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbom/g"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/tlog"
	"github.com/pilebones/go-udev/netlink"
	"github.com/prometheus/procfs"
	"github.com/shomali11/util/xhashes"
	"github.com/u-root/u-root/pkg/mount"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/*
VolumeServiceInterface defines the interface for managing volumes and mount points.

Copilot file rules:
- Always validate input parameters for mount and unmount operations.
- Always update disks map puntually after an operation that changes state.
*/

type VolumeServiceInterface interface {
	MountVolume(md *dto.MountPointData) errors.E
	UnmountVolume(id string, force bool) errors.E
	GetVolumesData() []*dto.Disk
	//PathHashToPath(pathhash string) (string, errors.E)
	GetDevicePathByDeviceID(deviceID string) (string, errors.E)
	//EjectDisk(diskID string) error
	PatchMountPointSettings(root string, path string, settingsPatch dto.MountPointData) (*dto.MountPointData, errors.E)
	// Test only
	MockSetProcfsGetMounts(f func() ([]*procfs.MountInfo, error))
	CreateBlockDevice(device string) error
}

type VolumeService struct {
	ctx        context.Context
	db         *gorm.DB
	refreshing atomic.Bool
	//broascasting    BroadcasterServiceInterface
	hardwareClient  HardwareServiceInterface
	fs_service      FilesystemServiceInterface
	shareService    ShareServiceInterface
	issueService    IssueServiceInterface
	state           *dto.ContextState
	sfGroup         singleflight.Group
	haService       HomeAssistantServiceInterface
	hdidleService   HDIdleServiceInterface
	eventBus        events.EventBusInterface
	convDto         converter.DtoToDbomConverterImpl
	convMDto        converter.MountToDtoImpl
	procfsGetMounts func() ([]*procfs.MountInfo, error)
	disks           *dto.DiskMap
	refreshVersion  uint32
	// test hooks for mount operations
	tryMountFunc func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error)
	doMountFunc  func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error)
	unmountFunc  func(target string, force, lazy bool) error
}

type VolumeServiceProps struct {
	fx.In
	Ctx context.Context
	Db  *gorm.DB
	//MountPointRepo    repository.MountPointPathRepositoryInterface
	HardwareClient    HardwareServiceInterface `optional:"true"`
	FilesystemService FilesystemServiceInterface
	ShareService      ShareServiceInterface
	IssueService      IssueServiceInterface
	State             *dto.ContextState
	HAService         HomeAssistantServiceInterface `optional:"true"`
	HDIdleService     HDIdleServiceInterface        `optional:"true"`
	EventBus          events.EventBusInterface
}

func NewVolumeService(
	lc fx.Lifecycle,
	in VolumeServiceProps,
) VolumeServiceInterface {
	p := &VolumeService{
		ctx:             in.Ctx,
		db:              in.Db,
		hardwareClient:  in.HardwareClient,
		fs_service:      in.FilesystemService,
		state:           in.State,
		shareService:    in.ShareService,
		issueService:    in.IssueService,
		haService:       in.HAService,
		hdidleService:   in.HDIdleService,
		eventBus:        in.EventBus,
		convDto:         converter.DtoToDbomConverterImpl{},
		convMDto:        converter.MountToDtoImpl{},
		procfsGetMounts: procfs.GetMounts,
		disks:           &dto.DiskMap{},
		refreshVersion:  0,
		tryMountFunc:    mount.TryMount,
		doMountFunc:     mount.Mount,
		unmountFunc:     mount.Unmount,
	}

	var unsubscribe [6]func()
	unsubscribe[0] = p.eventBus.OnPartition(p.handlePartitionEvent)
	unsubscribe[1] = p.eventBus.OnMountPoint(p.handleMountPointEvent)
	unsubscribe[2] = p.eventBus.OnHomeAssistant(func(ctx context.Context, hae events.HomeAssistantEvent) errors.E {
		tlog.DebugContext(ctx, "Home Assistant started event received, getVolumesData called")
		if hae.Type == events.EventTypes.START {
			err := p.getVolumesData()
			if err != nil {
				slog.ErrorContext(ctx, "Failed to refresh volumes data on Home Assistant start event", "err", err)
			}
		}
		return nil
	})
	unsubscribe[3] = p.eventBus.OnShare(func(ctx context.Context, se events.ShareEvent) errors.E {
		tlog.DebugContext(ctx, "Share event received update cache volumes", "event_type", se.Type, "share", se.Share)
		switch se.Type {
		case events.EventTypes.REMOVE:
			ok, disk := p.disks.RemoveMountPointShare(se.Share.Name)
			if !ok {
				slog.WarnContext(ctx, "Failed to remove share from mount point in cache", "share", se.Share.Name)
			} else {
				p.eventBus.EmitDisk(events.DiskEvent{
					Event: events.Event{
						Type: events.EventTypes.UPDATE,
					},
					Disk: disk,
				})
			}
		case events.EventTypes.ADD, events.EventTypes.UPDATE:

			disk, err := p.disks.AddMountPointShare(se.Share)
			if err != nil {
				if se.Share.Usage != "internal" {
					slog.WarnContext(ctx, "Failed to add/update share in mount point in cache", "share", se.Share, "err", err)
				}
				return nil
			}
			p.eventBus.EmitDisk(events.DiskEvent{
				Event: events.Event{
					Type: events.EventTypes.UPDATE,
				},
				Disk: disk,
			})
		}
		return nil
	})
	unsubscribe[4] = p.eventBus.OnSmart(func(ctx context.Context, se events.SmartEvent) errors.E {
		p.disks.AddSmartInfo(&se.SmartInfo)
		return nil
	})
	unsubscribe[5] = p.eventBus.OnPower(func(ctx context.Context, pe events.PowerEvent) errors.E {
		// Handle PowerEvent
		p.disks.AddHDIdleDevice(&pe.PowerInfo)
		return nil
	})
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			err := p.getVolumesData()
			if err != nil {
				return err
			}
			p.ctx.Value("wg").(*sync.WaitGroup).Go(func() {
				p.udevEventHandler()
			})
			return nil
		},
		OnStop: func(ctx context.Context) error {
			for _, unsub := range unsubscribe {
				if unsub != nil {
					unsub()
				}
			}
			return nil
		},
	})

	return p
}

func (self *VolumeService) persistMountPoint(md *dto.MountPointData) errors.E {
	dbom_mount_data := &dbom.MountPointPath{}
	err := self.convDto.MountPointDataToMountPointPath(*md, dbom_mount_data)
	if err != nil {
		return errors.WithStack(err)
	}
	err = self.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).
		Create(dbom_mount_data).Error
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (ms *VolumeService) MountVolume(md *dto.MountPointData) errors.E {
	// Early validation of required fields
	if ms.state.ProtectedMode {
		return errors.WithDetails(dto.ErrorOperationNotPermittedInProtectedMode,
			"Operation", "MountVolume",
			"Detail", "Mount operation is not permitted when ProtectedMode is enabled.",
		)
	}

	if md == nil {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"Message", "MountPointData is nil",
		)
	}

	if md.Path == "" {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"Message", "Mount point path is empty",
		)
	}

	if md.Root == "" {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"DeviceId", md.DeviceId,
			"Root", md.Root,
			"Message", "Mount point root is empty",
		)
	}

	if md.DeviceId == "" {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"Message", "Source device name is empty in request",
		)
	}

	if md.Partition == nil || md.Partition.Id == nil || *md.Partition.Id == "" {
		for _, disk := range *ms.disks {
			for _, part := range *disk.Partitions {
				if *part.Id == md.DeviceId {
					md.Partition = &part
					break
				}
			}
		}
	}

	if md.Partition == nil {
		return errors.WithDetails(dto.ErrorDeviceNotFound,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"Message", "Source device does not exist on the system",
		)
	}

	if md.Partition.DevicePath == nil || *md.Partition.DevicePath == "" {
		return errors.WithDetails(dto.ErrorDeviceNotFound,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"Message", "Source device does not exist on the system",
		)
	}

	ok, errS := osutil.IsMounted(md.Path)
	if errS != nil {
		// Note: IsMounted might fail if the path doesn't exist yet, which is fine before mounting.
		// Consider if this check needs refinement based on expected state.
		// For now, we proceed assuming an error here might be ignorable if ok is false.
		if ok { // Only return error if it claims to be mounted but check failed
			//slog.Error("Error checking if path is mounted", "path", dbom_mount_data.Path, "err", errS)
			return errors.WithDetails(dto.ErrorMountFail, "Detail", "Error checking mount status", "Path", md.Path, "Error", errS)
		}
		slog.DebugContext(ms.ctx, "osutil.IsMounted check failed, but path not mounted, proceeding", "path", md.Path, "err", errS)
		ok = false // Ensure ok is false if IsMounted errored
	}

	if ok {
		slog.WarnContext(ms.ctx, "Volume already mounted according to OS check", "device", md.DeviceId, "path", md.Path)
		return errors.WithDetails(dto.ErrorAlreadyMounted,
			"Device", md.DeviceId,
			"Path", md.Path,
			"Message", "Volume is already mounted",
		)
	}

	/*
		// Rename logic if path is already mounted (even if DB state was inconsistent)
		orgPath := md.Path
		if ok { // Only rename if osutil.IsMounted returned true
			slog.InfoContext(ms.ctx, "Attempting to rename mount path due to conflict", "original_path", orgPath)
			for i := 1; ; i++ {
				md.Path = orgPath + "_(" + strconv.Itoa(i) + ")"
				okCheck, errCheck := osutil.IsMounted(md.Path)
				if errCheck != nil {
					// Similar to above, error might be okay if path doesn't exist yet
					if okCheck {
								slog.ErrorContext(ms.ctx, "Error checking renamed path mount status", "path", md.Path, "err", errCheck)
						return errors.WithDetails(dto.ErrorMountFail, "Detail", "Error checking renamed mount status", "Path", md.Path, "Error", errCheck)
					}
					okCheck = false // Treat error as not mounted
				}
				if !okCheck {
					slog.InfoContext(ms.ctx, "Found available renamed path", "new_path", md.Path)
					break // Found an unused path
				}
				if i > 100 { // Safety break
					slog.ErrorContext(ms.ctx, "Could not find available renamed mount path after 100 attempts", "original_path", orgPath)
					return errors.WithDetails(dto.ErrorMountFail, "Path", orgPath, "Message", "Could not find available renamed mount path")
				}
			}
		}
	*/

	// Initialize flags if nil to avoid nil pointer dereference
	if md.Flags == nil {
		md.Flags = &dto.MountFlags{}
		slog.DebugContext(ms.ctx, "Initialized nil Flags to empty MountFlags", "device", md.DeviceId, "path", md.Path)
	}

	flags, data, err := ms.fs_service.MountFlagsToSyscallFlagAndData(*md.Flags)
	if err != nil {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"Device", md.DeviceId,
			"Path", md.Path,
			"Message", "Invalid Flags",
			"Error", err,
		)
	}

	mountFsType := ""
	if md.FSType != nil && *md.FSType != "" {
		mountFsType = ms.fs_service.ResolveLinuxFsModule(*md.FSType)
		if mountFsType == "" {
			mountFsType = *md.FSType
		}
	}

	slog.DebugContext(ms.ctx, "Attempting to mount volume", "device", md.DeviceId, "path", md.Path, "fstype", md.FSType, "mount_fstype", mountFsType, "flags", flags, "data", data)

	var mp *mount.MountPoint
	// Ensure secure directory permissions when creating mount point
	mountFunc := func() error { return os.MkdirAll(md.Path, 0o750) }

	// Final validation before mount - ensure DevicePath is not nil
	if md.Partition.DevicePath == nil || *md.Partition.DevicePath == "" {
		return errors.WithDetails(dto.ErrorDeviceNotFound,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"Message", "Device path is nil or empty, cannot mount",
		)
	} else if _, err := os.Stat(*md.Partition.DevicePath); err != nil {
		if os.IsPermission(err) {
			return errors.WithDetails(dto.ErrorOperationNotPermitted,
				"DeviceId", md.DeviceId,
				"Path", md.Path,
				"DevicePath", *md.Partition.DevicePath,
				"Message", "Permission denied to access device",
				"Error", err.Error(),
			)
		}
		return errors.WithDetails(dto.ErrorDeviceNotFound,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"DevicePath", *md.Partition.DevicePath,
			"Message", "Device path does not exist",
			"Error", err.Error(),
		)
	}

	// FIXME: Manage mount with different roots if needed
	if md.FSType == nil || *md.FSType == "" {
		// Use TryMount if FSType is not specified
		mp, errS = ms.tryMountFunc(*md.Partition.DevicePath, md.Path, data, flags, mountFunc)
	} else {
		// Use Mount if FSType is specified
		mp, errS = ms.doMountFunc(*md.Partition.DevicePath, md.Path, mountFsType, data, flags, mountFunc)
	}

	if errS != nil {
		// Provide detailed error message with all context

		fsTypeStr := "auto"
		if md.FSType != nil {
			fsTypeStr = *md.FSType
		}

		slog.ErrorContext(ms.ctx, "Failed to mount volume",
			"device_id", md.DeviceId,
			"device_path", *md.Partition.DevicePath,
			"fstype", fsTypeStr,
			"mount_fstype", mountFsType,
			"mount_path", md.Path,
			"flags", flags,
			"data", data,
			"mount_error", errS,
			"mountpoint_details", mp)

		// Attempt to clean up directory if we created it and mount failed
		if _, statErr := os.Stat(md.Path); statErr == nil {
			// Directory exists, try to remove it
			if removeErr := os.Remove(md.Path); removeErr != nil {
				slog.WarnContext(ms.ctx, "Failed to cleanup mount directory after mount failure",
					"path", md.Path,
					"cleanup_error", removeErr)
			}
		}

		return errors.WithDetails(dto.ErrorMountFail,
			"Detail", fmt.Sprintf("Mount command failed: %v", errS),
			"DeviceId", md.DeviceId,
			"DevicePath", *md.Partition.DevicePath,
			"MountPath", md.Path,
			"FSType", fsTypeStr,
			"MountFSType", mountFsType,
			"Flags", flags,
			"Data", data,
			"Error", errS.Error(),
		)
	} else {
		slog.InfoContext(ms.ctx, "Successfully mounted volume", "device", md.DeviceId, "path", md.Path, "fstype", md.FSType, "mount_fstype", mountFsType, "flags", mp.Flags, "data", mp.Data)
		// Update dbom_mount_data with details from the actual mount point if available
		errS = ms.convMDto.MountToMountPointData(mp, md, ms.disks)
		if errS != nil {
			return errors.WithDetails(dto.ErrorMountFail,
				"Detail", "Failed to convert mount details back to DTO after successful mount",
				"DeviceId", md.DeviceId,
				"MountPath", md.Path,
				"Error", errS.Error(),
			)
		}
		errS = ms.disks.AddOrUpdateMountPoint(*md.Partition.DiskId, *md.Partition.Id, *md)
		if errS != nil {
			slog.ErrorContext(ms.ctx, "Failed to add mount point to in-memory cache after successful mount", "device", md.DeviceId, "path", md.Path, "err", errS)
			return errors.WithDetails(dto.ErrorMountFail,
				"Detail", "Failed to add mount point to in-memory cache after successful mount",
				"DeviceId", md.DeviceId,
				"MountPath", md.Path,
				"Error", errS.Error(),
			)
		}
		ms.eventBus.EmitMountPoint(events.MountPointEvent{
			Event:      events.Event{Type: events.EventTypes.UPDATE},
			MountPoint: md,
		})
	}

	// Dismiss any existing failure notifications since the mount was successful
	ms.dismissAutomountNotification(md.DeviceId, "automount_failure")
	ms.dismissAutomountNotification(md.DeviceId, "unmounted_partition")

	return nil
}

/*
func (ms *VolumeService) PathHashToPath(pathhash string) (string, errors.E) {
	dbom_mount_data, err := gorm.G[dbom.MountPointPath](ms.db).Find(ms.ctx)
	if err != nil {
		return "", errors.WithStack(err)
	}
	for _, mount_data := range dbom_mount_data {
		if xhashes.SHA1(mount_data.Path) == pathhash {
			return mount_data.Path, nil
		}
	}
	return "", errors.New("PathHash not found")
}
*/

func (ms *VolumeService) UnmountVolume(path string, force bool) errors.E {
	// Look up mount point data from in-memory cache first
	md, ok := ms.disks.GetMountPointByPath(path)
	if ok && md.Share != nil && md.Share.Status.IsHAMounted {
		slog.DebugContext(ms.ctx, "Found mount point as HAMounted", "path", path)
		md.IsInvalid = true
		ms.eventBus.EmitShare(events.ShareEvent{
			Event: events.Event{Type: events.EventTypes.REMOVE},
			Share: md.Share,
		})
	} else if !ok {
		// Fallback to DB lookup if not found in cache
		slog.DebugContext(ms.ctx, "Mount point not found in cache, looking up in DB", "path", path)
		md = &dto.MountPointData{Path: path}
	}
	return ms.unmountVolume(md, force)
}

func (ms *VolumeService) unmountVolume(md *dto.MountPointData, force bool) errors.E {
	slog.DebugContext(ms.ctx, "Attempting to unmount volume", "path", md.Path, "force", force)
	unmountErr := errors.WithStack(ms.unmountFunc(md.Path, force, !force))
	if unmountErr != nil {
		slog.ErrorContext(ms.ctx, "Failed to unmount volume", "path", md.Path, "err", unmountErr)
		return errors.WithDetails(dto.ErrorUnmountFail, "Detail", unmountErr.Error(), "Path", md.Path, "Error", unmountErr)
	}

	slog.InfoContext(ms.ctx, "Successfully unmounted volume", "path", md.Path)

	if err := os.Remove(md.Path); err != nil {
		slog.WarnContext(ms.ctx, "Failed to remove mount point directory", "path", md.Path, "err", err)
	} else {
		slog.DebugContext(ms.ctx, "Removed mount point directory", "path", md.Path)
	}
	// Unmount succeeded
	if md != nil && md.Partition != nil && md.Partition.DiskId != nil && md.Partition.Id != nil {
		md.IsMounted = false
		ms.eventBus.EmitMountPoint(events.MountPointEvent{
			Event:      events.Event{Type: events.EventTypes.UPDATE},
			MountPoint: md,
		})
		ms.disks.AddOrUpdateMountPoint(*md.Partition.DiskId, *md.Partition.Id, *md)
	}

	return nil
}

func (self *VolumeService) udevEventHandler() {
	tlog.TraceContext(self.ctx, "Starting Udev event handler...")

	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		tlog.ErrorContext(self.ctx, "Unable to connect to Netlink Kobject UEvent socket", "err", err)
		return // Exit goroutine if connection fails
	}
	defer conn.Close()

	queue := make(chan netlink.UEvent, 10)
	errorChan := make(chan error, 1)
	quit := conn.Monitor(queue, errorChan, nil)
	tlog.TraceContext(self.ctx, "Udev monitor started successfully.")

	// Handling message from queue
	for {
		select {
		case <-self.ctx.Done():
			slog.InfoContext(self.ctx, "Udev event handler stopping due to context cancellation.", "err", self.ctx.Err())
			if quit != nil {
				close(quit)
			}
			return
		case uevent := <-queue:
			// Filter events - only interested in block devices for now
			if subsystem, ok := uevent.Env["SUBSYSTEM"]; ok && subsystem == "block" {
				action := uevent.Action
				devName, _ := uevent.Env["DEVNAME"]
				devType, _ := uevent.Env["DEVTYPE"]

				slog.DebugContext(self.ctx, "Received Udev block event", "action", action, "devname", devName, "devtype", devType, "env", uevent.Env)

				if devType != "disk" && devType != "partition" {
					slog.DebugContext(self.ctx, "Ignoring Udev event for non-disk/partition block device", "devname", devName, "devtype", devType)
					continue
				}
				// FIXME: Process Right events here sending events to refresh volumes data
				// Process block device events
				if action == "remove" && devType == "disk" {
					bus, _ := uevent.Env["ID_BUS"]
					suffix, _ := uevent.Env[".PART_SUFFIX"]
					serial, _ := uevent.Env["ID_SERIAL"]

					slog.InfoContext(self.ctx, "Processing block device removal event", "devname", devName, "bus", bus, "serial", serial, "suffix", suffix)
					self.disks.Remove(bus + "-" + serial + suffix)
				} else if devType == "disk" && action == "add" {
					slog.InfoContext(self.ctx, "Processing block device event", "action", action, "devname", devName)

					// Get current volumes data
					self.hardwareClient.InvalidateHardwareInfo()
					err := self.getVolumesData()
					if err != nil {
						slog.ErrorContext(self.ctx, "Failed to get volumes data after udev event", "err", err)
						continue
					}
				} else if devType == "disk" && action == "change" {
					slog.InfoContext(self.ctx, "Ignore: Processing block device change event", "action", action, "devname", devName)
					continue
				} else if devType == "partition" && action == "add" {
					slog.InfoContext(self.ctx, "Processing partition addition event", "action", action, "devname", devName)
					// TODO:Check if cache contain the partition. If yes retry mount process else InvalidateHarduer and getvolumeData
				} else if devType == "partition" && action == "remove" {
					slog.InfoContext(self.ctx, "Processing partition removal event", "action", action, "devname", devName)
					// TODO:Check if cache contain the partition. if yes umount and remove from cache
				}
			}
		case err := <-errorChan:
			// Parse errors from malformed uevents are expected and should not be treated as critical
			// These can occur from kernel/driver events with non-standard formatting
			if err != nil && strings.Contains(err.Error(), "unable to parse uevent") {
				// Extract more context if available
				errMsg := err.Error()
				if strings.Contains(errMsg, "invalid env data") {
					slog.DebugContext(self.ctx, "Ignoring malformed uevent with invalid env data",
						"err", err,
						"detail", "This can occur when kernel sends events with non-standard formatting")
				} else {
					slog.DebugContext(self.ctx, "Failed to parse uevent, skipping",
						"err", err,
						"detail", "Event format not recognized or incompatible")
				}
			} else {
				// Other errors (connection issues, etc.) are more serious
				slog.ErrorContext(self.ctx, "Error received from Udev monitor", "err", err)
			}
		}
	}
}

func (self *VolumeService) GetVolumesData() []*dto.Disk {
	if len(*self.disks) == 0 {
		err := self.getVolumesData()
		if err != nil {
			slog.ErrorContext(self.ctx, "Failed to get volumes data in GetVolumesData", "err", err)
			return []*dto.Disk{}
		}
	}
	value := slices.Collect(maps.Values(*self.disks))
	return value
}

// loadMountPointFromDB loads mount point data from the database for a partition
func (self *VolumeService) loadMountPointFromDB(part *dto.Partition) (map[string]*dto.MountPointData, errors.E) {
	if part.Id == nil || *part.Id == "" {
		return nil, nil
	}

	dmp, err := gorm.G[dbom.MountPointPath](self.db).
		Preload("ExportedShare", nil).
		Where(g.MountPointPath.DeviceId.Eq(*part.Id)).
		Find(self.ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(dmp) == 0 {
		tlog.TraceContext(self.ctx, "No mount point records found in DB for device", "device", *part.Id, "name", part.Name)
		return make(map[string]*dto.MountPointData), nil
	}

	tlog.TraceContext(self.ctx, "Found mount point records in DB for device", "device", *part.Id, "name", part.Name, "count", len(dmp))
	mountData, convErr := self.convDto.MountPointPathsToMountPointDataMap(dmp)
	if convErr != nil {
		slog.ErrorContext(self.ctx, "Failed to convert mount point data", "device", *part.Id, "err", convErr)
		return nil, errors.WithStack(convErr)
	}

	slog.DebugContext(self.ctx, "Loaded mount point from repository", "device", *part.Id, "mountData", mountData)
	return mountData, nil
}

// processPartitionMountData loads mount data from DB and initializes partition mount point data
/*
func (self *VolumeService) processPartitionMountData(disk *dto.Disk, pid string, part dto.Partition, emitEvents bool) error {
	if part.DiskId == nil || *part.DiskId == "" {
		part.DiskId = disk.Id
	}

	if part.DevicePath == nil || *part.DevicePath == "" {
		slog.DebugContext(self.ctx, "Skipping partition with nil or empty device path", "disk_id", *disk.Id)
		return nil
	}

	if part.MountPointData == nil {
		part.MountPointData = &map[string]dto.MountPointData{}
	}

	mountData, err := self.loadMountPointFromDB(&part)
	if err != nil {
		return err
	}

	if len(mountData) > 0 {
		for _, mountD := range mountData {
			if mountD1, ok := (*part.MountPointData)[mountD.Path]; ok {
				mountD.IsToMountAtStartup = mountD1.IsToMountAtStartup
				mountD.RefreshVersion = self.refreshVersion
				(*part.MountPointData)[mountD.Path] = mountD1
				if emitEvents {
					self.eventBus.EmitMountPoint(events.MountPointEvent{
						Event:      events.Event{Type: events.EventTypes.UPDATE},
						MountPoint: mountD,
					})
				}
			} else {
				mountD.RefreshVersion = self.refreshVersion
				(*part.MountPointData)[mountD.Path] = *mountD
				if emitEvents {
					self.eventBus.EmitMountPoint(events.MountPointEvent{
						Event:      events.Event{Type: events.EventTypes.ADD},
						MountPoint: mountD,
					})
				}
			}
		}
		(*disk.Partitions)[pid] = part
	}

	return nil
}
*/

/*
// processNewDisk adds a new disk to the cache and loads its partition mount data from DB
func (self *VolumeService) processNewDisk(disk dto.Disk) error {
	disk.RefreshVersion = self.refreshVersion
	if disk.Partitions != nil {
		for pid, part := range *disk.Partitions {
			if err := self.processPartitionMountData(&disk, pid, part, true); err != nil {
				slog.WarnContext(self.ctx, "Failed to process partition mount data for new disk", "disk_id", *disk.Id, "partition_id", pid, "err", err)
				continue
			}
			self.eventBus.EmitPartition(events.PartitionEvent{
				Event:     events.Event{Type: events.EventTypes.ADD},
				Partition: &part,
				Disk:      &disk,
			})
		}
		err := self.disks.Add(disk)
		if err != nil {
			return err
		}
	}
	return nil
}
*/

/*
// processExistingDisk updates an existing disk's refresh version and reloads mount data from DB
func (self *VolumeService) processExistingDisk(diskId string) error {
	existing, ok := self.disks.Get(diskId)
	if !ok {
		return errors.WithDetails(dto.ErrorNotFound, "Message", "disk not found", "DiskId", diskId)
	}
	existing.RefreshVersion = self.refreshVersion
	err := self.disks.Add(existing)
	if err != nil {
		return err
	}
	// Refresh mount data from DB for all partitions to ensure is_to_mount_at_startup is up to date
	if existing.Partitions != nil {
		for pid, part := range *existing.Partitions {
			if err := self.processPartitionMountData(&existing, pid, part, true); err != nil {
				slog.WarnContext(self.ctx, "Failed to process partition mount data for new disk", "disk_id", *existing.Id, "partition_id", pid, "err", err)
				continue
			}
		}
		self.eventBus.EmitDisk(events.DiskEvent{
			Event: events.Event{Type: events.EventTypes.UPDATE},
			Disk:  &existing,
		})
	}

	return nil
}
*/

/*
// processMountInfos updates partition mount states based on current procfs mount information
func (self *VolumeService) processMountInfos(mountInfos []*procfs.MountInfo) {
	for diskName, disk := range *self.disks {
		if disk.RefreshVersion != self.refreshVersion {
			// Disk not present in current scan, remove it
			removedDisk := disk
			self.disks.Remove(diskName)
			self.eventBus.EmitDisk(events.DiskEvent{
				Event: events.Event{Type: events.EventTypes.REMOVE},
				Disk:  &removedDisk,
			})
			continue
		}

		for pidx, part := range *disk.Partitions {
			if part.Id == nil || *part.Id == "" {
				slog.DebugContext(self.ctx, "Skipping partition with nil or empty device id", "disk_id", diskName, "partition_index", pidx)
				continue
			}

			if part.MountPointData == nil {
				part.MountPointData = &map[string]dto.MountPointData{}
			}

			// Update existing mount points with current mount info
			for _, prtstate := range mountInfos {
				if mpd, ok := (*part.MountPointData)[prtstate.MountPoint]; ok {
					if !mpd.IsMounted {
						mpd.IsMounted = true
						(*part.MountPointData)[prtstate.MountPoint] = mpd
						self.eventBus.EmitMountPoint(events.MountPointEvent{
							Event:      events.Event{Type: events.EventTypes.UPDATE},
							MountPoint: &mpd,
						})
					}
					mpd.RefreshVersion = self.refreshVersion
					(*part.MountPointData)[prtstate.MountPoint] = mpd
					continue
				}

				if prtstate.Source == *part.DevicePath || (part.LegacyDevicePath != nil && prtstate.Source == *part.LegacyDevicePath) {
					// Found matching mount info for partition
					iw := osutil.IsWritable(prtstate.MountPoint)
					mountPoint := dto.MountPointData{
						Path:             prtstate.MountPoint,
						DeviceId:         *part.Id,
						PathHash:         xhashes.SHA1(prtstate.MountPoint),
						IsWriteSupported: new(iw),
						IsMounted:        true,
						Flags:            &dto.MountFlags{},
						CustomFlags:      &dto.MountFlags{},
						FSType:           new(prtstate.FSType),
						Type:             "ADDON",
						Partition:        &part,
						RefreshVersion:   self.refreshVersion,
					}
					mountPoint.Flags.Scan(prtstate.Options)
					mountPoint.CustomFlags.Scan(prtstate.SuperOptions)
					(*part.MountPointData)[prtstate.MountPoint] = mountPoint
					self.eventBus.EmitMountPoint(events.MountPointEvent{
						Event:      events.Event{Type: events.EventTypes.ADD},
						MountPoint: &mountPoint,
					})
					continue
				}
			}

			// Mark mount points not found in procfs as unmounted
			for key, mpd := range *part.MountPointData {
				if mpd.RefreshVersion != self.refreshVersion {
					mpd.IsMounted = false
					mpd.RefreshVersion = self.refreshVersion
					(*part.MountPointData)[key] = mpd
					self.eventBus.EmitMountPoint(events.MountPointEvent{
						Event:      events.Event{Type: events.EventTypes.UPDATE},
						MountPoint: &mpd,
					})
				}
			}
			err := self.disks.AddPartition(*disk.Id, part)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to update partition in disk map", "disk_id", *disk.Id, "partition_id", *part.Id, "err", err)
			}
		}
	}
}
*/
// getVolumesData retrieves and synchronizes volume data with caching and concurrency control.
// Disks and partitions are read from the hardware client and enriched with local mount point data.
// It also syncs mount point data with database records, saving new entries and removing obsolete ones.
func (self *VolumeService) getVolumesData() errors.E {
	tlog.TraceContext(self.ctx, "Requesting GetVolumesData via singleflight...")

	_, err, _ := self.sfGroup.Do("GetVolumesData", func() (any, error) {
		self.refreshVersion++
		filesystemSupportCache := make(map[string]*dto.FilesystemInfo)

		tlog.TraceContext(self.ctx, "Executing GetVolumesData core logic (singleflight)...")

		// Skip hardware client if it's not initialized
		if self.hardwareClient == nil {
			slog.DebugContext(self.ctx, "Hardware client not initialized, continuing with empty disk list")
			return self.disks, nil
		}

		// Get Host Hardware
		hwDisks, errHw := self.hardwareClient.GetHardwareInfo()
		if errHw != nil {
			return nil, errHw
		}
		if hwDisks == nil {
			tlog.TraceContext(self.ctx, "Hardware client returned nil disks, continuing with empty disk list")
			return self.disks, nil
		}

		tlog.DebugContext(self.ctx, "Retrieved hardware disks from hardware client", "disk_count", len(hwDisks))
		// Disks processing
		for _, disk := range hwDisks {
			if disk.Partitions == nil {
				continue
			}
			tlog.TraceContext(self.ctx, "Processing disk from hardware client", "disk_id", *disk.Id, "partition_count", len(*disk.Partitions))
			disk.RefreshVersion = self.refreshVersion

			currentDisk, updateDisk := self.disks.Get(*disk.Id)

			err := self.disks.AddOrUpdate(&disk)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to update existing disk in cache", "disk_id", *disk.Id, "err", err)
			}

			for pid, part := range *disk.Partitions {
				if part.FsType != nil && *part.FsType != "" {
					if cached, ok := filesystemSupportCache[*part.FsType]; ok {
						part.FilesystemInfo = cached
					} else {
						info, err := self.fs_service.GetSupportAndInfo(self.ctx, *part.FsType)
						if err != nil || info == nil || info.Support == nil {
							part.FilesystemInfo = &dto.FilesystemInfo{}
						} else {
							part.FilesystemInfo = info
						}
						filesystemSupportCache[*part.FsType] = part.FilesystemInfo
					}
				}
				(*disk.Partitions)[pid] = part
				//			if err := self.processPartitionMountData(&disk, pid, part, true); err != nil {
				//				slog.WarnContext(self.ctx, "Failed to process partition mount data for new disk", "disk_id", *disk.Id, "partition_id", pid, "err", err)
				//				continue
				//			}
				if currentDisk != nil && updateDisk {
					self.eventBus.EmitPartition(events.PartitionEvent{
						Event:     events.Event{Type: events.EventTypes.UPDATE},
						Partition: &part,
						Disk:      &disk,
					})
				} else {
					self.eventBus.EmitPartition(events.PartitionEvent{
						Event:     events.Event{Type: events.EventTypes.ADD},
						Partition: &part,
						Disk:      &disk,
					})
				}
			}
		}
		return nil, nil
	})

	if err != nil {
		//slog.Error("Singleflight execution of GetVolumesData failed", "err", err, "shared", shared)
		return errors.WithStack(err)
	}

	return nil
}

func (self *VolumeService) handlePartitionEvent(ctx context.Context, e events.PartitionEvent) errors.E {

	tlog.TraceContext(ctx, "Processing partition event for mount data sync", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "event_type", e.Type)

	if e.Partition.DevicePath == nil || *e.Partition.DevicePath == "" {
		slog.DebugContext(ctx, "Skipping partition with nil or empty device path", "disk_id", *e.Disk.Id)
		return nil
	}
	if e.Partition.DiskId == nil || *e.Partition.DiskId == "" {
		e.Partition.DiskId = e.Disk.Id
	}

	mountData, err := self.loadMountPointFromDB(e.Partition)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to load mount point data from DB for partition", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "err", err)
		return err
	}
	// Add missing mount points from DB to in-memory cache
	for _, md := range mountData {
		//		if _, ok := self.disks.GetMountPoint(*e.Partition.DiskId, *e.Partition.Id, md.Path); !ok {
		err := self.disks.AddOrUpdateMountPoint(*e.Partition.DiskId, *e.Partition.Id, *md)
		if err != nil {
			slog.WarnContext(self.ctx, "Failed to add mount point to disk map during partition event handling", "disk_id", *e.Partition.DiskId, "partition_id", *e.Partition.Id, "mount_path", md.Path, "err", err)
			continue
		}
		//		}
	}

	// Get current mount information from procfs
	mountInfos, errS := self.procfsGetMounts()
	if errS != nil {
		slog.ErrorContext(ctx, "Failed to get current mount information from procfs", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "err", errS)
		return errors.WithStack(errS)
	}

	// Update existing mount points with current mount info
	tlog.TraceContext(ctx, "Synchronizing mount points for partition", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "mount_data_count", len(mountData), "procfs_mounts_count", len(mountInfos))
	for _, prtstate := range mountInfos {
		iw := osutil.IsWritable(prtstate.MountPoint)
		if mountPoint, ok := self.disks.GetMountPoint(*e.Partition.DiskId, *e.Partition.Id, prtstate.MountPoint); ok {
			tlog.TraceContext(ctx, "Found existing mount point in cache for partition, updating state", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "mount_path", mountPoint.Path, "is_mounted", mountPoint.IsMounted)
			oldstate := mountPoint.IsMounted
			mountPoint.IsMounted = true
			mountPoint.Root = prtstate.Root
			mountPoint.RefreshVersion = self.refreshVersion
			mountPoint.IsWriteSupported = new(iw)
			mountPoint.Flags.Scan(prtstate.Options)
			mountPoint.CustomFlags.Scan(prtstate.SuperOptions)
			mountPoint.FSType = new(prtstate.FSType)
			mountPoint.Type = "ADDON"
			err := self.disks.AddOrUpdateMountPoint(*e.Partition.DiskId, *e.Partition.Id, *mountPoint)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to add mount point to disk map", "disk_id", *e.Partition.DiskId, "partition_id", *e.Partition.Id, "mount_path", mountPoint.Path, "err", err)
				continue
			}
			if !oldstate {
				self.eventBus.EmitMountPoint(events.MountPointEvent{
					Event:      events.Event{Type: events.EventTypes.UPDATE},
					MountPoint: mountPoint,
				})
			}
			continue
		} else if prtstate.Source == *e.Partition.DevicePath || (e.Partition.LegacyDevicePath != nil && prtstate.Source == *e.Partition.LegacyDevicePath) {
			// Found matching mount info for partition

			mountPoint := dto.MountPointData{
				Path:     prtstate.MountPoint,
				Root:     prtstate.Root,
				DeviceId: *e.Partition.Id,
				//PathHash:         xhashes.SHA1(prtstate.MountPoint),
				IsWriteSupported: new(iw),
				IsMounted:        true,
				Flags:            &dto.MountFlags{},
				CustomFlags:      &dto.MountFlags{},
				FSType:           new(prtstate.FSType),
				Type:             "ADDON",
				Partition:        e.Partition,
				RefreshVersion:   self.refreshVersion,
			}
			mountPoint.Flags.Scan(prtstate.Options)
			mountPoint.CustomFlags.Scan(prtstate.SuperOptions)
			err := self.disks.AddOrUpdateMountPoint(*e.Partition.DiskId, *e.Partition.Id, mountPoint)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to add mount point to disk map", "disk_id", *e.Partition.DiskId, "partition_id", *e.Partition.Id, "mount_path", mountPoint.Path, "err", err)
				continue
			}
			self.eventBus.EmitMountPoint(events.MountPointEvent{
				Event:      events.Event{Type: events.EventTypes.ADD},
				MountPoint: &mountPoint,
			})
			continue
		}
	}

	tlog.TraceContext(ctx, "Marking stale mount points as unmounted for partition", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id)
	for _, mountPoint := range self.disks.GetAllMountPoints() {
		if mountPoint.RefreshVersion != self.refreshVersion && (mountPoint.IsMounted || (mountPoint.IsToMountAtStartup != nil && *mountPoint.IsToMountAtStartup)) {
			tlog.TraceContext(ctx, "Marking mount point as unmounted since not found in procfs mounts", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "mount_path", mountPoint.Path)
			mountPoint.IsMounted = false
			mountPoint.RefreshVersion = self.refreshVersion
			err := self.disks.AddOrUpdateMountPoint(*e.Partition.DiskId, *e.Partition.Id, *mountPoint)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to add mount point to disk map", "disk_id", *e.Partition.DiskId, "partition_id", *e.Partition.Id, "mount_path", mountPoint.Path, "err", err)
				continue
			}
			self.eventBus.EmitMountPoint(events.MountPointEvent{
				Event:      events.Event{Type: events.EventTypes.UPDATE},
				MountPoint: mountPoint,
			})
		}
	}
	tlog.TraceContext(ctx, "Done synchronizing mount points for partition", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id)

	return nil
}

func (self *VolumeService) handleMountPointEvent(ctx context.Context, e events.MountPointEvent) errors.E {
	if e.MountPoint.Type == "" {
		e.MountPoint.Type = inferMountPointType(e.MountPoint)
		slog.WarnContext(ctx, "Mount point type missing, defaulting", "mount_point", e.MountPoint.Path, "type", e.MountPoint.Type)
	}
	tlog.TraceContext(ctx, "Processing mount point event for persistence", "mount_point", e.MountPoint.Path, "device_id", e.MountPoint.DeviceId, "event_type", e.Type)
	err := self.persistMountPoint(e.MountPoint)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to persist mount point on event", "mount_point", e.MountPoint, "err", err)
		return err
	}
	if (e.Type == events.EventTypes.ADD || e.Type == events.EventTypes.UPDATE) && !e.MountPoint.IsMounted && e.MountPoint.IsToMountAtStartup != nil && *e.MountPoint.IsToMountAtStartup {
		slog.InfoContext(ctx, "New mount point added and not mounted, attempting to mount", "mount_point", e.MountPoint.Path, "device_id", e.MountPoint.DeviceId)
		err = self.MountVolume(e.MountPoint)
		if err != nil {
			if errors.Is(err, dto.ErrorAlreadyMounted) {
				slog.InfoContext(ctx, "Mount point already mounted during automount attempt", "mount_point", e.MountPoint.Path, "device_id", e.MountPoint.DeviceId)
				return nil
			}
			slog.ErrorContext(ctx, "Failed to mount volume on event", "mount_point", e.MountPoint, "err", err)
			self.createAutomountFailureNotification(e.MountPoint.Path, e.MountPoint.DeviceId, err)
		}
	}
	return nil
}

func inferMountPointType(mountPoint *dto.MountPointData) string {
	if mountPoint == nil {
		return "ADDON"
	}
	path := mountPoint.Root
	if path == "" {
		path = mountPoint.Path
	}
	if path == "" {
		return "ADDON"
	}
	if path == "/mnt" || strings.HasPrefix(path, "/mnt/") {
		return "ADDON"
	}
	return "HOST"
}

func (self *VolumeService) CreateBlockDevice(device string) error {
	// Controlla se il dispositivo esiste già
	if _, err := os.Stat(device); !os.IsNotExist(err) {
		slog.WarnContext(self.ctx, "Loop device already exists", "device", device)
		return nil
	}

	// Estrai i numeri major e minor dal nome del dispositivo
	major, minor, err := self.extractMajorMinor(device)
	if err != nil {
		return errors.Errorf("errore durante l'estrazione dei numeri major e minor: %v", err)
	}

	// Crea il dispositivo di blocco usando la syscall mknod
	dev := (major << 8) | minor
	err = syscall.Mknod(device, syscall.S_IFBLK|0660, dev)
	if err != nil {
		return errors.Errorf("errore durante la creazione del dispositivo di blocco: %v", err)
	}

	return nil
}

func (self *VolumeService) extractMajorMinor(device string) (int, int, error) {
	re := regexp.MustCompile(`/dev/loop(\d+)`)
	matches := re.FindStringSubmatch(device)
	if len(matches) != 2 {
		return 0, 0, fmt.Errorf("formato del dispositivo non valido")
	}

	minor, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, fmt.Errorf("errore durante la conversione del numero minor: %v", err)
	}

	// Il numero major per i dispositivi di loop è generalmente 7
	major := 7

	return major, minor, nil
}

func (ms *VolumeService) PatchMountPointSettings(root string, path string, patchData dto.MountPointData) (*dto.MountPointData, errors.E) {

	dbMountData, err := gorm.G[dbom.MountPointPath](ms.db).
		Where(g.MountPointPath.Root.Eq(root), g.MountPointPath.Path.Eq(path)).First(ms.ctx)
	if err != nil {
		return nil, errors.Wrapf(dto.ErrorNotFound, "mount configuration with root %s and path %s not found", root, path)
	}

	err = ms.convDto.MountPointDataToMountPointPath(patchData, &dbMountData)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	affected, err := gorm.G[*dbom.MountPointPath](ms.db).
		Where(g.MountPointPath.Root.Eq(root), g.MountPointPath.Path.Eq(path)).
		Updates(ms.ctx, &dbMountData)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(dto.ErrorNotFound, "mount configuration with root %s and path %s not found", root, path)
		}
		return nil, errors.WithStack(err)
	}
	if affected == 0 {
		return nil, errors.Wrapf(dto.ErrorNotFound, "mount configuration with root %s and path %s not found", root, path)
	}

	currentDto := dto.MountPointData{}
	if convErr := ms.convDto.MountPointPathToMountPointData(dbMountData, &currentDto, ms.GetVolumesData()); convErr != nil {
		return nil, errors.WithStack(convErr)
	}
	// Update cached mount point data
	if currentDto.Partition != nil && currentDto.Partition.DiskId != nil && currentDto.Partition.Id != nil {
		err := ms.disks.AddOrUpdateMountPoint(*currentDto.Partition.DiskId, *currentDto.Partition.Id, currentDto)
		if err != nil {
			slog.WarnContext(ms.ctx, "Failed to update mount point in cache", "err", err)
		}
	} else {
		// Fallback: partition could not be resolved.
		updated := false
		if existing, ok := ms.disks.GetMountPointByPath(path); ok {
			if existing.Partition != nil && existing.Partition.DiskId != nil && existing.Partition.Id != nil {
				existing.IsToMountAtStartup = currentDto.IsToMountAtStartup
				err := ms.disks.AddOrUpdateMountPoint(*existing.Partition.DiskId, *existing.Partition.Id, *existing)
				if err != nil {
					slog.WarnContext(ms.ctx, "Failed to update mount point in fallback cache update", "err", err)
				}
				updated = true
			}
		}
		if !updated {
			for dk, d := range *ms.disks {
				if d.Partitions == nil {
					continue
				}
				for pid, part := range *d.Partitions {
					if part.MountPointData == nil {
						continue
					}
					if existing, ok := (*part.MountPointData)[path]; ok {
						existing.IsToMountAtStartup = currentDto.IsToMountAtStartup
						err := ms.disks.AddOrUpdateMountPoint(dk, pid, existing)
						if err != nil {
							slog.WarnContext(ms.ctx, "Failed to update mount point in fallback cache update", "err", err)
						}
						updated = true
						break
					}
				}
				if updated {
					break
				}
			}
		}
	}
	ms.eventBus.EmitMountPoint(events.MountPointEvent{
		Event:      events.Event{Type: events.EventTypes.UPDATE},
		MountPoint: &currentDto,
	})
	return &currentDto, nil
}

// createAutomountFailureNotification creates a persistent notification for failed automount operations
func (self *VolumeService) createAutomountFailureNotification(mountPath, device string, err errors.E) {
	if self.haService == nil {
		slog.DebugContext(self.ctx, "Home Assistant service not available, skipping automount failure notification")
		return
	}

	notificationID := fmt.Sprintf("srat_automount_failure_%s", xhashes.SHA1(mountPath))
	title := "Automount Failed"

	var message string
	if errors.Is(err, dto.ErrorDeviceNotFound) {
		message = fmt.Sprintf("Device '%s' for mount point '%s' not found during startup. The device may have been removed or disconnected.", device, mountPath)
	} else if errors.Is(err, dto.ErrorMountFail) {
		message = fmt.Sprintf("Failed to mount device '%s' to '%s' during startup. Check device filesystem and permissions.", device, mountPath)
	} else {
		message = fmt.Sprintf("Automount failed for device '%s' to '%s': %s", device, mountPath, err.Error())
	}

	notifyErr := self.haService.CreatePersistentNotification(notificationID, title, message)
	if notifyErr != nil {
		slog.ErrorContext(self.ctx, "Failed to create automount failure notification", "mount_path", mountPath, "device", device, "err", notifyErr)
	} else {
		slog.InfoContext(self.ctx, "Created automount failure notification", "mount_path", mountPath, "device", device, "notification_id", notificationID)
	}
}

// DismissAutomountNotification dismisses an automount-related notification
func (self *VolumeService) dismissAutomountNotification(deviceId string, notificationType string) {
	if self.haService == nil {
		return
	}

	notificationID := fmt.Sprintf("srat_%s_%s", notificationType, xhashes.SHA1(deviceId))

	notifyErr := self.haService.DismissPersistentNotification(notificationID)
	if notifyErr != nil {
		slog.WarnContext(self.ctx, "Failed to dismiss automount notification", "mount_path", deviceId, "notification_type", notificationType, "err", notifyErr)
	} else {
		slog.DebugContext(self.ctx, "Dismissed automount notification", "mount_path", deviceId, "notification_type", notificationType, "notification_id", notificationID)
	}
}

func (ms *VolumeService) MockSetProcfsGetMounts(f func() ([]*procfs.MountInfo, error)) {
	ms.procfsGetMounts = f
}

// MockSetMountOps allows tests to override mount operations.
func (ms *VolumeService) MockSetMountOps(
	tryMount func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
	mountFn func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
	unmountFn func(target string, force, lazy bool) error,
) {
	if tryMount != nil {
		ms.tryMountFunc = tryMount
	}
	if mountFn != nil {
		ms.doMountFunc = mountFn
	}
	if unmountFn != nil {
		ms.unmountFunc = unmountFn
	}
}

func (ms *VolumeService) GetDevicePathByDeviceID(deviceID string) (string, errors.E) {
	md, ok := ms.disks.Get(deviceID)
	if !ok {
		return "", errors.WithDetails(dto.ErrorNotFound, "Message", "mount point not found", "DeviceId", deviceID)
	}
	return *md.DevicePath, nil
}
