package service

import (
	"archive/zip"
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
	"github.com/dianlight/srat/repository"
	"github.com/google/go-github/v75/github"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"golang.org/x/time/rate"
)

type UpdatePackage struct {
	CurrentExecutablePath *string
	OtherFilesPaths       []string
	TempDirPath           string
}

type UpgradeServiceInterface interface {
	GetUpgradeReleaseAsset(updateChannel *dto.UpdateChannel) (ass *dto.ReleaseAsset, err errors.E)
	DownloadAndExtractBinaryAsset(asset dto.BinaryAsset) (*UpdatePackage, errors.E)
	InstallUpdatePackage(updatePkg *UpdatePackage) errors.E
	InstallOverseerUpdate(updatePkg *UpdatePackage, overseerUpdatePath string) errors.E
	InstallUpdateLocal(updateChannel *dto.UpdateChannel) errors.E
	//InstallOverseerLocal(overseerUpdatePath string) errors.E
}

type UpgradeService struct {
	ctx           context.Context
	gh            *github.Client
	broadcaster   BroadcasterServiceInterface
	updateLimiter rate.Sometimes
	props_repo    repository.PropertyRepositoryInterface
	updateChannel *dto.UpdateChannel
}

const (
	localUpdateDir     = "/config"
	localUpdatePattern = "srat-"
)

type UpgradeServiceProps struct {
	fx.In
	PropsRepo   repository.PropertyRepositoryInterface
	Broadcaster BroadcasterServiceInterface
	Ctx         context.Context
	Gh          *github.Client
}

func NewUpgradeService(lc fx.Lifecycle, in UpgradeServiceProps) UpgradeServiceInterface {
	p := new(UpgradeService)
	p.updateLimiter = rate.Sometimes{Interval: 30 * time.Minute}
	p.ctx = in.Ctx
	p.broadcaster = in.Broadcaster
	p.props_repo = in.PropsRepo
	p.gh = in.Gh
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			value, err := p.props_repo.Value("UpdateChannel", false)
			if err != nil || value == nil {
				p.updateChannel = &dto.UpdateChannels.NONE
			} else {
				p.updateChannel = &dto.UpdateChannel{}
				errS := p.updateChannel.Scan(value)
				if errS != nil {
					slog.Warn("Unable to convert config value", "value", value, "type", fmt.Sprintf("%T", value), "err", errS)
					p.updateChannel = &dto.UpdateChannels.NONE
				}
			}
			p.ctx.Value("wg").(*sync.WaitGroup).Add(1)
			go func() {
				defer p.ctx.Value("wg").(*sync.WaitGroup).Done()
				p.run()
			}()
			return nil
		},
	})

	return p
}

func (self *UpgradeService) run() error {
	for {
		select {
		case <-self.ctx.Done():
			slog.Info("Run process closed", "err", self.ctx.Err())
			return errors.WithStack(self.ctx.Err())
		default:
			self.updateLimiter.Do(func() {
				slog.Debug("Version Checking...")
				self.notifyClient(dto.UpdateProgress{
					ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSCHECKING,
				})
				ass, err := self.GetUpgradeReleaseAsset(nil)
				if err != nil && !errors.Is(err, dto.ErrorNoUpdateAvailable) {
					slog.Error("Error checking for updates", "err", err)
				}
				if ass != nil {
					self.notifyClient(dto.UpdateProgress{
						ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE,
						LastRelease:    ass.LastRelease,
					})
				} else {
					self.notifyClient(dto.UpdateProgress{
						ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSNOUPGRDE,
					})
				}
			})
			time.Sleep(self.updateLimiter.Interval / 10)
		}
	}
}

func (self *UpgradeService) GetUpgradeReleaseAsset(updateChannel *dto.UpdateChannel) (ass *dto.ReleaseAsset, err errors.E) {
	if updateChannel == nil {
		updateChannel = self.updateChannel
	}

	if updateChannel != &dto.UpdateChannels.NONE && updateChannel != &dto.UpdateChannels.DEVELOP {
		myversion, err := semver.NewVersion(config.Version)
		if err != nil {
			slog.Error("Error parsing version", "current", config.Version, "err", err)
			return nil, errors.WithStack(err)
		}

		slog.Debug("Checking for updates...", "channel", updateChannel.String())
		releases, _, err := self.gh.Repositories.ListReleases(context.Background(), "dianlight", "srat", &github.ListOptions{
			Page:    1,
			PerPage: 5,
		})
		if err != nil {
			if _, ok := err.(*github.RateLimitError); ok {
				slog.Warn("Github API hit rate limit")
			}
			slog.Warn("Error getting releases", "err", err)
			return nil, errors.WithMessage(dto.ErrorNoUpdateAvailable, "No releases found")
		} else if len(releases) > 0 {
			for _, release := range releases {
				//log.Println(pretty.Sprintf("%v\n", release))
				if *release.Prerelease && updateChannel != &dto.UpdateChannels.PRERELEASE {
					//log.Printf("Skip Release %s", *release.TagName)
					continue
				}

				assertVersion, err := semver.NewVersion(*release.TagName)
				if err != nil {
					slog.Warn("Error parsing version", "version", *release.TagName, "err", err)
					continue
				}
				slog.Debug("Checking version", "current", config.Version, "release", *release.TagName)

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
					}
				}
			}
			if ass != nil {
				return ass, nil
			} else {
				return nil, errors.WithMessage(dto.ErrorNoUpdateAvailable, "No releases found")
			}
		} else {
			slog.Debug("No Releases found")
			return nil, errors.WithMessage(dto.ErrorNoUpdateAvailable, "No releases found")
		}
	} else {
		return nil, errors.WithMessage(dto.ErrorNoUpdateAvailable, "No releases check")
	}
}

func (self *UpgradeService) notifyClient(data dto.UpdateProgress) {
	self.broadcaster.BroadcastMessage(data)
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

	var success bool = false
	tmpDir, err := os.MkdirTemp("", "srat_update_*")
	if err != nil {
		errWrapped := errors.Wrap(err, "failed to create temporary directory")
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}
	defer func() {
		if !success {
			os.RemoveAll(tmpDir)
		}
	}()

	slog.Info("Starting download and extraction", "asset_name", asset.Name, "download_url", asset.BrowserDownloadURL, "temp_dir", tmpDir)

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

	slog.Debug("Downloading asset", "url", asset.BrowserDownloadURL, "destination", downloadedFilePath, "size", resp.ContentLength)
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
	slog.Info("Asset downloaded successfully", "path", downloadedFilePath)

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

	slog.Debug("Extracting asset", "source_zip", downloadedFilePath, "total_files", totalFiles)
	for _, f := range zipReader.File {
		targetPath := filepath.Join(tmpDir, f.Name)

		// Path traversal mitigation: Ensure the path is within the temp directory
		if !strings.HasPrefix(targetPath, filepath.Clean(tmpDir)+string(os.PathSeparator)) {
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
				slog.Info("Found matching executable in archive", "path", targetPath, "current_exe_name", currentExecutableName)
				executablePath = &targetPath
			} else {
				foundPaths = append(foundPaths, targetPath)
			}
		}
		extractedFilesCount++
		extractPercentage := 0
		if totalFiles > 0 {
			extractPercentage = (extractedFilesCount * 100) / totalFiles
		}
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSEXTRACTING, Progress: extractPercentage})
	}

	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSEXTRACTCOMPLETE, Progress: 100})
	slog.Info("Asset extracted successfully", "temp_dir", tmpDir)

	success = true // Mark as successful so defer doesn't clean up tmpDir
	return &UpdatePackage{
		CurrentExecutablePath: executablePath,
		OtherFilesPaths:       foundPaths,
		TempDirPath:           tmpDir,
	}, nil
}

func (self *UpgradeService) installBinaryTo(newExecutablePath string, destinationFile string) errors.E {
	if newExecutablePath == "" {
		return errors.New("invalid new executable path")
	}
	if destinationFile == "" {
		return errors.New("invalid destination file path")
	}
	slog.Info("Starting in-place update installation", "new_executable", newExecutablePath)
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSINSTALLING, Progress: 0})

	// Perform the update using standard library functions
	slog.Info("Applying update...", "target_executable", destinationFile, "source_new_executable", newExecutablePath)
	
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
			slog.Warn("Failed to backup old executable", "destination", destinationFile, "backup", oldSavePath, "error", err)
		} else {
			slog.Info("Backed up old executable", "backup", oldSavePath)
		}
	}

	// Step 6: Atomically rename temp file to destination
	// This works even if the destination is currently running on Unix systems
	err = os.Rename(tempPath, destinationFile)
	if err != nil {
		// Try to restore backup on error
		if _, statErr := os.Stat(oldSavePath); statErr == nil {
			if restoreErr := os.Rename(oldSavePath, destinationFile); restoreErr != nil {
				slog.Error("Failed to restore backup after rename failure", "error", restoreErr)
			}
		}
		errWrapped := errors.Wrapf(err, "failed to apply in-place update to %s from %s", destinationFile, tempPath)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return errWrapped
	}

	slog.Info("In-place update applied successfully. Application will need to be restarted.", "updated_executable", destinationFile)
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSINSTALLCOMPLETE, Progress: 100})
	return nil
}

func (self *UpgradeService) InstallUpdatePackage(updatePkg *UpdatePackage) errors.E {
	if updatePkg == nil || updatePkg.CurrentExecutablePath == nil || *updatePkg.CurrentExecutablePath == "" {
		err := errors.New("invalid update package or missing executable path for overseer update")
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
	return self.installBinaryTo(*updatePkg.CurrentExecutablePath, *basePath+"/"+filepath.Base(*updatePkg.CurrentExecutablePath))
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

	slog.Debug("Searching for local update binary", "searchDir", searchDir, "pattern", basePattern+"*", "currentExeModTime", currentExeModTime)

	var latestFilePaths []string
	var latestModTime time.Time

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("Local update directory does not exist", "dir", searchDir)
			return []string{}, time.Time{}, errors.WithMessagef(dto.ErrorNoUpdateAvailable, "local update directory %s not found", searchDir)
		}
		return []string{}, time.Time{}, errors.Wrapf(err, "failed to read local update directory %s", searchDir)
	}

	foundCandidate := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), basePattern) {
			slog.Debug("Skipping candidate local update binary", "path", entry.Name())
			continue
		}

		fullPath := filepath.Join(searchDir, entry.Name())
		info, errStat := entry.Info()
		if errStat != nil {
			slog.Warn("Failed to stat candidate local update file", "path", fullPath, "error", errStat)
			continue
		}
		if info.Mode().IsRegular() && info.ModTime().After(currentExeModTime) {
			slog.Debug("Found potential local update binary", "path", fullPath, "modTime", info.ModTime())
			latestFilePaths = append(latestFilePaths, fullPath)
			latestModTime = info.ModTime()
			foundCandidate = true
			slog.Debug("New latest local update binary candidate", "path", latestFilePaths, "modTime", latestModTime)
		} else {
			slog.Debug("Ignore file", "path", fullPath, "info", info)
		}
	}

	if !foundCandidate {
		slog.Info("No new local update binary found", "searchDir", searchDir, "pattern", basePattern+"*")
		return []string{}, time.Time{}, errors.WithMessage(dto.ErrorNoUpdateAvailable, "no new local update binary found")
	}

	slog.Info("Latest local update binary selected", "path", latestFilePaths, "modTime", latestModTime)
	return latestFilePaths, latestModTime, nil
}

func (self *UpgradeService) InstallUpdateLocal(updateChannel *dto.UpdateChannel) errors.E {
	if updateChannel == nil {
		updateChannel = self.updateChannel
	}

	if updateChannel == nil || *updateChannel != dto.UpdateChannels.DEVELOP {
		err := errors.Errorf("local updates are only allowed on the DEVELOP update channel not %s", updateChannel.String())
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: err.Error()})
		return err
	}

	slog.Info("Starting local update process.")
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSCHECKING})

	foundFilePaths, _, errFind := self.findLatestLocalBinaries(localUpdateDir, localUpdatePattern)
	if errFind != nil {
		errMsg := fmt.Sprintf("Local update: %s", errFind.Error())
		if errors.Is(errFind, dto.ErrorNoUpdateAvailable) {
			slog.Info("No local update found or directory missing.")
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSNOUPGRDE, ErrorMessage: errMsg})
		} else {
			slog.Error("Error finding local update binary.", "error", errFind)
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errMsg})
		}
		return errFind
	}

	var aerr errors.E = nil

	for _, foundFilePath := range foundFilePaths {
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE, LastRelease: filepath.Base(foundFilePath)})
		updatePkg := &UpdatePackage{CurrentExecutablePath: &foundFilePath, OtherFilesPaths: []string{}, TempDirPath: filepath.Dir(foundFilePath)}
		slog.Info("Prepared local update package", "executable", *updatePkg.CurrentExecutablePath)
		err := self.InstallUpdatePackage(updatePkg)
		if err != nil {
			slog.Error("Error installing local update package", "error", err)
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: err.Error()})
			aerr = errors.WithStack(err)
		}
	}
	return aerr
}

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
	slog.Info("Starting local overseer update process.", "overseerPath", overseerUpdatePath)
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
	slog.Info("Prepared local overseer update package", "executable", *updatePkg.CurrentExecutablePath)
	return self.InstallOverseerUpdate(updatePkg, overseerUpdatePath)
}
*/
