package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/data"
	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v68/github"
	"github.com/jpillora/overseer"
	"golang.org/x/time/rate"
)

type Health struct {
	Alive     bool   `json:"alive"`
	ReadOnly  bool   `json:"read_only"`
	Samba     int32  `json:"samba_pid"`
	LastError string `json:"last_error"`
}

var ctx = context.Background()
var healthData = &Health{
	Alive:     true,
	ReadOnly:  true,
	Samba:     -1,
	LastError: "",
}

type SRATReleaseAsset struct {
	UpdateStatus int8                      `json:"update_status"`
	LastRelease  *github.RepositoryRelease `json:"last_release,omitempty"`
	ArchAsset    *github.ReleaseAsset      `json:"arch,omitempty"`
}

var lastReleaseData = &SRATReleaseAsset{
	UpdateStatus: -1,
}

var UpdateLimiter = rate.Sometimes{Interval: 30 * time.Minute}

var (
	updateQueue      = map[string](chan *SRATReleaseAsset){}
	updateQueueMutex = sync.RWMutex{}
)

func HealthAndUpdateDataRefeshHandlers() {
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	var gh = github.NewClient(rateLimiter)
	for {
		healthData.ReadOnly = *data.ROMode
		if data.Config.UpdateChannel != config.None {
			UpdateLimiter.Do(func() {
				log.Printf("Checking for updates...%v", data.Config.UpdateChannel)
				releases, _, err := gh.Repositories.ListReleases(context.Background(), "dianlight", "srat", &github.ListOptions{
					Page:    1,
					PerPage: 5,
				})
				if err != nil {
					if _, ok := err.(*github.RateLimitError); ok {
						log.Println("Github API hit rate limit")
					}
				} else if len(releases) > 0 {
					for _, release := range releases {
						//log.Println(pretty.Sprintf("%v\n", release))
						if *release.Prerelease && data.Config.UpdateChannel == config.Stable {
							//log.Printf("Skip Prerelease %s", *release.TagName)
							continue
						} else if !*release.Prerelease && data.Config.UpdateChannel == config.Prerelease {
							//log.Printf("Skip Release %s", *release.TagName)
							continue
						}
						lastReleaseData.LastRelease = release
						// Serch for the asset corrisponfing the correct architecture
						for _, asset := range lastReleaseData.LastRelease.Assets {
							arch := runtime.GOARCH
							if arch == "arm64" {
								arch = "aarch64"
							}
							if asset.GetName() == fmt.Sprintf("srat_%s", arch) {
								lastReleaseData.ArchAsset = asset
								break
							}
						}
						break
					}
					log.Printf("Latest %s version is %s (Asset %s)", data.Config.UpdateChannel, *lastReleaseData.LastRelease.TagName, lastReleaseData.ArchAsset.GetName())
					notifyUpdate()
				} else {
					log.Println("No Releases found")
				}
			})
		}
		sambaProcess, err := GetSambaProcess()
		if err == nil && sambaProcess != nil {
			healthData.Samba = int32(sambaProcess.Pid)
		} else {
			healthData.Samba = -1
		}
		time.Sleep(5 * time.Second)
	}
}

// HealthCheckHandler godoc
//
//	@Summary		HealthCheck
//	@Description	HealthCheck
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	Health
//	@Failure		405	{object}	ResponseError
//	@Router			/health [get]
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// A very simple health check.
	w.Header().Set("Content-Type", "application/json")

	jsonResponse, jsonError := json.Marshal(healthData)

	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func HealthCheckWsHandler(request WebSocketMessageEnvelope, c chan *WebSocketMessageEnvelope) {
	for {

		var message WebSocketMessageEnvelope = WebSocketMessageEnvelope{
			Event: EventHeartbeat,
			Uid:   request.Uid,
			Data:  healthData,
		}
		c <- &message
		time.Sleep(5 * time.Second)
	}
}

func notifyUpdate() {
	updateQueueMutex.RLock()
	for _, v := range updateQueue {
		v <- lastReleaseData
	}
	updateQueueMutex.RUnlock()
}

func UpdateWsHandler(request WebSocketMessageEnvelope, c chan *WebSocketMessageEnvelope) {
	updateQueueMutex.Lock()
	if updateQueue[request.Uid] == nil {
		updateQueue[request.Uid] = make(chan *SRATReleaseAsset, 10)
	}
	updateQueue[request.Uid] <- lastReleaseData
	var queue = updateQueue[request.Uid]
	queue <- lastReleaseData
	updateQueueMutex.Unlock()
	//log.Printf("Handle recv: %s %s %d", request.Event, request.Uid, len(sharesQueue))
	for {
		smessage := &WebSocketMessageEnvelope{
			Event: EventUpdate,
			Uid:   request.Uid,
			Data:  <-queue,
		}
		//log.Printf("Handle send: %s %s %d", smessage.Event, smessage.Uid, len(c))
		c <- smessage
	}
}

type ProgressWriter struct {
	w io.Writer
	n atomic.Int64
}

func NewProgressWriter(w io.Writer) *ProgressWriter {
	return &ProgressWriter{w: w}
}

func (w *ProgressWriter) Write(b []byte) (n int, err error) {
	n, err = w.Write(b)
	w.n.Add(int64(n))
	return n, err
}

func (w *ProgressWriter) N() int64 {
	return w.n.Load()
}

// UpdateHandler godoc
//
//	@Summary		UpdateHandler
//	@Description	Start the update process
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	SRATReleaseAsset
//	@Failure		405	{object}	ResponseError
//	@Router			/update [put]
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	// A very simple health check.
	w.Header().Set("Content-Type", "application/json")

	log.Printf("Updating to version %s", *lastReleaseData.LastRelease.TagName)

	lastReleaseData.UpdateStatus = 0
	var gh = github.NewClient(nil)
	if lastReleaseData.ArchAsset == nil {
		fmt.Printf("Asset not found for architecture %s\n", runtime.GOARCH)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Asset not found for architecture " + runtime.GOARCH))
		return
	}

	rc, _, err := gh.Repositories.DownloadReleaseAsset(context.Background(), "dianlight", "srat", *lastReleaseData.ArchAsset.ID, http.DefaultClient)
	if err != nil {
		fmt.Printf("Error downloading release asset: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	//defer rc.Close()
	tmpFile, err := os.OpenFile(data.UpdateFilePath, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		fmt.Printf("Error creating temporary file: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	//defer tmpFile.Close()
	pw := NewProgressWriter(tmpFile)
	go func() {
		var by, err = io.Copy(tmpFile, rc)
		if err != nil {
			fmt.Printf("Error copying downloaded file to temporary file %s: %v\n", data.UpdateFilePath, err.Error())
			healthData.LastError = err.Error()
		}
		lastReleaseData.UpdateStatus = -1
		notifyUpdate()
		fmt.Printf("Update process completed %d vs %d\n", by, *lastReleaseData.ArchAsset.Size)
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
			fmt.Printf("Copied %d bytes progress %d%%\n", pw.N(), lastReleaseData.UpdateStatus)
			notifyUpdate()
		}
	}()

	jsonResponse, jsonError := json.Marshal(lastReleaseData)

	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// RestartHandler godoc
//
//	@Summary		RestartHandler
//	@Description	Restart the server ( useful in development )
//	@Tags			system
//	@Produce		json
//	@Success		204
//	@Failure		405	{object}	ResponseError
//	@Router			/restart [put]
func RestartHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)

	log.Println("Restarting server...")
	overseer.Restart()
}
