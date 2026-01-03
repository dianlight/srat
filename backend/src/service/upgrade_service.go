package service

import (
	"archive/zip"
	"bytes"
	"context"
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
	"github.com/google/go-github/v80/github"
	"github.com/minio/selfupdate"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"golang.org/x/time/rate"
)

type UpdatePackage struct {
	CurrentExecutablePath *string
	OtherFilesPaths       []string
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
}

type UpgradeService struct {
	ctx            context.Context
	gh             *github.Client
	broadcaster    BroadcasterServiceInterface
	updateLimiter  rate.Sometimes
	state          *dto.ContextState
	shutdowner     fx.Shutdowner
	fileWatcherCtx context.Context
	fileWatcherCancel context.CancelFunc
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

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			p.ctx.Value("wg").(*sync.WaitGroup).Add(1)
			go func() {
				defer p.ctx.Value("wg").(*sync.WaitGroup).Done()
				p.run()
			}()

			// Start file watcher for develop channel
			if p.state.UpdateChannel == dto.UpdateChannels.DEVELOP {
				p.ctx.Value("wg").(*sync.WaitGroup).Add(1)
				go func() {
					defer p.ctx.Value("wg").(*sync.WaitGroup).Done()
					p.watchForDevelopUpdates()
				}()
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

func (self *UpgradeService) run() error {
	for {
		select {
		case <-self.ctx.Done():
			slog.InfoContext(self.ctx, "Run process closed", "err", self.ctx.Err())
			return errors.WithStack(self.ctx.Err())
		case <-time.After(self.updateLimiter.Interval):
			slog.DebugContext(self.ctx, "Version Checking...")
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
					LastRelease:    ass.LastRelease,
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

// watchForDevelopUpdates watches the UpdateDataDir for new binary files in develop channel
// When a new binary is detected, it applies the update and restarts if running under s6
func (self *UpgradeService) watchForDevelopUpdates() {
	if self.state.UpdateDataDir == "" {
		slog.WarnContext(self.ctx, "UpdateDataDir not set, file watcher for develop updates disabled")
		return
	}

	slog.InfoContext(self.ctx, "Starting file watcher for develop channel updates", "watch_dir", self.state.UpdateDataDir)

	// Get the current executable name to watch for
	exePath, err := os.Executable()
	if err != nil {
		slog.ErrorContext(self.ctx, "Failed to get current executable path", "err", err)
		return
	}
	exeName := filepath.Base(exePath)
	watchPath := filepath.Join(self.state.UpdateDataDir, exeName)

	// Track the last modification time to detect changes
	var lastModTime time.Time
	if info, err := os.Stat(watchPath); err == nil {
		lastModTime = info.ModTime()
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-self.fileWatcherCtx.Done():
			slog.InfoContext(self.ctx, "File watcher stopped")
			return
		case <-ticker.C:
			// Check if the file exists and has been modified
			info, err := os.Stat(watchPath)
			if err != nil {
				// File doesn't exist yet or can't be accessed
				continue
			}

			// Check if file has been modified since last check
			if info.ModTime().After(lastModTime) && info.Size() > 0 {
				lastModTime = info.ModTime()
				slog.InfoContext(self.ctx, "Detected new update file in develop channel", "file", watchPath, "size", info.Size())

				// Small delay to ensure file write is complete
				time.Sleep(100 * time.Millisecond)

				// Create update package
				updatePkg := &UpdatePackage{
					CurrentExecutablePath: &watchPath,
					OtherFilesPaths:       []string{},
				}

				// Notify that update is available
				self.notifyClient(dto.UpdateProgress{
					ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE,
					LastRelease:    "develop-" + info.ModTime().Format("20060102-150405"),
				})

				// Apply the update (this will copy to the running location)
				slog.InfoContext(self.ctx, "Installing develop channel update", "source", watchPath)
				err := self.InstallUpdatePackage(updatePkg)
				if err != nil {
					slog.ErrorContext(self.ctx, "Failed to install develop update", "err", err)
					continue
				}

				// If AutoUpdate is enabled or we're in develop mode, restart
				if self.state.AutoUpdate || self.state.UpdateChannel == dto.UpdateChannels.DEVELOP {
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
				}
			}
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
						myversion = *assertVersion
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

const progressReportThresholdPercentage = 5

type progressReader struct {
	reader                 io.Reader
	totalSize              int64
	readSoFar              int64
	onProgress             func(percentage int)
	lastReportedPercentage int
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.readSoFar += int64(n)
	if pr.totalSize > 0 {
		percentage := int((float64(pr.readSoFar) / float64(pr.totalSize)) * 100)
		if percentage >= pr.lastReportedPercentage+progressReportThresholdPercentage || (percentage == 100 && pr.lastReportedPercentage != 100) {
			if pr.onProgress != nil {
				pr.onProgress(percentage)
			}
			pr.lastReportedPercentage = percentage
		}
	}
	return
}

func (self *UpgradeService) DownloadAndExtractBinaryAsset(asset dto.BinaryAsset) (*UpdatePackage, errors.E) {
	currentExe, err := os.Executable()
	if err != nil {
		errWrapped := errors.Wrap(err, "failed to get current executable path")
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}
	currentExecutableName := filepath.Base(currentExe)

	tmpDir, err := os.MkdirTemp("", "srat_update_*")
	if err != nil {
		errWrapped := errors.Wrap(err, "failed to create temporary directory")
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}
	defer func() {
		os.RemoveAll(tmpDir)
	}()

	slog.InfoContext(self.ctx, "Starting download and extraction", "asset_name", asset.Name, "download_url", asset.BrowserDownloadURL, "temp_dir", tmpDir)

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

	downloadedFilePath := filepath.Join(tmpDir, asset.Name)
	downloadedFile, err := os.Create(downloadedFilePath)
	if err != nil {
		errWrapped := errors.Wrapf(err, "failed to create file for downloaded asset %s", downloadedFilePath)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}
	defer downloadedFile.Close()

	pr := &progressReader{
		reader:    resp.Body,
		totalSize: resp.ContentLength,
		onProgress: func(percentage int) {
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING, Progress: percentage})
		},
		lastReportedPercentage: 0,
	}

	slog.DebugContext(self.ctx, "Downloading asset", "url", asset.BrowserDownloadURL, "destination", downloadedFilePath, "size", resp.ContentLength)
	_, err = io.Copy(downloadedFile, pr)
	if err != nil {
		errWrapped := errors.Wrapf(err, "failed to write downloaded asset to file %s", downloadedFilePath)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}
	// Ensure 100% is reported
	if pr.lastReportedPercentage != 100 && resp.ContentLength > 0 { // only if total size was known
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING, Progress: 100})
	}
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSDOWNLOADCOMPLETE, Progress: 100})
	slog.InfoContext(self.ctx, "Asset downloaded successfully", "path", downloadedFilePath)

	// --- Extraction Phase ---
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSEXTRACTING, Progress: 0})

	zipReader, err := zip.OpenReader(downloadedFilePath)
	if err != nil {
		errWrapped := errors.Wrapf(err, "failed to open downloaded zip asset %s", downloadedFilePath)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}
	defer zipReader.Close()

	totalFiles := len(zipReader.File)
	var extractedFilesCount int
	var executablePath *string
	var foundPaths []string

	slog.DebugContext(self.ctx, "Extracting asset", "source_zip", downloadedFilePath, "total_files", totalFiles)
	extractPercentage := 0

	for _, f := range zipReader.File {
		targetPath := filepath.Join(self.state.UpdateDataDir, f.Name)

		// Path traversal mitigation: Ensure the path is within the temp directory
		if !strings.HasPrefix(targetPath, filepath.Clean(self.state.UpdateDataDir)+string(os.PathSeparator)) {
			errWrapped := errors.Errorf("invalid file path in zip: '%s' attempts to escape temporary directory", f.Name)
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
			return nil, errWrapped
		}

		if f.FileInfo().IsDir() {
			// Use a safe default when creating directories; adjust later if needed
			if err := os.MkdirAll(targetPath, 0o750); err != nil {
				errWrapped := errors.Wrapf(err, "failed to create directory %s during extraction", targetPath)
				self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
				return nil, errWrapped
			}
		} else {
			// Ensure parent directory exists with safe permissions
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
				errWrapped := errors.Wrapf(err, "failed to create parent directory for %s during extraction", targetPath)
				self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
				return nil, errWrapped
			}

			srcFile, errOpen := f.Open()
			if errOpen != nil {
				errWrapped := errors.Wrapf(errOpen, "failed to open file %s from zip archive", f.Name)
				self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
				return nil, errWrapped
			}

			destFile, errCreate := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if errCreate != nil {
				srcFile.Close()
				errWrapped := errors.Wrapf(errCreate, "failed to create destination file %s during extraction", targetPath)
				self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
				return nil, errWrapped
			}

			if _, errCopy := io.Copy(destFile, srcFile); errCopy != nil {
				srcFile.Close()
				destFile.Close()
				errWrapped := errors.Wrapf(errCopy, "failed to copy content to %s during extraction", targetPath)
				self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
				return nil, errWrapped
			}
			srcFile.Close()
			destFile.Close()

			if filepath.Base(targetPath) == currentExecutableName {
				slog.InfoContext(self.ctx, "Found matching executable in archive", "path", targetPath, "current_exe_name", currentExecutableName)
				executablePath = &targetPath
			} else {
				foundPaths = append(foundPaths, targetPath)
			}
		}
		extractedFilesCount++
		if totalFiles > 0 {
			extractPercentage = (extractedFilesCount * 100) / totalFiles
		}
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSEXTRACTING, Progress: extractPercentage})
	}

	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSEXTRACTCOMPLETE, Progress: 100})
	slog.InfoContext(self.ctx, "Asset extracted successfully", "temp_dir", tmpDir)

	return &UpdatePackage{
		CurrentExecutablePath: executablePath,
		OtherFilesPaths:       foundPaths,
	}, nil
}

func (self *UpgradeService) installBinaryTo(newExecutablePath string, destinationFile string) errors.E {
	if newExecutablePath == "" {
		return errors.New("invalid new executable path")
	}
	if destinationFile == "" {
		return errors.New("invalid destination file path")
	}
	slog.InfoContext(self.ctx, "Starting in-place update installation", "new_executable", newExecutablePath)
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSINSTALLING, Progress: 0})

	// Perform the update using standard library functions
	slog.InfoContext(self.ctx, "Applying update...", "target_executable", destinationFile, "source_new_executable", newExecutablePath)

	// Step 1: Open the new executable file
	newExeFile, err := os.Open(newExecutablePath)
	if err != nil {
		errWrapped := errors.Wrapf(err, "failed to open new executable file: %s", newExecutablePath)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return errWrapped
	}
	defer newExeFile.Close()

	// Step 2: Create a temporary file in the same directory as the target
	// This ensures atomic rename works (must be on same filesystem)
	targetDir := filepath.Dir(destinationFile)
	tempFile, err := os.CreateTemp(targetDir, ".srat_update_*")
	if err != nil {
		errWrapped := errors.Wrapf(err, "failed to create temporary file in %s", targetDir)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return errWrapped
	}
	tempPath := tempFile.Name()
	defer func() {
		// Clean up temp file if it still exists (error case)
		if _, err := os.Stat(tempPath); err == nil {
			os.Remove(tempPath)
		}
	}()

	// Step 3: Copy the new binary to the temp file
	_, err = io.Copy(tempFile, newExeFile)
	tempFile.Close() // Close before chmod
	if err != nil {
		errWrapped := errors.Wrapf(err, "failed to copy new executable to temp file %s", tempPath)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return errWrapped
	}

	// Step 4: Make the temp file executable
	err = os.Chmod(tempPath, 0755)
	if err != nil {
		errWrapped := errors.Wrapf(err, "failed to make temp file executable: %s", tempPath)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return errWrapped
	}

	// Step 5: Backup the old binary (if it exists)
	oldSavePath := destinationFile + ".old"
	if _, err := os.Stat(destinationFile); err == nil {
		// Destination exists, back it up
		// Remove any existing backup first
		os.Remove(oldSavePath)
		if err := os.Rename(destinationFile, oldSavePath); err != nil {
			// Log warning but don't fail - backup is optional
			slog.WarnContext(self.ctx, "Failed to backup old executable", "destination", destinationFile, "backup", oldSavePath, "error", err)
		} else {
			slog.InfoContext(self.ctx, "Backed up old executable", "backup", oldSavePath)
		}
	}

	// Step 6: Atomically rename temp file to destination
	// This works even if the destination is currently running on Unix systems
	err = os.Rename(tempPath, destinationFile)
	if err != nil {
		// Try to restore backup on error
		if _, statErr := os.Stat(oldSavePath); statErr == nil {
			if restoreErr := os.Rename(oldSavePath, destinationFile); restoreErr != nil {
				slog.ErrorContext(self.ctx, "Failed to restore backup after rename failure", "error", restoreErr)
			}
		}
		errWrapped := errors.Wrapf(err, "failed to apply in-place update to %s from %s", destinationFile, tempPath)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return errWrapped
	}

	slog.InfoContext(self.ctx, "In-place update applied successfully. Application will need to be restarted.", "updated_executable", destinationFile)
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSINSTALLCOMPLETE, Progress: 100})
	return nil
}

func (self *UpgradeService) InstallUpdatePackage(updatePkg *UpdatePackage) errors.E {
	if updatePkg == nil || updatePkg.CurrentExecutablePath == nil || *updatePkg.CurrentExecutablePath == "" {
		err := errors.New("invalid update package or missing executable path")
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: err.Error()})
		return err
	}

	basePath, err := self.getCurrentExecutablePath()
	if err != nil {
		return err
	}

	for i := range updatePkg.OtherFilesPaths {
		err := self.installBinaryTo(updatePkg.OtherFilesPaths[i], *basePath+"/"+filepath.Base(updatePkg.OtherFilesPaths[i]))
		if err != nil {
			return err
		}
	}
	err = self.installBinaryTo(*updatePkg.CurrentExecutablePath, *basePath+"/"+filepath.Base(*updatePkg.CurrentExecutablePath))
	if err != nil {
		return err
	}
	if self.state.UpdateFilePath != "" {
		err = self.installBinaryTo(*updatePkg.CurrentExecutablePath, self.state.UpdateFilePath)
		if err != nil {
			return err
		}
	}
	return nil
}

// ApplyUpdateAndRestart applies the update using selfupdate with signature verification
// and restarts the process if running under s6
func (self *UpgradeService) ApplyUpdateAndRestart(updatePkg *UpdatePackage) errors.E {
	if updatePkg == nil || updatePkg.CurrentExecutablePath == nil || *updatePkg.CurrentExecutablePath == "" {
		err := errors.New("invalid update package or missing executable path")
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: err.Error()})
		return err
	}

	slog.InfoContext(self.ctx, "Applying update with signature verification", "new_executable", *updatePkg.CurrentExecutablePath)
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSINSTALLING, Progress: 0})

	// Open the new binary file
	newBinary, err := os.Open(*updatePkg.CurrentExecutablePath)
	if err != nil {
		errWrapped := errors.Wrapf(err, "failed to open new binary: %s", *updatePkg.CurrentExecutablePath)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return errWrapped
	}
	defer newBinary.Close()

	// Check if signature file exists (for signed updates)
	signatureFile := *updatePkg.CurrentExecutablePath + ".minisig"
	var opts selfupdate.Options
	if _, statErr := os.Stat(signatureFile); statErr == nil {
		slog.InfoContext(self.ctx, "Signature file found, will verify update", "signature_file", signatureFile)

		// Create verifier with embedded public key
		verifier := selfupdate.NewVerifier()
		if err := verifier.LoadFromFile(signatureFile, updatekey.UpdatePublicKey); err != nil {
			errWrapped := errors.Wrapf(err, "failed to load signature from %s", signatureFile)
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
			return errWrapped
		}

		opts = selfupdate.Options{
			Verifier: verifier,
		}
	} else {
		// Signature file not found
		if self.state.UpdateChannel == dto.UpdateChannels.DEVELOP {
			slog.WarnContext(self.ctx, "Signature file not found, proceeding without verification (develop channel)", "expected_signature_file", signatureFile)
			opts = selfupdate.Options{}
		} else {
			// For non-develop channels, reject unsigned updates
			errWrapped := errors.Errorf("signature file not found for non-develop channel: %s", signatureFile)
			self.notifyClient(dto.UpdateProgress{
				ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR,
				ErrorMessage:   "Update is not signed. Signature verification is required for this update channel.",
			})
			slog.ErrorContext(self.ctx, "Update rejected: signature file missing for non-develop channel", "channel", self.state.UpdateChannel.String(), "expected_file", signatureFile)
			return errWrapped
		}
	}

	// Apply the update
	err = selfupdate.Apply(newBinary, opts)
	if err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			errWrapped := errors.Wrapf(err, "failed to apply update and rollback failed: %v", rerr)
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
			return errWrapped
		}
		errWrapped := errors.Wrap(err, "failed to apply update (rolled back successfully)")
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return errWrapped
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

func (self *UpgradeService) getCurrentExecutablePath() (*string, errors.E) {
	currentExe, err := os.Executable()
	if err != nil {
		errWrapped := errors.Wrap(err, "failed to get current executable path for in-place update")
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}
	destinationFile := filepath.Dir(currentExe)
	return &destinationFile, nil
}

/*

func (self *UpgradeService) InstallOverseerUpdate(updatePkg *UpdatePackage, overseerUpdatePath string) errors.E {
	if updatePkg == nil || updatePkg.CurrentExecutablePath == nil || *updatePkg.CurrentExecutablePath == "" {
		err := errors.New("invalid update package or missing executable path for overseer update")
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: err.Error()})
		return err
	}
	if overseerUpdatePath == "" {
		err := errors.New("overseer update path cannot be empty")
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: err.Error()})
		return err
	}
	basePath, err := self.getCurrentExecutablePath()
	if err != nil {
		return err
	}

	for i := range updatePkg.OtherFilesPaths {
		err := self.installBinaryTo(updatePkg.OtherFilesPaths[i], *basePath+"/"+filepath.Base(updatePkg.OtherFilesPaths[i]))
		if err != nil {
			return err
		}
	}
	return self.installBinaryTo(*updatePkg.CurrentExecutablePath, overseerUpdatePath)
}
*/

/*
// findLatestLocalBinaries searches for the newest binary matching the pattern in the search directory
// that is newer than the current executable.
func (self *UpgradeService) findLatestLocalBinaries(searchDir string, basePattern string) (filePath []string, modTime time.Time, errRet errors.E) {
	currentExePath, err := os.Executable()
	if err != nil {
		return []string{}, time.Time{}, errors.Wrap(err, "failed to get current executable path")
	}
	currentExeStat, err := os.Stat(currentExePath)
	if err != nil {
		return []string{}, time.Time{}, errors.Wrap(err, "failed to stat current executable")
	}
	currentExeModTime := currentExeStat.ModTime()

	slog.DebugContext(self.ctx, "Searching for local update binary", "searchDir", searchDir, "pattern", basePattern+"*", "currentExeModTime", currentExeModTime)

	var latestFilePaths []string
	var latestModTime time.Time

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		if os.IsNotExist(err) {
			slog.WarnContext(self.ctx, "Local update directory does not exist", "dir", searchDir)
			return []string{}, time.Time{}, errors.WithMessagef(dto.ErrorNoUpdateAvailable, "local update directory %s not found", searchDir)
		}
		return []string{}, time.Time{}, errors.Wrapf(err, "failed to read local update directory %s", searchDir)
	}

	foundCandidate := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), basePattern) {
			slog.DebugContext(self.ctx, "Skipping candidate local update binary", "path", entry.Name())
			continue
		}

		fullPath := filepath.Join(searchDir, entry.Name())
		info, errStat := entry.Info()
		if errStat != nil {
			slog.WarnContext(self.ctx, "Failed to stat candidate local update file", "path", fullPath, "error", errStat)
			continue
		}
		if info.Mode().IsRegular() && info.ModTime().After(currentExeModTime) {
			slog.DebugContext(self.ctx, "Found potential local update binary", "path", fullPath, "modTime", info.ModTime())
			latestFilePaths = append(latestFilePaths, fullPath)
			latestModTime = info.ModTime()
			foundCandidate = true
			slog.DebugContext(self.ctx, "New latest local update binary candidate", "path", latestFilePaths, "modTime", latestModTime)
		} else {
			slog.DebugContext(self.ctx, "Ignore file", "path", fullPath, "info", info)
		}
	}

	if !foundCandidate {
		slog.InfoContext(self.ctx, "No new local update binary found", "searchDir", searchDir, "pattern", basePattern+"*")
		return []string{}, time.Time{}, errors.WithMessage(dto.ErrorNoUpdateAvailable, "no new local update binary found")
	}

	slog.InfoContext(self.ctx, "Latest local update binary selected", "path", latestFilePaths, "modTime", latestModTime)
	return latestFilePaths, latestModTime, nil
}
*/
/*

func (self *UpgradeService) InstallUpdateLocal(updateChannel *dto.UpdateChannel) errors.E {
	if updateChannel == nil {
		updateChannel = self.state.UpdateChannel
	}

	if updateChannel == nil || *updateChannel != dto.UpdateChannels.DEVELOP {
		err := errors.Errorf("local updates are only allowed on the DEVELOP update channel not %s", updateChannel.String())
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: err.Error()})
		return err
	}

	slog.InfoContext(self.ctx, "Starting local update process.")
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSCHECKING})

	foundFilePaths, _, errFind := self.findLatestLocalBinaries(localUpdateDir, localUpdatePattern)
	if errFind != nil {
		errMsg := fmt.Sprintf("Local update: %s", errFind.Error())
		if errors.Is(errFind, dto.ErrorNoUpdateAvailable) {
			slog.InfoContext(self.ctx, "No local update found or directory missing.")
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSNOUPGRDE, ErrorMessage: errMsg})
		} else {
			slog.ErrorContext(self.ctx, "Error finding local update binary.", "error", errFind)
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errMsg})
		}
		return errFind
	}

	var aerr errors.E = nil

	for _, foundFilePath := range foundFilePaths {
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE, LastRelease: filepath.Base(foundFilePath)})
		updatePkg := &UpdatePackage{CurrentExecutablePath: &foundFilePath, OtherFilesPaths: []string{}, TempDirPath: filepath.Dir(foundFilePath)}
		slog.InfoContext(self.ctx, "Prepared local update package", "executable", *updatePkg.CurrentExecutablePath)
		err := self.InstallUpdatePackage(updatePkg)
		if err != nil {
			slog.ErrorContext(self.ctx, "Error installing local update package", "error", err)
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: err.Error()})
			aerr = errors.WithStack(err)
		}
	}
	return aerr
}
*/

/*
func (self *UpgradeService) InstallOverseerLocal(overseerUpdatePath string) errors.E {
	if self.updateChannel == nil || *self.updateChannel != dto.UpdateChannels.DEVELOP {
		err := errors.New("local overseer updates are only allowed on the DEVELOP update channel")
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: err.Error()})
		return err
	}
	if overseerUpdatePath == "" {
		err := errors.New("overseer update path cannot be empty for local overseer update")
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: err.Error()})
		return err
	}
	slog.InfoContext(self.ctx, "Starting local overseer update process.", "overseerPath", overseerUpdatePath)
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSCHECKING})

	foundFilePath, _, errFind := self.findLatestLocalBinary(localUpdateDir, localUpdatePattern)
	if errFind != nil { // Error handling similar to InstallUpdateLocal
		errMsg := fmt.Sprintf("Local overseer update: %s", errFind.Error())
		status := dto.UpdateProcessStates.UPDATESTATUSERROR
		if errors.Is(errFind, dto.ErrorNoUpdateAvailable) {
			status = dto.UpdateProcessStates.UPDATESTATUSNOUPGRDE
		}
		self.notifyClient(dto.UpdateProgress{ProgressStatus: status, ErrorMessage: errMsg})
		return errFind
	}

	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE, LastRelease: filepath.Base(foundFilePath)})
	updatePkg := &UpdatePackage{CurrentExecutablePath: &foundFilePath, OtherFilesPaths: []string{}, TempDirPath: filepath.Dir(foundFilePath)}
	slog.InfoContext(self.ctx, "Prepared local overseer update package", "executable", *updatePkg.CurrentExecutablePath)
	return self.InstallOverseerUpdate(updatePkg, overseerUpdatePath)
}
*/
