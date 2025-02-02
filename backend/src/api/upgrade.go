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

	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/utility"
	"github.com/google/go-github/v68/github"
)

type UpgradeHanler struct {
	ctx     context.Context
	apictx  *ContextState
	upgader service.UpgradeServiceInterface
}

func NewUpgradeHanler(ctx context.Context, apictx *ContextState, upgader service.UpgradeServiceInterface) *UpgradeHanler {

	p := new(UpgradeHanler)
	p.ctx = ctx
	p.apictx = apictx
	p.upgader = upgader
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
//	@Success		200 {object}	dto.ReleaseAsset
//	@Failure		405	{object}	ErrorResponse
//	@Router			/update [put]
func (handler *UpgradeHanler) UpdateHandler(w http.ResponseWriter, r *http.Request) {

	lastReleaseData := handler.upgader.GetLastReleaseData()
	log.Printf("Updating to version %s", *lastReleaseData.LastRelease.TagName)

	lastReleaseData.UpdateStatus = 0
	var gh = github.NewClient(nil)
	if lastReleaseData.ArchAsset == nil {
		HttpJSONReponse(w, fmt.Errorf("No asset found for architecture %s", runtime.GOARCH), nil)
		return
	}

	rc, _, err := gh.Repositories.DownloadReleaseAsset(context.Background(), "dianlight", "srat", *lastReleaseData.ArchAsset.ID, http.DefaultClient)
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
	pw := utility.NewProgressWriter(tmpFile)
	go func() {
		var by, err = io.Copy(tmpFile, rc)
		if err != nil {
			HttpJSONReponse(w, err, nil)
		}
		lastReleaseData.UpdateStatus = -1
		slog.Debug(fmt.Sprintf("Update process completed %d vs %d\n", by, *lastReleaseData.ArchAsset.Size))
		tmpFile.Close()
		rc.Close()
	}()

	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			if lastReleaseData.UpdateStatus == -1 {
				break
			}
			lastReleaseData.UpdateStatus = int8((int(pw.N()) / (*lastReleaseData.ArchAsset.Size)) * 100)
			slog.Debug(fmt.Sprintf("Copied %d bytes progress %d%%\n", pw.N(), lastReleaseData.UpdateStatus))
		}
	}()

	HttpJSONReponse(w, lastReleaseData, nil)
}
