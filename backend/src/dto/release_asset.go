package dto

import (
	"github.com/google/go-github/v68/github"
)

type ReleaseAsset struct {
	UpdateStatus int8                      `json:"update_status"`
	LastRelease  *github.RepositoryRelease `json:"last_release,omitempty"`
	ArchAsset    *github.ReleaseAsset      `json:"arch,omitempty"`
}
