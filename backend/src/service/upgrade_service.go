package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/updatekey"
	"github.com/fsnotify/fsnotify"
	"github.com/google/go-github/v82/github"
	"github.com/vvair/selfupdate"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"golang.org/x/time/rate"
)

type UpdateFile struct {
	Path      string
	Signature []byte
	Size      int64
}

type UpdatePackage struct {
	//CurrentExecutablePath *string
	//ReleaseAsset *dto.BinaryAsset
	FilesPaths []UpdateFile
	//TempDirPath           string
}

type UpgradeServiceInterface interface {
	// Check for Upgrade based on current version and update channel
	GetUpgradeReleaseAsset() (ass *dto.ReleaseAsset, err errors.E)
	// Download upgrade assets (.zip) and extract in Data directory
	DownloadAndExtractBinaryAsset(asset dto.BinaryAsset) (*UpdatePackage, errors.E)
	// Install update using selfupdate library with signature verification
	InstallUpdatePackage(updatePkg *UpdatePackage) errors.E
	// Apply update to the current binary and restart if running under s6
	ApplyUpdateAndRestart(updatePkg *UpdatePackage) errors.E
	GetProcessStatus(parentPid int32) *dto.ProcessStatus
}

type UpgradeService struct {
	ctx               context.Context
	gh                *github.Client
	broadcaster       BroadcasterServiceInterface
	updateLimiter     rate.Sometimes
	state             *dto.ContextState
	shutdowner        fx.Shutdowner
	fileWatcherCtx    context.Context
	fileWatcherCancel context.CancelFunc
	verifier          *selfupdate.Verifier
}

type UpgradeServiceProps struct {
	fx.In
	State       *dto.ContextState
	Broadcaster BroadcasterServiceInterface `optional:"true"`
	Ctx         context.Context
	Gh          *github.Client
	Shutdowner  fx.Shutdowner
}

func NewUpgradeService(lc fx.Lifecycle, in UpgradeServiceProps) (UpgradeServiceInterface, error) {
	p := new(UpgradeService)
	p.updateLimiter = rate.Sometimes{Interval: 30 * time.Minute}
	p.ctx = in.Ctx
	p.broadcaster = in.Broadcaster
	p.state = in.State
	p.gh = in.Gh
	p.shutdowner = in.Shutdowner
	p.fileWatcherCtx, p.fileWatcherCancel = context.WithCancel(in.Ctx)

	p.verifier = selfupdate.NewVerifier()

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if p.state.UpdateChannel != dto.UpdateChannels.NONE {
				p.ctx.Value("wg").(*sync.WaitGroup).Add(1)
				go func() {
					defer p.ctx.Value("wg").(*sync.WaitGroup).Done()
					err := p.updater()
					if err != nil {
						slog.ErrorContext(p.ctx, "Error in run loop", "err", err)
					}
				}()

				// Start file watcher for develop channel
				if p.state.UpdateChannel == dto.UpdateChannels.DEVELOP {
					p.ctx.Value("wg").(*sync.WaitGroup).Add(1)
					go func() {
						defer p.ctx.Value("wg").(*sync.WaitGroup).Done()
						p.watchForDevelopUpdates()
					}()
					slog.DebugContext(p.ctx, "File watcher for develop updates started")
				}
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if p.fileWatcherCancel != nil {
				p.fileWatcherCancel()
			}
			return nil
		},
	})

	return p, nil
}

func (self *UpgradeService) updater() error {
	for {
		select {
		case <-self.ctx.Done():
			slog.InfoContext(self.ctx, "Run process closed", "err", self.ctx.Err())
			return errors.WithStack(self.ctx.Err())
		case <-time.After(self.updateLimiter.Interval):
			slog.DebugContext(self.ctx, "Version Checking...", "channel", self.state.UpdateChannel.String())
			self.notifyClient(dto.UpdateProgress{
				ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSCHECKING,
			})
			ass, err := self.GetUpgradeReleaseAsset()
			if err != nil && !errors.Is(err, dto.ErrorNoUpdateAvailable) {
				slog.ErrorContext(self.ctx, "Error checking for updates", "err", err)
			}
			if ass != nil {
				self.state.UpdateAvailable = true
				self.notifyClient(dto.UpdateProgress{
					ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE,
					ReleaseAsset:   ass,
				})

				// Auto-update if enabled
				if self.state.AutoUpdate {
					slog.InfoContext(self.ctx, "Auto-update enabled, downloading and installing update", "release", ass.LastRelease)
					updatePkg, err := self.DownloadAndExtractBinaryAsset(ass.ArchAsset)
					if err != nil {
						slog.ErrorContext(self.ctx, "Error downloading update during auto-update", "err", err)
						continue
					}
					err = self.ApplyUpdateAndRestart(updatePkg)
					if err != nil {
						slog.ErrorContext(self.ctx, "Error applying update during auto-update", "err", err)
					}
					// If we successfully apply and restart, this code won't be reached
				}
			} else {
				self.state.UpdateAvailable = false
				self.notifyClient(dto.UpdateProgress{
					ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSNOUPGRADE,
				})
			}

		}
	}
}

// watchForDevelopUpdates watches the UpdateDataDir for new binary files in develop channel using fsnotify
// When a new binary is detected, it applies the update and restarts if running under s6
func (self *UpgradeService) watchForDevelopUpdates() {
	if self.state.UpdateDataDir == "" {
		slog.WarnContext(self.ctx, "UpdateDataDir not set, file watcher for develop updates disabled")
		return
	}

	slog.InfoContext(self.ctx, "Starting file watcher for develop channel updates using fsnotify", "watch_dir", self.state.UpdateDataDir)

	// Get the current executable name to watch for
	exePath, err := os.Executable()
	if err != nil {
		slog.ErrorContext(self.ctx, "Failed to get current executable path", "err", err)
		return
	}
	exeName := filepath.Base(exePath)
	watchPath := filepath.Join(self.state.UpdateDataDir, exeName)

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.ErrorContext(self.ctx, "Failed to create file watcher", "err", err)
		return
	}
	defer watcher.Close()

	// Add the directory to watch
	err = watcher.Add(self.state.UpdateDataDir)
	if err != nil {
		slog.ErrorContext(self.ctx, "Failed to add directory to watcher", "dir", self.state.UpdateDataDir, "err", err)
		return
	}

	// Track the last modification time to avoid processing the same event multiple times
	var lastModTime time.Time
	if info, err := os.Stat(watchPath); err == nil {
		lastModTime = info.ModTime()
	}

	// Debounce timer to handle multiple events for the same file write
	var debounceTimer *time.Timer
	const debounceDelay = 500 * time.Millisecond

	for {
		select {
		case <-self.fileWatcherCtx.Done():
			slog.InfoContext(self.ctx, "File watcher stopped")
			return

		case event, ok := <-watcher.Events:
			if !ok {
				slog.WarnContext(self.ctx, "File watcher events channel closed")
				return
			}

			// Only process events for our target binary
			if filepath.Base(event.Name) != exeName {
				continue
			}

			// We're interested in Write and Create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// Check if file has been modified since last check
				info, err := os.Stat(watchPath)
				if err != nil {
					// File might have been deleted or moved
					slog.ErrorContext(self.ctx, "Failed to stat watched file", "file", watchPath, "err", err)
					continue
				}

				if info.ModTime().After(lastModTime) && info.Size() > 0 {
					//slog.DebugContext(self.ctx, "File modification detected", "file", event.Name, "op", event.Op.String())

					// Cancel any existing debounce timer
					if debounceTimer != nil {
						debounceTimer.Stop()
					}

					// Create a new debounce timer
					debounceTimer = time.AfterFunc(debounceDelay, func() {
						// Re-check modification time after debounce
						currentInfo, err := os.Stat(watchPath)
						if err != nil {
							slog.ErrorContext(self.ctx, "Failed to stat file after debounce", "err", err)
							return
						}

						if currentInfo.ModTime().Equal(lastModTime) {
							slog.DebugContext(self.ctx, "No change in modification time after debounce, skipping update", "file", watchPath)
							// File hasn't changed since we first saw the event
							return
						}

						lastModTime = currentInfo.ModTime()
						slog.InfoContext(self.ctx, "Detected new update file in develop channel", "file", watchPath, "size", currentInfo.Size())

						// Create update package
						updatePkg := &UpdatePackage{
							FilesPaths: []UpdateFile{{
								Path:      watchPath,
								Size:      currentInfo.Size(),
								Signature: nil,
							}},
						}

						// Notify that update is available
						self.notifyClient(dto.UpdateProgress{
							ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE,
							Progress:       0,
							ReleaseAsset: &dto.ReleaseAsset{
								ArchAsset: dto.BinaryAsset{
									Name: filepath.Base(watchPath),
									Size: int(currentInfo.Size()),
								},
							},
						})

						// Apply the update (this will copy to the running location)
						slog.InfoContext(self.ctx, "Installing develop channel update", "source", watchPath)
						if err = self.InstallUpdatePackage(updatePkg); err != nil {
							slog.ErrorContext(self.ctx, "Failed to install develop update", "err", err)
							return
						}

						// We're in develop mode, restart

						slog.InfoContext(self.ctx, "Triggering restart after develop update install")
						self.notifyClient(dto.UpdateProgress{
							ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSINSTALLCOMPLETE,
							Progress:       100,
							ErrorMessage:   "Update installed, restarting...",
						})

						// Give time for the message to be sent
						time.Sleep(500 * time.Millisecond)

						// Check if running under s6 and restart
						if self.isRunningUnderS6() {
							slog.InfoContext(self.ctx, "Running under s6, initiating graceful shutdown for restart")
							if err := self.shutdowner.Shutdown(); err != nil {
								slog.ErrorContext(self.ctx, "Failed to trigger graceful shutdown", "err", err)
								// Fallback to os.Exit if shutdowner fails
								os.Exit(0)
							}
						} else {
							slog.InfoContext(self.ctx, "Not running under s6, manual restart required")
							self.notifyClient(dto.UpdateProgress{
								ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSINSTALLCOMPLETE,
								Progress:       100,
								ErrorMessage:   "Update installed, please restart the service manually",
							})
						}

					})
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				slog.WarnContext(self.ctx, "File watcher errors channel closed")
				return
			}
			slog.ErrorContext(self.ctx, "File watcher error", "err", err)
		}
	}
}

func (self *UpgradeService) GetUpgradeReleaseAsset() (ass *dto.ReleaseAsset, err errors.E) {

	if self.state.UpdateChannel.String() == dto.UpdateChannels.NONE.String() {
		// if channel NONE return error not update dto.ErrorNoUpdateAvailable
		return nil, errors.WithMessage(dto.ErrorNoUpdateAvailable, "No releases check", "channel", self.state.UpdateChannel.String())
	} else if self.state.UpdateChannel.String() == dto.UpdateChannels.DEVELOP.String() {
		// if channel DEVELOP first search in /config directory but don't do update check
		return nil, errors.WithMessage(dto.ErrorNoUpdateAvailable, "No releases check", "channel", self.state.UpdateChannel.String())
	} else {
		// if channel PRERELEASE or RELEASE search online on github
		myversion := config.GetCurrentBinaryVersion()

		if myversion == nil {
			return nil, errors.Errorf("Current binary version is not set")
		}

		slog.InfoContext(self.ctx, "Checking for updates", "current", config.Version, "channel", self.state.UpdateChannel.String())
		releases, _, err := self.gh.Repositories.ListReleases(context.Background(), "dianlight", "srat", &github.ListOptions{
			Page:    1,
			PerPage: 5,
		})
		if err != nil {
			if _, ok := err.(*github.RateLimitError); ok {
				slog.WarnContext(self.ctx, "Github API hit rate limit")
			}
			slog.WarnContext(self.ctx, "Error getting releases", "err", err)
			return nil, errors.WithMessage(dto.ErrorNoUpdateAvailable, "No releases found")
		} else if len(releases) > 0 {
			for _, release := range releases {
				slog.DebugContext(self.ctx, "Found Release", "tag_name", release.GetTagName(), "prerelease", release.GetPrerelease())
				if release.GetPrerelease() && self.state.UpdateChannel.String() != dto.UpdateChannels.PRERELEASE.String() {
					slog.DebugContext(self.ctx, "Skip PreRelease", "tag_name", release.GetTagName())
					continue
				}

				assertVersion, err := semver.NewVersion(*release.TagName)
				if err != nil {
					slog.WarnContext(self.ctx, "Error parsing version", "version", *release.TagName, "err", err)
					continue
				}
				slog.DebugContext(self.ctx, "Checking version", "current", myversion, "release", *release.TagName, "compare", myversion.Compare(assertVersion))

				if myversion.GreaterThanEqual(assertVersion) {
					continue
				}

				// Serch for the asset corrisponfing the correct architecture
				for _, asset := range release.Assets {
					arch := runtime.GOARCH
					switch arch {
					case "arm64":
						arch = "aarch64"
					case "amd64":
						arch = "x86_64"
					}
					slog.InfoContext(self.ctx, "Checking asset", "asset_name", asset.GetName(), "expected_name", fmt.Sprintf("srat_%s.zip", arch))
					if asset.GetName() == fmt.Sprintf("srat_%s.zip", arch) {
						var archAsset dto.BinaryAsset
						conv := converter.GitHubToDtoImpl{}
						err = conv.ReleaseAssetToBinaryAsset(asset, &archAsset)
						if err != nil {
							return nil, errors.WithStack(err)
						}
						ass = &dto.ReleaseAsset{
							LastRelease: *release.TagName,
							ArchAsset:   archAsset,
						}
						myversion = assertVersion
						slog.InfoContext(self.ctx, "Found upgrade release asset", "release", *release.TagName, "asset_name", asset.GetName())
						break
					}
				}
			}
			if ass != nil {
				return ass, nil
			} else {
				return nil, errors.WithMessage(dto.ErrorNoUpdateAvailable, "No releases found")
			}
		} else {
			slog.DebugContext(self.ctx, "No Releases found")
			return nil, errors.WithMessage(dto.ErrorNoUpdateAvailable, "No releases found")
		}
	}
}

func (self *UpgradeService) notifyClient(data dto.UpdateProgress) {
	if self.broadcaster != nil {
		self.broadcaster.BroadcastMessage(data)
	}
}

// ProgressTracker implements io.Writer to track bytes processed
type progressTracker struct {
	Total      uint64
	readed     uint64
	Target     string
	OnProgress func(target string, readed uint64, total uint64)
}

func (pt *progressTracker) Write(p []byte) (int, error) {
	n := len(p)
	pt.readed += uint64(n)
	if pt.OnProgress != nil {
		pt.OnProgress(pt.Target, pt.readed, pt.Total)
	}
	return n, nil
}

func (self *UpgradeService) extractFile(f *zip.File, dest string) (*UpdateFile, errors.E) {

	// Verify file validity
	if f == nil {
		return nil, errors.Errorf("file entry is nil")
	}
	if f.FileInfo().IsDir() {
		return nil, errors.Errorf("directories are not supported in update package: %s", f.Name)
	}
	if f.FileHeader.Comment == "" {
		return nil, errors.Errorf("file has no signature in comment: %s", f.Name)
	}

	// Build the local path
	path := filepath.Join(dest, f.Name)

	// Check for ZipSlip vulnerability
	if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) && dest != "." {
		return nil, errors.Errorf("illegal file path: %s", path)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return nil, errors.WithStack(err)
	}

	// Open file in zip
	rc, err := f.Open()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rc.Close()

	// Create local file
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer out.Close()

	size := f.FileHeader.UncompressedSize64
	tracker := &progressTracker{
		Total:  size,
		Target: path,
		OnProgress: func(target string, downloaded, total uint64) {
			if total > 0 {
				percent := float64(downloaded) / float64(total) * 100
				self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING, Progress: percent})
				slog.DebugContext(self.ctx, "Extracting", "target", target, "percent", percent, "downloaded", downloaded, "total", total)
			} else {
				slog.DebugContext(self.ctx, "Extracting", "target", target, "downloaded_bytes", downloaded, "total_bytes", "unknown")
			}
		},
	}

	// Copy content
	//
	//_, err = io.Copy(out, io.TeeReader(rc, tracker))
	err = self.verifier.Load([]byte(f.FileHeader.Comment), []byte(updatekey.UpdatePublicKey))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = selfupdate.Apply(io.TeeReader(rc, tracker), selfupdate.Options{
		TargetPath: path,
		Verifier:   self.verifier,
	})
	if err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			slog.ErrorContext(self.ctx, "Failed to rollback from bad update", "error", rerr)
		}
		return nil, errors.WithStack(err)
	}

	return &UpdateFile{
		Path:      path,
		Signature: []byte(f.FileHeader.Comment),
		Size:      f.FileInfo().Size(),
	}, errors.WithStack(err)
}

// DownloadAndExtractBinaryAsset downloads the binary asset from the given URL, extracts it to a UpdateDataDir, and returns the UpdatePackage to the extracted executable.
func (self *UpgradeService) DownloadAndExtractBinaryAsset(asset dto.BinaryAsset) (*UpdatePackage, errors.E) {
	slog.InfoContext(self.ctx, "Starting download and extraction", "asset_name", asset.Name, "download_url", asset.BrowserDownloadURL)

	// --- Download Phase ---
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING, Progress: 0})

	req, err := http.NewRequestWithContext(self.ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		errWrapped := errors.Wrapf(err, "failed to create request for %s", asset.BrowserDownloadURL)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		errWrapped := errors.Wrapf(err, "failed to download asset from %s", asset.BrowserDownloadURL)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errWrapped := errors.Errorf("failed to download asset: received status code %d from %s", resp.StatusCode, asset.BrowserDownloadURL)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}

	size := uint64(resp.ContentLength)
	tracker := &progressTracker{
		Total:  size,
		Target: asset.Name,
		OnProgress: func(target string, downloaded, total uint64) {
			if total > 0 {
				percent := float64(downloaded) / float64(total) * 100
				self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING, Progress: percent})
				slog.DebugContext(self.ctx, "Downloading", "target", target, "percent", percent, "downloaded", downloaded, "total", total)
			} else {
				slog.DebugContext(self.ctx, "Downloading", "target", target, "downloaded_bytes", downloaded, "total_bytes", "unknown")
			}
		},
	}

	downloadedFilePath, err := os.CreateTemp("", "download-*.zip")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create temporary file for download")
	}
	defer os.Remove(downloadedFilePath.Name())
	defer downloadedFilePath.Close()

	// Wrap the response body with our progress tracker
	_, err = io.Copy(downloadedFilePath, io.TeeReader(resp.Body, tracker))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to save temp file")
	}

	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSDOWNLOADCOMPLETE, Progress: 100})
	slog.InfoContext(self.ctx, "Asset downloaded successfully", "path", downloadedFilePath.Name())

	// --- Verify Checksum ---
	if asset.Digest != "" {
		slog.InfoContext(self.ctx, "Verifying downloaded asset checksum", "expected_digest", asset.Digest)
		downloadedFilePath.Seek(0, io.SeekStart) // Reset file pointer

		if strings.HasPrefix(asset.Digest, "sha256:") {
			h := sha256.New()
			if _, err := io.Copy(h, downloadedFilePath); err != nil {
				errWrapped := errors.Wrapf(err, "failed to compute checksum for downloaded asset %v", downloadedFilePath.Name())
				self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
				return nil, errWrapped
			}

			computedDigest := fmt.Sprintf("sha256:%x", h.Sum(nil))
			slog.InfoContext(self.ctx, "Computed checksum", "computed_digest", computedDigest)

			if computedDigest != asset.Digest {
				errWrapped := errors.Errorf("checksum mismatch for downloaded asset %#v: expected %s, got %s", downloadedFilePath.Name(), asset.Digest, computedDigest)
				self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
				return nil, errWrapped
			}
			slog.InfoContext(self.ctx, "Checksum verification passed for downloaded asset")
			downloadedFilePath.Seek(0, io.SeekStart)
		} else {
			errWrapped := errors.Errorf("unsupported digest format for asset %v: %s", downloadedFilePath.Name(), asset.Digest)
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
			return nil, errWrapped
		}
	} else {
		slog.ErrorContext(self.ctx, "No expected digest provided, for checksum verification", "asset_name", asset.Name, "digest", asset.Digest)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: "No expected digest provided, for checksum verification"})
		return nil, errors.New("no expected digest provided, for checksum verification")
	}

	// --- Extraction Phase ---
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSEXTRACTING, Progress: 0})

	zipReader, err := zip.OpenReader(downloadedFilePath.Name())
	if err != nil {
		errWrapped := errors.Wrapf(err, "failed to open downloaded zip asset %v", downloadedFilePath)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}
	defer zipReader.Close()

	totalFiles := len(zipReader.File)

	if totalFiles == 0 {
		errWrapped := errors.Errorf("downloaded zip asset %v is empty", *downloadedFilePath)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}

	var extractedFilesCount int
	var foundPaths []UpdateFile

	slog.DebugContext(self.ctx, "Extracting asset", "source_zip", downloadedFilePath, "total_files", totalFiles)
	extractPercentage := 0.0

	for _, f := range zipReader.File {
		path, err := self.extractFile(f, self.state.UpdateDataDir)
		if err != nil {
			errWrapped := errors.Wrapf(err, "failed to extract file %s from zip:%s", f.Name, err.Error())
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
			return nil, errWrapped
		}

		foundPaths = append(foundPaths, *path)

		extractedFilesCount++
		if totalFiles > 0 {
			extractPercentage = float64((extractedFilesCount * 100) / totalFiles)
		}
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSEXTRACTING, Progress: extractPercentage})
	}

	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSEXTRACTCOMPLETE, Progress: 100})
	slog.InfoContext(self.ctx, "Asset extracted successfully", "temp_dir", self.state.UpdateDataDir, "extracted_to", self.state.UpdateDataDir)

	// Verify that the current executable was found in the archive

	return &UpdatePackage{
		FilesPaths: foundPaths,
	}, nil
}

func (self *UpgradeService) InstallUpdatePackage(updatePkg *UpdatePackage) errors.E {

	if updatePkg == nil || len(updatePkg.FilesPaths) == 0 {
		return errors.New("invalid update package")
	}

	// chrach all FilesPaths for existence
	for _, path := range updatePkg.FilesPaths {
		if _, err := os.Stat(path.Path); os.IsNotExist(err) {
			return errors.Errorf("update package file does not exist: %s", path.Path)
		}
	}

	currentFile, targetDir, err := self.getCurrentExecutablePath()
	if err != nil {
		return err
	}
	slog.InfoContext(self.ctx, "Installing update package", "target_directory", *targetDir)
	for _, path := range updatePkg.FilesPaths {
		targetPath := filepath.Join(*targetDir, filepath.Base(path.Path))
		tracker := &progressTracker{
			Total:  uint64(path.Size),
			Target: targetPath,
			OnProgress: func(target string, progress, total uint64) {
				if total > 0 {
					percent := float64(progress) / float64(total) * 100
					self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING, Progress: percent})
					slog.DebugContext(self.ctx, "Installing", "target", target, "percent", percent, "progress", progress, "total", total)
				} else {
					slog.DebugContext(self.ctx, "Installing", "target", target, "progress", progress, "total_bytes", "unknown")
				}
			},
		}
		var currentVersion *semver.Version
		newVersion := config.GetBinaryVersion(path.Path)
		if targetPath == *currentFile {
			currentVersion = config.GetCurrentBinaryVersion()
		} else {
			currentVersion = config.GetBinaryVersion(targetPath)
		}

		if currentVersion == nil && newVersion == nil {
			slog.DebugContext(self.ctx, "No version info found, proceeding with installation assuming not exe file", "target_path", targetPath)
		} else {
			if newVersion.LessThan(currentVersion) ||
				(newVersion.Equal(currentVersion) && self.state.UpdateChannel != dto.UpdateChannels.DEVELOP) {
				slog.InfoContext(self.ctx, "New binary has same version or older as current, skipping installation", "Current", currentVersion.String(), "NewVersion", newVersion.String())
				self.notifyClient(dto.UpdateProgress{
					ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSNOUPGRADE,
					ErrorMessage:   fmt.Sprintf("Binary version %s is already installed", newVersion.String()),
				})
				return errors.New("binary version is same as current version")
			}
			slog.InfoContext(self.ctx, "Installing new version", "current", currentVersion.String(), "new", newVersion.String())
		}

		rc, err := os.Open(path.Path)
		if err != nil {
			return errors.WithStack(err)
		}
		defer rc.Close()

		// Copy content
		if path.Signature != nil {

			errS := self.verifier.Load(path.Signature, []byte(updatekey.UpdatePublicKey))
			if errS != nil {
				return errors.WithStack(errS)
			}
			errS = selfupdate.Apply(io.TeeReader(rc, tracker), selfupdate.Options{
				TargetPath: targetPath,
				Verifier:   self.verifier,
			})
			if errS != nil {
				if rerr := selfupdate.RollbackError(errS); rerr != nil {
					slog.ErrorContext(self.ctx, "Failed to rollback from bad update", "error", rerr)
				}
				return errors.WithStack(errS)
			}
		} else if self.state.UpdateChannel == dto.UpdateChannels.DEVELOP {
			// In develop channel, allow unsigned updates (for testing)
			slog.WarnContext(self.ctx, "Installing unsigned update in develop channel", "target_path", targetPath)
			errS := selfupdate.Apply(io.TeeReader(rc, tracker), selfupdate.Options{
				TargetPath: targetPath,
			})
			if errS != nil {
				if rerr := selfupdate.RollbackError(errS); rerr != nil {
					slog.ErrorContext(self.ctx, "Failed to rollback from bad update", "error", rerr)
				}
				return errors.WithStack(errS)
			}
		} else {
			errWrapped := errors.Errorf("missing signature for update file: %s", path.Path)
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
			return errWrapped
		}
		slog.InfoContext(self.ctx, "Update package installed successfully")

	}
	return nil
}

// ApplyUpdateAndRestart applies the update using selfupdate with signature verification
// and restarts the process if running under s6
func (self *UpgradeService) ApplyUpdateAndRestart(updatePkg *UpdatePackage) errors.E {

	err := self.InstallUpdatePackage(updatePkg)
	if err != nil {
		return err
	}

	slog.InfoContext(self.ctx, "Update applied successfully")
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSINSTALLCOMPLETE, Progress: 100})

	// Check if we're running under s6 and restart if so
	if self.isRunningUnderS6() {
		slog.InfoContext(self.ctx, "Running under s6, initiating graceful shutdown for restart")
		self.notifyClient(dto.UpdateProgress{
			ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSINSTALLCOMPLETE,
			Progress:       100,
			ErrorMessage:   "Update complete, restarting service...",
		})
		// Give time for the message to be sent
		time.Sleep(500 * time.Millisecond)
		// Trigger graceful shutdown via fx.Shutdowner
		if err := self.shutdowner.Shutdown(); err != nil {
			slog.ErrorContext(self.ctx, "Failed to trigger graceful shutdown", "err", err)
			// Fallback to os.Exit if shutdowner fails
			os.Exit(0)
		}
	} else {
		slog.InfoContext(self.ctx, "Not running under s6, manual restart required")
		self.notifyClient(dto.UpdateProgress{
			ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSINSTALLCOMPLETE,
			Progress:       100,
			ErrorMessage:   "Update complete, please restart the service manually",
		})
	}

	return nil
}

// isRunningUnderS6 checks if the current process is running under s6 supervision
func (self *UpgradeService) isRunningUnderS6() bool {
	// Check for s6 environment variables
	if os.Getenv("S6_VERSION") != "" {
		return true
	}

	// Check if parent process is s6-supervise
	ppid := os.Getppid()
	if ppid <= 1 {
		return false
	}

	// Read the parent process name
	cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", ppid)
	cmdline, err := os.ReadFile(cmdlinePath)
	if err != nil {
		// If we can't read the parent process, assume not under s6
		return false
	}

	// Convert null-terminated strings to regular string
	cmdlineStr := string(bytes.ReplaceAll(cmdline, []byte{0}, []byte(" ")))

	// Check if parent process contains "s6-supervise"
	return strings.Contains(cmdlineStr, "s6-supervise")
}

func (self *UpgradeService) getCurrentExecutablePath() (currentExe *string, currentDir *string, err errors.E) {
	currentExe_, errS := os.Executable()
	if errS != nil {
		errWrapped := errors.Wrap(errS, "failed to get current executable path for in-place update")
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, nil, errWrapped
	}
	destinationFile := filepath.Dir(currentExe_)
	return &currentExe_, &destinationFile, nil
}

func (self *UpgradeService) GetProcessStatus(parentPid int32) *dto.ProcessStatus {

	if self.state.UpdateChannel != dto.UpdateChannels.NONE {
		return nil
	}

	status := &dto.ProcessStatus{
		Pid:           -parentPid, // Negative PID indicates this is a subprocess
		Name:          "updater_" + (self.state.UpdateChannel.String()),
		CreateTime:    time.Time{},
		CPUPercent:    0.0,
		MemoryPercent: 0.0,
		OpenFiles:     0,
		Connections:   0,
		Status:        []string{"idle"},
		IsRunning:     true,
	}

	return status
}
