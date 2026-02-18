package api

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/tlog"
)

// FilesystemHandler handles filesystem-related API endpoints
type FilesystemHandler struct {
	fsService     service.FilesystemServiceInterface
	volumeService service.VolumeServiceInterface
}

// NewFilesystemHandler creates a new FilesystemHandler
func NewFilesystemHandler(
	fsService service.FilesystemServiceInterface,
	volumeService service.VolumeServiceInterface,
) *FilesystemHandler {
	return &FilesystemHandler{
		fsService:     fsService,
		volumeService: volumeService,
	}
}

// RegisterFilesystemHandler registers all filesystem-related endpoints
func (h *FilesystemHandler) RegisterFilesystemHandler(api huma.API) {
	huma.Get(api, "/filesystems", h.ListFilesystems, huma.OperationTags("filesystems"))
	huma.Post(api, "/filesystem/format", h.FormatPartition, huma.OperationTags("filesystems"))
	huma.Post(api, "/filesystem/check", h.CheckPartition, huma.OperationTags("filesystems"))
	huma.Post(api, "/filesystem/check/abort", h.AbortCheckPartition, huma.OperationTags("filesystems"))
	huma.Get(api, "/filesystem/state", h.GetPartitionState, huma.OperationTags("filesystems"))
	huma.Get(api, "/filesystem/label", h.GetPartitionLabel, huma.OperationTags("filesystems"))
	huma.Put(api, "/filesystem/label", h.SetPartitionLabel, huma.OperationTags("filesystems"))
}

// Helper function to get DiskMap from VolumeService
func (h *FilesystemHandler) getDiskMap() dto.DiskMap {
	volumes := h.volumeService.GetVolumesData()
	diskMap := make(dto.DiskMap)
	for _, disk := range volumes {
		if disk.Id != nil {
			diskMap[*disk.Id] = disk
		}
	}
	return diskMap
}

// ListFilesystems returns all supported filesystems with their capabilities
func (h *FilesystemHandler) ListFilesystems(
	ctx context.Context,
	input *struct{},
) (*struct{ Body []dto.FilesystemInfo }, error) {
	tlog.DebugContext(ctx, "Listing all filesystems with capabilities")

	// Get all supported filesystem types
	fsTypes := h.fsService.ListSupportedTypes()
	result := make([]dto.FilesystemInfo, 0, len(fsTypes))

	// Get capabilities for each filesystem using the new GetSupportAndInfo method
	for _, fsType := range fsTypes {
		info, err := h.fsService.GetSupportAndInfo(ctx, fsType)
		if err != nil {
			tlog.WarnContext(ctx, "Failed to get filesystem info", "filesystem", fsType, "error", err)
			continue
		}

		result = append(result, *info)
	}

	return &struct{ Body []dto.FilesystemInfo }{Body: result}, nil
}

// FormatPartitionInput contains the input for formatting a partition
type FormatPartitionInput struct {
	// PartitionID is the unique partition identifier
	PartitionID string `path:"partition_id" json:"partitionId" doc:"Unique partition identifier"`

	// FilesystemType is the type of filesystem to format (e.g., ext4, ntfs)
	FilesystemType string `json:"filesystemType" doc:"Filesystem type to format (ext4, vfat, ntfs, btrfs, xfs, etc.)"`

	// Label is the optional filesystem label
	Label string `json:"label,omitempty" doc:"Optional filesystem label"`

	// Force forces formatting even if the device appears to be in use
	Force bool `json:"force,omitempty" default:"false" doc:"Force formatting even if device appears in use"`

	// AdditionalOptions contains filesystem-specific formatting options
	AdditionalOptions map[string]string `json:"additionalOptions,omitempty" doc:"Filesystem-specific formatting options"`
}

// FormatPartition formats a partition with the specified filesystem
func (h *FilesystemHandler) FormatPartition(
	ctx context.Context,
	input *struct {
		Body FormatPartitionInput
	},
) (*struct{ Body dto.CheckResult }, error) {
	req := input.Body
	tlog.InfoContext(ctx, "Formatting partition",
		"partition", req.PartitionID,
		"filesystem", req.FilesystemType,
		"label", req.Label,
		"force", req.Force)

	// Find the partition using DiskMap directly
	diskMap := h.getDiskMap()
	partition, _, found := diskMap.GetPartitionByID(req.PartitionID)
	if !found {
		tlog.ErrorContext(ctx, "Partition not found", "partition_id", req.PartitionID)
		return nil, huma.Error404NotFound("Partition not found")
	}

	// Get device path using DiskMap method
	devicePath := diskMap.GetPartitionDevicePath(partition)
	if devicePath == "" {
		return nil, huma.Error400BadRequest("Partition has no valid device path")
	}

	// Use filesystem service to format the partition
	result, formatErr := h.fsService.FormatPartition(ctx, devicePath, req.FilesystemType, dto.FormatOptions{
		Label:             req.Label,
		Force:             req.Force,
		AdditionalOptions: req.AdditionalOptions,
	})

	if formatErr != nil {
		tlog.ErrorContext(ctx, "Failed to format partition",
			"partition", req.PartitionID,
			"device", devicePath,
			"error", formatErr)
		return nil, huma.Error500InternalServerError("Failed to format partition", formatErr)
	}

	tlog.InfoContext(ctx, "Successfully formatted partition",
		"partition", req.PartitionID,
		"filesystem", req.FilesystemType,
		"device", devicePath)

	return &struct{ Body dto.CheckResult }{Body: *result}, nil
}

// CheckPartitionInput contains the input for checking a partition
type CheckPartitionInput struct {
	// PartitionID is the unique partition identifier
	PartitionID string `query:"partition_id" json:"partitionId" doc:"Unique partition identifier"`

	// AutoFix automatically fixes errors if possible
	AutoFix bool `query:"auto_fix" json:"autoFix,omitempty" default:"false" doc:"Automatically fix errors if possible"`

	// Force forces check even if filesystem appears clean
	Force bool `query:"force" json:"force,omitempty" default:"false" doc:"Force check even if filesystem appears clean"`

	// Verbose enables verbose output
	Verbose bool `query:"verbose" json:"verbose,omitempty" default:"false" doc:"Enable verbose output"`
}

// AbortCheckPartitionInput contains the input for canceling a filesystem check
type AbortCheckPartitionInput struct {
	// PartitionID is the unique partition identifier
	PartitionID string `json:"partitionId" doc:"Unique partition identifier"`
}

// CheckPartition checks a partition's filesystem for errors
func (h *FilesystemHandler) CheckPartition(
	ctx context.Context,
	input *struct {
		Body CheckPartitionInput
	},
) (*struct{ Body dto.CheckResult }, error) {
	req := input.Body
	tlog.InfoContext(ctx, "Checking partition filesystem",
		"partition", req.PartitionID,
		"auto_fix", req.AutoFix,
		"force", req.Force)

	// Find the partition using DiskMap directly
	diskMap := h.getDiskMap()
	partition, _, found := diskMap.GetPartitionByID(req.PartitionID)
	if !found {
		return nil, huma.Error404NotFound("Partition not found")
	}

	// Get device path and filesystem type
	devicePath := diskMap.GetPartitionDevicePath(partition)
	if devicePath == "" {
		return nil, huma.Error400BadRequest("Partition has no valid device path")
	}

	fsType := ""
	if partition.FsType != nil {
		fsType = *partition.FsType
	}
	if fsType == "" {
		return nil, huma.Error400BadRequest("Partition has no filesystem type")
	}

	// Use filesystem service to check the partition
	result, checkErr := h.fsService.CheckPartition(ctx, devicePath, fsType, dto.CheckOptions{
		AutoFix: req.AutoFix,
		Force:   req.Force,
		Verbose: req.Verbose,
	})

	if checkErr != nil {
		tlog.ErrorContext(ctx, "Failed to check partition",
			"partition", req.PartitionID,
			"device", devicePath,
			"error", checkErr)
		return nil, huma.Error500InternalServerError("Failed to check partition", checkErr)
	}

	tlog.InfoContext(ctx, "Filesystem check completed",
		"partition", req.PartitionID,
		"success", result.Success,
		"errors_found", result.ErrorsFound,
		"errors_fixed", result.ErrorsFixed)

	return &struct{ Body dto.CheckResult }{Body: *result}, nil
}

// AbortCheckPartition cancels a running filesystem check operation
func (h *FilesystemHandler) AbortCheckPartition(
	ctx context.Context,
	input *struct {
		Body AbortCheckPartitionInput
	},
) (*struct {
	Body struct {
		Success bool `json:"success"`
	}
}, error) {
	req := input.Body
	tlog.InfoContext(ctx, "Aborting filesystem check", "partition", req.PartitionID)

	// Find the partition using DiskMap directly
	diskMap := h.getDiskMap()
	partition, _, found := diskMap.GetPartitionByID(req.PartitionID)
	if !found {
		return nil, huma.Error404NotFound("Partition not found")
	}

	// Get device path
	devicePath := diskMap.GetPartitionDevicePath(partition)
	if devicePath == "" {
		return nil, huma.Error400BadRequest("Partition has no valid device path")
	}

	abortErr := h.fsService.AbortCheckPartition(ctx, devicePath)
	if abortErr != nil {
		if errors.Is(abortErr, dto.ErrorNotFound) {
			return nil, huma.Error404NotFound("Check operation not running")
		}
		if errors.Is(abortErr, dto.ErrorConflict) {
			return nil, huma.Error409Conflict("Another operation is running for this device")
		}
		tlog.ErrorContext(ctx, "Failed to abort filesystem check", "partition", req.PartitionID, "device", devicePath, "error", abortErr)
		return nil, huma.Error500InternalServerError("Failed to abort filesystem check", abortErr)
	}

	return &struct {
		Body struct {
			Success bool `json:"success"`
		}
	}{
		Body: struct {
			Success bool `json:"success"`
		}{Success: true},
	}, nil
}

// PartitionStateInput contains the input for getting partition state
type PartitionStateInput struct {
	// PartitionID is the unique partition identifier
	PartitionID string `query:"partition_id" json:"partitionId" doc:"Unique partition identifier"`
}

// GetPartitionState gets the current state of a partition's filesystem
func (h *FilesystemHandler) GetPartitionState(
	ctx context.Context,
	input *PartitionStateInput,
) (*struct{ Body dto.FilesystemState }, error) {
	tlog.DebugContext(ctx, "Getting partition state", "partition", input.PartitionID)

	// Find the partition using DiskMap directly
	diskMap := h.getDiskMap()
	partition, _, found := diskMap.GetPartitionByID(input.PartitionID)
	if !found {
		return nil, huma.Error404NotFound("Partition not found")
	}

	// Get device path and filesystem type
	devicePath := diskMap.GetPartitionDevicePath(partition)
	if devicePath == "" {
		return nil, huma.Error400BadRequest("Partition has no valid device path")
	}

	fsType := ""
	if partition.FsType != nil {
		fsType = *partition.FsType
	}
	if fsType == "" {
		return nil, huma.Error400BadRequest("Partition has no filesystem type")
	}

	// Use filesystem service to get partition state
	state, stateErr := h.fsService.GetPartitionState(ctx, devicePath, fsType)
	if stateErr != nil {
		if errors.Is(stateErr, dto.ErrorUnsupportedFilesystem) {
			tlog.WarnContext(ctx, "Unsupported filesystem type for partition state",
				"partition", input.PartitionID,
				"device", devicePath,
				"filesystem_type", fsType)
			return nil, huma.Error400BadRequest("Unsupported filesystem type for this partition")
		}
		tlog.ErrorContext(ctx, "Failed to get partition state",
			"partition", input.PartitionID,
			"device", devicePath,
			"error", stateErr)
		return nil, huma.Error500InternalServerError("Failed to get partition state", stateErr)
	}

	return &struct{ Body dto.FilesystemState }{Body: *state}, nil
}

// PartitionLabelInput contains the input for getting/setting partition label
type PartitionLabelInput struct {
	// PartitionID is the unique partition identifier
	PartitionID string `query:"partition_id" json:"partitionId" doc:"Unique partition identifier"`
}

// SetPartitionLabelInput contains the input for setting partition label
type SetPartitionLabelInput struct {
	// PartitionID is the unique partition identifier
	PartitionID string `json:"partitionId" doc:"Unique partition identifier"`

	// Label is the new filesystem label
	Label string `json:"label" doc:"New filesystem label"`
}

// GetPartitionLabel gets the label of a partition's filesystem
func (h *FilesystemHandler) GetPartitionLabel(
	ctx context.Context,
	input *PartitionLabelInput,
) (*struct {
	Body struct {
		Label string `json:"label"`
	}
}, error) {
	tlog.DebugContext(ctx, "Getting partition label", "partition", input.PartitionID)

	// Find the partition using DiskMap directly
	diskMap := h.getDiskMap()
	partition, _, found := diskMap.GetPartitionByID(input.PartitionID)
	if !found {
		return nil, huma.Error404NotFound("Partition not found")
	}

	// Get device path and filesystem type
	devicePath := diskMap.GetPartitionDevicePath(partition)
	if devicePath == "" {
		return nil, huma.Error400BadRequest("Partition has no valid device path")
	}

	fsType := ""
	if partition.FsType != nil {
		fsType = *partition.FsType
	}
	if fsType == "" {
		return nil, huma.Error400BadRequest("Partition has no filesystem type")
	}

	// Use filesystem service to get the label
	label, labelErr := h.fsService.GetPartitionLabel(ctx, devicePath, fsType)
	if labelErr != nil {
		tlog.ErrorContext(ctx, "Failed to get partition label",
			"partition", input.PartitionID,
			"device", devicePath,
			"error", labelErr)
		return nil, huma.Error500InternalServerError("Failed to get partition label", labelErr)
	}

	return &struct {
		Body struct {
			Label string `json:"label"`
		}
	}{
		Body: struct {
			Label string `json:"label"`
		}{Label: label},
	}, nil
}

// SetPartitionLabel sets the label of a partition's filesystem
func (h *FilesystemHandler) SetPartitionLabel(
	ctx context.Context,
	input *struct {
		Body SetPartitionLabelInput
	},
) (*struct {
	Body struct {
		Success bool `json:"success"`
	}
}, error) {
	req := input.Body
	tlog.InfoContext(ctx, "Setting partition label",
		"partition", req.PartitionID,
		"label", req.Label)

	// Find the partition using DiskMap directly
	diskMap := h.getDiskMap()
	partition, _, found := diskMap.GetPartitionByID(req.PartitionID)
	if !found {
		return nil, huma.Error404NotFound("Partition not found")
	}

	// Get device path and filesystem type
	devicePath := diskMap.GetPartitionDevicePath(partition)
	if devicePath == "" {
		return nil, huma.Error400BadRequest("Partition has no valid device path")
	}

	fsType := ""
	if partition.FsType != nil {
		fsType = *partition.FsType
	}
	if fsType == "" {
		return nil, huma.Error400BadRequest("Partition has no filesystem type")
	}

	// Use filesystem service to set the label
	labelErr := h.fsService.SetPartitionLabel(ctx, devicePath, fsType, req.Label)
	if labelErr != nil {
		tlog.ErrorContext(ctx, "Failed to set partition label",
			"partition", req.PartitionID,
			"device", devicePath,
			"label", req.Label,
			"error", labelErr)
		return nil, huma.Error500InternalServerError("Failed to set partition label", labelErr)
	}

	tlog.InfoContext(ctx, "Successfully set partition label",
		"partition", req.PartitionID,
		"label", req.Label)

	return &struct {
		Body struct {
			Success bool `json:"success"`
		}
	}{
		Body: struct {
			Success bool `json:"success"`
		}{Success: true},
	}, nil
}
