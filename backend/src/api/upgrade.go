package api

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/utility"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
	"github.com/google/go-github/v69/github"
	"github.com/ztrue/tracerr"
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

func (p *UpgradeHanler) Routers(srv *fuego.Server) error {
	fuego.Put(srv, "/update", p.UpdateHandler, option.Description("Start the update process"), option.Tags("system"))
	return nil
}

func (handler *UpgradeHanler) UpdateHandler(c fuego.ContextNoBody) (*dto.UpdateProgress, error) {

	lastReleaseData := handler.upgader.GetLastReleaseData()
	log.Printf("Updating to version %s", lastReleaseData.LastRelease)

	var gh = github.NewClient(nil)
	if lastReleaseData.ArchAsset.Size == 0 {
		return nil, fuego.NotFoundError{
			Title:  "No asset found",
			Detail: fmt.Sprintf("No asset found for architecture %s", runtime.GOARCH),
		}
	}

	rc, _, err := gh.Repositories.DownloadReleaseAsset(context.Background(), "dianlight", "srat", lastReleaseData.ArchAsset.ID, http.DefaultClient)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	//defer rc.Close()
	tmpFile, err := os.OpenFile(handler.apictx.UpdateFilePath, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	//defer tmpFile.Close()
	handler.pw = utility.NewProgressWriter(tmpFile, lastReleaseData.ArchAsset.Size)
	go func() {
		defer tmpFile.Close()
		defer rc.Close()
		handler.progress = dto.UpdateProgress{
			ProgressStatus: 0,
			LastRelease:    lastReleaseData.LastRelease,
		}
		go handler.notifyClient()
		var by, err = io.Copy(tmpFile, rc)
		if err != nil {
			slog.Error(fmt.Sprintf("Error copying file %s", err))
			handler.progress.UpdateError = err.Error()
		}
		slog.Debug(fmt.Sprintf("Update process completed %d vs %d\n", by, lastReleaseData.ArchAsset.Size))
	}()

	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
		}
	}()

	return &handler.progress, nil
}

func (handler *UpgradeHanler) notifyClient() {
	for {
		select {
		case <-handler.ctx.Done():
			slog.Error("Upgrade process closed", "err", handler.ctx.Err())
			return
		case n := <-handler.pw.P:
			slog.Debug(fmt.Sprintf("Notified client of progress update, bytes written: %d", n))
			handler.progress.ProgressStatus = handler.pw.Percent()
			slog.Debug(fmt.Sprintf("Copied %d bytes progress %d%%\n", handler.pw.Written(), handler.progress.ProgressStatus))
			var event dto.EventMessageEnvelope
			event.Event = dto.EventUpdate
			event.Data = handler.progress
			handler.broadcaster.BroadcastMessage(&event)
			if handler.progress.ProgressStatus >= 100 {
				slog.Info("Update process completed successfully")
				return
			}
		}
	}
}
