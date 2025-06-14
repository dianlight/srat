package api

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/utility"
	"gitlab.com/tozd/go/errors"
)

type UpgradeHanler struct {
	ctx         context.Context
	apictx      *dto.ContextState
	upgader     service.UpgradeServiceInterface
	broadcaster service.BroadcasterServiceInterface
	progress    dto.UpdateProgress
	pw          *utility.ProgressWriter
}

func NewUpgradeHanler(ctx context.Context, apictx *dto.ContextState, upgader service.UpgradeServiceInterface, broadcaster service.BroadcasterServiceInterface) *UpgradeHanler {

	p := new(UpgradeHanler)
	p.ctx = ctx
	p.apictx = apictx
	p.upgader = upgader
	p.broadcaster = broadcaster
	return p
}

func (self *UpgradeHanler) RegisterUpgradeHanler(api huma.API) {
	huma.Put(api, "/update", self.UpdateHandler, huma.OperationTags("system"))
	huma.Get(api, "/update", self.GetUpdateInfoHandler, huma.OperationTags("system"))
	huma.Get(api, "/update_channels", self.GetUpdateChannelsHandler, huma.OperationTags("system"))
}

// GetUpdateInfoHandler checks for available updates and returns information about the release asset.
//
//	@Summary		Check for available updates
//	@Description	Retrieves information about the latest available release asset based on the current update channel.
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	struct{Body dto.ReleaseAsset}	"Information about the available update."
//	@Failure		404	{object}	huma.ErrorModel					"No update available or error finding assets."
//	@Failure		500	{object}	huma.ErrorModel					"Internal server error."
//	@Router			/update [get]
func (handler *UpgradeHanler) GetUpdateInfoHandler(ctx context.Context, input *struct{}) (*struct{ Body dto.ReleaseAsset }, error) {
	slog.Debug("Handling GET /update request")
	asset, err := handler.upgader.GetUpgradeReleaseAsset(nil)
	if err != nil {
		if errors.Is(err, dto.ErrorNoUpdateAvailable) {
			slog.Info("No update available", "error", err)
			return nil, huma.Error404NotFound(err.Error())
		}
		slog.Error("Error getting upgrade release asset", "error", err)
		return nil, errors.Wrap(err, "failed to get upgrade release asset")
	}

	if asset == nil { // Should ideally be covered by ErrorNoUpdateAvailable
		slog.Info("No update asset found, though no explicit error was returned.")
		return nil, huma.Error404NotFound("No update asset found.")
	}

	slog.Debug("Update asset found", "release", asset.LastRelease, "asset_name", asset.ArchAsset.Name)
	return &struct{ Body dto.ReleaseAsset }{Body: *asset}, nil
}

func (handler *UpgradeHanler) UpdateHandler(ctx context.Context, input *struct{}) (*struct{ Body dto.UpdateProgress }, error) {
	assets, err := handler.upgader.GetUpgradeReleaseAsset(nil)
	if err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("Unable to find update assets %#v", err.Error()))
	}

	log.Printf("Updating to version %s", assets.LastRelease)

	handler.ctx.Value("wg").(*sync.WaitGroup).Add(1)
	go func() {
		defer handler.ctx.Value("wg").(*sync.WaitGroup).Done()

		updatePkg, err := handler.upgader.DownloadAndExtractBinaryAsset(assets.ArchAsset)
		if err != nil {
			slog.Error("Error downloading and extracting binary asset", "err", err)
			return
		}
		err = handler.upgader.InstallUpdatePackage(updatePkg)
		if err != nil {
			slog.Error("Error installing update package", "err", err)
			return
		}
	}()

	return &struct{ Body dto.UpdateProgress }{Body: dto.UpdateProgress{
		ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING,
		Progress:       0,
		ErrorMessage:   "",
		LastRelease:    assets.LastRelease,
	}}, nil
}

// GetUpdateChannelsHandler returns a list of available update channels.
//
//	@Summary		List available update channels
//	@Description	Retrieves a list of all defined update channels in the system.
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	struct{Body []dto.UpdateChannel}	"A list of available update channels."
//	@Failure		500	{object}	huma.ErrorModel					"Internal server error."
//	@Router			/update_channels [get]
func (handler *UpgradeHanler) GetUpdateChannelsHandler(ctx context.Context, input *struct{}) (*struct{ Body []dto.UpdateChannel }, error) {
	slog.Debug("Handling GET /update_channels request")

	channels := dto.UpdateChannels.All()

	return &struct{ Body []dto.UpdateChannel }{Body: channels}, nil
}
