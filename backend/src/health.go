package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/data"
	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v68/github"
	"golang.org/x/time/rate"
)

type Health struct {
	Alive       bool   `json:"alive"`
	ReadOnly    bool   `json:"read_only"`
	Samba       int32  `json:"samba_pid"`
	LastRelease string `json:"last_release"`
}

var ctx = context.Background()
var healthData = &Health{
	Alive:       true,
	ReadOnly:    true,
	Samba:       -1,
	LastRelease: "",
}

var UpdateLimiter = rate.Sometimes{Interval: 30 * time.Minute}

func HealthDataRefeshHandlers() {
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
						healthData.LastRelease = *release.TagName
						break
					}
					log.Printf("Latest %s version is %s", data.Config.UpdateChannel, healthData.LastRelease)
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
			Event: "heartbeat",
			Uid:   request.Uid,
			Data:  healthData,
		}
		c <- &message
		time.Sleep(5 * time.Second)
	}
}
