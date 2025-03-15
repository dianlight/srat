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

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
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

func (self *UpgradeHanler) RegisterUpgradeHanler(api huma.API) {
	huma.Put(api, "/update", self.UpdateHandler, huma.OperationTags("system"))
}

// UpdateHandler handles the update process for the application.
// It retrieves the latest release data, downloads the release asset,
// and writes it to a temporary file while tracking the progress.
//
// Parameters:
//   - ctx: The context for the request.
//   - input: A pointer to an input struct (currently unused).
//
// Returns:
//   - A pointer to a struct containing the update progress body.
//   - An error if any occurs during the update process.
//
// The function performs the following steps:
//  1. Retrieves the latest release data.
//  2. Logs the version being updated to.
//  3. Checks if the release asset size is zero and returns a 404 error if true.
//  4. Downloads the release asset from GitHub.
//  5. Opens a temporary file for writing the update.
//  6. Initializes a progress writer to track the update progress.
//  7. Starts a goroutine to copy the downloaded asset to the temporary file and update the progress.
//  8. Starts a goroutine to periodically sleep (purpose unclear).
func (handler *UpgradeHanler) UpdateHandler(ctx context.Context, input *struct{}) (*struct{ Body dto.UpdateProgress }, error) {

	lastReleaseData := handler.upgader.GetLastReleaseData()
	log.Printf("Updating to version %s", lastReleaseData.LastRelease)

	var gh = github.NewClient(nil)
	if lastReleaseData.ArchAsset.Size == 0 {
		return nil, huma.Error404NotFound(fmt.Sprintf("No asset found for architecture %s", runtime.GOARCH))
	}

	rc, _, err := gh.Repositories.DownloadReleaseAsset(context.Background(), "dianlight", "srat", lastReleaseData.ArchAsset.ID, http.DefaultClient)
	if err != nil {
		return nil, err
	}
	//defer rc.Close()
	tmpFile, err := os.OpenFile(handler.apictx.UpdateFilePath, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
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

	return nil, nil
}

// notifyClient listens for progress updates and notifies the client accordingly.
// It runs an infinite loop that waits for either the context to be done or for progress updates.
// When the context is done, it logs an error message and returns.
// When a progress update is received, it logs the progress, updates the progress status,
// broadcasts the progress to the client, and checks if the update process is complete.
// If the update process is complete, it logs a success message and returns.
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
