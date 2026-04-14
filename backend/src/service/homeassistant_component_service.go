package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/tlog"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

const developCustomComponentArchivePath = "/addon_configs/local_sambanas2/upgrade/srat.zip"
const customComponentRestartRepairID = "custom_component_restart_required"

// HomeAssistantComponentServiceInterface exposes read-only status inspection for
// the SRAT Home Assistant custom component lifecycle.
type HomeAssistantComponentServiceInterface interface {
	GetStatus() (*dto.HomeAssistantCustomComponentStatus, error)
	SyncIssueStatus(status *dto.HomeAssistantCustomComponentStatus) error
	InstallOrUpgrade() error
	InstallOrUpgradeFromZip(zipArchive []byte) error
	Uninstall() error
	UpsertRestartRequiredRepair(ctx context.Context) error
	DismissRestartRequiredRepair(ctx context.Context) error
	DismissAddonConfigIssue(ctx context.Context) error
}

type HomeAssistantComponentService struct {
	ctx           context.Context
	state         *dto.ContextState
	issueService  IssueServiceInterface
	repairService RepairServiceInterface
	haService     HomeAssistantServiceInterface
	broadcaster   BroadcasterServiceInterface
}

type HomeAssistantComponentServiceProps struct {
	fx.In
	Ctx           context.Context
	State         *dto.ContextState
	IssueService  IssueServiceInterface         `optional:"true"`
	RepairService RepairServiceInterface        `optional:"true"`
	HAService     HomeAssistantServiceInterface `optional:"true"`
	Broadcaster   BroadcasterServiceInterface   `optional:"true"`
}

// NewHomeAssistantComponentService builds a status service for SRAT custom component.
func NewHomeAssistantComponentService(in HomeAssistantComponentServiceProps) HomeAssistantComponentServiceInterface {
	return &HomeAssistantComponentService{
		ctx:           in.Ctx,
		state:         in.State,
		issueService:  in.IssueService,
		repairService: in.RepairService,
		haService:     in.HAService,
		broadcaster:   in.Broadcaster,
	}
}

type customComponentManifest struct {
	Version string `json:"version"`
}

func (s *HomeAssistantComponentService) GetStatus() (*dto.HomeAssistantCustomComponentStatus, error) {
	root := dto.DefaultCustomComponentsPath
	if s.state != nil && s.state.CustomComponentsPath != "" {
		root = s.state.CustomComponentsPath
	}

	installPath := filepath.Join(root, dto.CustomComponentSRATName)
	manifestPath := filepath.Join(installPath, "manifest.json")

	status := &dto.HomeAssistantCustomComponentStatus{
		Component:    dto.HomeAssistantComponentSRAT,
		InstallPath:  installPath,
		ManifestPath: manifestPath,
	}

	if info, err := os.Stat(installPath); err == nil {
		status.Installed = info.IsDir()
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if status.Installed {
		installedVersion, err := readManifestVersion(manifestPath)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		status.InstalledVersion = installedVersion
	}

	if s.state != nil && s.state.HAWsComponent != nil && s.state.HAWsComponent.Component == dto.HomeAssistantComponentSRAT {
		status.Connected = true
		status.ConnectedVersion = &s.state.HAWsComponent.Version
		status.HAVersion = s.state.HAWsComponent.HAVersion
		status.EntryID = s.state.HAWsComponent.EntryID
		connectedAt := s.state.HAWsComponent.ConnectedAt
		if !connectedAt.IsZero() {
			status.ConnectedAt = new(connectedAt)
		}
	}

	if archive, err := internal.GetEmbeddedCustomComponentZip(); err == nil {
		if v, err := readManifestVersionFromCustomComponentArchive(archive); err == nil {
			status.LatestVersion = &v
		}
	}
	if status.LatestVersion == nil {
		v := config.Version
		status.LatestVersion = &v
	}

	status.CanInstall = !status.Installed
	status.CanUpgrade = status.Installed
	status.CanUninstall = status.Installed

	return status, nil
}

func (s *HomeAssistantComponentService) Uninstall() error {
	root := dto.DefaultCustomComponentsPath
	if s.state != nil && s.state.CustomComponentsPath != "" {
		root = s.state.CustomComponentsPath
	}

	installPath := filepath.Join(root, dto.CustomComponentSRATName)
	if _, err := os.Stat(installPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	err := os.RemoveAll(installPath)
	if err != nil {
		return err
	}

	go func() {
		err := s.UpsertRestartRequiredRepair(s.ctx)
		if err != nil {
			tlog.WarnContext(s.ctx, "Failed to upsert restart required repair after custom component uninstall", "error", err)
		}
	}()

	return nil
}

func (s *HomeAssistantComponentService) UpsertRestartRequiredRepair(ctx context.Context) error {
	if s.repairService != nil {
		cmd := dto.RepairCommandMessage{
			CommandID:      uuid.NewString(),
			RepairID:       customComponentRestartRepairID,
			Action:         dto.RepairCommandActionUpsert,
			TranslationKey: customComponentRestartRepairID,
			Severity:       dto.RepairIssueSeverityWarning,
			IsFixable:      true,
			IsPersistent:   true,
		}

		_, createErr := s.repairService.Create(cmd)
		if createErr != nil {
			if _, updateErr := s.repairService.Update(cmd); updateErr != nil {
				tlog.WarnContext(ctx, "Unable to create or refresh custom component restart repair", "repair_id", customComponentRestartRepairID, "error", createErr, "update_error", updateErr)
				return updateErr
			}
		}

		if s.broadcaster != nil {
			s.broadcaster.BroadcastMessage(cmd)
		} else {
			tlog.WarnContext(ctx, "Broadcaster service not available to broadcast custom component restart repair update", "repair_id", customComponentRestartRepairID)
		}

		return nil
	}

	if s.haService != nil {
		err := s.haService.CreatePersistentNotification(
			customComponentRestartRepairID,
			"Restart Home Assistant required",
			"SRAT custom component changes require a Home Assistant restart to fully apply.",
		)
		if err != nil {
			tlog.WarnContext(ctx, "Unable to create restart-required persistent notification", "notification_id", customComponentRestartRepairID, "error", err)
			return err
		}
	}

	return nil
}

func (s *HomeAssistantComponentService) DismissRestartRequiredRepair(ctx context.Context) error {
	return s.dismissRepairIssue(ctx, customComponentRestartRepairID)
}

func (s *HomeAssistantComponentService) DismissAddonConfigIssue(ctx context.Context) error {
	return s.dismissRepairIssue(ctx, "addon_config_changed")
}

func (s *HomeAssistantComponentService) dismissRepairIssue(ctx context.Context, repairID string) error {
	if s.repairService != nil {
		dismissErr := s.repairService.Delete(repairID)
		if dismissErr != nil && !errors.Is(dismissErr, dto.ErrorNotFound) {
			tlog.WarnContext(ctx, "Unable to dismiss repair", "repair_id", repairID, "error", dismissErr)
			return dismissErr
		}

		if s.broadcaster != nil {
			s.broadcaster.BroadcastMessage(dto.RepairCommandMessage{
				CommandID: uuid.NewString(),
				RepairID:  repairID,
				Action:    dto.RepairCommandActionDelete,
			})
		}

		return nil
	}

	if s.haService != nil {
		dismissErr := s.haService.DismissPersistentNotification(repairID)
		if dismissErr != nil && !errors.Is(dismissErr, dto.ErrorNotFound) {
			tlog.WarnContext(ctx, "Unable to dismiss persistent notification", "notification_id", repairID, "error", dismissErr)
			return dismissErr
		}
	}

	return nil
}

func (s *HomeAssistantComponentService) SyncIssueStatus(status *dto.HomeAssistantCustomComponentStatus) error {
	if status == nil || s.issueService == nil {
		return nil
	}

	if !status.Installed && !status.Connected {
		existing, err := s.issueService.FindByTitle(dto.HomeAssistantComponentMissingIssueTitle)
		if err != nil {
			return err
		}
		if existing == nil {
			severity := dto.IssueSeverities.ISSUESEVERITYWARNING
			return s.issueService.Create(&dto.Issue{
				Title:          dto.HomeAssistantComponentMissingIssueTitle,
				Description:    "SRAT custom component is not installed under /config/custom_components/srat and no active websocket connection from Home Assistant is present.",
				ResolutionLink: dto.HomeAssistantComponentMissingIssueResolutionLink,
				Severity:       &severity,
			})
		}

		return nil
	}

	err := s.issueService.ResolveByTitle(dto.HomeAssistantComponentMissingIssueTitle)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return nil
}

func (s *HomeAssistantComponentService) InstallOrUpgrade() error {
	zipArchive, err := s.resolveCustomComponentArchive()
	if err != nil {
		return err
	}

	return s.InstallOrUpgradeFromZip(zipArchive)
}

func (s *HomeAssistantComponentService) InstallOrUpgradeFromZip(zipArchive []byte) error {
	if len(zipArchive) == 0 {
		return fmt.Errorf("custom component archive is empty")
	}

	root := dto.DefaultCustomComponentsPath
	if s.state != nil && s.state.CustomComponentsPath != "" {
		root = s.state.CustomComponentsPath
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	installPath := filepath.Join(root, dto.CustomComponentSRATName)
	if err := os.RemoveAll(installPath); err != nil {
		return err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipArchive), int64(len(zipArchive)))
	if err != nil {
		return err
	}

	for _, file := range zipReader.File {
		cleanedName := strings.TrimPrefix(filepath.Clean(file.Name), "/")
		if cleanedName == "." {
			continue
		}

		destination := filepath.Join(root, cleanedName)
		cleanRoot := filepath.Clean(root)
		if !strings.HasPrefix(destination, cleanRoot+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in archive: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}

		source, err := file.Open()
		if err != nil {
			return err
		}

		mode := file.Mode()
		if mode == 0 {
			mode = 0o644
		}

		target, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			source.Close()
			return err
		}

		_, copyErr := io.Copy(target, source)
		closeSourceErr := source.Close()
		closeTargetErr := target.Close()

		if copyErr != nil {
			return copyErr
		}
		if closeSourceErr != nil {
			return closeSourceErr
		}
		if closeTargetErr != nil {
			return closeTargetErr
		}
	}

	if _, err := os.Stat(filepath.Join(installPath, "manifest.json")); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("custom component archive missing %s/manifest.json", dto.CustomComponentSRATName)
		}
		return err
	}

	go func() {
		err := s.UpsertRestartRequiredRepair(s.ctx)
		if err != nil {
			tlog.WarnContext(s.ctx, "Failed to upsert restart required repair after custom component install/upgrade", "error", err)
		}
	}()

	return nil
}

func (s *HomeAssistantComponentService) resolveCustomComponentArchive() ([]byte, error) {
	if s.state != nil && s.state.UpdateChannel.String() == dto.UpdateChannels.DEVELOP.String() {
		status, err := s.GetStatus()
		if err != nil {
			return nil, err
		}

		if developArchive, used, err := s.resolveDevelopCustomComponentArchive(status.InstalledVersion); err != nil {
			return nil, err
		} else if used {
			return developArchive, nil
		}
	}

	embeddedArchive, err := internal.GetEmbeddedCustomComponentZip()
	if err != nil {
		return nil, err
	}

	return embeddedArchive, nil
}

func (s *HomeAssistantComponentService) resolveDevelopCustomComponentArchive(installedVersion *string) ([]byte, bool, error) {
	archive, err := os.ReadFile(developCustomComponentArchivePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	developVersion, err := readManifestVersionFromCustomComponentArchive(archive)
	if err != nil {
		return nil, false, err
	}

	if installedVersion == nil || *installedVersion == "" {
		return archive, true, nil
	}

	useArchive, err := versionLessOrEqual(*installedVersion, developVersion)
	if err != nil {
		return nil, false, err
	}

	if !useArchive {
		return nil, false, nil
	}

	return archive, true, nil
}

func readManifestVersionFromCustomComponentArchive(zipArchive []byte) (string, error) {
	if len(zipArchive) == 0 {
		return "", fmt.Errorf("custom component archive is empty")
	}

	archiveReader, err := zip.NewReader(bytes.NewReader(zipArchive), int64(len(zipArchive)))
	if err != nil {
		return "", err
	}

	for _, file := range archiveReader.File {
		cleanedName := strings.TrimPrefix(filepath.Clean(file.Name), "/")
		if cleanedName != filepath.Join(dto.CustomComponentSRATName, "manifest.json") {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return "", err
		}

		manifest := customComponentManifest{}
		decodeErr := json.NewDecoder(rc).Decode(&manifest)
		closeErr := rc.Close()
		if decodeErr != nil {
			return "", decodeErr
		}
		if closeErr != nil {
			return "", closeErr
		}

		if manifest.Version == "" {
			return "", fmt.Errorf("custom component archive manifest has empty version")
		}

		return manifest.Version, nil
	}

	return "", fmt.Errorf("custom component archive missing %s/manifest.json", dto.CustomComponentSRATName)
}

func versionLessOrEqual(currentVersion string, candidateVersion string) (bool, error) {
	current, err := semver.NewVersion(strings.TrimPrefix(currentVersion, "v"))
	if err != nil {
		return false, err
	}

	candidate, err := semver.NewVersion(strings.TrimPrefix(candidateVersion, "v"))
	if err != nil {
		return false, err
	}

	return current.LessThan(candidate) || current.Equal(candidate), nil
}

func readManifestVersion(manifestPath string) (*string, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	manifest := customComponentManifest{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	if manifest.Version == "" {
		return nil, nil
	}

	version := manifest.Version
	return &version, nil
}
