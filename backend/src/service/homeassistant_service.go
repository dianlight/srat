package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/core_api"
	"github.com/dianlight/tlog"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

type HomeAssistantServiceInterface interface {
	SendDiskEntities(disks *[]*dto.Disk) error
	SendSambaStatusEntity(status *dto.SambaStatus) error
	SendSambaProcessStatusEntity(status *dto.ServerProcessStatus) error
	SendVolumeStatusEntity(data *[]*dto.Disk) error
	SendDiskHealthEntities(diskHealth *dto.DiskHealth) error
	CreatePersistentNotification(notificationID, title, message string) error
	DismissPersistentNotification(notificationID string) error
}

type HomeAssistantService struct {
	ctx            context.Context
	state          *dto.ContextState
	coreClient     core_api.ClientWithResponsesInterface
	settingService SettingServiceInterface
	//propRepo                repository.PropertyRepositoryInterface
	notificationTracker     map[string]string // Maps notificationID to last sent date
	notificationTrackerLock sync.RWMutex
}

type HomeAssistantServiceParams struct {
	fx.In
	Ctx        context.Context
	State      *dto.ContextState
	CoreClient core_api.ClientWithResponsesInterface `optional:"true"`
	//PropRepo   repository.PropertyRepositoryInterface
	SettingService SettingServiceInterface
}

func NewHomeAssistantService(params HomeAssistantServiceParams) HomeAssistantServiceInterface {
	return &HomeAssistantService{
		ctx:            params.Ctx,
		state:          params.State,
		coreClient:     params.CoreClient,
		settingService: params.SettingService,
		//propRepo:            params.PropRepo,
		notificationTracker: make(map[string]string),
	}
}

func (s *HomeAssistantService) SendDiskEntities(disks *[]*dto.Disk) error {
	setting, _ := s.settingService.Load()
	if setting == nil || setting.ExportStatsToHA == nil || !*setting.ExportStatsToHA {
		return nil
	}

	if s.coreClient == nil || disks == nil || s.state.HACoreReady == false {
		slog.DebugContext(s.ctx, "Skipping sending disk entities to Home Assistant", "core_client", s.coreClient != nil, "disks", disks != nil)
		return nil
	}

	for _, disk := range *disks {
		if err := s.sendDiskEntity(*disk); err != nil {
			slog.WarnContext(s.ctx, "Failed to send disk entity to Home Assistant", "disk", disk.Id, "error", err)
		}

		if disk.Partitions != nil {
			for _, partition := range *disk.Partitions {
				if err := s.sendPartitionEntity(partition, *disk); err != nil {
					slog.WarnContext(s.ctx, "Failed to send partition entity to Home Assistant", "partition", partition.Id, "error", err)
				}
			}
		}
	}

	return nil
}

func (s *HomeAssistantService) SendSambaStatusEntity(status *dto.SambaStatus) error {
	setting, _ := s.settingService.Load()
	if setting == nil || setting.ExportStatsToHA == nil || !*setting.ExportStatsToHA {
		return nil
	}

	if s.coreClient == nil || status == nil || s.state.HACoreReady == false {
		slog.DebugContext(s.ctx, "Skipping sending Samba status entity to Home Assistant", "core_client", s.coreClient != nil, "status", status != nil)
		return nil
	}

	entityId := "sensor.srat_samba_status"

	// Prepare attributes
	attributes := map[string]interface{}{
		"icon":          "mdi:folder-network",
		"friendly_name": "SRAT Samba Status",
		"device_class":  "connectivity",
		"version":       status.Version,
		"smb_conf":      status.SmbConf,
		"timestamp":     status.Timestamp.Time.Format("2006-01-02T15:04:05Z07:00"),
		"session_count": len(status.Sessions),
		"tcon_count":    len(status.Tcons),
	}

	state := "connected"
	if len(status.Sessions) == 0 {
		state = "idle"
	}

	entity := core_api.EntityState{
		EntityId:   &entityId,
		State:      &state,
		Attributes: &attributes,
	}

	resp, err := s.coreClient.PostEntityStateWithResponse(s.ctx, entityId, entity)
	if err != nil {
		return errors.Wrap(err, "failed to send samba status entity")
	}

	if resp.StatusCode() >= 400 {
		return errors.Errorf("failed to send samba status entity: HTTP %d", resp.StatusCode())
	} else {
	}

	tlog.TraceContext(s.ctx, "Sent Samba status entity to Home Assistant", "entity_id", entityId, "state", state, "response", string(resp.Body))
	return nil
}

func (s *HomeAssistantService) SendSambaProcessStatusEntity(status *dto.ServerProcessStatus) error {
	setting, _ := s.settingService.Load()
	if setting == nil || setting.ExportStatsToHA == nil || !*setting.ExportStatsToHA {
		return nil
	}

	if s.coreClient == nil || status == nil || s.state.HACoreReady == false {
		slog.DebugContext(s.ctx, "Skipping sending Samba process status entity to Home Assistant", "core_client", s.coreClient != nil, "status", status != nil)
		return nil
	}

	entityId := "sensor.srat_samba_process_status"

	// Prepare attributes
	attributes := map[string]interface{}{
		"icon":          "mdi:cog",
		"friendly_name": "SRAT Samba Process Status",
		"device_class":  "running",
	}

	// Dynamically add process information
	runningCount := 0
	for processName, processStatus := range *status {
		// Add running status for each process
		attributes[processName+"_running"] = processStatus.IsRunning

		// Add detailed process information if running
		if processStatus.IsRunning {
			attributes[processName+"_pid"] = processStatus.Pid
			attributes[processName+"_cpu_percent"] = processStatus.CPUPercent
			attributes[processName+"_memory_percent"] = processStatus.MemoryPercent
			runningCount++
		}
	}

	// Determine overall state
	state := "stopped"

	if runningCount >= len(*status) {
		state = "running"
	} else if runningCount > 0 {
		state = "partial"
	}

	entity := core_api.EntityState{
		EntityId:   &entityId,
		State:      &state,
		Attributes: &attributes,
	}

	resp, err := s.coreClient.PostEntityStateWithResponse(s.ctx, entityId, entity)
	if err != nil {
		return errors.Wrap(err, "failed to send samba process status entity")
	}

	if resp.StatusCode() >= 400 {
		return errors.Errorf("failed to send Samba process status entity to Home Assistant: %s", string(resp.Body))
	}

	tlog.TraceContext(s.ctx, "Sent Samba process status entity to Home Assistant", "entity_id", entityId, "state", state)
	return nil
}

func (s *HomeAssistantService) SendVolumeStatusEntity(data *[]*dto.Disk) error {
	setting, _ := s.settingService.Load()
	if setting == nil || setting.ExportStatsToHA == nil || !*setting.ExportStatsToHA {
		return nil
	}

	if s.coreClient == nil || data == nil || s.state.HACoreReady == false {
		slog.DebugContext(s.ctx, "Skipping sending volume status entity to Home Assistant", "core_client", s.coreClient != nil, "data", data != nil)
		return nil
	}

	entityId := "sensor.srat_volume_status"

	totalDisks := len(*data)
	totalPartitions := 0
	mountedPartitions := 0
	sharedPartitions := 0

	for _, disk := range *data {
		if disk.Partitions != nil {
			totalPartitions += len(*disk.Partitions)
			for _, partition := range *disk.Partitions {
				if partition.MountPointData != nil {
					for _, mp := range *partition.MountPointData {
						if mp.IsMounted {
							mountedPartitions++
						}
						if mp.Share != nil && mp.Share.Disabled != nil && !*mp.Share.Disabled {
							sharedPartitions++
						}
					}
				}
			}
		}
	}

	attributes := map[string]interface{}{
		"icon":               "mdi:harddisk",
		"friendly_name":      "SRAT Volume Status",
		"total_disks":        totalDisks,
		"total_partitions":   totalPartitions,
		"mounted_partitions": mountedPartitions,
		"shared_partitions":  sharedPartitions,
	}

	state := fmt.Sprintf("%d", totalDisks)

	entity := core_api.EntityState{
		EntityId:   &entityId,
		State:      &state,
		Attributes: &attributes,
	}

	resp, err := s.coreClient.PostEntityStateWithResponse(s.ctx, entityId, entity)
	if err != nil {
		return errors.Wrap(err, "failed to send volume status entity")
	}

	if resp.StatusCode() >= 400 {
		return errors.Errorf("failed to send volume status entity to Home Assistant: %s", string(resp.Body))
	}

	tlog.TraceContext(s.ctx, "Sent Volume status entity to Home Assistant", "entity_id", entityId, "total_disks", totalDisks)
	return nil
}

func (s *HomeAssistantService) sendDiskEntity(disk dto.Disk) error {
	if disk.Id == nil {
		return errors.New("disk ID is required")
	}

	entityId := fmt.Sprintf("sensor.srat_disk_%s", sanitizeEntityId(*disk.Id))

	attributes := map[string]interface{}{
		"icon":          "mdi:harddisk",
		"friendly_name": fmt.Sprintf("SRAT Disk %s", getStringOrDefault(disk.Id, "unknown")),
	}

	if disk.LegacyDeviceName != nil {
		attributes["device"] = *disk.LegacyDeviceName
	}
	if disk.Model != nil {
		attributes["model"] = *disk.Model
	}
	if disk.Vendor != nil {
		attributes["vendor"] = *disk.Vendor
	}
	if disk.Serial != nil {
		attributes["serial"] = *disk.Serial
	}
	if disk.Size != nil {
		attributes["size_bytes"] = *disk.Size
		attributes["size_gb"] = fmt.Sprintf("%.2f", float64(*disk.Size)/1024/1024/1024)
	}
	if disk.ConnectionBus != nil {
		attributes["connection_bus"] = *disk.ConnectionBus
	}
	if disk.Removable != nil {
		attributes["removable"] = *disk.Removable
	}
	if disk.Ejectable != nil {
		attributes["ejectable"] = *disk.Ejectable
	}

	state := "connected"
	if disk.Partitions != nil {
		attributes["partition_count"] = len(*disk.Partitions)
	}

	entity := core_api.EntityState{
		EntityId:   &entityId,
		State:      &state,
		Attributes: &attributes,
	}

	resp, err := s.coreClient.PostEntityStateWithResponse(s.ctx, entityId, entity)
	if err != nil {
		return errors.Wrap(err, "failed to send disk entity")
	}

	if resp.StatusCode() >= 400 {
		return errors.Errorf("failed to send disk entity to Home Assistant: %s", string(resp.Body))
	}

	tlog.TraceContext(s.ctx, "Sent disk entity to Home Assistant", "entity_id", entityId, "disk_id", *disk.Id)
	return nil
}

func (s *HomeAssistantService) sendPartitionEntity(partition dto.Partition, disk dto.Disk) error {
	if partition.Id == nil {
		return errors.New("partition ID is required")
	}

	entityId := fmt.Sprintf("sensor.srat_partition_%s", sanitizeEntityId(*partition.Id))

	diskName := getStringOrDefault(disk.Id, "unknown")
	partitionName := getStringOrDefault(partition.Name, getStringOrDefault(partition.Id, "unknown"))

	attributes := map[string]interface{}{
		"icon":          "mdi:folder",
		"friendly_name": fmt.Sprintf("SRAT Partition %s", partitionName),
		"disk_id":       diskName,
	}

	if partition.LegacyDeviceName != nil {
		attributes["device"] = *partition.LegacyDeviceName
	}
	if partition.Name != nil {
		attributes["name"] = *partition.Name
	}
	if partition.Size != nil {
		attributes["size_bytes"] = *partition.Size
		attributes["size_gb"] = fmt.Sprintf("%.2f", float64(*partition.Size)/1024/1024/1024)
	}
	if partition.System != nil {
		attributes["system"] = *partition.System
	}

	state := "unmounted"
	mountedCount := 0
	shareCount := 0

	if partition.MountPointData != nil {
		for _, mp := range *partition.MountPointData {
			if mp.IsMounted {
				mountedCount++
				attributes["mount_path"] = mp.Path
			}
			if mp.Share != nil && mp.Share.Disabled != nil && !*mp.Share.Disabled {
				shareCount++
			}
		}
	}

	if mountedCount > 0 {
		if shareCount > 0 {
			state = "shared"
		} else {
			state = "mounted"
		}
	}

	attributes["mounted_count"] = mountedCount
	attributes["share_count"] = shareCount

	entity := core_api.EntityState{
		EntityId:   &entityId,
		State:      &state,
		Attributes: &attributes,
	}

	resp, err := s.coreClient.PostEntityStateWithResponse(s.ctx, entityId, entity)
	if err != nil {
		return errors.Wrap(err, "failed to send partition entity")
	}

	if resp.StatusCode() >= 400 {
		return errors.Errorf("failed to send partition entity to Home Assistant: %s", string(resp.Body))
	}

	tlog.TraceContext(s.ctx, "Sent partition entity to Home Assistant", "entity_id", entityId, "partition_id", *partition.Id, "state", state)
	return nil
}

func (s *HomeAssistantService) SendDiskHealthEntities(diskHealth *dto.DiskHealth) error {
	setting, _ := s.settingService.Load()
	if setting == nil || setting.ExportStatsToHA == nil || !*setting.ExportStatsToHA {
		return nil
	}

	if s.coreClient == nil || diskHealth == nil || s.state.HACoreReady == false {
		return nil
	}

	// Send global disk health entity
	if err := s.sendGlobalDiskHealthEntity(diskHealth); err != nil {
		slog.WarnContext(s.ctx, "Failed to send global disk health entity to Home Assistant", "error", err)
	}

	// Send individual disk I/O entities
	if diskHealth.PerDiskIO != nil {
		for _, diskIO := range diskHealth.PerDiskIO {
			if err := s.sendDiskIOEntity(diskIO); err != nil {
				slog.WarnContext(s.ctx, "Failed to send disk I/O entity to Home Assistant", "device", diskIO.DeviceName, "error", err)
			}
		}
	}

	// Send partition health entities
	for diskName, partitions := range diskHealth.PerPartitionInfo {
		for _, partition := range partitions {
			if err := s.sendPartitionHealthEntity(diskName, partition); err != nil {
				slog.WarnContext(s.ctx, "Failed to send partition health entity to Home Assistant", "device", partition.Device, "error", err)
			}
		}
	}

	return nil
}

func (s *HomeAssistantService) sendGlobalDiskHealthEntity(diskHealth *dto.DiskHealth) error {
	entityId := "sensor.srat_global_disk_health"

	attributes := map[string]interface{}{
		"icon":                "mdi:harddisk-plus",
		"friendly_name":       "SRAT Global Disk Health",
		"device_class":        "frequency",
		"unit_of_measurement": "IOPS",
		"total_iops":          diskHealth.Global.TotalIOPS,
		"read_latency_ms":     diskHealth.Global.TotalReadLatency,
		"write_latency_ms":    diskHealth.Global.TotalWriteLatency,
	}

	// Determine state based on IOPS
	state := fmt.Sprintf("%.2f", diskHealth.Global.TotalIOPS)

	entity := core_api.EntityState{
		EntityId:   &entityId,
		State:      &state,
		Attributes: &attributes,
	}

	resp, err := s.coreClient.PostEntityStateWithResponse(s.ctx, entityId, entity)
	if err != nil {
		return errors.Wrap(err, "failed to send global disk health entity")
	}

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return errors.Errorf("failed to send global disk health entity: HTTP %d payload %#v response %s", resp.StatusCode(), entity, string(resp.Body))
	}

	tlog.TraceContext(s.ctx, "Sent global disk health entity to Home Assistant", "entity_id", entityId, "iops", diskHealth.Global.TotalIOPS)
	return nil
}

func (s *HomeAssistantService) sendDiskIOEntity(diskIO dto.DiskIOStats) error {
	entityId := fmt.Sprintf("sensor.srat_disk_io_%s", sanitizeEntityId(diskIO.DeviceName))

	attributes := map[string]interface{}{
		"icon":                "mdi:chart-line",
		"friendly_name":       fmt.Sprintf("SRAT Disk I/O %s", diskIO.DeviceName),
		"device_class":        "frequency",
		"unit_of_measurement": "IOPS",
		"device_name":         diskIO.DeviceName,
		"read_iops":           diskIO.ReadIOPS,
		"write_iops":          diskIO.WriteIOPS,
		"read_latency_ms":     diskIO.ReadLatency,
		"write_latency_ms":    diskIO.WriteLatency,
	}

	if diskIO.DeviceDescription != "" {
		attributes["device_description"] = diskIO.DeviceDescription
	}

	// Add SMART data if available
	if diskIO.SmartData != nil {
		attributes["smart_temperature"] = diskIO.SmartData.Temperature.Value
		attributes["smart_power_on_hours"] = diskIO.SmartData.PowerOnHours.Value
		attributes["smart_power_cycle_count"] = diskIO.SmartData.PowerCycleCount.Value
	}

	// Calculate total IOPS as state
	totalIOPS := diskIO.ReadIOPS + diskIO.WriteIOPS
	state := fmt.Sprintf("%.2f", totalIOPS)

	entity := core_api.EntityState{
		EntityId:   &entityId,
		State:      &state,
		Attributes: &attributes,
	}

	resp, err := s.coreClient.PostEntityStateWithResponse(s.ctx, entityId, entity)
	if err != nil {
		return errors.Wrap(err, "failed to send disk I/O entity")
	}

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return errors.Errorf("failed to send disk I/O entity: HTTP %d payload %#v response %s", resp.StatusCode(), entity, string(resp.Body))
	}

	tlog.TraceContext(s.ctx, "Sent disk I/O entity to Home Assistant", "entity_id", entityId, "device", diskIO.DeviceName, "iops", totalIOPS)
	return nil
}

func (s *HomeAssistantService) sendPartitionHealthEntity(diskName string, partition dto.PerPartitionInfo) error {
	// Create a more specific entity ID combining disk and partition info
	deviceSanitized := sanitizeEntityId(partition.Device)
	entityId := fmt.Sprintf("sensor.srat_partition_health_%s", deviceSanitized)

	attributes := map[string]interface{}{
		"icon":                "mdi:folder-information",
		"friendly_name":       fmt.Sprintf("SRAT Partition Health %s", partition.Device),
		"device_class":        "data_size",
		"unit_of_measurement": "bytes",
		"device":              partition.Device,
		"mount_point":         partition.MountPoint,
		"fstype":              partition.FSType,
		"total_space_bytes":   partition.TotalSpace,
		"free_space_bytes":    partition.FreeSpace,
		"fsck_needed":         partition.FsckNeeded,
		"fsck_supported":      partition.FsckSupported,
		"disk_name":           diskName,
	}

	// Calculate usage percentage
	usagePercent := 0.0
	if partition.TotalSpace > 0 {
		usagePercent = float64(partition.TotalSpace-partition.FreeSpace) / float64(partition.TotalSpace) * 100
		attributes["usage_percent"] = fmt.Sprintf("%.2f", usagePercent)
	}

	// State represents free space in bytes
	state := fmt.Sprintf("%d", partition.FreeSpace)

	entity := core_api.EntityState{
		EntityId:   &entityId,
		State:      &state,
		Attributes: &attributes,
	}

	resp, err := s.coreClient.PostEntityStateWithResponse(s.ctx, entityId, entity)
	if err != nil {
		return errors.Wrap(err, "failed to send partition health entity")
	}

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return errors.Errorf("failed to send partition health entity: HTTP %d payload %+v response %s", resp.StatusCode(), spew.Sdump(entity), string(resp.Body))
	}

	tlog.TraceContext(s.ctx, "Sent partition health entity to Home Assistant", "entity_id", entityId, "device", partition.Device, "free_space", partition.FreeSpace)
	return nil
}

func (s *HomeAssistantService) CreatePersistentNotification(notificationID, title, message string) error {
	if s.coreClient == nil {
		slog.DebugContext(s.ctx, "Skipping persistent notification creation - no Home Assistant client available")
		return nil
	}

	// Check if notification was already sent today
	today := time.Now().Format("2006-01-02")

	s.notificationTrackerLock.RLock()
	lastSent, exists := s.notificationTracker[notificationID]
	s.notificationTrackerLock.RUnlock()

	if exists && lastSent == today {
		slog.DebugContext(s.ctx, "Skipping notification - already sent today", "notification_id", notificationID, "date", today)
		return nil
	}

	serviceData := core_api.ServiceData{
		NotificationId: &notificationID,
		Title:          &title,
		Message:        &message,
	}

	resp, err2 := s.coreClient.CallServiceWithResponse(s.ctx, "persistent_notification", "create", serviceData)
	if err2 != nil {
		return errors.Wrap(err2, "failed to call persistent_notification.create service")
	}

	if resp.HTTPResponse.StatusCode < 200 || resp.HTTPResponse.StatusCode >= 300 {
		return errors.Errorf("failed to create persistent notification: HTTP %d", resp.HTTPResponse.StatusCode)
	}

	// Track that notification was sent today
	s.notificationTrackerLock.Lock()
	s.notificationTracker[notificationID] = today
	s.notificationTrackerLock.Unlock()

	slog.DebugContext(s.ctx, "Created persistent notification in Home Assistant", "notification_id", notificationID, "title", title)
	return nil
}

func (s *HomeAssistantService) DismissPersistentNotification(notificationID string) error {
	if s.coreClient == nil {
		slog.DebugContext(s.ctx, "Skipping persistent notification dismissal - no Home Assistant client available")
		return nil
	}

	serviceData := core_api.ServiceData{
		NotificationId: &notificationID,
	}

	resp, err := s.coreClient.CallServiceWithResponse(s.ctx, "persistent_notification", "dismiss", serviceData)
	if err != nil {
		return errors.Wrap(err, "failed to call persistent_notification.dismiss service")
	}

	if resp.HTTPResponse.StatusCode < 200 || resp.HTTPResponse.StatusCode >= 300 {
		return errors.Errorf("failed to dismiss persistent notification: HTTP %d", resp.HTTPResponse.StatusCode)
	}

	// Clear tracking to allow recreation on the same day
	s.notificationTrackerLock.Lock()
	delete(s.notificationTracker, notificationID)
	s.notificationTrackerLock.Unlock()

	slog.DebugContext(s.ctx, "Dismissed persistent notification in Home Assistant", "notification_id", notificationID)
	return nil
}

// Helper functions

var nonAlphaNumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func sanitizeEntityId(id string) string {
	// Replace all non-alphanumeric characters with underscores for valid entity IDs
	sanitized := nonAlphaNumericRegex.ReplaceAllString(id, "_")
	sanitized = strings.ToLower(sanitized)
	return sanitized
}

func getStringOrDefault(ptr *string, defaultValue string) string {
	if ptr != nil {
		return *ptr
	}
	return defaultValue
}
