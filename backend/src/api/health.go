package api

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v68/github"
	"github.com/ztrue/tracerr"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"
)

type HealthHanler struct {
	ctx                    context.Context
	OutputEventsCount      uint64
	OutputEventsInterleave time.Duration
	dto.HealthPing
	gh            *github.Client
	updateChannel dto.UpdateChannel
	broadcaster   service.BroadcasterServiceInterface
}

func NewHealthHandler(ctx context.Context, apictx *ContextState, broadcaster service.BroadcasterServiceInterface) *HealthHanler {

	p := new(HealthHanler)
	p.Alive = true
	p.AliveTime = time.Now()
	p.ReadOnly = apictx.ReadOnlyMode
	p.SambaProcessStatus.Pid = -1
	p.LastError = ""
	p.ctx = ctx
	p.broadcaster = broadcaster
	p.OutputEventsCount = 0
	if apictx.Heartbeat > 0 {
		p.OutputEventsInterleave = time.Duration(apictx.Heartbeat) * time.Second
	} else {
		p.OutputEventsInterleave = 5 * time.Second
	}
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil)
	if err != nil {
		panic(tracerr.Errorf("Error: %v\n", err))
	}
	p.gh = github.NewClient(rateLimiter)
	var properties dbom.Properties
	properties.Load()
	value, err := properties.GetValue("UpdateChannel")
	if err != nil {
		log.Printf("Error: %v\n", err)
		return p
	}
	p.updateChannel = dto.UpdateChannel(value.(string))
	go p.run()
	return p
}

func (broker *HealthHanler) Patterns() []server.RouteDetail {
	return []server.RouteDetail{
		{Pattern: "/healt", Method: "GET", Handler: broker.HealthCheckHandler},
	}
}

// HealthCheckHandler godoc
//
//	@Summary		HealthCheck
//	@Description	HealthCheck
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	dto.HealthPing
//	@Failure		405	{object}	ErrorResponse
//	@Router			/health [get]
func (self *HealthHanler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	HttpJSONReponse(w, self, nil)
}

func (self *HealthHanler) EventEmitter(ctx context.Context, data dto.HealthPing) error {
	msg := dto.EventMessageEnvelope{
		Event: dto.EventHeartbeat,
		Data:  data,
	}
	_, err := self.broadcaster.BroadcastMessage(&msg)
	if err != nil {
		slog.Error("Error broadcasting health message: %w", "err", err)
		return tracerr.Wrap(err)
	}
	self.OutputEventsCount++
	return nil
}

func (self *HealthHanler) checkSoftwareVersion() error {
	if self.updateChannel != dto.None {
		UpdateLimiter.Do(func() {
			slog.Debug("Checking for updates...", "channel", self.updateChannel)
			releases, _, err := self.gh.Repositories.ListReleases(context.Background(), "dianlight", "srat", &github.ListOptions{
				Page:    1,
				PerPage: 5,
			})
			if err != nil {
				if _, ok := err.(*github.RateLimitError); ok {
					slog.Warn("Github API hit rate limit")
				}
			} else if len(releases) > 0 {
				for _, release := range releases {
					//log.Println(pretty.Sprintf("%v\n", release))
					if *release.Prerelease && self.updateChannel == dto.Stable {
						//log.Printf("Skip Prerelease %s", *release.TagName)
						continue
					} else if !*release.Prerelease && self.updateChannel == dto.Prerelease {
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
			} else {
				slog.Debug("No Releases found")
				lastReleaseData = &dto.ReleaseAsset{
					UpdateStatus: -1,
				}
			}
		})
	} else {
		lastReleaseData = &dto.ReleaseAsset{
			UpdateStatus: -1,
		}
	}
	return nil
}

func (self *HealthHanler) checkSamba() {
	sambaProcess, err := GetSambaProcess()
	if err == nil && sambaProcess != nil {
		var conv converter.ProcessToDtoImpl
		conv.ProcessToSambaProcessStatus(sambaProcess, &self.HealthPing.SambaProcessStatus)
	} else {
		healthData.SambaProcessStatus.Pid = -1
	}
}

func (self *HealthHanler) run() error {
	for {
		select {
		case <-self.ctx.Done():
			slog.Debug("Run process closed", "err", self.ctx.Err())
			return tracerr.Wrap(self.ctx.Err())
		default:
			// FIXME: Implement background process to retrieve all health information
			slog.Debug("Richiesto aggiornamento per Healthy")
			self.checkSoftwareVersion()
			self.checkSamba()
			err := self.EventEmitter(self.ctx, self.HealthPing)
			if err != nil {
				slog.Error("Error emitting health message: %w", "err", err)
			}
			time.Sleep(self.OutputEventsInterleave)
		}
	}
}
