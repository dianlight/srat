package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/core_api"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

type HomeAssistantServiceInterface interface {
	SendDiskEntities(disks *[]dto.Disk) error
	SendSambaStatusEntity(status *dto.SambaStatus) error
	SendSambaProcessStatusEntity(status *dto.SambaProcessStatus) error
	SendVolumeStatusEntity(data *[]dto.Disk) error
}

type HomeAssistantService struct {
	ctx        context.Context
	config     *dto.ContextState
	coreClient core_api.ClientWithResponsesInterface
}

type HomeAssistantServiceParams struct {
	fx.In
	Ctx        context.Context
	Config     *dto.ContextState
	CoreClient core_api.ClientWithResponsesInterface `optional:"true"`
}

func NewHomeAssistantService(params HomeAssistantServiceParams) HomeAssistantServiceInterface {
	return &HomeAssistantService{
		ctx:        params.Ctx,
		config:     params.Config,
		coreClient: params.CoreClient,
	}
}

func (s *HomeAssistantService) SendDiskEntities(disks *[]dto.Disk) error {
	if s.coreClient == nil || disks == nil {
		slog.Debug("Skipping sending disk entities to Home Assistant", "core_client", s.coreClient != nil, "disks", disks != nil)
		return nil
	}

	for _, disk := range *disks {
		if err := s.sendDiskEntity(disk); err != nil {
			slog.Warn("Failed to send disk entity to Home Assistant", "disk", disk.Id, "error", err)
		}

		if disk.Partitions != nil {
			for _, partition := range *disk.Partitions {
				if err := s.sendPartitionEntity(partition, disk); err != nil {
					slog.Warn("Failed to send partition entity to Home Assistant", "partition", partition.Id, "error", err)
				}
			}
		}
	}

	return nil
}

func (s *HomeAssistantService) SendSambaStatusEntity(status *dto.SambaStatus) error {
	if s.coreClient == nil || status == nil {
		slog.Debug("Skipping sending Samba status entity to Home Assistant", "core_client", s.coreClient != nil, "status", status != nil)
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

	slog.Debug("Sent Samba status entity to Home Assistant", "entity_id", entityId, "state", state, "response", string(resp.Body))
	return nil
}

func (s *HomeAssistantService) SendSambaProcessStatusEntity(status *dto.SambaProcessStatus) error {
	if s.coreClient == nil || status == nil {
		slog.Debug("Skipping sending Samba process status entity to Home Assistant", "core_client", s.coreClient != nil, "status", status != nil)
		return nil
	}

	entityId := "sensor.srat_samba_process_status"

	// Prepare attributes
	attributes := map[string]interface{}{
		"icon":          "mdi:cog",
		"friendly_name": "SRAT Samba Process Status",
		"device_class":  "running",
		"smbd_running":  status.Smbd.IsRunning,
		"nmbd_running":  status.Nmbd.IsRunning,
		"wsdd2_running": status.Wsdd2.IsRunning,
		"avahi_running": status.Avahi.IsRunning,
	}

	// Add detailed process information
	if status.Smbd.IsRunning {
		attributes["smbd_pid"] = status.Smbd.Pid
		attributes["smbd_cpu_percent"] = status.Smbd.CPUPercent
		attributes["smbd_memory_percent"] = status.Smbd.MemoryPercent
	}
	if status.Nmbd.IsRunning {
		attributes["nmbd_pid"] = status.Nmbd.Pid
		attributes["nmbd_cpu_percent"] = status.Nmbd.CPUPercent
		attributes["nmbd_memory_percent"] = status.Nmbd.MemoryPercent
	}

	// Determine overall state
	state := "stopped"
	runningCount := 0
	if status.Smbd.IsRunning {
		runningCount++
	}
	if status.Nmbd.IsRunning {
		runningCount++
	}
	if status.Wsdd2.IsRunning {
		runningCount++
	}
	if status.Avahi.IsRunning {
		runningCount++
	}

	if runningCount >= 2 {
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
		return errors.Errorf("failed to send samba process status entity: HTTP %d", resp.StatusCode())
	}

	slog.Debug("Sent Samba process status entity to Home Assistant", "entity_id", entityId, "state", state)
	return nil
}

func (s *HomeAssistantService) SendVolumeStatusEntity(data *[]dto.Disk) error {
	if s.coreClient == nil || data == nil {
		slog.Debug("Skipping sending volume status entity to Home Assistant", "core_client", s.coreClient != nil, "data", data != nil)
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
						for _, share := range mp.Shares {
							if share.Disabled != nil && !*share.Disabled {
								sharedPartitions++
							}
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
		return errors.Errorf("failed to send volume status entity: HTTP %d", resp.StatusCode())
	}

	slog.Debug("Sent Volume status entity to Home Assistant", "entity_id", entityId, "total_disks", totalDisks)
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

	if disk.Device != nil {
		attributes["device"] = *disk.Device
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
		return errors.Errorf("failed to send disk entity: HTTP %d", resp.StatusCode())
	}

	slog.Debug("Sent disk entity to Home Assistant", "entity_id", entityId, "disk_id", *disk.Id)
	return nil
}

func (s *HomeAssistantService) sendPartitionEntity(partition dto.Partition, disk dto.Disk) error {
	if partition.Id == nil {
		return errors.New("partition ID is required")
	}

	entityId := fmt.Sprintf("sensor.srat_partition_%s", sanitizeEntityId(*partition.Id))

	diskName := getStringOrDefault(disk.Id, "unknown")
	partitionName := getStringOrDefault(partition.Name, getStringOrDefault(partition.Device, "unknown"))

	attributes := map[string]interface{}{
		"icon":          "mdi:folder",
		"friendly_name": fmt.Sprintf("SRAT Partition %s", partitionName),
		"disk_id":       diskName,
	}

	if partition.Device != nil {
		attributes["device"] = *partition.Device
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
			for _, share := range mp.Shares {
				if share.Disabled != nil && !*share.Disabled {
					shareCount++
				}
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
		return errors.Errorf("failed to send partition entity: HTTP %d", resp.StatusCode())
	}

	slog.Debug("Sent partition entity to Home Assistant", "entity_id", entityId, "partition_id", *partition.Id, "state", state)
	return nil
}

// Helper functions

func sanitizeEntityId(id string) string {
	// Replace special characters with underscores for valid entity IDs
	sanitized := strings.ReplaceAll(id, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	sanitized = strings.ToLower(sanitized)
	return sanitized
}

func getStringOrDefault(ptr *string, defaultValue string) string {
	if ptr != nil {
		return *ptr
	}
	return defaultValue
}
