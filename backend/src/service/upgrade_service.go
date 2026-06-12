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
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/internal/updatekey"
	"github.com/dianlight/srat/internal/urlutil"
	"github.com/dianlight/tlog"
	"github.com/fsnotify/fsnotify"
	"github.com/google/go-github/v88/github"
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
				if wg, ok := p.ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup); ok && wg != nil {
					wg.Go(func() {
						err := p.updater()
						if err != nil {
							slog.ErrorContext(p.ctx, "Error in run loop", "err", err)
						}
					})
				}

				// Start file watcher for develop channel
				if p.state.UpdateChannel == dto.UpdateChannels.DEVELOP {
					if wg, ok := p.ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup); ok && wg != nil {
						wg.Go(func() {
							p.watchForDevelopUpdates()
						})
					}
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
			return nil
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

// developWatchNames is the fixed set of binary names watched in develop channel.
// All three server variants plus srat-cli are watched so that any build:remote
// deployment (regardless of --variant) triggers an update.
var developWatchNames = map[string]struct{}{
	"srat-cli":           {},
	"srat-server-static": {},
	"srat-server-musl":   {},
	"srat-server-glib":   {},
}

// watchForDevelopUpdates watches UpdateDataDir for any of the known binary variant
// files and installs them when changed. All three server variants (static, musl,
// glib) and srat-cli are watched so every build:remote --variant=* deployment
// triggers the update regardless of which variant is currently running.
func (self *UpgradeService) watchForDevelopUpdates() {
	if self.state.UpdateDataDir == "" {
		slog.WarnContext(self.ctx, "UpdateDataDir not set, file watcher for develop updates disabled")
		return
	}

	slog.InfoContext(self.ctx, "Starting file watcher for develop channel updates using fsnotify", "watch_dir", self.state.UpdateDataDir)

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

	// Seed last-seen modification times for all watched files so we don't
	// re-install binaries that were already present when the watcher starts.
	seenModTimes := make(map[string]time.Time)
	for name := range developWatchNames {
		p := filepath.Join(self.state.UpdateDataDir, name)
		if info, statErr := os.Stat(p); statErr == nil {
			seenModTimes[name] = info.ModTime()
		}
	}

	// pendingMu protects pendingFiles, which accumulates names of files that
	// arrived since the last debounce install. The event loop writes it;
	// the AfterFunc callback reads+clears it — both under the mutex.
	var pendingMu sync.Mutex
	pendingFiles := make(map[string]struct{})

	// Debounce timer: a single timer is reset for every incoming event so that
	// a rapid burst (e.g. rsync of srat-cli + srat-server-musl) is batched.
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

			baseName := filepath.Base(event.Name)
			if _, watched := developWatchNames[baseName]; !watched {
				continue
			}

			if event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Create == 0 {
				continue
			}

			filePath := filepath.Join(self.state.UpdateDataDir, baseName)
			info, statErr := os.Stat(filePath)
			if statErr != nil {
				slog.ErrorContext(self.ctx, "Failed to stat watched file", "file", filePath, "err", statErr)
				continue
			}

			if !info.ModTime().After(seenModTimes[baseName]) || info.Size() == 0 {
				continue
			}

			// Mark this file as pending and (re)set the debounce timer.
			pendingMu.Lock()
			pendingFiles[baseName] = struct{}{}
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDelay, func() {
				// Drain the pending set under lock.
				pendingMu.Lock()
				toInstall := make([]string, 0, len(pendingFiles))
				for name := range pendingFiles {
					toInstall = append(toInstall, name)
				}
				pendingFiles = make(map[string]struct{})
				pendingMu.Unlock()

				if len(toInstall) == 0 {
					return
				}

				// Build UpdatePackage from all changed files, updating seenModTimes.
				var files []UpdateFile
				for _, name := range toInstall {
					p := filepath.Join(self.state.UpdateDataDir, name)
					fi, statErr := os.Stat(p)
					if statErr != nil {
						slog.WarnContext(self.ctx, "File gone after debounce, skipping", "file", p, "err", statErr)
						continue
					}
					seenModTimes[name] = fi.ModTime()
					files = append(files, UpdateFile{Path: p, Size: fi.Size()})
				}
				if len(files) == 0 {
					return
				}

				slog.InfoContext(self.ctx, "Detected updated files in develop channel", "files", toInstall)
				self.notifyClient(dto.UpdateProgress{
					ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE,
					Progress:       0,
					ReleaseAsset: &dto.ReleaseAsset{
						ArchAsset: dto.BinaryAsset{Name: toInstall[0], Size: int(files[0].Size)},
					},
				})

				if err := self.InstallUpdatePackage(&UpdatePackage{FilesPaths: files}); err != nil {
					slog.ErrorContext(self.ctx, "Failed to install develop update", "err", err)
					return
				}

				slog.InfoContext(self.ctx, "Triggering restart after develop update install")
				self.notifyClient(dto.UpdateProgress{
					ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSINSTALLCOMPLETE,
					Progress:       100,
					ErrorMessage:   "Update installed, restarting...",
				})

				time.Sleep(500 * time.Millisecond)

				if self.isRunningUnderS6() {
					slog.InfoContext(self.ctx, "Running under s6, initiating graceful shutdown for restart")
					if shutErr := self.shutdowner.Shutdown(); shutErr != nil {
						slog.ErrorContext(self.ctx, "Failed to trigger graceful shutdown", "err", shutErr)
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
			pendingMu.Unlock()

		case watchErr, ok := <-watcher.Errors:
			if !ok {
				slog.WarnContext(self.ctx, "File watcher errors channel closed")
				return
			}
			slog.ErrorContext(self.ctx, "File watcher error", "err", watchErr)
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
				tlog.TraceContext(self.ctx, "Found Release", "tag_name", release.GetTagName(), "prerelease", release.GetPrerelease())
				if release.GetPrerelease() && self.state.UpdateChannel.String() != dto.UpdateChannels.PRERELEASE.String() {
					tlog.TraceContext(self.ctx, "Skip PreRelease", "tag_name", release.GetTagName())
					continue
				}

				assertVersion, err := semver.NewVersion(*release.TagName)
				if err != nil {
					slog.WarnContext(self.ctx, "Error parsing version", "version", *release.TagName, "err", err)
					continue
				}
				tlog.TraceContext(self.ctx, "Checking version", "current", myversion, "release", *release.TagName, "compare", myversion.Compare(assertVersion))

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

	// Handle symlinks: extract them without requiring a signature comment.
	// Zip symlinks (stored with zip -y) have ModeSymlink set; their content is the link target.
	if f.FileInfo().Mode()&os.ModeSymlink != 0 {
		path := filepath.Join(dest, f.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) && dest != "." {
			return nil, errors.Errorf("illegal file path: %s", path)
		}

		if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
			return nil, errors.WithStack(err)
		}

		rc, err := f.Open()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		defer rc.Close()

		targetBytes, err := io.ReadAll(rc)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		target := strings.TrimSpace(string(targetBytes))

		// Remove any existing file or symlink at this path before creating the new one
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "failed to remove existing path for symlink: %s", path)
		}

		if err := os.Symlink(target, path); err != nil {
			return nil, errors.Wrapf(err, "failed to create symlink %s -> %s", path, target)
		}

		slog.DebugContext(self.ctx, "Extracted symlink", "path", path, "target", target)
		return &UpdateFile{
			Path:      path,
			Signature: nil, // Symlinks are not signed
			Size:      0,
		}, nil
	}

	if f.Comment == "" {
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

	size := f.UncompressedSize64
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
	err = self.verifier.Load([]byte(f.Comment), []byte(updatekey.UpdatePublicKey))
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
		Signature: []byte(f.Comment),
		Size:      f.FileInfo().Size(),
	}, errors.WithStack(err)
}

// DownloadAndExtractBinaryAsset downloads the binary asset from the given URL, extracts it to a UpdateDataDir, and returns the UpdatePackage to the extracted executable.
func (self *UpgradeService) DownloadAndExtractBinaryAsset(asset dto.BinaryAsset) (*UpdatePackage, errors.E) {
	slog.InfoContext(self.ctx, "Starting download and extraction", "asset_name", asset.Name, "download_url", asset.BrowserDownloadURL)

	// --- Download Phase ---
	self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING, Progress: 0})

	// Validate download URL to prevent SSRF (G704)
	if err := urlutil.ValidateURL(asset.BrowserDownloadURL, []string{"github.com", "objects.githubusercontent.com"}); err != nil {
		errWrapped := errors.Errorf("untrusted download URL: %s", asset.BrowserDownloadURL)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}

	req, err := http.NewRequestWithContext(self.ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		errWrapped := errors.Wrapf(err, "failed to create request for %s", asset.BrowserDownloadURL)
		self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
		return nil, errWrapped
	}

	resp, err := http.DefaultClient.Do(req) // #nosec G704
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
		if _, err := downloadedFilePath.Seek(0, io.SeekStart); err != nil { // Reset file pointer
			errWrapped := errors.Wrapf(err, "failed to reset downloaded file pointer")
			self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
			return nil, errWrapped
		}

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
			if _, err := downloadedFilePath.Seek(0, io.SeekStart); err != nil {
				errWrapped := errors.Wrapf(err, "failed to reset downloaded file pointer after checksum validation")
				self.notifyClient(dto.UpdateProgress{ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSERROR, ErrorMessage: errWrapped.Error()})
				return nil, errWrapped
			}
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
		if _, err := os.Lstat(path.Path); os.IsNotExist(err) {
			return errors.Errorf("update package file does not exist: %s", path.Path)
		}
	}

	currentFile, targetDir, err := self.getCurrentExecutablePath()
	if err != nil {
		return err
	}
	slog.InfoContext(self.ctx, "Installing update package", "target_directory", *targetDir)
	for _, path := range updatePkg.FilesPaths {
		// Skip symlink entries — they are recreated by updateServerSymlink after all binaries install.
		info, statErr := os.Lstat(path.Path)
		if statErr != nil {
			return errors.Wrapf(statErr, "failed to stat update file: %s", path.Path)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			slog.DebugContext(self.ctx, "Skipping symlink in install loop; will be updated after binary installation", "path", path.Path)
			continue
		}

		targetPath := filepath.Join(*targetDir, filepath.Base(path.Path))
		tracker := &progressTracker{
			Total:  uint64(path.Size),
			Target: targetPath,
			OnProgress: func(target string, progress, total uint64) {
				if total > 0 {
					percent := float64(progress) / float64(total) * 100
					percent = float64(int(percent*10)) / 10 // Round to 1 decimal place
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
				return errors.Errorf("binary version is same as current version %s", currentVersion.String())
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

	// After all binaries are installed, update the srat-server symlink to the best variant
	// for the current system (musl, glibc, or static fallback).
	if err := self.updateServerSymlink(*targetDir, updatePkg); err != nil {
		return errors.Wrap(err, "failed to update srat-server symlink")
	}

	return nil
}

// updateServerSymlink detects the best srat-server variant for the current system
// and atomically updates the srat-server symlink in targetDir to point to it.
func (self *UpgradeService) updateServerSymlink(targetDir string, updatePkg *UpdatePackage) error {
	variant := detectBestServerVariant(targetDir, updatePkg)
	symlinkPath := filepath.Join(targetDir, "srat-server")

	// Atomically replace the existing symlink via a temp path + os.Rename.
	// This avoids any window where srat-server is absent or points to the wrong target.
	tempPath := symlinkPath + ".tmp"
	if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean up temp symlink %s: %w", tempPath, err)
	}
	if err := os.Symlink(variant, tempPath); err != nil {
		return fmt.Errorf("failed to create temp symlink -> %s: %w", variant, err)
	}
	if err := os.Rename(tempPath, symlinkPath); err != nil {
		_ = os.Remove(tempPath) // best-effort cleanup on rename failure
		return fmt.Errorf("failed to atomically replace srat-server symlink: %w", err)
	}

	slog.InfoContext(self.ctx, "Updated srat-server symlink", "variant", variant, "path", symlinkPath)
	return nil
}

// detectBestServerVariant returns the filename of the best available srat-server variant
// for the current system: musl dynamic, glibc dynamic, or static (always safe fallback).
// It only considers variants that are present in the UpdatePackage to avoid selecting
// stale binaries from previous installations.
func detectBestServerVariant(targetDir string, updatePkg *UpdatePackage) string {
	arch := runtime.GOARCH

	// Build a set of variant names present in the update package
	packagedVariants := make(map[string]bool)
	if updatePkg != nil {
		for _, file := range updatePkg.FilesPaths {
			baseName := filepath.Base(file.Path)
			if baseName == "srat-server-musl" || baseName == "srat-server-glib" || baseName == "srat-server-static" {
				packagedVariants[baseName] = true
			}
		}
	}

	// Prefer musl dynamic variant if present in package and system has a musl dynamic linker
	if packagedVariants["srat-server-musl"] {
		if _, err := os.Stat(filepath.Join(targetDir, "srat-server-musl")); err == nil {
			var muslLinker string
			switch arch {
			case "amd64":
				muslLinker = "/lib/ld-musl-x86_64.so.1"
			case "arm64":
				muslLinker = "/lib/ld-musl-aarch64.so.1"
			}
			if muslLinker != "" {
				if _, err := os.Stat(muslLinker); err == nil {
					return "srat-server-musl"
				}
			}
		}
	}

	// Prefer glibc dynamic variant if present in package and system has a glibc dynamic linker
	if packagedVariants["srat-server-glib"] {
		if _, err := os.Stat(filepath.Join(targetDir, "srat-server-glib")); err == nil {
			glibcIndicators := []string{
				"/lib64/ld-linux-x86-64.so.2",      // x86_64 glibc dynamic linker
				"/lib/ld-linux-aarch64.so.1",       // aarch64 glibc dynamic linker
				"/lib/x86_64-linux-gnu/libc.so.6",  // Debian/Ubuntu x86_64
				"/lib/aarch64-linux-gnu/libc.so.6", // Debian/Ubuntu aarch64
			}
			for _, indicator := range glibcIndicators {
				if _, err := os.Stat(indicator); err == nil {
					return "srat-server-glib"
				}
			}
		}
	}

	// Static binary always works regardless of libc availability
	return "srat-server-static"
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
