package api

import (
	"context"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/service/filesystem"
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
	huma.Get(api, "/filesystem/state", h.GetPartitionState, huma.OperationTags("filesystems"))
	huma.Get(api, "/filesystem/label", h.GetPartitionLabel, huma.OperationTags("filesystems"))
	huma.Put(api, "/filesystem/label", h.SetPartitionLabel, huma.OperationTags("filesystems"))
}

// FilesystemInfo combines filesystem type information with capability details
type FilesystemInfo struct {
	// Name is the filesystem type name
	Name string `json:"name"`

	// Type is the filesystem type (same as name for consistency)
	Type string `json:"type"`

	// Description provides a human-readable description of the filesystem
	Description string `json:"description,omitempty"`

	// MountFlags contains standard mount flags
	MountFlags dto.MountFlags `json:"mountFlags"`

	// CustomMountFlags contains filesystem-specific mount flags
	CustomMountFlags dto.MountFlags `json:"customMountFlags"`

	// Support contains filesystem capability information
	Support *dto.FilesystemSupport `json:"support,omitempty"`
}

// ListFilesystems returns all supported filesystems with their capabilities
func (h *FilesystemHandler) ListFilesystems(
	ctx context.Context,
	input *struct{},
) (*struct{ Body []FilesystemInfo }, error) {
	tlog.DebugContext(ctx, "Listing all filesystems with capabilities")

	// Get all supported filesystem types
	fsTypes := h.fsService.ListSupportedTypes()
	result := make([]FilesystemInfo, 0, len(fsTypes))

	// Get capabilities for each filesystem
	for _, fsType := range fsTypes {
		adapter, err := h.fsService.GetAdapter(fsType)
		if err != nil {
			tlog.WarnContext(ctx, "Failed to get adapter", "filesystem", fsType, "error", err)
			continue
		}

		// Get support information
		support, err := adapter.IsSupported(ctx)
		if err != nil {
			tlog.WarnContext(ctx, "Failed to check support", "filesystem", fsType, "error", err)
			continue
		}

		// Get standard and custom mount flags
		standardFlags, _ := h.fsService.GetStandardMountFlags()
		customFlags, _ := h.fsService.GetFilesystemSpecificMountFlags(fsType)

		result = append(result, FilesystemInfo{
			Name:             adapter.GetName(),
			Type:             fsType,
			Description:      adapter.GetDescription(),
			MountFlags:       standardFlags,
			CustomMountFlags: customFlags,
			Support:          &support,
		})
	}

	return &struct{ Body []FilesystemInfo }{Body: result}, nil
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

	// Find the partition
	partition, err := h.findPartitionByID(req.PartitionID)
	if err != nil {
		tlog.ErrorContext(ctx, "Partition not found", "partition_id", req.PartitionID, "error", err)
		return nil, huma.Error404NotFound("Partition not found", err)
	}

	// Get device path
	devicePath := h.getPartitionDevicePath(partition)
	if devicePath == "" {
		return nil, huma.Error400BadRequest("Partition has no valid device path")
	}

	// Get filesystem adapter
	adapter, err := h.fsService.GetAdapter(req.FilesystemType)
	if err != nil {
		tlog.ErrorContext(ctx, "Unsupported filesystem type",
			"filesystem", req.FilesystemType, "error", err)
		return nil, huma.Error400BadRequest(fmt.Sprintf("Unsupported filesystem type: %s", req.FilesystemType))
	}

	// Check if formatting is supported
	support, err := adapter.IsSupported(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to check filesystem support", err)
	}

	if !support.CanFormat {
		msg := fmt.Sprintf("Filesystem %s cannot be formatted on this system", req.FilesystemType)
		if len(support.MissingTools) > 0 {
			msg += fmt.Sprintf(". Missing tools: %v. Install package: %s",
				support.MissingTools, support.AlpinePackage)
		}
		return nil, huma.Error422UnprocessableEntity(msg)
	}

	// Format the partition
	formatErr := adapter.Format(ctx, devicePath, dto.FormatOptions{
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

	return &struct{ Body dto.CheckResult }{
		Body: dto.CheckResult{
			Success:  true,
			Message:  fmt.Sprintf("Successfully formatted %s as %s", devicePath, req.FilesystemType),
			ExitCode: 0,
		},
	}, nil
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

	// Find the partition
	partition, err := h.findPartitionByID(req.PartitionID)
	if err != nil {
		return nil, huma.Error404NotFound("Partition not found", err)
	}

	// Get device path and filesystem type
	devicePath := h.getPartitionDevicePath(partition)
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

	// Get filesystem adapter
	adapter, err := h.fsService.GetAdapter(fsType)
	if err != nil {
		return nil, huma.Error400BadRequest(fmt.Sprintf("Unsupported filesystem type: %s", fsType))
	}

	// Check if checking is supported
	support, err := adapter.IsSupported(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to check filesystem support", err)
	}

	if !support.CanCheck {
		msg := fmt.Sprintf("Filesystem %s cannot be checked on this system", fsType)
		if len(support.MissingTools) > 0 {
			msg += fmt.Sprintf(". Missing tools: %v. Install package: %s",
				support.MissingTools, support.AlpinePackage)
		}
		return nil, huma.Error422UnprocessableEntity(msg)
	}

	// Check the filesystem
	result, checkErr := adapter.Check(ctx, devicePath, dto.CheckOptions{
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

	return &struct{ Body dto.CheckResult }{Body: result}, nil
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

	// Find the partition
	partition, err := h.findPartitionByID(input.PartitionID)
	if err != nil {
		return nil, huma.Error404NotFound("Partition not found", err)
	}

	// Get device path and filesystem type
	devicePath := h.getPartitionDevicePath(partition)
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

	// Get filesystem adapter
	adapter, err := h.fsService.GetAdapter(fsType)
	if err != nil {
		return nil, huma.Error400BadRequest(fmt.Sprintf("Unsupported filesystem type: %s", fsType))
	}

	// Check if state retrieval is supported
	support, err := adapter.IsSupported(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to check filesystem support", err)
	}

	if !support.CanGetState {
		return nil, huma.Error422UnprocessableEntity(
			fmt.Sprintf("Filesystem %s does not support state retrieval", fsType))
	}

	// Get the state
	state, stateErr := adapter.GetState(ctx, devicePath)
	if stateErr != nil {
		tlog.ErrorContext(ctx, "Failed to get partition state",
			"partition", input.PartitionID,
			"device", devicePath,
			"error", stateErr)
		return nil, huma.Error500InternalServerError("Failed to get partition state", stateErr)
	}

	return &struct{ Body dto.FilesystemState }{Body: state}, nil
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
) (*struct{ Body struct{ Label string `json:"label"` } }, error) {
	tlog.DebugContext(ctx, "Getting partition label", "partition", input.PartitionID)

	// Find the partition
	partition, err := h.findPartitionByID(input.PartitionID)
	if err != nil {
		return nil, huma.Error404NotFound("Partition not found", err)
	}

	// Get device path and filesystem type
	devicePath := h.getPartitionDevicePath(partition)
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

	// Get filesystem adapter
	adapter, err := h.fsService.GetAdapter(fsType)
	if err != nil {
		return nil, huma.Error400BadRequest(fmt.Sprintf("Unsupported filesystem type: %s", fsType))
	}

	// Get the label
	label, labelErr := adapter.GetLabel(ctx, devicePath)
	if labelErr != nil {
		tlog.ErrorContext(ctx, "Failed to get partition label",
			"partition", input.PartitionID,
			"device", devicePath,
			"error", labelErr)
		return nil, huma.Error500InternalServerError("Failed to get partition label", labelErr)
	}

	return &struct{ Body struct{ Label string `json:"label"` } }{
		Body: struct{ Label string `json:"label"` }{Label: label},
	}, nil
}

// SetPartitionLabel sets the label of a partition's filesystem
func (h *FilesystemHandler) SetPartitionLabel(
	ctx context.Context,
	input *struct {
		Body SetPartitionLabelInput
	},
) (*struct{ Body struct{ Success bool `json:"success"` } }, error) {
	req := input.Body
	tlog.InfoContext(ctx, "Setting partition label",
		"partition", req.PartitionID,
		"label", req.Label)

	// Find the partition
	partition, err := h.findPartitionByID(req.PartitionID)
	if err != nil {
		return nil, huma.Error404NotFound("Partition not found", err)
	}

	// Get device path and filesystem type
	devicePath := h.getPartitionDevicePath(partition)
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

	// Get filesystem adapter
	adapter, err := h.fsService.GetAdapter(fsType)
	if err != nil {
		return nil, huma.Error400BadRequest(fmt.Sprintf("Unsupported filesystem type: %s", fsType))
	}

	// Check if label setting is supported
	support, err := adapter.IsSupported(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to check filesystem support", err)
	}

	if !support.CanSetLabel {
		msg := fmt.Sprintf("Filesystem %s does not support label changes", fsType)
		if len(support.MissingTools) > 0 {
			msg += fmt.Sprintf(". Missing tools: %v. Install package: %s",
				support.MissingTools, support.AlpinePackage)
		}
		return nil, huma.Error422UnprocessableEntity(msg)
	}

	// Set the label
	labelErr := adapter.SetLabel(ctx, devicePath, req.Label)
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

	return &struct{ Body struct{ Success bool `json:"success"` } }{
		Body: struct{ Success bool `json:"success"` }{Success: true},
	}, nil
}

// Helper functions

// findPartitionByID finds a partition by its unique ID across all disks
func (h *FilesystemHandler) findPartitionByID(partitionID string) (*dto.Partition, error) {
	volumes := h.volumeService.GetVolumesData()
	for _, disk := range volumes {
		if disk.Partitions != nil {
			if partition, found := (*disk.Partitions)[partitionID]; found {
				return &partition, nil
			}
		}
	}
	return nil, fmt.Errorf("partition not found: %s", partitionID)
}

// getPartitionDevicePath gets the best available device path for a partition
func (h *FilesystemHandler) getPartitionDevicePath(partition *dto.Partition) string {
	// Prefer persistent device path
	if partition.DevicePath != nil && *partition.DevicePath != "" {
		return *partition.DevicePath
	}
	// Fallback to legacy device path
	if partition.LegacyDevicePath != nil && *partition.LegacyDevicePath != "" {
		return *partition.LegacyDevicePath
	}
	return ""
}
