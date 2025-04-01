package service

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/google/go-github/v69/github"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/time/rate"
)

type UpgradeServiceInterface interface {
	checkSoftwareVersion() error
	GetLastReleaseData() *dto.ReleaseAsset
}

type UpgradeService struct {
	ctx             context.Context
	gh              *github.Client
	broadcaster     BroadcasterServiceInterface
	lastReleaseData dto.ReleaseAsset
	updateLimiter   rate.Sometimes
	//updateQueue      map[string](chan *dto.ReleaseAsset)
	//updateQueueMutex sync.RWMutex
}

func NewUpgradeService(ctx context.Context, broadcaster BroadcasterServiceInterface) UpgradeServiceInterface {
	p := new(UpgradeService)
	p.updateLimiter = rate.Sometimes{Interval: 30 * time.Minute}
	p.ctx = ctx
	//p.updateQueue = make(map[string](chan *dto.ReleaseAsset))
	//p.updateQueueMutex = sync.RWMutex{}
	p.lastReleaseData = dto.ReleaseAsset{}
	p.broadcaster = broadcaster
	rateLimiter := github_ratelimit.NewClient(nil)
	p.gh = github.NewClient(rateLimiter)
	go p.run()
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
				err := self.checkSoftwareVersion()
				if err != nil {
					slog.Error("Error checking for updates: %w", "err", err)
				}
				err = self.EventEmitter(self.ctx, self.lastReleaseData)
				if err != nil {
					slog.Error("Error emitting vrsion message: %w", "err", err)
				}
			})
			time.Sleep(time.Second * 10)
		}
	}
}

func (self *UpgradeService) checkSoftwareVersion() error {
	var properties dbom.Properties
	properties.Load()
	value, err := properties.GetValue("UpdateChannel")
	if err != nil {
		return errors.WithStack(err)
	}
	updateChannel := dto.UpdateChannel(value.(string))

	if updateChannel != dto.None {
		slog.Debug("Checking for updates...", "channel", updateChannel)
		releases, _, err := self.gh.Repositories.ListReleases(context.Background(), "dianlight", "srat", &github.ListOptions{
			Page:    1,
			PerPage: 5,
		})
		if err != nil {
			if _, ok := err.(*github.RateLimitError); ok {
				slog.Warn("Github API hit rate limit")
			}
		} else if len(releases) > 0 {
			conv := converter.GitHubToDtoImpl{}
			for _, release := range releases {
				//log.Println(pretty.Sprintf("%v\n", release))
				if *release.Prerelease && updateChannel == dto.Stable {
					//log.Printf("Skip Prerelease %s", *release.TagName)
					continue
				} else if !*release.Prerelease && updateChannel == dto.Prerelease {
					//log.Printf("Skip Release %s", *release.TagName)
					continue
				}
				self.lastReleaseData.LastRelease = *release.TagName
				// Serch for the asset corrisponfing the correct architecture
				for _, asset := range release.Assets {
					arch := runtime.GOARCH
					if arch == "arm64" {
						arch = "aarch64"
					}
					if asset.GetName() == fmt.Sprintf("srat_%s", arch) {
						err = conv.ReleaseAssetToBinaryAsset(asset, &self.lastReleaseData.ArchAsset)
						if err != nil {
							return errors.WithStack(err)
						}
						break
					}
				}
				break
			}
		} else {
			slog.Debug("No Releases found")
			self.lastReleaseData = dto.ReleaseAsset{}
		}
	} else {
		self.lastReleaseData = dto.ReleaseAsset{}
	}
	return nil
}

func (self *UpgradeService) EventEmitter(ctx context.Context, data dto.ReleaseAsset) error {
	_, err := self.broadcaster.BroadcastMessage(data)
	if err != nil {
		slog.Error("Error broadcasting update message: %w", "err", err)
		return errors.WithStack(err)
	}
	return nil
}

func (self *UpgradeService) GetLastReleaseData() *dto.ReleaseAsset {
	return &self.lastReleaseData
}
