package service

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/dianlight/srat/dto"
)

// HomeAssistantComponentServiceInterface exposes read-only status inspection for
// the SRAT Home Assistant custom component lifecycle.
type HomeAssistantComponentServiceInterface interface {
	GetStatus() (*dto.HomeAssistantCustomComponentStatus, error)
	Uninstall() error
}

type HomeAssistantComponentService struct {
	state *dto.ContextState
}

// NewHomeAssistantComponentService builds a status service for SRAT custom component.
func NewHomeAssistantComponentService(state *dto.ContextState) HomeAssistantComponentServiceInterface {
	return &HomeAssistantComponentService{state: state}
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

	return os.RemoveAll(installPath)
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
