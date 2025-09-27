package converter

import (
	"github.com/dianlight/srat/dto"
	"github.com/google/go-github/v75/github"
)

// goverter:converter
// goverter:output:file ./github_to_dto_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:default:update
type GitHubToDto interface {
	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:map LastRelease.TagName LastRelease
	// goverter:ignore LastRelease ArchAsset
	RepositoryReleaseToReleaseAsset(source *github.RepositoryRelease, target *dto.ReleaseAsset) error

	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	ReleaseAssetToBinaryAsset(source *github.ReleaseAsset, target *dto.BinaryAsset) error
}
