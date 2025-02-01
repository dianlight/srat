package api

import (
	"bufio"
	"context"
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

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/google/go-github/v68/github"
	"github.com/jaypipes/ghw"
	"github.com/jinzhu/copier"
	"github.com/jpillora/overseer"
	"golang.org/x/time/rate"
)

var ctx = context.Background()
var healthData = &dto.HealthPing{
	Alive:              true,
	ReadOnly:           true,
	SambaProcessStatus: dto.SambaProcessStatus{Pid: -1},
	LastError:          "",
}

var lastReleaseData = &dto.ReleaseAsset{
	UpdateStatus: -1,
}

var UpdateLimiter = rate.Sometimes{Interval: 30 * time.Minute}

var (
	updateQueue      = map[string](chan *dto.ReleaseAsset){}
	updateQueueMutex = sync.RWMutex{}
)

/*
// HealthAndUpdateDataRefeshHandlers periodically refreshes health data and checks for updates.
// It performs the following tasks:
// - Updates the read-only status of the system.
// - Checks for new releases on GitHub based on the configured update channel.
// - Updates the lastReleaseData with the latest release information.
// - Checks the status of the Samba process.
//
// This function runs indefinitely in a loop, with a 5-second pause between iterations.
// It uses a rate limiter to manage GitHub API requests and respects the configured update channel.
func HealthAndUpdateDataRefeshHandlers(ctx context.Context) {
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	var properties dbom.Properties
	err = properties.Load()
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}
	var settings dto.Settings
	var conv converter.DtoToDbomConverterImpl
	err = conv.PropertiesToSettings(properties, &settings)
	// err = mapper.Map(context.Background(), &settings, properties)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	//context_state := (&status.ContextState{}).FromContext(ctx)
	context_state := StateFromContext(ctx)

	var gh = github.NewClient(rateLimiter)
	for {
		healthData.ReadOnly = context_state.ReadOnlyMode
		if settings.UpdateChannel != dto.None {
			go UpdateLimiter.Do(func() {
				log.Printf("Checking for updates...%v", settings.UpdateChannel)
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
						if *release.Prerelease && settings.UpdateChannel == dto.Stable {
							//log.Printf("Skip Prerelease %s", *release.TagName)
							continue
						} else if !*release.Prerelease && settings.UpdateChannel == dto.Prerelease {
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
					notifyUpdate(ctx)
				} else {
					log.Println("No Releases found")
					lastReleaseData = &dto.ReleaseAsset{
						UpdateStatus: -1,
					}
					notifyUpdate(ctx)
				}
			})
		} else {
			lastReleaseData = &dto.ReleaseAsset{
				UpdateStatus: -1,
			}
			notifyUpdate(ctx)
		}
		sambaProcess, err := GetSambaProcess()
		if err == nil && sambaProcess != nil {
			var conv converter.ProcessToDtoImpl
			conv.ProcessToSambaProcessStatus(sambaProcess, &healthData.SambaProcessStatus)
		} else {
			healthData.SambaProcessStatus.Pid = -1
		}
		time.Sleep(5 * time.Second)
	}
}
*/
/*
// HealthCheckHandler godoc
//
//	@Summary		HealthCheck
//	@Description	HealthCheck
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	dto.HealthPing
//	@Failure		405	{object}	ErrorResponse
//	@Router			/health [get]
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	HttpJSONReponse(w, healthData, nil)
}
*/

/*
// HealthCheckWsHandler handles WebSocket connections for health check updates.
// It continuously sends health status updates to the client every 5 seconds.
//
// Parameters:
//   - request: WebSocketMessageEnvelope containing the initial request information.
//   - c: A channel of *WebSocketMessageEnvelope used to send messages back to the WebSocket client.
//
// The function runs indefinitely, sending health updates until the WebSocket connection is closed.
func HealthCheckWsHandler(ctx context.Context, request dto.WebSocketMessageEnvelope, c chan *dto.WebSocketMessageEnvelope) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var message dto.WebSocketMessageEnvelope = dto.WebSocketMessageEnvelope{
				Event: dto.EventHeartbeat,
				Uid:   request.Uid,
				Data:  healthData,
			}
			c <- &message
			time.Sleep(5 * time.Second)
		}
	}
}
*/

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
func DirtyWsHandler(ctx context.Context, request dto.WebSocketMessageEnvelope, c chan *dto.WebSocketMessageEnvelope) {
	var oldDritySectionState dto.DataDirtyTracker
	//context_state := (&dto.Status{}).FromContext(ctx)
	context_state := StateFromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if oldDritySectionState != context_state.DataDirtyTracker {
				var message dto.WebSocketMessageEnvelope = dto.WebSocketMessageEnvelope{
					Event: dto.EventHeartbeat,
					Uid:   request.Uid,
					Data:  &context_state.DataDirtyTracker,
				}
				c <- &message
				copier.Copy(&oldDritySectionState, context_state.DataDirtyTracker)
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
func notifyUpdate(_ context.Context) {
	//var sse dto.EventMessageEnvelope
	//sse.Event = dto.EventHeartbeat
	//sse.Data = lastReleaseData
	// FIXME: Use SSEBroker to broadcast the message instead of sending to all channels
	//state := StateFromContext(ctx)
	//state.SSEBroker.BroadcastMessage(&sse)

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
func UpdateWsHandler(ctx context.Context, request dto.WebSocketMessageEnvelope, c chan *dto.WebSocketMessageEnvelope) {
	updateQueueMutex.Lock()
	if updateQueue[request.Uid] == nil {
		updateQueue[request.Uid] = make(chan *dto.ReleaseAsset, 10)
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
			smessage := &dto.WebSocketMessageEnvelope{
				Event: dto.EventUpdate,
				Uid:   request.Uid,
				Data:  <-queue,
			}
			c <- smessage
		}
	}
}

type ProgressWriter struct { // FIXME: Don't work!!!
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
//	@Success		200 {object}	dto.ReleaseAsset
//	@Failure		405	{object}	ErrorResponse
//	@Router			/update [put]
func UpdateHandler(w http.ResponseWriter, r *http.Request) {

	log.Printf("Updating to version %s", *lastReleaseData.LastRelease.TagName) // FIXME: Move latest realase to context

	//ctx := (&dto.Status{}).FromContext(r.Context())
	ctx := StateFromContext(r.Context())

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
	tmpFile, err := os.OpenFile(ctx.UpdateFilePath, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	//defer tmpFile.Close()
	pw := NewProgressWriter(tmpFile)
	go func() {
		var by, err = io.Copy(tmpFile, rc)
		if err != nil {
			fmt.Printf("Error copying downloaded file to temporary file %s: %v\n", ctx.UpdateFilePath, err.Error())
			healthData.LastError = err.Error()
		}
		lastReleaseData.UpdateStatus = -1
		notifyUpdate(r.Context())
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
			notifyUpdate(r.Context())
		}
	}()

	HttpJSONReponse(w, lastReleaseData, nil)
}

// RestartHandler godoc
//
//	@Summary		RestartHandler
//	@Description	Restart the server ( useful in development )
//	@Tags			system
//	@Produce		json
//	@Success		204
//	@Failure		405	{object}	ErrorResponse
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
//	@Success		200 {object}	dto.NetworkInfo
//	@Failure		405	{object}	ErrorResponse
//	@Router			/nics [get]
func GetNICsHandler(w http.ResponseWriter, r *http.Request) {

	net, err := ghw.Network()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	var info dto.NetworkInfo
	var conv converter.NetToDtoImpl
	err = conv.NetInfoToNetworkInfo(*net, &info)
	//err = mapper.Map(context.Background(), &info, net)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	HttpJSONReponse(w, info, nil)
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
//	@Description	Return all supported fs
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	dto.FilesystemTypes
//	@Failure		405	{object}	ErrorResponse
//	@Router			/filesystems [get]
func GetFSHandler(w http.ResponseWriter, r *http.Request) {

	fs, err := getFileSystems()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	var xfs dto.FilesystemTypes
	for _, fsi := range fs {
		xfs = append(xfs, dto.FilesystemType(fsi))
	}
	//err = mapper.Map(context.Background(), &xfs, fs)
	//if err != nil {
	//	HttpJSONReponse(w, err, nil)
	//	return
	//}
	HttpJSONReponse(w, xfs, nil)
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

	/*
		volumes, err := GetVolumesData()
		if err != nil {
			log.Printf("Error persisting volumes state: %v\n", err)
			return tracerr.Wrap(err)
		}
		for _, partition := range volumes.Partitions {
			if partition.MountPoint != "" {
				var flags = &dto.MounDataFlags{}
				flags.Scan(partition.MountFlags)
				adata := dbom.MountPointPath{
					Path: partition.MountPoint,
					//Label:  partition.Label,
					Source: partition.Name,
					FSType: partition.Type,
					Flags:  *flags,
				}
				//pretty.Println(adata)
				err = adata.Save()
				if err != nil {
					log.Printf("Error persisting volume data: %v\n", err)
					return tracerr.Wrap(err)
				}
			}
		}
	*/
	return nil
}

func PersistSharesState() error {

	return nil
}
