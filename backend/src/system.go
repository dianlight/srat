package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/data"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dm"
	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v68/github"
	"github.com/jaypipes/ghw"
	"github.com/jinzhu/copier"
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

// HealthAndUpdateDataRefeshHandlers periodically refreshes health data and checks for updates.
// It performs the following tasks:
// - Updates the read-only status of the system.
// - Checks for new releases on GitHub based on the configured update channel.
// - Updates the lastReleaseData with the latest release information.
// - Checks the status of the Samba process.
//
// This function runs indefinitely in a loop, with a 5-second pause between iterations.
// It uses a rate limiter to manage GitHub API requests and respects the configured update channel.
func HealthAndUpdateDataRefeshHandlers() {
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	var gh = github.NewClient(rateLimiter)
	for {
		healthData.ReadOnly = *data.ROMode
		if data.Config.UpdateChannel != dm.None {
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
						if *release.Prerelease && data.Config.UpdateChannel == dm.Stable {
							//log.Printf("Skip Prerelease %s", *release.TagName)
							continue
						} else if !*release.Prerelease && data.Config.UpdateChannel == dm.Prerelease {
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
					//log.Printf("Latest %s version is %s (Asset %s)", data.Config.UpdateChannel, *lastReleaseData.LastRelease.TagName, lastReleaseData.ArchAsset.GetName())
					notifyUpdate()
				} else {
					log.Println("No Releases found")
					lastReleaseData = &SRATReleaseAsset{
						UpdateStatus: -1,
					}
					notifyUpdate()
				}
			})
		} else {
			lastReleaseData = &SRATReleaseAsset{
				UpdateStatus: -1,
			}
			notifyUpdate()
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

// HealthCheckWsHandler handles WebSocket connections for health check updates.
// It continuously sends health status updates to the client every 5 seconds.
//
// Parameters:
//   - request: WebSocketMessageEnvelope containing the initial request information.
//   - c: A channel of *WebSocketMessageEnvelope used to send messages back to the WebSocket client.
//
// The function runs indefinitely, sending health updates until the WebSocket connection is closed.
func HealthCheckWsHandler(ctx context.Context, request WebSocketMessageEnvelope, c chan *WebSocketMessageEnvelope) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var message WebSocketMessageEnvelope = WebSocketMessageEnvelope{
				Event: EventHeartbeat,
				Uid:   request.Uid,
				Data:  healthData,
			}
			c <- &message
			time.Sleep(5 * time.Second)
		}
	}
}

// DirtyWsHandler handles WebSocket connections for monitoring changes in the dirty state of configuration sections.
// It continuously checks for changes in the DirtySectionState and sends updates to the client when changes occur.
//
// Parameters:
//   - ctx: A context.Context for handling cancellation of the WebSocket connection.
//   - request: A WebSocketMessageEnvelope containing the initial request information.
//   - c: A channel of *WebSocketMessageEnvelope used to send messages back to the WebSocket client.
//
// The function runs indefinitely, sending updates only when the DirtySectionState changes,
// until the WebSocket connection is closed or the context is cancelled.
func DirtyWsHandler(ctx context.Context, request WebSocketMessageEnvelope, c chan *WebSocketMessageEnvelope) {
	var oldDritySectionState config.ConfigSectionDirtySate
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if oldDritySectionState != data.DirtySectionState {
				var message WebSocketMessageEnvelope = WebSocketMessageEnvelope{
					Event: EventHeartbeat,
					Uid:   request.Uid,
					Data:  data.DirtySectionState,
				}
				c <- &message
				copier.Copy(&oldDritySectionState, data.DirtySectionState)
				//log.Printf("%v %v\n", oldDritySectionState, data.DirtySectionState)
			}
		}
		time.Sleep(1 * time.Second)
	}
}

// notifyUpdate sends the latest release data to all registered update channels.
// This function is used to notify all clients waiting for update information
// when new release data becomes available.
//
// The function acquires a read lock on the updateQueueMutex to safely iterate
// over the updateQueue. It then sends the lastReleaseData to each channel in
// the queue.
//
// This function does not take any parameters and does not return any values.
func notifyUpdate() {
	updateQueueMutex.RLock()
	for _, v := range updateQueue {
		v <- lastReleaseData
	}
	updateQueueMutex.RUnlock()
}

// UpdateWsHandler handles WebSocket connections for update notifications.
// It manages a queue for each client to receive update information and
// continuously sends updates to the connected client.
//
// Parameters:
//   - ctx: A context.Context for handling cancellation of the WebSocket connection.
//   - request: A WebSocketMessageEnvelope containing the initial request information,
//     including a unique identifier (Uid) for the client.
//   - c: A channel of *WebSocketMessageEnvelope used to send messages back to the WebSocket client.
//
// The function runs indefinitely, sending updates when available, until the WebSocket
// connection is closed or the context is cancelled. It does not return any value.
func UpdateWsHandler(ctx context.Context, request WebSocketMessageEnvelope, c chan *WebSocketMessageEnvelope) {
	updateQueueMutex.Lock()
	if updateQueue[request.Uid] == nil {
		updateQueue[request.Uid] = make(chan *SRATReleaseAsset, 10)
	}
	var queue = updateQueue[request.Uid]
	queue <- lastReleaseData
	updateQueueMutex.Unlock()
	for {
		select {
		case <-ctx.Done():
			delete(updateQueue, request.Uid)
			return
		default:
			smessage := &WebSocketMessageEnvelope{
				Event: EventUpdate,
				Uid:   request.Uid,
				Data:  <-queue,
			}
			c <- smessage
		}
	}
}

type ProgressWriter struct {
	w io.Writer
	n atomic.Int64
}

// NewProgressWriter creates a new ProgressWriter that wraps the provided io.Writer.
func NewProgressWriter(w io.Writer) *ProgressWriter {
	return &ProgressWriter{w: w}
}

// Write writes the provided bytes to the underlying writer and updates the progress counter.
func (w *ProgressWriter) Write(b []byte) (n int, err error) {
	n, err = w.Write(b)
	w.n.Add(int64(n))
	return n, err
}

// N returns the total number of bytes written by the ProgressWriter.
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

// GetNICsHandler godoc
//
//	@Summary		GetNICsHandler
//	@Description	Return all network interfaces
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	net.Info
//	@Failure		405	{object}	ResponseError
//	@Router			/nics [get]
func GetNICsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	net, err := ghw.Network()
	if err != nil {
		DoResponseError(http.StatusInternalServerError, w, "Unable to fetch nics", err.Error())
		return
	}

	DoResponse(http.StatusOK, w, net)
}

// ReadLinesOffsetN reads contents from file and splits them by new line.
// The offset tells at which line number to start.
// The count determines the number of lines to read (starting from offset):
// n >= 0: at most n lines
// n < 0: whole file
// Source: https://github.com/shirou/gopsutil
func readLinesOffsetN(filename string, offset uint, n int) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return []string{""}, err
	}
	defer f.Close()

	var ret []string

	r := bufio.NewReader(f)
	for i := uint(0); i < uint(n)+offset || n < 0; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				ret = append(ret, strings.Trim(line, "\n"))
			}
			break
		}
		if i < offset {
			continue
		}
		ret = append(ret, strings.Trim(line, "\n"))
	}

	return ret, nil
}

// Source: https://github.com/shirou/gopsutil
func getFileSystems() ([]string, error) {
	filename := "/proc/filesystems"
	lines, err := readLinesOffsetN(filename, 0, -1)
	if err != nil {
		return nil, err
	}
	var ret []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "nodev") {
			ret = append(ret, strings.TrimSpace(line))
			continue
		}
		t := strings.Split(line, "\t")
		if len(t) != 2 || t[1] != "zfs" {
			continue
		}
		ret = append(ret, strings.TrimSpace(t[1]))
	}

	return ret, nil
}

// GetFSHandler godoc
//
//	@Summary		GetFSHandler
//	@Description	Return all network interfaces
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	[]string
//	@Failure		405	{object}	ResponseError
//	@Router			/filesystems [get]
func GetFSHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	fs, err := getFileSystems()
	if err != nil {
		DoResponseError(http.StatusInternalServerError, w, "Unable to fetch nics", err.Error())
		return
	}

	DoResponse(http.StatusOK, w, fs)
}

// PersistVolumesState saves the current state of mounted volumes to persistent storage.
// It retrieves volume data, processes each mounted partition, and saves the mount point data.
//
// The function performs the following steps:
// 1. Retrieves volume data using GetVolumesData().
// 2. Iterates through each partition with a mount point.
// 3. Creates a MountPointData struct for each mounted partition.
// 4. Saves the MountPointData using SaveMountPointData().
//
// Returns:
//   - error: nil if the operation was successful, otherwise an error describing what went wrong
//     during the retrieval of volume data or while saving mount point data.
func PersistVolumesState() error {
	volumes, err := GetVolumesData()
	if err != nil {
		log.Printf("Error persisting volumes state: %v\n", err)
		return err
	}
	for _, partition := range volumes.Partitions {
		if partition.MountPoint != "" {
			var flags = &data.MounDataFlags{}
			flags.Scan(partition.MountFlags)
			adata := dbom.MountPointData{
				Path:   partition.MountPoint,
				Label:  partition.Label,
				Name:   partition.Name,
				FSType: partition.Type,
				Flags:  *flags,
			}
			//pretty.Println(adata)
			err = adata.Save()
			if err != nil {
				log.Printf("Error persisting volume data: %v\n", err)
				return err
			}
		}
	}
	return nil
}

func PersistSharesState() error {
	/*
		volumes, err := GetVolumesData()
		if err != nil {
			log.Printf("Error persisting shared state: %v\n", err) // FIXME: Implement me!
			return err
		}
		for _, partition := range volumes.Partitions {
			if partition.MountPoint != "" {
				var flags = &config.MounDataFlags{}
				flags.Scan(partition.MountFlags)
				adata := config.MountPointData{
					Path:   partition.MountPoint,
					Label:  partition.Label,
					Name:   partition.Name,
					FSType: partition.Type,
					Flags:  *flags,
				}
				//pretty.Println(adata)
				err = adata.Save()
				if err != nil {
					log.Printf("Error persisting volume data: %v\n", err)
					return err
				}
			}
		}*/
	return nil
}
