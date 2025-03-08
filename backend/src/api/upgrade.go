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
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/utility"
	"github.com/google/go-github/v69/github"
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

func (handler *UpgradeHanler) Patterns() []server.RouteDetail {
	return []server.RouteDetail{
		{Pattern: "/update", Method: "PUT", Handler: handler.UpdateHandler},
	}
}

// UpdateHandler godoc
//
//	@Summary		UpdateHandler
//	@Description	Start the update process
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	dto.UpdateProgress
//	@Failure		405	{object}	dto.ErrorInfo
//	@Router			/update [put]
func (handler *UpgradeHanler) UpdateHandler(w http.ResponseWriter, r *http.Request) {

	lastReleaseData := handler.upgader.GetLastReleaseData()
	log.Printf("Updating to version %s", lastReleaseData.LastRelease)

	var gh = github.NewClient(nil)
	if lastReleaseData.ArchAsset.Size == 0 {
		HttpJSONReponse(w, fmt.Errorf("No asset found for architecture %s", runtime.GOARCH), nil)
		return
	}

	rc, _, err := gh.Repositories.DownloadReleaseAsset(context.Background(), "dianlight", "srat", lastReleaseData.ArchAsset.ID, http.DefaultClient)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	//defer rc.Close()
	tmpFile, err := os.OpenFile(handler.apictx.UpdateFilePath, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
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

	HttpJSONReponse(w, handler.progress, nil)
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
			handler.broadcaster.BroadcastMessage(handler.progress)
			if handler.progress.ProgressStatus >= 100 {
				slog.Info("Update process completed successfully")
				return
			}
		}
	}
}
